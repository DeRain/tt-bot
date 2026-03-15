package bot

import (
	"context"
	"fmt"
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
	// Verify the detail view shows the updated (stopped) state.
	if !sender.hasEditText("Stopped") {
		t.Fatalf("expected stopped state label in detail view after pause, got edits: %v", sender.editTexts())
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
	// Verify the detail view shows the updated (downloading) state.
	if !sender.hasEditText("Downloading") {
		t.Fatalf("expected downloading state label in detail view after resume, got edits: %v", sender.editTexts())
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

// ---------------------------------------------------------------------------
// handleRemoveConfirmCallback tests (TEST-7, AC-2.1, AC-2.2)
// ---------------------------------------------------------------------------

func TestCallback_RemoveConfirm_ShowsConfirmationView(t *testing.T) {
	sender := &mockSender{}
	hash := strings.Repeat("a", 40)
	qbtClient := &mockQBTClient{
		torrents: []qbt.Torrent{
			{Hash: hash, Name: "Ubuntu 24.04", State: "downloading"},
		},
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	update := newCallbackUpdate(1, "cb-rm", "rm:a:1:"+hash)
	h.HandleUpdate(context.Background(), update)

	// AC-2.1: Confirmation view contains torrent name.
	if !sender.hasEditText("Ubuntu 24.04") {
		t.Fatalf("expected torrent name in confirmation view, got edits: %v", sender.editTexts())
	}
	if !sender.hasEditText("Remove") {
		t.Fatalf("expected 'Remove' in confirmation view, got edits: %v", sender.editTexts())
	}

	// AC-2.2: no DeleteTorrents call should have been made (non-destructive).
	if len(qbtClient.deletedHashes) != 0 {
		t.Fatalf("expected no DeleteTorrents call on confirm view, got %v", qbtClient.deletedHashes)
	}
}

func TestCallback_RemoveConfirm_TorrentNotFound_NavigatesToList(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{torrents: []qbt.Torrent{}}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	hash := strings.Repeat("b", 40)
	update := newCallbackUpdate(1, "cb-rm-nf", "rm:a:1:"+hash)
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
		t.Fatal("expected 'not found' callback answer when torrent missing on remove confirm")
	}
}

func TestCallback_RemoveConfirm_InvalidFormat(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	update := newCallbackUpdate(1, "cb-rm-bad", "rm:invalid")
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
		t.Fatal("expected 'Invalid' callback answer for malformed rm:")
	}
}

// ---------------------------------------------------------------------------
// handleRemoveDeleteCallback tests (TEST-8, AC-3.1, AC-4.1, AC-5.1, AC-5.2)
// ---------------------------------------------------------------------------

func TestCallback_RemoveDelete_NoFiles_CallsDeleteAndNavigatesToList(t *testing.T) {
	sender := &mockSender{}
	hash := strings.Repeat("c", 40)
	qbtClient := &mockQBTClient{
		torrents: []qbt.Torrent{
			{Hash: "other", Name: "Remaining Torrent"},
		},
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	update := newCallbackUpdate(1, "cb-rd", "rd:a:1:"+hash)
	h.HandleUpdate(context.Background(), update)

	// AC-3.1: DeleteTorrents called with deleteFiles=false.
	if len(qbtClient.deletedHashes) != 1 || qbtClient.deletedHashes[0] != hash {
		t.Fatalf("expected DeleteTorrents(%s, false), got hashes=%v", hash, qbtClient.deletedHashes)
	}
	if qbtClient.deletedFiles {
		t.Error("expected deleteFiles=false for rd: callback")
	}

	// AC-5.1: navigates back to the list view.
	if !sender.hasEditText("page 1/") {
		t.Fatalf("expected list view after removal, got edits: %v", sender.editTexts())
	}
}

func TestCallback_RemoveDelete_WithFiles_CallsDeleteWithFilesTrue(t *testing.T) {
	sender := &mockSender{}
	hash := strings.Repeat("d", 40)
	qbtClient := &mockQBTClient{
		torrents: []qbt.Torrent{
			{Hash: hash, Name: "To Remove"},
		},
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	update := newCallbackUpdate(1, "cb-rf", "rf:a:1:"+hash)
	h.HandleUpdate(context.Background(), update)

	// AC-4.1: DeleteTorrents called with deleteFiles=true.
	if len(qbtClient.deletedHashes) != 1 || qbtClient.deletedHashes[0] != hash {
		t.Fatalf("expected DeleteTorrents(%s, true), got hashes=%v", hash, qbtClient.deletedHashes)
	}
	if !qbtClient.deletedFiles {
		t.Error("expected deleteFiles=true for rf: callback")
	}
}

func TestCallback_RemoveDelete_EmptyListAfterDeletion_ShowsEmptyListMessage(t *testing.T) {
	sender := &mockSender{}
	hash := strings.Repeat("e", 40)
	qbtClient := &mockQBTClient{
		torrents: []qbt.Torrent{}, // list is empty after deletion
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	update := newCallbackUpdate(1, "cb-rd-empty", "rd:a:1:"+hash)
	h.HandleUpdate(context.Background(), update)

	// AC-5.2: empty list shows the empty-list message, not an error.
	if !sender.hasEditText("No torrents found") {
		t.Fatalf("expected empty list message after last torrent removed, got edits: %v", sender.editTexts())
	}
}

func TestCallback_RemoveDelete_DeleteError_AnswersWithError(t *testing.T) {
	sender := &mockSender{}
	hash := strings.Repeat("f", 40)
	qbtClient := &mockQBTClient{
		deleteErr: fmt.Errorf("qbt unavailable"),
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	update := newCallbackUpdate(1, "cb-rd-err", "rd:a:1:"+hash)
	h.HandleUpdate(context.Background(), update)

	found := false
	for _, msg := range sender.sentMessages {
		if ca, ok := msg.(tgbotapi.CallbackConfig); ok {
			if strings.Contains(ca.Text, "Error") {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("expected error callback answer when DeleteTorrents fails")
	}
}

// ---------------------------------------------------------------------------
// handleRemoveCancelCallback tests (TEST-9, AC-6.1, AC-6.2)
// ---------------------------------------------------------------------------

func TestCallback_RemoveCancel_ReturnsToDetailView(t *testing.T) {
	sender := &mockSender{}
	hash := strings.Repeat("a", 40)
	qbtClient := &mockQBTClient{
		torrents: []qbt.Torrent{
			{Hash: hash, Name: "Ubuntu 24.04", State: "downloading"},
		},
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	update := newCallbackUpdate(1, "cb-rc", "rc:a:1:"+hash)
	h.HandleUpdate(context.Background(), update)

	// AC-6.1: navigates back to the detail view for the same torrent.
	if !sender.hasEditText("Ubuntu 24.04") {
		t.Fatalf("expected torrent detail view after cancel, got edits: %v", sender.editTexts())
	}
	if !sender.hasEditText("Size:") {
		t.Fatalf("expected detail fields (Size:) after cancel, got edits: %v", sender.editTexts())
	}

	// AC-6.2: no DeleteTorrents call should have been made.
	if len(qbtClient.deletedHashes) != 0 {
		t.Fatalf("expected no DeleteTorrents call on cancel, got %v", qbtClient.deletedHashes)
	}
}

func TestCallback_RemoveCancel_TorrentNotFound_NavigatesToList(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{torrents: []qbt.Torrent{}}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	hash := strings.Repeat("b", 40)
	update := newCallbackUpdate(1, "cb-rc-nf", "rc:a:1:"+hash)
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
		t.Fatal("expected 'not found' callback answer when torrent missing on cancel")
	}
}

// ---------------------------------------------------------------------------
// Callback routing tests (TEST-10, AC-1.1, AC-2.1, AC-5.1, AC-6.1)
// ---------------------------------------------------------------------------

func TestCallback_RemovePrefixesRoutedCorrectly(t *testing.T) {
	hash := strings.Repeat("a", 40)

	tests := []struct {
		name     string
		data     string
		wantEdit string
	}{
		{
			name:     "rm: shows confirmation",
			data:     "rm:a:1:" + hash,
			wantEdit: "Remove torrent?",
		},
		{
			name:     "rc: returns to detail",
			data:     "rc:a:1:" + hash,
			wantEdit: "Size:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := &mockSender{}
			qbtClient := &mockQBTClient{
				torrents: []qbt.Torrent{
					{Hash: hash, Name: "Test Torrent", State: "downloading", Size: 1024},
				},
			}
			auth := NewAuthorizer([]int64{1})
			h := New(context.Background(), sender, qbtClient, auth, "test-token")

			update := newCallbackUpdate(1, "cb-route", tt.data)
			h.HandleUpdate(context.Background(), update)

			if !sender.hasEditText(tt.wantEdit) {
				t.Errorf("data=%q: expected edit containing %q, got edits: %v", tt.data, tt.wantEdit, sender.editTexts())
			}
		})
	}
}

func TestCallback_RemoveDelete_Routing_BothPrefixes(t *testing.T) {
	hash := strings.Repeat("d", 40)

	tests := []struct {
		name            string
		data            string
		wantDeleteFiles bool
	}{
		{"rd: deleteFiles=false", "rd:a:1:" + hash, false},
		{"rf: deleteFiles=true", "rf:a:1:" + hash, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := &mockSender{}
			qbtClient := &mockQBTClient{
				torrents: []qbt.Torrent{},
			}
			auth := NewAuthorizer([]int64{1})
			h := New(context.Background(), sender, qbtClient, auth, "test-token")

			update := newCallbackUpdate(1, "cb-del-route", tt.data)
			h.HandleUpdate(context.Background(), update)

			if len(qbtClient.deletedHashes) != 1 {
				t.Fatalf("expected 1 delete call, got %d", len(qbtClient.deletedHashes))
			}
			if qbtClient.deletedFiles != tt.wantDeleteFiles {
				t.Errorf("deleteFiles = %v, want %v", qbtClient.deletedFiles, tt.wantDeleteFiles)
			}
		})
	}
}

// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// parseFileSelectCallback validation tests (input validation)
// ---------------------------------------------------------------------------

func TestParseFileSelectCallback_NegativeFileIndex_ReturnsError(t *testing.T) {
	hash := strings.Repeat("a", 40)
	// fs:<hash>:<fileIndex>:<filePage>:<filterChar>:<listPage>
	data := hash + ":-1:1:a:1"
	_, _, _, _, _, err := parseFileSelectCallback(data)
	if err == nil {
		t.Fatal("expected error for negative fileIndex, got nil")
	}
}

func TestParseFileSelectCallback_ValidData_ReturnsNoError(t *testing.T) {
	hash := strings.Repeat("a", 40)
	data := hash + ":5:2:a:1"
	gotHash, fileIndex, filePage, filterChar, listPage, err := parseFileSelectCallback(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotHash != hash {
		t.Errorf("hash = %q, want %q", gotHash, hash)
	}
	if fileIndex != 5 {
		t.Errorf("fileIndex = %d, want 5", fileIndex)
	}
	if filePage != 2 {
		t.Errorf("filePage = %d, want 2", filePage)
	}
	if filterChar != "a" {
		t.Errorf("filterChar = %q, want 'a'", filterChar)
	}
	if listPage != 1 {
		t.Errorf("listPage = %d, want 1", listPage)
	}
}

func TestCallback_FileSelect_NegativeFileIndex_AnswersInvalid(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	hash := strings.Repeat("a", 40)
	// Negative fileIndex: fs:<hash>:-1:<filePage>:<filterChar>:<listPage>
	update := newCallbackUpdate(1, "cb-fs-neg", "fs:"+hash+":-1:1:a:1")
	h.HandleUpdate(context.Background(), update)

	found := false
	for _, msg := range sender.sentMessages {
		if ca, ok := msg.(tgbotapi.CallbackConfig); ok && strings.Contains(ca.Text, "Invalid") {
			found = true
		}
	}
	if !found {
		t.Fatal("expected 'Invalid' callback answer for negative fileIndex in fs:")
	}
}

// ---------------------------------------------------------------------------
// parseFilePriorityCallback validation tests (input validation)
// ---------------------------------------------------------------------------

func TestParseFilePriorityCallback_NegativeFileIndex_ReturnsError(t *testing.T) {
	hash := strings.Repeat("b", 40)
	// fp:<hash>:<fileIndex>:<priority>:<filePage>:<filterChar>:<listPage>
	data := hash + ":-1:1:1:a:1"
	_, _, _, _, _, _, err := parseFilePriorityCallback(data)
	if err == nil {
		t.Fatal("expected error for negative fileIndex, got nil")
	}
}

func TestParseFilePriorityCallback_ValidData_ReturnsNoError(t *testing.T) {
	hash := strings.Repeat("b", 40)
	data := hash + ":3:6:2:a:1"
	gotHash, fileIndex, priority, filePage, filterChar, listPage, err := parseFilePriorityCallback(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotHash != hash {
		t.Errorf("hash = %q, want %q", gotHash, hash)
	}
	if fileIndex != 3 {
		t.Errorf("fileIndex = %d, want 3", fileIndex)
	}
	if priority != 6 {
		t.Errorf("priority = %d, want 6", priority)
	}
	if filePage != 2 {
		t.Errorf("filePage = %d, want 2", filePage)
	}
	if filterChar != "a" {
		t.Errorf("filterChar = %q, want 'a'", filterChar)
	}
	if listPage != 1 {
		t.Errorf("listPage = %d, want 1", listPage)
	}
}

// ---------------------------------------------------------------------------
// isValidFilePriority tests (input validation)
// ---------------------------------------------------------------------------

func TestIsValidFilePriority(t *testing.T) {
	cases := []struct {
		p    int
		want bool
	}{
		{0, true},  // FilePrioritySkip
		{1, true},  // FilePriorityNormal
		{6, true},  // FilePriorityHigh
		{7, true},  // FilePriorityMaximum
		{2, false}, // invalid
		{5, false}, // invalid
		{-1, false},
		{8, false},
	}
	for _, c := range cases {
		if got := isValidFilePriority(c.p); got != c.want {
			t.Errorf("isValidFilePriority(%d) = %v, want %v", c.p, got, c.want)
		}
	}
}

// ---------------------------------------------------------------------------
// handleFilePriorityCallback validation tests (input validation)
// ---------------------------------------------------------------------------

func TestCallback_FilePriority_InvalidPriority_AnswersInvalidPriority(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	hash := strings.Repeat("c", 40)
	// priority=5 is invalid (not in {0,1,6,7})
	// fp:<hash>:<fileIndex>:<priority>:<filePage>:<filterChar>:<listPage>
	update := newCallbackUpdate(1, "cb-fp-badpri", "fp:"+hash+":0:5:1:a:1")
	h.HandleUpdate(context.Background(), update)

	found := false
	for _, msg := range sender.sentMessages {
		if ca, ok := msg.(tgbotapi.CallbackConfig); ok && strings.Contains(ca.Text, "Invalid priority") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected 'Invalid priority' callback answer, got messages: %v", sender.sentMessages)
	}
	// SetFilePriority must NOT have been called.
	if len(qbtClient.setPriorityRecords) != 0 {
		t.Fatalf("expected SetFilePriority not called, got %d calls", len(qbtClient.setPriorityRecords))
	}
}

func TestCallback_FilePriority_NegativeFileIndex_AnswersInvalid(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	hash := strings.Repeat("c", 40)
	// fp:<hash>:-1:<priority>:<filePage>:<filterChar>:<listPage>
	update := newCallbackUpdate(1, "cb-fp-negidx", "fp:"+hash+":-1:1:1:a:1")
	h.HandleUpdate(context.Background(), update)

	found := false
	for _, msg := range sender.sentMessages {
		if ca, ok := msg.(tgbotapi.CallbackConfig); ok && strings.Contains(ca.Text, "Invalid") {
			found = true
		}
	}
	if !found {
		t.Fatal("expected 'Invalid' callback answer for negative fileIndex in fp:")
	}
	if len(qbtClient.setPriorityRecords) != 0 {
		t.Fatalf("expected SetFilePriority not called, got %d calls", len(qbtClient.setPriorityRecords))
	}
}

func TestCallback_FilePriority_ValidCall_SetsFilePriority(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	hash := strings.Repeat("d", 40)
	// priority=6 (FilePriorityHigh), fileIndex=2
	update := newCallbackUpdate(1, "cb-fp-ok", "fp:"+hash+":2:6:1:a:1")
	h.HandleUpdate(context.Background(), update)

	if len(qbtClient.setPriorityRecords) != 1 {
		t.Fatalf("expected 1 SetFilePriority call, got %d", len(qbtClient.setPriorityRecords))
	}
	rec := qbtClient.setPriorityRecords[0]
	if rec.hash != hash {
		t.Errorf("hash = %q, want %q", rec.hash, hash)
	}
	if len(rec.indices) != 1 || rec.indices[0] != 2 {
		t.Errorf("indices = %v, want [2]", rec.indices)
	}
	if rec.priority != 6 {
		t.Errorf("priority = %d, want 6", rec.priority)
	}
}

// ---------------------------------------------------------------------------
// TestDetailKeyboardFilesButton (TEST-5, AC-5.1)
// ---------------------------------------------------------------------------

// TestDetailKeyboardFilesButton verifies that the detail keyboard returned by
// formatter.TorrentDetailKeyboard contains a "Files" button with a fl: callback.
// This test is in callback_test.go because it exercises the formatter via the
// handler, satisfying the TEST-5 target in verification.md.
func TestDetailKeyboardFilesButton(t *testing.T) {
	sender := &mockSender{}
	hash := strings.Repeat("a", 40)
	qbtClient := &mockQBTClient{
		torrents: []qbt.Torrent{
			{Hash: hash, Name: "Ubuntu 24.04", State: "downloading"},
		},
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	// Trigger detail view via sel: callback.
	update := newCallbackUpdate(1, "cb-detail-files", "sel:a:1:"+hash)
	h.HandleUpdate(context.Background(), update)

	// The edited message must have an inline keyboard with a fl: button.
	found := false
	for _, msg := range sender.sentMessages {
		if em, ok := msg.(tgbotapi.EditMessageTextConfig); ok {
			if em.ReplyMarkup == nil {
				continue
			}
			for _, row := range em.ReplyMarkup.InlineKeyboard {
				for _, btn := range row {
					if btn.CallbackData != nil && strings.HasPrefix(*btn.CallbackData, "fl:") {
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

// ---------------------------------------------------------------------------
// fl: callback tests (TEST-6, AC-1.1, AC-1.2, AC-1.3)
// ---------------------------------------------------------------------------

// TestCallbackFL_ShowsFileList verifies that fl:<filterChar>:<listPage>:<hash>
// calls ListFiles and edits the message with file list content (AC-1.1, AC-1.2).
func TestCallbackFL_ShowsFileList(t *testing.T) {
	sender := &mockSender{}
	hash := strings.Repeat("a", 40)
	qbtClient := &mockQBTClient{
		torrents: []qbt.Torrent{
			{Hash: hash, Name: "My Show"},
		},
		torrentFiles: map[string][]qbt.TorrentFile{
			hash: {
				{Index: 0, Name: "Season 1/ep01.mkv", Size: 1 << 30, Progress: 0.5, Priority: qbt.FilePriorityNormal},
			},
		},
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	// fl:<filterChar>:<listPage>:<hash>
	update := newCallbackUpdate(1, "cb-fl", "fl:a:1:"+hash)
	h.HandleUpdate(context.Background(), update)

	// AC-1.2: torrent name appears in the file list header.
	if !sender.hasEditText("My Show") {
		t.Fatalf("expected torrent name in file list header, got edits: %v", sender.editTexts())
	}
	// AC-1.1: file name (last component) appears.
	if !sender.hasEditText("ep01.mkv") {
		t.Fatalf("expected file name in file list, got edits: %v", sender.editTexts())
	}
}

// TestCallbackFL_ListFilesError verifies that when ListFiles returns an error
// the bot answers with a user-friendly message and does not crash (AC-1.3).
func TestCallbackFL_ListFilesError(t *testing.T) {
	sender := &mockSender{}
	hash := strings.Repeat("b", 40)
	qbtClient := &mockQBTClient{
		torrents:     []qbt.Torrent{{Hash: hash, Name: "Broken"}},
		listFilesErr: fmt.Errorf("qbt unavailable"),
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	update := newCallbackUpdate(1, "cb-fl-err", "fl:a:1:"+hash)
	h.HandleUpdate(context.Background(), update)

	found := false
	for _, msg := range sender.sentMessages {
		if ca, ok := msg.(tgbotapi.CallbackConfig); ok && strings.Contains(ca.Text, "Failed to load files") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected 'Failed to load files' answer on ListFiles error, got: %v", sender.sentMessages)
	}
}

// ---------------------------------------------------------------------------
// pg:fl: callback tests (TEST-6, AC-3.2)
// ---------------------------------------------------------------------------

// TestCallbackPgFL_NavigatesToCorrectPage verifies that pg:fl: calls ListFiles
// and renders the requested file page (AC-3.2).
func TestCallbackPgFL_NavigatesToCorrectPage(t *testing.T) {
	sender := &mockSender{}
	hash := strings.Repeat("c", 40)

	// 8 files so we have 2 pages (5 + 3).
	files := make([]qbt.TorrentFile, 8)
	for i := range files {
		files[i] = qbt.TorrentFile{
			Index:    i,
			Name:     fmt.Sprintf("file%02d.mkv", i),
			Size:     1 << 20,
			Progress: 0.0,
			Priority: qbt.FilePriorityNormal,
		}
	}
	qbtClient := &mockQBTClient{
		torrents:     []qbt.Torrent{{Hash: hash, Name: "Show"}},
		torrentFiles: map[string][]qbt.TorrentFile{hash: files},
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	// pg:fl:<hash>:<filePage>:<filterChar>:<listPage>
	update := newCallbackUpdate(1, "cb-pgfl", "pg:fl:"+hash+":2:a:1")
	h.HandleUpdate(context.Background(), update)

	// Page 2 header should contain "Page 2/2".
	if !sender.hasEditText("Page 2/2") {
		t.Fatalf("expected 'Page 2/2' in file list, got edits: %v", sender.editTexts())
	}
}

// ---------------------------------------------------------------------------
// fs: callback tests (TEST-6, AC-4.1, AC-4.3)
// ---------------------------------------------------------------------------

// TestCallbackFS_ShowsPriorityKeyboard verifies that fs: edits the message to
// show a priority selection keyboard without calling SetFilePriority (AC-4.1).
func TestCallbackFS_ShowsPriorityKeyboard(t *testing.T) {
	sender := &mockSender{}
	hash := strings.Repeat("d", 40)
	qbtClient := &mockQBTClient{
		torrentFiles: map[string][]qbt.TorrentFile{
			hash: {
				{Index: 0, Name: "file.mkv", Size: 1 << 20, Progress: 0.5, Priority: qbt.FilePriorityNormal},
			},
		},
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	// fs:<hash>:<fileIndex>:<filePage>:<filterChar>:<listPage>
	update := newCallbackUpdate(1, "cb-fs", "fs:"+hash+":0:1:a:1")
	h.HandleUpdate(context.Background(), update)

	// Must not have called SetFilePriority.
	if len(qbtClient.setPriorityRecords) != 0 {
		t.Fatalf("fs: must not call SetFilePriority, got %d calls", len(qbtClient.setPriorityRecords))
	}
	// Must edit the message (showing priority selection).
	if len(sender.editTexts()) == 0 {
		t.Fatal("expected message edit to show priority keyboard")
	}
}

// ---------------------------------------------------------------------------
// fp: callback with re-render (TEST-6, AC-4.2, AC-4.4)
// ---------------------------------------------------------------------------

// TestCallbackFP_SetsFilePriorityAndRefreshes verifies that a valid fp: callback
// calls SetFilePriority and then re-fetches the file list (AC-4.2).
func TestCallbackFP_SetsFilePriorityAndRefreshes(t *testing.T) {
	sender := &mockSender{}
	hash := strings.Repeat("e", 40)
	qbtClient := &mockQBTClient{
		torrents: []qbt.Torrent{{Hash: hash, Name: "Show"}},
		torrentFiles: map[string][]qbt.TorrentFile{
			hash: {
				{Index: 0, Name: "ep01.mkv", Size: 1 << 20, Progress: 0.8, Priority: qbt.FilePriorityNormal},
			},
		},
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	// fp:<hash>:<fileIndex>:<priority>:<filePage>:<filterChar>:<listPage>
	// Set fileIndex=0 to priority=0 (Skip).
	update := newCallbackUpdate(1, "cb-fp-refresh", "fp:"+hash+":0:0:1:a:1")
	h.HandleUpdate(context.Background(), update)

	if len(qbtClient.setPriorityRecords) != 1 {
		t.Fatalf("expected 1 SetFilePriority call, got %d", len(qbtClient.setPriorityRecords))
	}
	if qbtClient.setPriorityRecords[0].priority != qbt.FilePrioritySkip {
		t.Errorf("priority = %d, want 0 (Skip)", qbtClient.setPriorityRecords[0].priority)
	}
	// Must edit message to show refreshed file list.
	if !sender.hasEditText("ep01.mkv") {
		t.Fatalf("expected file list refresh after fp:, got edits: %v", sender.editTexts())
	}
}

// TestCallbackFP_SetPriorityError_AnswersWithError verifies that when
// SetFilePriority returns an error the bot answers with a user-friendly
// message (AC-4.4).
func TestCallbackFP_SetPriorityError_AnswersWithError(t *testing.T) {
	sender := &mockSender{}
	hash := strings.Repeat("f", 40)
	qbtClient := &mockQBTClient{
		setFilePriorityErr: fmt.Errorf("qbt unavailable"),
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	update := newCallbackUpdate(1, "cb-fp-err", "fp:"+hash+":0:1:1:a:1")
	h.HandleUpdate(context.Background(), update)

	found := false
	for _, msg := range sender.sentMessages {
		if ca, ok := msg.(tgbotapi.CallbackConfig); ok && strings.Contains(ca.Text, "Failed to set priority") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected 'Failed to set priority' answer on SetFilePriority error, got: %v", sender.sentMessages)
	}
}

// ---------------------------------------------------------------------------
// bk:fl: callback tests (TEST-6, AC-5.2)
// ---------------------------------------------------------------------------

// TestCallbackBkFL_ReturnsToDetailView verifies that bk:fl:<filterChar>:<listPage>:<hash>
// edits the message to show the torrent detail view (AC-5.2).
func TestCallbackBkFL_ReturnsToDetailView(t *testing.T) {
	sender := &mockSender{}
	hash := strings.Repeat("a", 40)
	qbtClient := &mockQBTClient{
		torrents: []qbt.Torrent{
			{Hash: hash, Name: "My Torrent", State: "downloading", Size: 2 << 30},
		},
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	// bk:fl:<filterChar>:<listPage>:<hash>
	update := newCallbackUpdate(1, "cb-bkfl", "bk:fl:a:1:"+hash)
	h.HandleUpdate(context.Background(), update)

	if !sender.hasEditText("My Torrent") {
		t.Fatalf("expected torrent detail view after bk:fl:, got edits: %v", sender.editTexts())
	}
	if !sender.hasEditText("Size:") {
		t.Fatalf("expected detail fields (Size:) after bk:fl:, got edits: %v", sender.editTexts())
	}
}

// TestCallbackBkFL_InvalidFormat verifies that a malformed bk:fl: callback is
// answered with an error and does not panic.
func TestCallbackBkFL_InvalidFormat(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	update := newCallbackUpdate(1, "cb-bkfl-bad", "bk:fl:invalid")
	h.HandleUpdate(context.Background(), update)

	found := false
	for _, msg := range sender.sentMessages {
		if ca, ok := msg.(tgbotapi.CallbackConfig); ok && strings.Contains(ca.Text, "Invalid") {
			found = true
		}
	}
	if !found {
		t.Fatal("expected 'Invalid' callback answer for malformed bk:fl:")
	}
}

// ---------------------------------------------------------------------------
// awaitStateChange tests
// ---------------------------------------------------------------------------

func TestAwaitStateChange_DetectsChange(t *testing.T) {
	hash := strings.Repeat("c", 40)
	qbtClient := &mockQBTClient{
		torrents: []qbt.Torrent{
			{Hash: hash, Name: "Test", State: "downloading"},
		},
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), nil, qbtClient, auth, "test-token")

	// Simulate state change after 50ms.
	go func() {
		time.Sleep(50 * time.Millisecond)
		qbtClient.mu.Lock()
		qbtClient.torrents[0].State = "stoppedDL"
		qbtClient.mu.Unlock()
	}()

	torrent, changed := h.awaitStateChange(context.Background(), hash, "downloading")
	if !changed {
		t.Fatal("expected state change to be detected")
	}
	if torrent.State != "stoppedDL" {
		t.Fatalf("expected state stoppedDL, got %s", torrent.State)
	}
}

func TestAwaitStateChange_ContextCanceled(t *testing.T) {
	hash := strings.Repeat("d", 40)
	qbtClient := &mockQBTClient{
		torrents: []qbt.Torrent{
			{Hash: hash, Name: "Test", State: "downloading"},
		},
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), nil, qbtClient, auth, "test-token")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_, changed := h.awaitStateChange(ctx, hash, "downloading")
	if changed {
		t.Fatal("expected no state change on canceled context")
	}
}

func TestAwaitStateChange_TorrentDisappears(t *testing.T) {
	qbtClient := &mockQBTClient{
		torrents: []qbt.Torrent{}, // No torrents.
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), nil, qbtClient, auth, "test-token")

	hash := strings.Repeat("e", 40)
	_, changed := h.awaitStateChange(context.Background(), hash, "downloading")
	if changed {
		t.Fatal("expected no change when torrent is not found")
	}
}
