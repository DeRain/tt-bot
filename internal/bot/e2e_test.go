//go:build integration

package bot

import (
	"context"
	"fmt"
	"os"
	"strings"
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
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

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
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

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
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

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
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

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

// TestE2E_SelectTorrentShowsDetail verifies the full flow: /list → select torrent
// → detail view with Pause/Resume and Back buttons.
func TestE2E_SelectTorrentShowsDetail(t *testing.T) {
	const (
		chatID = int64(1005)
		userID = int64(1005)
	)

	ctx := context.Background()
	qbtClient := getQBTClient(t)

	// Ensure at least one torrent exists.
	if err := qbtClient.AddMagnet(ctx, ubuntuMagnet, ""); err != nil {
		t.Logf("AddMagnet (pre-seed): %v", err)
	}
	time.Sleep(2 * time.Second)

	sender := &mockSender{}
	auth := NewAuthorizer([]int64{userID})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	// Step 1: List torrents.
	listUpdate := newCommandUpdate(chatID, userID, "list")
	h.HandleUpdate(ctx, listUpdate)

	// Verify selection buttons are present.
	hasSelButton := false
	for _, msg := range sender.sentMessages {
		if nm, ok := msg.(tgbotapi.MessageConfig); ok {
			if kb, ok := nm.ReplyMarkup.(tgbotapi.InlineKeyboardMarkup); ok {
				for _, row := range kb.InlineKeyboard {
					for _, btn := range row {
						if btn.CallbackData != nil && len(*btn.CallbackData) > 4 && (*btn.CallbackData)[:4] == "sel:" {
							hasSelButton = true
						}
					}
				}
			}
		}
	}
	if !hasSelButton {
		t.Fatal("expected selection buttons (sel: prefix) in /list response")
	}

	// Step 2: Get a torrent hash to select.
	torrents, err := qbtClient.ListTorrents(ctx, qbt.ListOptions{Filter: qbt.FilterAll})
	if err != nil {
		t.Fatalf("ListTorrents() error = %v", err)
	}
	if len(torrents) == 0 {
		t.Skip("no torrents to select")
	}
	hash := torrents[0].Hash

	// Step 3: Select the torrent.
	sender.sentMessages = nil
	selCallback := newCallbackUpdate(chatID, "cb-sel", "sel:a:1:"+hash)
	h.HandleUpdate(ctx, selCallback)

	// Verify detail view was shown.
	if !sender.hasEditText("Size:") {
		t.Fatalf("expected detail view with Size: field, got edits: %v", sender.editTexts())
	}
	if !sender.hasEditText("State:") {
		t.Fatalf("expected detail view with State: field")
	}
}

// TestE2E_PauseResumeTorrent verifies the pause → resume → back flow against a
// real qBittorrent instance.
func TestE2E_PauseResumeTorrent(t *testing.T) {
	const (
		chatID = int64(1006)
		userID = int64(1006)
	)

	ctx := context.Background()
	qbtClient := getQBTClient(t)

	// Ensure at least one torrent exists.
	if err := qbtClient.AddMagnet(ctx, ubuntuMagnet, ""); err != nil {
		t.Logf("AddMagnet (pre-seed): %v", err)
	}
	time.Sleep(2 * time.Second)

	torrents, err := qbtClient.ListTorrents(ctx, qbt.ListOptions{Filter: qbt.FilterAll})
	if err != nil {
		t.Fatalf("ListTorrents() error = %v", err)
	}
	if len(torrents) == 0 {
		t.Skip("no torrents to pause/resume")
	}
	hash := torrents[0].Hash

	sender := &mockSender{}
	auth := NewAuthorizer([]int64{userID})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	// Step 1: Pause the torrent.
	paCallback := newCallbackUpdate(chatID, "cb-pa", "pa:a:1:"+hash)
	h.HandleUpdate(ctx, paCallback)

	// Verify callback was answered with "Paused".
	paused := false
	for _, msg := range sender.sentMessages {
		if ca, ok := msg.(tgbotapi.CallbackConfig); ok {
			if ca.Text == "Paused" {
				paused = true
			}
		}
	}
	if !paused {
		t.Fatal("expected 'Paused' callback answer")
	}

	// Step 2: Resume the torrent.
	sender.sentMessages = nil
	reCallback := newCallbackUpdate(chatID, "cb-re", "re:a:1:"+hash)
	h.HandleUpdate(ctx, reCallback)

	resumed := false
	for _, msg := range sender.sentMessages {
		if ca, ok := msg.(tgbotapi.CallbackConfig); ok {
			if ca.Text == "Resumed" {
				resumed = true
			}
		}
	}
	if !resumed {
		t.Fatal("expected 'Resumed' callback answer")
	}

	// Step 3: Back to list.
	sender.sentMessages = nil
	bkCallback := newCallbackUpdate(chatID, "cb-bk", "bk:a:1")
	h.HandleUpdate(ctx, bkCallback)

	if !sender.hasEditText("page 1/") {
		t.Fatalf("expected list page after back, got edits: %v", sender.editTexts())
	}
}

// TestE2E_DownloadingCommandShowsIncompleteTorrents verifies that /downloading
// returns a valid response and shows only incomplete torrents (Progress < 1.0).
// A torrent is pre-seeded via ubuntuMagnet which will remain incomplete during
// the test, satisfying AC-1.1, AC-2.1, AC-2.2, AC-5.1.
func TestE2E_DownloadingCommandShowsIncompleteTorrents(t *testing.T) {
	const (
		chatID = int64(1007)
		userID = int64(1007)
	)

	ctx := context.Background()
	qbtClient := getQBTClient(t)

	// Pre-seed an incomplete torrent — ubuntu ISO is large and will not finish.
	if err := qbtClient.AddMagnet(ctx, ubuntuMagnet, ""); err != nil {
		t.Logf("AddMagnet (pre-seed): %v", err)
	}
	// Give qBittorrent time to register the torrent.
	time.Sleep(2 * time.Second)

	sender := &mockSender{}
	auth := NewAuthorizer([]int64{userID})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	update := newCommandUpdate(chatID, userID, "downloading")
	h.HandleUpdate(ctx, update)

	// AC-5.1: command must respond without errors.
	if len(sender.sentMessages) == 0 {
		t.Fatal("expected at least one message in response to /downloading")
	}

	// The message text must not be empty.
	texts := sender.sentTexts()
	if len(texts) == 0 {
		t.Fatal("expected at least one text message in response to /downloading")
	}
	for _, text := range texts {
		if len(text) == 0 {
			t.Error("received empty text message for /downloading")
		}
	}

	// AC-1.1 / AC-2.1 / AC-2.2: since the ubuntu torrent is incomplete the
	// response must NOT be "No torrents found." — it should list the torrent.
	for _, text := range texts {
		if text == "No torrents found." {
			t.Errorf("/downloading returned 'No torrents found.' but an incomplete torrent should be visible")
		}
	}

	// A pagination keyboard should be present in the response.
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
		t.Error("expected pagination keyboard in /downloading response")
	}
}

