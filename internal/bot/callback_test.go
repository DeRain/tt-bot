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
			ID:   callbackID,
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

// ---------------------------------------------------------------------------
// Filter character mapping tests (TASK-9)
// ---------------------------------------------------------------------------

func TestFilterCharToFilter(t *testing.T) {
	cases := []struct {
		char   string
		want   qbt.TorrentFilter
		wantOK bool
	}{
		{"a", qbt.FilterAll, true},
		{"c", qbt.FilterActive, true},
		{"x", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		got, ok := filterCharToFilter(c.char)
		if ok != c.wantOK || got != c.want {
			t.Errorf("filterCharToFilter(%q) = (%q, %v), want (%q, %v)", c.char, got, ok, c.want, c.wantOK)
		}
	}
}

func TestFilterCharToPrefix(t *testing.T) {
	if p := filterCharToPrefix("a"); p != "all" {
		t.Errorf("filterCharToPrefix('a') = %q, want 'all'", p)
	}
	if p := filterCharToPrefix("c"); p != "act" {
		t.Errorf("filterCharToPrefix('c') = %q, want 'act'", p)
	}
}

func TestFilterToChar(t *testing.T) {
	if c := filterToChar(qbt.FilterAll); c != "a" {
		t.Errorf("filterToChar(FilterAll) = %q, want 'a'", c)
	}
	if c := filterToChar(qbt.FilterActive); c != "c" {
		t.Errorf("filterToChar(FilterActive) = %q, want 'c'", c)
	}
}

// ---------------------------------------------------------------------------
// handleSelectCallback tests (TASK-10)
// ---------------------------------------------------------------------------

func TestCallback_Select_ShowsDetailView(t *testing.T) {
	sender := &mockSender{}
	hash := strings.Repeat("a", 40)
	qbtClient := &mockQBTClient{
		torrents: []qbt.Torrent{
			{Hash: hash, Name: "Ubuntu 24.04", State: "downloading", Size: 2 * 1024 * 1024 * 1024},
		},
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	update := newCallbackUpdate(1, "cb-sel", "sel:a:1:"+hash)
	h.HandleUpdate(context.Background(), update)

	if !sender.hasEditText("Ubuntu 24.04") {
		t.Fatalf("expected torrent name in detail view, got edits: %v", sender.editTexts())
	}
	if !sender.hasEditText("⬇️ Downloading") {
		t.Fatalf("expected mapped state label in detail view")
	}
}

func TestCallback_Select_TorrentNotFound(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{torrents: []qbt.Torrent{}}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	hash := strings.Repeat("d", 40)
	update := newCallbackUpdate(1, "cb-sel-nf", "sel:a:1:"+hash)
	h.HandleUpdate(context.Background(), update)

	found := false
	for _, msg := range sender.sentMessages {
		if ca, ok := msg.(tgbotapi.CallbackConfig); ok {
			if strings.Contains(ca.Text, "not found") {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("expected 'Torrent not found' callback answer")
	}
}

// ---------------------------------------------------------------------------
// handlePauseCallback / handleResumeCallback tests (TASK-11)
// ---------------------------------------------------------------------------

func TestCallback_Pause_CallsPauseAndRefreshes(t *testing.T) {
	sender := &mockSender{}
	hash := strings.Repeat("a", 40)
	qbtClient := &mockQBTClient{
		torrents: []qbt.Torrent{
			{Hash: hash, Name: "Ubuntu", State: "downloading"},
		},
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	update := newCallbackUpdate(1, "cb-pa", "pa:a:1:"+hash)
	h.HandleUpdate(context.Background(), update)

	if len(qbtClient.pausedHashes) != 1 || qbtClient.pausedHashes[0] != hash {
		t.Fatalf("expected PauseTorrents called with %s, got %v", hash, qbtClient.pausedHashes)
	}
	if !sender.hasEditText("Ubuntu") {
		t.Fatalf("expected detail view refresh after pause")
	}
}

func TestCallback_Resume_CallsResumeAndRefreshes(t *testing.T) {
	sender := &mockSender{}
	hash := strings.Repeat("b", 40)
	qbtClient := &mockQBTClient{
		torrents: []qbt.Torrent{
			{Hash: hash, Name: "Fedora", State: "pausedDL"},
		},
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	update := newCallbackUpdate(1, "cb-re", "re:a:1:"+hash)
	h.HandleUpdate(context.Background(), update)

	if len(qbtClient.resumedHashes) != 1 || qbtClient.resumedHashes[0] != hash {
		t.Fatalf("expected ResumeTorrents called with %s, got %v", hash, qbtClient.resumedHashes)
	}
	if !sender.hasEditText("Fedora") {
		t.Fatalf("expected detail view refresh after resume")
	}
}

// ---------------------------------------------------------------------------
// handleBackCallback tests (TASK-12)
// ---------------------------------------------------------------------------

func TestCallback_Back_ReturnsToList(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{
		torrents: []qbt.Torrent{
			{Hash: "h1", Name: "Torrent A"},
			{Hash: "h2", Name: "Torrent B"},
		},
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	update := newCallbackUpdate(1, "cb-bk", "bk:a:1")
	h.HandleUpdate(context.Background(), update)

	if !sender.hasEditText("page 1/1") {
		t.Fatalf("expected page 1/1 in list, got edits: %v", sender.editTexts())
	}
}

func TestCallback_Pause_InvalidFormat(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	update := newCallbackUpdate(1, "cb-pa-bad", "pa:invalid")
	h.HandleUpdate(context.Background(), update)

	found := false
	for _, msg := range sender.sentMessages {
		if ca, ok := msg.(tgbotapi.CallbackConfig); ok {
			if strings.Contains(ca.Text, "Invalid") {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("expected 'Invalid' callback answer for malformed pause")
	}
}

func TestCallback_Select_InvalidFilter(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	hash := strings.Repeat("a", 40)
	update := newCallbackUpdate(1, "cb-sel-bad", "sel:x:1:"+hash)
	h.HandleUpdate(context.Background(), update)

	found := false
	for _, msg := range sender.sentMessages {
		if ca, ok := msg.(tgbotapi.CallbackConfig); ok {
			if strings.Contains(ca.Text, "Invalid filter") {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("expected 'Invalid filter' callback answer")
	}
}

func TestCallback_Back_InvalidFormat(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	update := newCallbackUpdate(1, "cb-bk-bad", "bk:invalid")
	h.HandleUpdate(context.Background(), update)

	found := false
	for _, msg := range sender.sentMessages {
		if ca, ok := msg.(tgbotapi.CallbackConfig); ok {
			if strings.Contains(ca.Text, "Invalid") {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("expected 'Invalid' callback answer for malformed back")
	}
}

// ---------------------------------------------------------------------------
// Selection keyboard integration (TASK-13)
// ---------------------------------------------------------------------------

func TestCallback_PaginationAll_IncludesSelectionKeyboard(t *testing.T) {
	sender := &mockSender{}
	torrents := []qbt.Torrent{
		{Hash: strings.Repeat("a", 40), Name: "Torrent A"},
		{Hash: strings.Repeat("b", 40), Name: "Torrent B"},
	}
	qbtClient := &mockQBTClient{torrents: torrents}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	update := newCallbackUpdate(1, "cb-pg", "pg:all:1")
	h.HandleUpdate(context.Background(), update)

	// Check that the edited message has an inline keyboard with selection buttons.
	found := false
	for _, msg := range sender.sentMessages {
		if em, ok := msg.(tgbotapi.EditMessageTextConfig); ok {
			if em.ReplyMarkup != nil {
				for _, row := range em.ReplyMarkup.InlineKeyboard {
					for _, btn := range row {
						if btn.CallbackData != nil && strings.HasPrefix(*btn.CallbackData, "sel:") {
							found = true
						}
					}
				}
			}
		}
	}
	if !found {
		t.Fatal("expected selection keyboard buttons (sel: prefix) in pagination response")
	}
}

// ---------------------------------------------------------------------------
// FilterDownloading char mapping tests (TASK-2)
// ---------------------------------------------------------------------------

func TestFilterCharToFilter_Downloading(t *testing.T) {
	got, ok := filterCharToFilter("d")
	if !ok {
		t.Fatal("filterCharToFilter('d') should return ok=true")
	}
	if got != qbt.FilterDownloading {
		t.Errorf("filterCharToFilter('d') = %q, want %q", got, qbt.FilterDownloading)
	}
}

func TestFilterCharToPrefix_Downloading(t *testing.T) {
	if p := filterCharToPrefix("d"); p != "dw" {
		t.Errorf("filterCharToPrefix('d') = %q, want 'dw'", p)
	}
}

func TestFilterToChar_Downloading(t *testing.T) {
	if c := filterToChar(qbt.FilterDownloading); c != "d" {
		t.Errorf("filterToChar(FilterDownloading) = %q, want 'd'", c)
	}
}

// ---------------------------------------------------------------------------
// pg:dw: pagination callback (TASK-3)
// ---------------------------------------------------------------------------

func TestCallback_PaginationDownloading_FetchesCorrectPage(t *testing.T) {
	sender := &mockSender{}
	// 6 torrents, all incomplete (Progress < 1.0), so 2 pages (5+1).
	torrents := make([]qbt.Torrent, 6)
	for i := range torrents {
		torrents[i] = qbt.Torrent{
			Hash:     "h" + string(rune('0'+i)),
			Name:     "Downloading " + string(rune('A'+i)),
			Progress: 0.5,
		}
	}
	qbtClient := &mockQBTClient{torrents: torrents}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	update := newCallbackUpdate(1, "cb-dw", "pg:dw:2")
	h.HandleUpdate(context.Background(), update)

	if !sender.hasEditText("page 2/2") {
		t.Fatalf("expected page 2/2 in edited message, got edits: %v", sender.editTexts())
	}
}

// ---------------------------------------------------------------------------
// FilterUploading char mapping tests (TEST-2)
// ---------------------------------------------------------------------------

func TestFilterCharToFilter_Uploading(t *testing.T) {
	got, ok := filterCharToFilter("u")
	if !ok {
		t.Fatal("filterCharToFilter('u') should return ok=true")
	}
	if got != qbt.FilterUploading {
		t.Errorf("filterCharToFilter('u') = %q, want %q", got, qbt.FilterUploading)
	}
}

func TestFilterCharToPrefix_Uploading(t *testing.T) {
	if p := filterCharToPrefix("u"); p != "up" {
		t.Errorf("filterCharToPrefix('u') = %q, want 'up'", p)
	}
}

func TestFilterToChar_Uploading(t *testing.T) {
	if c := filterToChar(qbt.FilterUploading); c != "u" {
		t.Errorf("filterToChar(FilterUploading) = %q, want 'u'", c)
	}
}

// ---------------------------------------------------------------------------
// pg:up: pagination callback (TEST-3)
// ---------------------------------------------------------------------------

func TestCallback_PaginationUploading_FetchesCorrectPage(t *testing.T) {
	sender := &mockSender{}
	// 6 completed torrents (Progress == 1.0), so 2 pages (5+1).
	torrents := make([]qbt.Torrent, 6)
	for i := range torrents {
		torrents[i] = qbt.Torrent{
			Hash:     "h" + string(rune('0'+i)),
			Name:     "Seeding " + string(rune('A'+i)),
			Progress: 1.0,
		}
	}
	qbtClient := &mockQBTClient{torrents: torrents}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	update := newCallbackUpdate(1, "cb-up", "pg:up:2")
	h.HandleUpdate(context.Background(), update)

	if !sender.hasEditText("page 2/2") {
		t.Fatalf("expected page 2/2 in edited message, got edits: %v", sender.editTexts())
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
