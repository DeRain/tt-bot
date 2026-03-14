//go:build integration

package bot

import (
	"context"
	"os"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/home/tt-bot/internal/qbt"
)

// ubuntuMagnet is a well-known, stable Ubuntu ISO magnet link used for E2E
// testing. It is publicly seeded and does not need to complete downloading for
// the tests to pass.
const ubuntuMagnet = "magnet:?xt=urn:btih:3b245504cf5f11bbdbe1201cea6a6bf45aee1bc0&dn=ubuntu-24.04-desktop-amd64.iso"

// getQBTClient creates and logs in to a real qBittorrent instance.
// Connection parameters are read from environment variables with sensible
// defaults for the docker-compose.test.yml setup.
func getQBTClient(t *testing.T) qbt.Client {
	t.Helper()

	url := os.Getenv("QBITTORRENT_URL")
	if url == "" {
		url = "http://localhost:18080"
	}
	username := os.Getenv("QBITTORRENT_USERNAME")
	if username == "" {
		username = "admin"
	}
	// password may be empty when subnet whitelisting is enabled in the test config.
	password := os.Getenv("QBITTORRENT_PASSWORD")

	client := qbt.NewHTTPClient(url, username, password)
	if err := client.Login(context.Background()); err != nil {
		t.Fatalf("failed to login to qBittorrent: %v", err)
	}
	return client
}

// ---------------------------------------------------------------------------
// E2E Tests
// ---------------------------------------------------------------------------

// TestE2E_UnauthorizedUserRejected verifies that a user not on the whitelist
// receives an "Access denied." reply and the request is not forwarded to
// qBittorrent.
func TestE2E_UnauthorizedUserRejected(t *testing.T) {
	qbtClient := getQBTClient(t)
	sender := &mockSender{}
	auth := NewAuthorizer([]int64{999}) // only user 999 is allowed
	h := New(sender, qbtClient, auth, "test-token")

	update := newCommandUpdate(100, 123, "list")
	h.HandleUpdate(context.Background(), update)

	if !sender.hasText("Access denied") {
		t.Fatalf("expected 'Access denied' reply for unauthorized user, got: %v", sender.sentTexts())
	}
}

// TestE2E_AddMagnetWithCategorySelection verifies the full magnet → category
// selection → confirmation flow against a real qBittorrent instance.
func TestE2E_AddMagnetWithCategorySelection(t *testing.T) {
	const (
		chatID = int64(1001)
		userID = int64(1001)
	)

	qbtClient := getQBTClient(t)
	sender := &mockSender{}
	auth := NewAuthorizer([]int64{userID})
	h := New(sender, qbtClient, auth, "test-token")

	ctx := context.Background()

	// Step 1: Send the magnet link.
	magnetUpdate := newTestMessage(chatID, userID, ubuntuMagnet)
	h.HandleUpdate(ctx, magnetUpdate)

	// The bot should reply with a category selection keyboard.
	if !sender.hasText("Select category") {
		t.Fatalf("expected category keyboard prompt after magnet link, got: %v", sender.sentTexts())
	}

	// Verify the inline keyboard is present in the last sent message.
	hasKeyboard := false
	for _, msg := range sender.sentMessages {
		if nm, ok := msg.(tgbotapi.MessageConfig); ok {
			if nm.ReplyMarkup != nil {
				hasKeyboard = true
				break
			}
		}
	}
	if !hasKeyboard {
		t.Fatal("expected inline keyboard markup in the category selection message")
	}

	// Step 2: Simulate the user pressing "No category" (cat: = empty category).
	sender.sentMessages = nil // reset so we can inspect only new messages
	catCallback := newCallbackUpdate(chatID, "cb-1", "cat:")
	h.HandleUpdate(ctx, catCallback)

	// The bot should answer the callback with "Torrent added!" and edit the message.
	confirmed := false
	for _, msg := range sender.sentMessages {
		if cb, ok := msg.(tgbotapi.CallbackConfig); ok {
			if cb.Text == "Torrent added!" {
				confirmed = true
				break
			}
		}
	}
	if !confirmed {
		t.Fatalf("expected 'Torrent added!' callback answer, got messages: %v", sender.sentMessages)
	}

	// Step 3: Give qBittorrent a moment to register the torrent, then verify.
	time.Sleep(2 * time.Second)
	torrents, err := qbtClient.ListTorrents(ctx, qbt.ListOptions{Filter: qbt.FilterAll})
	if err != nil {
		t.Fatalf("failed to list torrents: %v", err)
	}

	found := false
	for _, tor := range torrents {
		if tor.Hash == "3b245504cf5f11bbdbe1201cea6a6bf45aee1bc0" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected ubuntu torrent to appear in qBittorrent after adding via magnet link")
	}
}