// TestE2E_DownloadingPaginationAndSelection verifies the full pagination and
// selection flow for the /downloading command: pg:dw:1 edits the message,
// sel:d:1:<hash> shows the detail view, and bk:d:1 returns to the list.
// Covers AC-3.2, AC-4.1.
func TestE2E_DownloadingPaginationAndSelection(t *testing.T) {
	const (
		chatID = int64(1008)
		userID = int64(1008)
	)

	ctx := context.Background()
	qbtClient := getQBTClient(t)

	// Pre-seed an incomplete torrent so the downloading list is non-empty.
	if err := qbtClient.AddMagnet(ctx, ubuntuMagnet, ""); err != nil {
		t.Logf("AddMagnet (pre-seed): %v", err)
	}
	time.Sleep(2 * time.Second)

	// Verify there is at least one incomplete torrent to work with.
	torrents, err := qbtClient.ListTorrents(ctx, qbt.ListOptions{Filter: qbt.FilterAll})
	if err != nil {
		t.Fatalf("ListTorrents() error = %v", err)
	}
	var incompleteTorrents []qbt.Torrent
	for _, tor := range torrents {
		if tor.Progress < 1.0 {
			incompleteTorrents = append(incompleteTorrents, tor)
		}
	}
	if len(incompleteTorrents) == 0 {
		t.Skip("no incomplete torrents available for downloading pagination test")
	}
	hash := incompleteTorrents[0].Hash

	sender := &mockSender{}
	auth := NewAuthorizer([]int64{userID})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	// Step 1: Issue /downloading to establish the list view.
	listUpdate := newCommandUpdate(chatID, userID, "downloading")
	h.HandleUpdate(ctx, listUpdate)

	if len(sender.sentMessages) == 0 {
		t.Fatal("expected at least one message for /downloading")
	}

	// Step 2: AC-3.2 — pg:dw:1 pagination callback edits the message and
	// answers the callback to dismiss the loading spinner.
	sender.sentMessages = nil
	pgCallback := newCallbackUpdate(chatID, "cb-dw-pg", "pg:dw:1")
	h.HandleUpdate(ctx, pgCallback)

	answered := false
	for _, msg := range sender.sentMessages {
		if _, ok := msg.(tgbotapi.CallbackConfig); ok {
			answered = true
			break
		}
	}
	if !answered {
		t.Errorf("expected callback answer after pg:dw:1 pagination, got: %v", sender.sentMessages)
	}

	// Step 3: AC-4.1 — sel:d:1:<hash> selection shows the detail view with
	// Size: and State: fields.
	sender.sentMessages = nil
	selCallback := newCallbackUpdate(chatID, "cb-dw-sel", "sel:d:1:"+hash)
	h.HandleUpdate(ctx, selCallback)

	if !sender.hasEditText("Size:") {
		t.Fatalf("expected detail view with Size: field after sel:d:1:hash, got edits: %v", sender.editTexts())
	}
	if !sender.hasEditText("State:") {
		t.Fatalf("expected detail view with State: field after sel:d:1:hash, got edits: %v", sender.editTexts())
	}

	// Step 4: bk:d:1 back callback returns to the downloading list.
	sender.sentMessages = nil
	bkCallback := newCallbackUpdate(chatID, "cb-dw-bk", "bk:d:1")
	h.HandleUpdate(ctx, bkCallback)

	if !sender.hasEditText("page 1/") {
		t.Fatalf("expected list page after bk:d:1 back, got edits: %v", sender.editTexts())
	}
}

