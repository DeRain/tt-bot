package bot

import (
	"context"
	"strings"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/home/tt-bot/internal/qbt"
)

// ---------------------------------------------------------------------------
// Callback helper
// ---------------------------------------------------------------------------

func newCallbackUpdate(chatID int64, callbackID, data string) tgbotapi.Update {
	return tgbotapi.Update{
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID: callbackID,
			From: &tgbotapi.User{ID: chatID},
			Message: &tgbotapi.Message{
				MessageID: 42,
				Chat:      &tgbotapi.Chat{ID: chatID},
			},
			Data: data,
		},
	}
}

// editTexts returns the text of all EditMessageTextConfig messages sent.
func (m *mockSender) editTexts() []string {
	var texts []string
	for _, msg := range m.sentMessages {
		if em, ok := msg.(tgbotapi.EditMessageTextConfig); ok {
			texts = append(texts, em.Text)
		}
	}
	return texts
}

// hasEditText reports whether any edited message contains sub.
func (m *mockSender) hasEditText(sub string) bool {
	for _, t := range m.editTexts() {
		if strings.Contains(t, sub) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestCallback_CategoryWithPendingMagnet_CallsAddMagnet(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	magnet := "magnet:?xt=urn:btih:abc123"
	h.storePending(1, &PendingTorrent{MagnetLink: magnet, CreatedAt: time.Now()})

	update := newCallbackUpdate(1, "cb1", "cat:Movies")
	h.HandleUpdate(context.Background(), update)

	if len(qbtClient.magnets) != 1 {
		t.Fatalf("expected 1 magnet added, got %d", len(qbtClient.magnets))
	}
	if qbtClient.magnets[0] != magnet {
		t.Errorf("expected magnet %q, got %q", magnet, qbtClient.magnets[0])
	}

	if !sender.hasEditText("Movies") {
		t.Fatalf("expected confirmation message with category, got edits: %v", sender.editTexts())
	}
}

func TestCallback_CategoryWithNoPending_ReturnsError(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	// No pending torrent stored.
	update := newCallbackUpdate(1, "cb2", "cat:Movies")
	h.HandleUpdate(context.Background(), update)

	// AddMagnet should not have been called.
	if len(qbtClient.magnets) != 0 {
		t.Fatalf("expected 0 magnets added, got %d", len(qbtClient.magnets))
	}

	// A callback answer should have been sent with an error message.
	found := false
	for _, msg := range sender.sentMessages {
		if ca, ok := msg.(tgbotapi.CallbackConfig); ok {
			if strings.Contains(ca.Text, "No pending torrent") {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("expected 'No pending torrent' error in callback answer")
	}
}

func TestCallback_PaginationAll_FetchesCorrectPage(t *testing.T) {
	sender := &mockSender{}
	// Create 7 torrents so we have 2 pages (5+2).
	torrents := make([]qbt.Torrent, 7)
	for i := range torrents {
		torrents[i] = qbt.Torrent{Hash: "h" + string(rune('0'+i)), Name: "Torrent " + string(rune('A'+i))}
	}
	qbtClient := &mockQBTClient{torrents: torrents}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	update := newCallbackUpdate(1, "cb3", "pg:all:2")
	h.HandleUpdate(context.Background(), update)

	if !sender.hasEditText("page 2/2") {
		t.Fatalf("expected page 2/2 in edited message, got edits: %v", sender.editTexts())
	}
}

func TestCallback_PaginationActive_FetchesCorrectPage(t *testing.T) {
	sender := &mockSender{}
	torrents := make([]qbt.Torrent, 6)
	for i := range torrents {
		torrents[i] = qbt.Torrent{Hash: "h" + string(rune('0'+i)), Name: "Active " + string(rune('A'+i))}
	}
	qbtClient := &mockQBTClient{torrents: torrents}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	update := newCallbackUpdate(1, "cb4", "pg:act:2")
	h.HandleUpdate(context.Background(), update)

	if !sender.hasEditText("page 2/2") {
		t.Fatalf("expected page 2/2 in edited message, got edits: %v", sender.editTexts())
	}
}

func TestCallback_Noop_JustAnswers(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	update := newCallbackUpdate(1, "cb5", "noop")
	h.HandleUpdate(context.Background(), update)

	// Should have sent exactly one message: the callback answer.
	if len(sender.sentMessages) != 1 {
		t.Fatalf("expected 1 sent message for noop, got %d", len(sender.sentMessages))
	}
	if _, ok := sender.sentMessages[0].(tgbotapi.CallbackConfig); !ok {
		t.Fatalf("expected CallbackConfig for noop, got %T", sender.sentMessages[0])
	}
}

func TestCallback_CategoryWithNoCategory_ShowsGenericConfirm(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	h.storePending(1, &PendingTorrent{MagnetLink: "magnet:?xt=urn:btih:fff", CreatedAt: time.Now()})

	// "cat:" with empty category name (the "No category" button).
	update := newCallbackUpdate(1, "cb6", "cat:")
	h.HandleUpdate(context.Background(), update)

	if !sender.hasEditText("Torrent added!") {
		t.Fatalf("expected generic confirmation, got edits: %v", sender.editTexts())
	}
}