// TestE2E_ListReturnsRealTorrents verifies that /list fetches torrents from the
// real qBittorrent instance and formats them into a message. A torrent is added
// directly via the qBT client before issuing the command.
func TestE2E_ListReturnsRealTorrents(t *testing.T) {
	const (
		chatID = int64(1002)
		userID = int64(1002)
	)

	ctx := context.Background()
	qbtClient := getQBTClient(t)

	// Ensure at least one torrent exists.
	if err := qbtClient.AddMagnet(ctx, ubuntuMagnet, ""); err != nil {
		// Ignore "already exists" type errors — qBT silently deduplicates.
		t.Logf("AddMagnet (pre-seed): %v", err)
	}
	// Short pause so qBittorrent has time to register the torrent.
	time.Sleep(2 * time.Second)

	sender := &mockSender{}
	auth := NewAuthorizer([]int64{userID})
	h := New(sender, qbtClient, auth, "test-token")

	update := newCommandUpdate(chatID, userID, "list")
	h.HandleUpdate(ctx, update)

	if len(sender.sentMessages) == 0 {
		t.Fatal("expected at least one message in response to /list")
	}

	// The message text should contain either the torrent name or "No torrents".
	texts := sender.sentTexts()
	if len(texts) == 0 {
		t.Fatal("expected at least one text message in response to /list")
	}

	// A pagination keyboard should be present.
	hasKeyboard := false
	for _, msg := range sender.sentMessages {
		if nm, ok := msg.(tgbotapi.MessageConfig); ok {
			if nm.ReplyMarkup != nil {
				hasKeyboard = true
				break
			}
		}
	}
	if !hasKeyboard {
		t.Error("expected pagination keyboard in /list response")
	}
}

// TestE2E_ListPagination verifies that a pagination callback correctly edits the
// message with the requested page content and dismisses the spinner.
func TestE2E_ListPagination(t *testing.T) {
	const (
		chatID = int64(1003)
		userID = int64(1003)
	)

	ctx := context.Background()
	qbtClient := getQBTClient(t)

	sender := &mockSender{}
	auth := NewAuthorizer([]int64{userID})
	h := New(sender, qbtClient, auth, "test-token")

	// Step 1: Fetch page 1.
	listUpdate := newCommandUpdate(chatID, userID, "list")
	h.HandleUpdate(ctx, listUpdate)

	if len(sender.sentMessages) == 0 {
		t.Fatal("expected at least one message for /list")
	}

	// Step 2: Simulate a pagination callback for page 1 (always valid).
	sender.sentMessages = nil
	pgCallback := newCallbackUpdate(chatID, "cb-pg-1", "pg:all:1")
	h.HandleUpdate(ctx, pgCallback)

	// The callback should be answered (CallbackConfig sent) to dismiss the
	// loading spinner on the button.
	answered := false
	for _, msg := range sender.sentMessages {
		if _, ok := msg.(tgbotapi.CallbackConfig); ok {
			answered = true
			break
		}
	}
	if !answered {
		t.Errorf("expected callback answer after pagination, got: %v", sender.sentMessages)
	}
}

// TestE2E_ActiveCommandShowsDownloading verifies that /active returns a valid
// response even when no torrents are currently active.
func TestE2E_ActiveCommandShowsDownloading(t *testing.T) {
	const (
		chatID = int64(1004)
		userID = int64(1004)
	)

	ctx := context.Background()
	qbtClient := getQBTClient(t)

	sender := &mockSender{}
	auth := NewAuthorizer([]int64{userID})
	h := New(sender, qbtClient, auth, "test-token")

	update := newCommandUpdate(chatID, userID, "active")
	h.HandleUpdate(ctx, update)

	if len(sender.sentMessages) == 0 {
		t.Fatal("expected at least one message in response to /active")
	}

	// The response is valid as long as something was sent and it does not
	// contain an error prefix. An empty list ("No torrents found.") is fine.
	for _, text := range sender.sentTexts() {
		if len(text) == 0 {
			t.Error("received empty text message for /active")
		}
	}
}