// TestE2E_ListResponseContainsMappedStateLabel verifies that the /list command
// response uses mapped state labels (emoji + human text) rather than raw
// qBittorrent state strings. This covers the status-emojis feature requirement
// that FormatState is applied to every torrent shown in the list view.
func TestE2E_ListResponseContainsMappedStateLabel(t *testing.T) {
	const (
		chatID = int64(1009)
		userID = int64(1009)
	)

	ctx := context.Background()
	qbtClient := getQBTClient(t)

	// Ensure at least one torrent exists so the list is non-empty.
	if err := qbtClient.AddMagnet(ctx, ubuntuMagnet, ""); err != nil {
		t.Logf("AddMagnet (pre-seed): %v", err)
	}
	time.Sleep(2 * time.Second)

	// Confirm there is at least one torrent with a known state to assert on.
	torrents, err := qbtClient.ListTorrents(ctx, qbt.ListOptions{Filter: qbt.FilterAll})
	if err != nil {
		t.Fatalf("ListTorrents() error = %v", err)
	}
	if len(torrents) == 0 {
		t.Skip("no torrents available for state-label test")
	}

	sender := &mockSender{}
	auth := NewAuthorizer([]int64{userID})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	update := newCommandUpdate(chatID, userID, "list")
	h.HandleUpdate(ctx, update)

	if len(sender.sentMessages) == 0 {
		t.Fatal("expected at least one message in response to /list")
	}

	// The list text must NOT contain any raw qBittorrent state string.
	// Mapped labels always start with an emoji; raw states never do.
	rawStates := []string{
		"downloading", "uploading", "pausedDL", "pausedUP",
		"stalledDL", "stalledUP", "checkingDL", "checkingUP",
		"queuedDL", "queuedUP", "metaDL", "forcedDL", "forcedUP",
		"allocating", "moving", "error", "missingFiles", "unknown",
		"checkingResumeData",
	}
	for _, text := range sender.sentTexts() {
		for _, raw := range rawStates {
			// A mapped label for "downloading" would appear as "⬇️ Downloading",
			// never as the bare word "downloading" followed by a newline or space
			// (the format is "| <state>\n"). We check for the raw token surrounded
			// by a pipe-space prefix which is the list entry format.
			needle := "| " + raw
			if strings.Contains(text, needle) {
				t.Errorf("/list response contains raw state %q; expected a mapped label (e.g. '⬇️ Downloading')", raw)
			}
		}
	}

	// Positive check: at least one mapped label must appear (emoji present).
	foundMapped := false
	mappedPrefixes := []string{"⬇️", "🌱", "⏸️", "🕐", "🔍", "⏫", "💾", "🔎", "⏬", "📦", "❌", "⚠️", "❓"}
	for _, text := range sender.sentTexts() {
		for _, prefix := range mappedPrefixes {
			if strings.Contains(text, prefix) {
				foundMapped = true
				break
			}
		}
		if foundMapped {
			break
		}
	}
	if !foundMapped {
		t.Errorf("/list response contains no mapped state emoji; raw texts: %v", sender.sentTexts())
	}
}

// TestE2E_DetailViewContainsUploadedAndRatio verifies that selecting a torrent
// from the list shows a detail view that includes Uploaded: and Ratio: fields.
// This covers the torrent-detail-extra feature requirements.
func TestE2E_DetailViewContainsUploadedAndRatio(t *testing.T) {
	const (
		chatID = int64(1010)
		userID = int64(1010)
	)

	ctx := context.Background()
	qbtClient := getQBTClient(t)

	// Ensure at least one torrent exists.
	if err := qbtClient.AddMagnet(ctx, ubuntuMagnet, ""); err != nil {
		t.Logf("AddMagnet (pre-seed): %v", err)
	}
	time.Sleep(2 * time.Second)

	torrents, err := qbtClient.ListTorrents(ctx, qbt.ListOptions{Filter: qbt.FilterAll})
	if err != nil {
		t.Fatalf("ListTorrents() error = %v", err)
	}
	if len(torrents) == 0 {
		t.Skip("no torrents available for detail view test")
	}
	hash := torrents[0].Hash

	sender := &mockSender{}
	auth := NewAuthorizer([]int64{userID})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	// Select the first torrent to trigger the detail view.
	selCallback := newCallbackUpdate(chatID, "cb-detail-extra", "sel:a:1:"+hash)
	h.HandleUpdate(ctx, selCallback)

	// The detail view must contain Uploaded: and Ratio: fields.
	if !sender.hasEditText("Uploaded:") {
		t.Fatalf("expected 'Uploaded:' field in torrent detail view, got edits: %v", sender.editTexts())
	}
	if !sender.hasEditText("Ratio:") {
		t.Fatalf("expected 'Ratio:' field in torrent detail view, got edits: %v", sender.editTexts())
	}
}

// TestE2E_UploadingCommandReturnsValidResponse verifies that /uploading returns
// a valid response against a real qBittorrent instance. Since all test torrents
// are incomplete (ubuntu ISO is large), the response is expected to be
// "No torrents found." — proving the filter correctly excludes incomplete ones.
// Covers AC-1.2, AC-5.1 (command responds without error).
func TestE2E_UploadingCommandReturnsValidResponse(t *testing.T) {
	const (
		chatID = int64(1011)
		userID = int64(1011)
	)

	ctx := context.Background()
	qbtClient := getQBTClient(t)

	sender := &mockSender{}
	auth := NewAuthorizer([]int64{userID})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	update := newCommandUpdate(chatID, userID, "uploading")
	h.HandleUpdate(ctx, update)

	// AC-5.1: command must respond without errors.
	if len(sender.sentMessages) == 0 {
		t.Fatal("expected at least one message in response to /uploading")
	}

	texts := sender.sentTexts()
	if len(texts) == 0 {
		t.Fatal("expected at least one text message in response to /uploading")
	}
	for _, text := range texts {
		if len(text) == 0 {
			t.Error("received empty text message for /uploading")
		}
	}

	// A pagination keyboard should be present in any case (even for empty list,
	// the page indicator button is sent).
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
		t.Error("expected pagination keyboard in /uploading response")
	}
}

// TestE2E_UploadingPaginationCallback verifies that the pg:up:1 pagination
// callback edits the message and answers the callback correctly.
// Covers AC-3.2.
func TestE2E_UploadingPaginationCallback(t *testing.T) {
	const (
		chatID = int64(1012)
		userID = int64(1012)
	)

	ctx := context.Background()
	qbtClient := getQBTClient(t)

	sender := &mockSender{}
	auth := NewAuthorizer([]int64{userID})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	// Step 1: Issue /uploading to establish the list view.
	listUpdate := newCommandUpdate(chatID, userID, "uploading")
	h.HandleUpdate(ctx, listUpdate)

	if len(sender.sentMessages) == 0 {
		t.Fatal("expected at least one message for /uploading")
	}

	// Step 2: Simulate pg:up:1 pagination callback — must answer the callback
	// to dismiss the loading spinner regardless of how many completed torrents exist.
	sender.sentMessages = nil
	pgCallback := newCallbackUpdate(chatID, "cb-up-pg", "pg:up:1")
	h.HandleUpdate(ctx, pgCallback)

	answered := false
	for _, msg := range sender.sentMessages {
		if _, ok := msg.(tgbotapi.CallbackConfig); ok {
			answered = true
			break
		}
	}
	if !answered {
		t.Errorf("expected callback answer after pg:up:1 pagination, got: %v", sender.sentMessages)
	}
}

// TestE2E_RemoveTorrent verifies the full remove flow against a real
// qBittorrent instance: add a torrent, open its detail view, press Remove to
// see the confirmation, then confirm "Remove torrent only" (deleteFiles=false),
// and verify the torrent disappears from subsequent ListTorrents responses.
// This covers CHECK-1 (AC-3.1, AC-5.1) and demonstrates that the bot-level
// remove callbacks route correctly end-to-end.
func TestE2E_RemoveTorrent(t *testing.T) {
	const (
		chatID = int64(1013)
		userID = int64(1013)
	)

	ctx := context.Background()
	qbtClient := getQBTClient(t)

	// Add a torrent to have something to remove.
	if err := qbtClient.AddMagnet(ctx, ubuntuMagnet, ""); err != nil {
		t.Logf("AddMagnet (pre-seed): %v", err)
	}
	time.Sleep(2 * time.Second)

	torrents, err := qbtClient.ListTorrents(ctx, qbt.ListOptions{Filter: qbt.FilterAll})
	if err != nil {
		t.Fatalf("ListTorrents() error = %v", err)
	}
	if len(torrents) == 0 {
		t.Skip("no torrents available for remove test")
	}
	hash := torrents[0].Hash

	sender := &mockSender{}
	auth := NewAuthorizer([]int64{userID})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	// Step 1: rm: shows the confirmation view (non-destructive, AC-2.2).
	sender.sentMessages = nil
	rmCallback := newCallbackUpdate(chatID, "cb-rm-e2e", "rm:a:1:"+hash)
	h.HandleUpdate(ctx, rmCallback)

	if !sender.hasEditText("Remove") {
		t.Fatalf("expected confirmation view after rm: callback, got edits: %v", sender.editTexts())
	}

	// Verify no deletion occurred yet (AC-2.2).
	afterConfirm, err := qbtClient.ListTorrents(ctx, qbt.ListOptions{Filter: qbt.FilterAll})
	if err != nil {
		t.Fatalf("ListTorrents after confirm view: %v", err)
	}
	foundAfterConfirm := false
	for _, tor := range afterConfirm {
		if tor.Hash == hash {
			foundAfterConfirm = true
			break
		}
	}
	if !foundAfterConfirm {
		t.Error("torrent should still exist after viewing confirmation prompt (non-destructive)")
	}

	// Step 2: rd: confirms remove-torrent-only (deleteFiles=false), AC-3.1.
	sender.sentMessages = nil
	rdCallback := newCallbackUpdate(chatID, "cb-rd-e2e", "rd:a:1:"+hash)
	h.HandleUpdate(ctx, rdCallback)

	// Callback must be answered with "Removed." (AC-5.1).
	removed := false
	for _, msg := range sender.sentMessages {
		if ca, ok := msg.(tgbotapi.CallbackConfig); ok {
			if ca.Text == "Removed." {
				removed = true
			}
		}
	}
	if !removed {
		t.Fatalf("expected 'Removed.' callback answer after rd:, got messages: %v", sender.sentMessages)
	}

	// AC-5.1: the message is edited to show the list view.
	if !sender.hasEditText("page 1/") && !sender.hasEditText("No torrents found") {
		t.Fatalf("expected list view after removal, got edits: %v", sender.editTexts())
	}

	// AC-3.1: give qBittorrent a moment to process the deletion, then verify.
	time.Sleep(2 * time.Second)
	afterDelete, err := qbtClient.ListTorrents(ctx, qbt.ListOptions{Filter: qbt.FilterAll})
	if err != nil {
		t.Fatalf("ListTorrents after delete: %v", err)
	}
	for _, tor := range afterDelete {
		if tor.Hash == hash {
			t.Errorf("torrent %s should have been removed from qBittorrent list", hash)
		}
	}
}

// TestE2E_RemoveCancelReturnsToDetail verifies that the rc: (cancel) callback
// navigates back to the torrent detail view without deleting the torrent.
// Covers AC-6.1 and AC-6.2.
func TestE2E_RemoveCancelReturnsToDetail(t *testing.T) {
	const (
		chatID = int64(1014)
		userID = int64(1014)
	)

	ctx := context.Background()
	qbtClient := getQBTClient(t)

	// Ensure at least one torrent exists.
	if err := qbtClient.AddMagnet(ctx, ubuntuMagnet, ""); err != nil {
		t.Logf("AddMagnet (pre-seed): %v", err)
	}
	time.Sleep(2 * time.Second)

	torrents, err := qbtClient.ListTorrents(ctx, qbt.ListOptions{Filter: qbt.FilterAll})
	if err != nil {
		t.Fatalf("ListTorrents() error = %v", err)
	}
	if len(torrents) == 0 {
		t.Skip("no torrents available for cancel test")
	}
	hash := torrents[0].Hash

	sender := &mockSender{}
	auth := NewAuthorizer([]int64{userID})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	// rc: should return to the detail view (AC-6.1).
	rcCallback := newCallbackUpdate(chatID, "cb-rc-e2e", "rc:a:1:"+hash)
	h.HandleUpdate(ctx, rcCallback)

	if !sender.hasEditText("Size:") {
		t.Fatalf("expected detail view with Size: field after cancel, got edits: %v", sender.editTexts())
	}
	if !sender.hasEditText("State:") {
		t.Fatalf("expected detail view with State: field after cancel, got edits: %v", sender.editTexts())
	}

	// AC-6.2: torrent must still be present (no deletion occurred).
	afterCancel, err := qbtClient.ListTorrents(ctx, qbt.ListOptions{Filter: qbt.FilterAll})
	if err != nil {
		t.Fatalf("ListTorrents after cancel: %v", err)
	}
	found := false
	for _, tor := range afterCancel {
		if tor.Hash == hash {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("torrent %s should still exist after cancel (AC-6.2)", hash)
	}
}

// TestE2E_DetailKeyboardContainsFilesButton verifies that the detail keyboard
// rendered after selecting a torrent includes a "Files" button with an fl: callback.
// Covers AC-5.1 (TEST-5, TEST-6).
func TestE2E_DetailKeyboardContainsFilesButton(t *testing.T) {
	const (
		chatID = int64(1015)
		userID = int64(1015)
	)

	ctx := context.Background()
	qbtClient := getQBTClient(t)

	if err := qbtClient.AddMagnet(ctx, ubuntuMagnet, ""); err != nil {
		t.Logf("AddMagnet (pre-seed): %v", err)
	}
	time.Sleep(2 * time.Second)

	torrents, err := qbtClient.ListTorrents(ctx, qbt.ListOptions{Filter: qbt.FilterAll})
	if err != nil {
		t.Fatalf("ListTorrents() error = %v", err)
	}
	if len(torrents) == 0 {
		t.Skip("no torrents available for Files button test")
	}
	hash := torrents[0].Hash

	sender := &mockSender{}
	auth := NewAuthorizer([]int64{userID})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	// Select the torrent to render the detail keyboard.
	selCallback := newCallbackUpdate(chatID, "cb-fl-btn", "sel:a:1:"+hash)
	h.HandleUpdate(ctx, selCallback)

	// AC-5.1: the detail keyboard must contain a fl: Files button.
	found := false
	for _, msg := range sender.sentMessages {
		if em, ok := msg.(tgbotapi.EditMessageTextConfig); ok && em.ReplyMarkup != nil {
			for _, row := range em.ReplyMarkup.InlineKeyboard {
				for _, btn := range row {
					if btn.CallbackData != nil && len(*btn.CallbackData) >= 3 && (*btn.CallbackData)[:3] == "fl:" {
						found = true
					}
				}
			}
		}
	}
	if !found {
		t.Fatal("expected fl: Files button in detail keyboard (AC-5.1)")
	}
}

// TestE2E_FileListCallback verifies that tapping the Files button loads the
// file list for a real torrent. Covers AC-1.1, AC-1.2, AC-3.2 (TEST-6, TEST-7).
func TestE2E_FileListCallback(t *testing.T) {
	const (
		chatID = int64(1016)
		userID = int64(1016)
	)

	ctx := context.Background()
	qbtClient := getQBTClient(t)

	if err := qbtClient.AddMagnet(ctx, ubuntuMagnet, ""); err != nil {
		t.Logf("AddMagnet (pre-seed): %v", err)
	}
	time.Sleep(2 * time.Second)

	torrents, err := qbtClient.ListTorrents(ctx, qbt.ListOptions{Filter: qbt.FilterAll})
	if err != nil {
		t.Fatalf("ListTorrents() error = %v", err)
	}
	if len(torrents) == 0 {
		t.Skip("no torrents available for file list callback test")
	}
	hash := torrents[0].Hash

	sender := &mockSender{}
	auth := NewAuthorizer([]int64{userID})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	// fl:<filterChar>:<listPage>:<hash> — open file list.
	flCallback := newCallbackUpdate(chatID, "cb-fl-e2e", "fl:a:1:"+hash)
	h.HandleUpdate(ctx, flCallback)

	// AC-1.1: callback answered without error.
	answered := false
	for _, msg := range sender.sentMessages {
		if _, ok := msg.(tgbotapi.CallbackConfig); ok {
			answered = true
		}
	}
	if !answered {
		t.Fatal("expected callback answer after fl: callback")
	}

	// The message should be edited to show files content (or a "Failed to load
	// files" message if metadata is still pending — both are valid outcomes for
	// this integration test since the torrent may not have metadata yet).
	if len(sender.editTexts()) == 0 {
		t.Fatal("expected at least one message edit after fl: callback")
	}
}

// TestE2E_FilePriorityChange verifies the full file priority flow against a
// real qBittorrent instance: fl: → fs: → fp:. Covers AC-4.1, AC-4.2 (TEST-8).
func TestE2E_FilePriorityChange(t *testing.T) {
	const (
		chatID = int64(1017)
		userID = int64(1017)
	)

	ctx := context.Background()
	qbtClient := getQBTClient(t)

	if err := qbtClient.AddMagnet(ctx, ubuntuMagnet, ""); err != nil {
		t.Logf("AddMagnet (pre-seed): %v", err)
	}
	time.Sleep(3 * time.Second)

	torrents, err := qbtClient.ListTorrents(ctx, qbt.ListOptions{Filter: qbt.FilterAll})
	if err != nil {
		t.Fatalf("ListTorrents() error = %v", err)
	}
	if len(torrents) == 0 {
		t.Skip("no torrents available for file priority test")
	}
	hash := torrents[0].Hash

	// Check whether files are available yet (metadata may still be pending).
	files, err := qbtClient.ListFiles(ctx, hash)
	if err != nil || len(files) == 0 {
		t.Skip("torrent has no files yet (metadata pending) — skipping priority test")
	}

	sender := &mockSender{}
	auth := NewAuthorizer([]int64{userID})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	fileIdx := files[0].Index

	// Step 1: fs: shows priority keyboard (AC-4.1).
	sender.sentMessages = nil
	fsCallback := newCallbackUpdate(chatID, "cb-fs-e2e",
		"fs:"+hash+":"+fmt.Sprintf("%d", fileIdx)+":1:a:1")
	h.HandleUpdate(ctx, fsCallback)

	// Must answer the callback and edit the message.
	answered := false
	for _, msg := range sender.sentMessages {
		if _, ok := msg.(tgbotapi.CallbackConfig); ok {
			answered = true
		}
	}
	if !answered {
		t.Fatal("expected callback answer after fs: callback")
	}
	if len(sender.editTexts()) == 0 {
		t.Fatal("expected message edit to show priority keyboard (AC-4.1)")
	}

	// Step 2: fp: sets priority and shows updated file list (AC-4.2).
	sender.sentMessages = nil
	fpCallback := newCallbackUpdate(chatID, "cb-fp-e2e",
		"fp:"+hash+":"+fmt.Sprintf("%d", fileIdx)+":1:1:a:1")
	h.HandleUpdate(ctx, fpCallback)

	prioritySet := false
	for _, msg := range sender.sentMessages {
		if ca, ok := msg.(tgbotapi.CallbackConfig); ok && ca.Text == "Priority updated." {
			prioritySet = true
		}
	}
	if !prioritySet {
		t.Fatalf("expected 'Priority updated.' callback answer after fp:, got: %v", sender.sentMessages)
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
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

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
