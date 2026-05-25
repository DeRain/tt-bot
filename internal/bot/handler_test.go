package bot

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/home/tt-bot/internal/qbt"
)

// ---------------------------------------------------------------------------
// Mock Sender
// ---------------------------------------------------------------------------

// mockSender records every message sent via Send and returns configurable
// GetFile results.
type mockSender struct {
	sentMessages []tgbotapi.Chattable
	fileToReturn tgbotapi.File
	fileErr      error
	requestErr   error
	nextMsgID    int
}

func (m *mockSender) Send(msg tgbotapi.Chattable) (tgbotapi.Message, error) {
	m.sentMessages = append(m.sentMessages, msg)
	m.nextMsgID++
	return tgbotapi.Message{MessageID: m.nextMsgID}, nil
}

func (m *mockSender) Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
	m.sentMessages = append(m.sentMessages, c)
	if m.requestErr != nil {
		return nil, m.requestErr
	}
	return &tgbotapi.APIResponse{Ok: true}, nil
}

func (m *mockSender) GetFile(config tgbotapi.FileConfig) (tgbotapi.File, error) {
	return m.fileToReturn, m.fileErr
}

// sentTexts returns the text of all NewMessage calls recorded by the mock.
func (m *mockSender) sentTexts() []string {
	var texts []string
	for _, msg := range m.sentMessages {
		if nm, ok := msg.(tgbotapi.MessageConfig); ok {
			texts = append(texts, nm.Text)
		}
	}
	return texts
}

// hasText reports whether any sent message contains sub as a substring.
func (m *mockSender) hasText(sub string) bool {
	for _, t := range m.sentTexts() {
		if strings.Contains(t, sub) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Mock qbt.Client
// ---------------------------------------------------------------------------

// mockQBTClient is a minimal in-memory implementation of qbt.Client for tests.
type mockQBTClient struct {
	loginErr          error
	magnets           []string
	files             []string
	torrents          []qbt.Torrent
	categories        []qbt.Category
	addMagnetErr      error
	addTorrentFileErr error
	pausedHashes      []string
	resumedHashes     []string
	pauseErr          error
	resumeErr         error
	deleteErr         error
	deletedHashes     []string
	deletedFiles      bool

	// torrentFiles maps torrent hash to the list of files returned by ListFiles.
	torrentFiles       map[string][]qbt.TorrentFile
	listFilesErr       error
	setFilePriorityErr error
	// setPriorityRecords tracks calls made to SetFilePriority.
	setPriorityRecords []setPriorityCall

	searchJobID        int
	searchErr          error
	searchStatus       string
	searchResults      []qbt.SearchResult
	searchTotal        int
	stopSearchCalled   bool
	deleteSearchCalled bool

	mu sync.Mutex
}

// setPriorityCall records a single call to SetFilePriority.
type setPriorityCall struct {
	hash     string
	indices  []int
	priority qbt.FilePriority
}

func (m *mockQBTClient) Login(_ context.Context) error { return m.loginErr }

func (m *mockQBTClient) AddMagnet(_ context.Context, magnet, _ string) error {
	if m.addMagnetErr != nil {
		return m.addMagnetErr
	}
	m.magnets = append(m.magnets, magnet)
	return nil
}

func (m *mockQBTClient) AddTorrentFile(_ context.Context, filename string, _ io.Reader, _ string) error {
	if m.addTorrentFileErr != nil {
		return m.addTorrentFileErr
	}
	m.files = append(m.files, filename)
	return nil
}

func (m *mockQBTClient) ListTorrents(_ context.Context, opts qbt.ListOptions) ([]qbt.Torrent, error) {
	m.mu.Lock()
	torrents := make([]qbt.Torrent, len(m.torrents))
	copy(torrents, m.torrents)
	m.mu.Unlock()

	// Apply offset and limit for pagination simulation.
	if opts.Offset > len(torrents) {
		return []qbt.Torrent{}, nil
	}
	torrents = torrents[opts.Offset:]
	if opts.Limit > 0 && opts.Limit < len(torrents) {
		torrents = torrents[:opts.Limit]
	}
	return torrents, nil
}

func (m *mockQBTClient) Categories(_ context.Context) ([]qbt.Category, error) {
	return m.categories, nil
}

func (m *mockQBTClient) PauseTorrents(_ context.Context, hashes []string) error {
	if m.pauseErr != nil {
		return m.pauseErr
	}
	m.pausedHashes = append(m.pausedHashes, hashes...)
	// Simulate state transition after a short delay.
	go func() {
		time.Sleep(50 * time.Millisecond)
		m.mu.Lock()
		defer m.mu.Unlock()
		for i, t := range m.torrents {
			for _, h := range hashes {
				if t.Hash == h {
					m.torrents[i].State = "stoppedDL"
				}
			}
		}
	}()
	return nil
}

func (m *mockQBTClient) ResumeTorrents(_ context.Context, hashes []string) error {
	if m.resumeErr != nil {
		return m.resumeErr
	}
	m.resumedHashes = append(m.resumedHashes, hashes...)
	// Simulate state transition after a short delay.
	go func() {
		time.Sleep(50 * time.Millisecond)
		m.mu.Lock()
		defer m.mu.Unlock()
		for i, t := range m.torrents {
			for _, h := range hashes {
				if t.Hash == h {
					m.torrents[i].State = "downloading"
				}
			}
		}
	}()
	return nil
}

func (m *mockQBTClient) DeleteTorrents(_ context.Context, hashes []string, deleteFiles bool) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.deletedHashes = append(m.deletedHashes, hashes...)
	m.deletedFiles = deleteFiles
	return nil
}

func (m *mockQBTClient) ListFiles(_ context.Context, hash string) ([]qbt.TorrentFile, error) {
	if m.listFilesErr != nil {
		return nil, m.listFilesErr
	}
	if m.torrentFiles != nil {
		return m.torrentFiles[hash], nil
	}
	return nil, nil
}

func (m *mockQBTClient) SetFilePriority(_ context.Context, hash string, fileIndices []int, priority qbt.FilePriority) error {
	if m.setFilePriorityErr != nil {
		return m.setFilePriorityErr
	}
	m.setPriorityRecords = append(m.setPriorityRecords, setPriorityCall{
		hash:     hash,
		indices:  fileIndices,
		priority: priority,
	})
	return nil
}

func (m *mockQBTClient) StartSearch(_ context.Context, _ string) (int, error) {
	if m.searchErr != nil {
		return 0, m.searchErr
	}
	return m.searchJobID, nil
}

func (m *mockQBTClient) SearchStatus(_ context.Context, _ int) (string, error) {
	if m.searchStatus != "" {
		return m.searchStatus, nil
	}
	return "Stopped", nil
}

func (m *mockQBTClient) SearchResults(_ context.Context, _ int, _, _ int) ([]qbt.SearchResult, int, error) {
	return m.searchResults, m.searchTotal, nil
}

func (m *mockQBTClient) StopSearch(_ context.Context, _ int) error {
	m.stopSearchCalled = true
	return nil
}

func (m *mockQBTClient) DeleteSearch(_ context.Context, _ int) error {
	m.deleteSearchCalled = true
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestMessage(chatID, userID int64, text string) tgbotapi.Update {
	return tgbotapi.Update{
		Message: &tgbotapi.Message{
			Chat:     &tgbotapi.Chat{ID: chatID},
			From:     &tgbotapi.User{ID: userID},
			Text:     text,
			Entities: []tgbotapi.MessageEntity{},
		},
	}
}

func newCommandUpdate(chatID, userID int64, command string) tgbotapi.Update {
	text := "/" + command
	return tgbotapi.Update{
		Message: &tgbotapi.Message{
			Chat: &tgbotapi.Chat{ID: chatID},
			From: &tgbotapi.User{ID: userID},
			Text: text,
			Entities: []tgbotapi.MessageEntity{
				{Type: "bot_command", Offset: 0, Length: len(text)},
			},
		},
	}
}

//nolint:unused // used by e2e_test.go (integration-tagged)
func newCommandUpdateWithArgs(chatID, userID int64, command string, args ...string) tgbotapi.Update {
	cmdPrefix := "/" + command
	text := cmdPrefix
	if len(args) > 0 {
		text = cmdPrefix + " " + strings.Join(args, " ")
	}
	return tgbotapi.Update{
		Message: &tgbotapi.Message{
			Chat: &tgbotapi.Chat{ID: chatID},
			From: &tgbotapi.User{ID: userID},
			Text: text,
			Entities: []tgbotapi.MessageEntity{
				{Type: "bot_command", Offset: 0, Length: len(cmdPrefix)},
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestHandler_UnauthorizedUser(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{999}) // user 1 is NOT allowed
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	update := newTestMessage(1, 1, "hello")
	h.HandleUpdate(context.Background(), update)

	if !sender.hasText("Access denied") {
		t.Fatalf("expected 'Access denied' reply, got: %v", sender.sentTexts())
	}
}

func TestHandler_HelpCommand(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	update := newCommandUpdate(1, 1, "help")
	h.HandleUpdate(context.Background(), update)

	if !sender.hasText("/list") {
		t.Fatalf("expected help text with /list, got: %v", sender.sentTexts())
	}
}

func TestHandler_StartCommand(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	update := newCommandUpdate(1, 1, "start")
	h.HandleUpdate(context.Background(), update)

	if !sender.hasText("/list") {
		t.Fatalf("expected help text in response to /start, got: %v", sender.sentTexts())
	}
}

func TestHandler_ListCommand_NoTorrents(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{torrents: []qbt.Torrent{}}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	update := newCommandUpdate(1, 1, "list")
	h.HandleUpdate(context.Background(), update)

	if !sender.hasText("No torrents found") {
		t.Fatalf("expected 'No torrents found', got: %v", sender.sentTexts())
	}
}

func TestHandler_ListCommand_WithTorrents(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{
		torrents: []qbt.Torrent{
			{Hash: "abc", Name: "My Torrent", State: "downloading", Progress: 0.5},
		},
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	update := newCommandUpdate(1, 1, "list")
	h.HandleUpdate(context.Background(), update)

	if !sender.hasText("My Torrent") {
		t.Fatalf("expected torrent name in response, got: %v", sender.sentTexts())
	}
}

func TestHandler_MagnetLink_StoresPendingAndShowsCategories(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{
		categories: []qbt.Category{{Name: "Movies"}, {Name: "TV"}},
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	magnet := "magnet:?xt=urn:btih:abc123&dn=test"
	update := newTestMessage(1, 1, magnet)
	h.HandleUpdate(context.Background(), update)

	// Verify a message was sent (the category keyboard prompt).
	if !sender.hasText("Select category") {
		t.Fatalf("expected category keyboard prompt, got: %v", sender.sentTexts())
	}

	// Verify the pending torrent was stored.
	h.mu.Lock()
	pt, ok := h.pending[1]
	h.mu.Unlock()

	if !ok {
		t.Fatal("expected pending torrent to be stored")
	}
	if pt.MagnetLink != magnet {
		t.Errorf("expected magnet %q stored, got %q", magnet, pt.MagnetLink)
	}
}

// ---------------------------------------------------------------------------
// /downloading command tests (TASK-4)
// ---------------------------------------------------------------------------

func TestHandler_DownloadingCommand_ShowsOnlyIncomplete(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{
		torrents: []qbt.Torrent{
			{Hash: "h1", Name: "Incomplete Torrent", Progress: 0.5},
			{Hash: "h2", Name: "Completed Torrent", Progress: 1.0},
		},
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	update := newCommandUpdate(1, 1, "downloading")
	h.HandleUpdate(context.Background(), update)

	if !sender.hasText("Incomplete Torrent") {
		t.Fatalf("expected incomplete torrent in response, got: %v", sender.sentTexts())
	}
	for _, text := range sender.sentTexts() {
		if strings.Contains(text, "Completed Torrent") {
			t.Fatalf("completed torrent should not appear in /downloading response, got: %v", sender.sentTexts())
		}
	}
}

func TestHandler_DownloadingCommand_NoIncomplete_ShowsNoTorrents(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{
		torrents: []qbt.Torrent{
			{Hash: "h1", Name: "Seeded Torrent", Progress: 1.0},
		},
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	update := newCommandUpdate(1, 1, "downloading")
	h.HandleUpdate(context.Background(), update)

	if !sender.hasText("No torrents found") {
		t.Fatalf("expected 'No torrents found', got: %v", sender.sentTexts())
	}
}

func TestHandler_DownloadingCommand_PausedIncomplete_Appears(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{
		torrents: []qbt.Torrent{
			{Hash: "h1", Name: "Paused Download", Progress: 0.3, State: "pausedDL"},
		},
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	update := newCommandUpdate(1, 1, "downloading")
	h.HandleUpdate(context.Background(), update)

	if !sender.hasText("Paused Download") {
		t.Fatalf("expected paused incomplete torrent in response, got: %v", sender.sentTexts())
	}
}

// ---------------------------------------------------------------------------
// /uploading command tests (TEST-4, TEST-5)
// ---------------------------------------------------------------------------

func TestHandler_UploadingCommand_ShowsOnlyCompleted(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{
		torrents: []qbt.Torrent{
			{Hash: "h1", Name: "Seeding Torrent", Progress: 1.0, State: "uploading"},
			{Hash: "h2", Name: "Incomplete Torrent", Progress: 0.5},
		},
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	update := newCommandUpdate(1, 1, "uploading")
	h.HandleUpdate(context.Background(), update)

	if !sender.hasText("Seeding Torrent") {
		t.Fatalf("expected completed torrent in response, got: %v", sender.sentTexts())
	}
	for _, text := range sender.sentTexts() {
		if strings.Contains(text, "Incomplete Torrent") {
			t.Fatalf("incomplete torrent should not appear in /uploading response, got: %v", sender.sentTexts())
		}
	}
}

func TestHandler_UploadingCommand_NoCompleted_ShowsNoTorrents(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{
		torrents: []qbt.Torrent{
			{Hash: "h1", Name: "Downloading Torrent", Progress: 0.3},
		},
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	update := newCommandUpdate(1, 1, "uploading")
	h.HandleUpdate(context.Background(), update)

	if !sender.hasText("No torrents found") {
		t.Fatalf("expected 'No torrents found', got: %v", sender.sentTexts())
	}
}

func TestHandler_UploadingCommand_PausedUP_Excluded(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{
		torrents: []qbt.Torrent{
			{Hash: "h1", Name: "Paused Seed", Progress: 1.0, State: "pausedUP"},
		},
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	update := newCommandUpdate(1, 1, "uploading")
	h.HandleUpdate(context.Background(), update)

	for _, text := range sender.sentTexts() {
		if strings.Contains(text, "Paused Seed") {
			t.Fatalf("pausedUP torrent should not appear in /uploading response, got: %v", sender.sentTexts())
		}
	}
	if !sender.hasText("No torrents found") {
		t.Fatalf("expected 'No torrents found' when only pausedUP torrents exist, got: %v", sender.sentTexts())
	}
}

func TestHandler_UploadingCommand_StalledUP_Appears(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{
		torrents: []qbt.Torrent{
			{Hash: "h1", Name: "Stalled Seed", Progress: 1.0, State: "stalledUP"},
		},
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	update := newCommandUpdate(1, 1, "uploading")
	h.HandleUpdate(context.Background(), update)

	if !sender.hasText("Stalled Seed") {
		t.Fatalf("expected stalled seeding torrent in response, got: %v", sender.sentTexts())
	}
}

func TestHandler_UploadingCommand_InBotCommands(t *testing.T) {
	found := false
	for _, cmd := range BotCommands {
		if cmd.Command == "uploading" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected 'uploading' command in BotCommands")
	}
}

func TestHandler_UploadingCommand_InHelpText(t *testing.T) {
	help := HelpText()
	if !strings.Contains(help, "/uploading") {
		t.Fatalf("expected '/uploading' in help text, got: %s", help)
	}
}

func TestHandler_MagnetLink_MidText(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{
		categories: []qbt.Category{{Name: "Movies"}},
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	update := newTestMessage(1, 1, "here is my link magnet:?xt=urn:btih:deadbeef thanks")
	h.HandleUpdate(context.Background(), update)

	h.mu.Lock()
	pt, ok := h.pending[1]
	h.mu.Unlock()

	if !ok {
		t.Fatal("expected pending torrent from mid-text magnet")
	}
	if !strings.HasPrefix(pt.MagnetLink, "magnet:?") {
		t.Errorf("unexpected magnet stored: %q", pt.MagnetLink)
	}
	// Should not contain trailing space.
	if strings.Contains(pt.MagnetLink, " ") {
		t.Errorf("magnet link should not contain spaces: %q", pt.MagnetLink)
	}
}

// ---------------------------------------------------------------------------
// Auto-Refresh Tests
// ---------------------------------------------------------------------------

func TestFilterPrefixForView(t *testing.T) {
	tests := []struct {
		filter   qbt.TorrentFilter
		expected string
	}{
		{qbt.FilterAll, "all"},
		{qbt.FilterActive, "act"},
		{qbt.FilterDownloading, "dw"},
		{qbt.FilterUploading, "up"},
	}

	for _, tt := range tests {
		t.Run(string(tt.filter), func(t *testing.T) {
			lv := &LiveView{Filter: tt.filter}
			if got := filterPrefixForView(lv); got != tt.expected {
				t.Errorf("filterPrefixForView(%s) = %q, want %q", tt.filter, got, tt.expected)
			}
		})
	}
}

func TestLiveViewRegisterDeregister(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	chatID := int64(42)
	lv := &LiveView{
		ChatID:    chatID,
		MessageID: 100,
		ViewType:  ViewList,
		Filter:    qbt.FilterAll,
		Page:      1,
	}

	// Register.
	h.registerLiveView(chatID, lv)

	h.liveViewsMu.Lock()
	if got, ok := h.liveViews[chatID]; !ok || got != lv {
		h.liveViewsMu.Unlock()
		t.Fatal("expected live view to be registered")
	}
	h.liveViewsMu.Unlock()

	// Register another — replaces.
	lv2 := &LiveView{ChatID: chatID, MessageID: 200, ViewType: ViewDetail}
	h.registerLiveView(chatID, lv2)

	h.liveViewsMu.Lock()
	if got := h.liveViews[chatID]; got != lv2 {
		h.liveViewsMu.Unlock()
		t.Fatal("expected live view to be replaced")
	}
	h.liveViewsMu.Unlock()

	// Deregister.
	h.deregisterLiveView(chatID, 200)

	h.liveViewsMu.Lock()
	if _, ok := h.liveViews[chatID]; ok {
		h.liveViewsMu.Unlock()
		t.Fatal("expected live view to be deregistered")
	}
	h.liveViewsMu.Unlock()

	// Deregister non-existent is a no-op.
	h.deregisterLiveView(chatID, 200)
}

func TestRefreshLiveView_ListView_Changed(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{
		torrents: []qbt.Torrent{
			{Hash: "a", Name: "Torrent A", Progress: 0.5, DLSpeed: 1024, State: "downloading"},
		},
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	chatID := int64(1)
	lv := &LiveView{
		ChatID:    chatID,
		MessageID: 100,
		ViewType:  ViewList,
		Filter:    qbt.FilterAll,
		Page:      1,
		// LastContentHash empty → will trigger edit.
	}
	h.registerLiveView(chatID, lv)

	err := h.refreshLiveView(context.Background(), lv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have called Request (editMessageText).
	if !sender.hasRequest() {
		t.Fatal("expected editMessageText to be called via Request")
	}

	// LastContentHash should be updated.
	if lv.LastContentHash == "" {
		t.Fatal("expected LastContentHash to be updated")
	}
}

func TestRefreshLiveView_ListView_Unchanged(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{
		torrents: []qbt.Torrent{
			{Hash: "a", Name: "Torrent A", Progress: 0.5, DLSpeed: 1024, State: "downloading"},
		},
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	chatID := int64(1)
	lv := &LiveView{
		ChatID:    chatID,
		MessageID: 100,
		ViewType:  ViewList,
		Filter:    qbt.FilterAll,
		Page:      1,
	}
	h.registerLiveView(chatID, lv)

	// First refresh — sets LastContentHash.
	err := h.refreshLiveView(context.Background(), lv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	hashAfterFirst := lv.LastContentHash
	sender.reset()

	// Second refresh — should NOT edit (hash matches).
	err = h.refreshLiveView(context.Background(), lv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sender.hasRequest() {
		t.Fatal("expected no edit on unchanged content")
	}
	if lv.LastContentHash != hashAfterFirst {
		t.Fatal("expected LastContentHash unchanged")
	}
}

func TestRefreshLiveView_DetailView_Found(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{
		torrents: []qbt.Torrent{
			{Hash: "abc123", Name: "Movie", Progress: 0.8, DLSpeed: 2048, State: "downloading"},
		},
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	chatID := int64(1)
	lv := &LiveView{
		ChatID:      chatID,
		MessageID:   100,
		ViewType:    ViewDetail,
		TorrentHash: "abc123",
		FilterChar:  "a",
		Page:        1,
	}
	h.registerLiveView(chatID, lv)

	err := h.refreshLiveView(context.Background(), lv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !sender.hasRequest() {
		t.Fatal("expected editMessageText for detail view")
	}
}

func TestRefreshLiveView_DetailView_NotFound(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{
		torrents: []qbt.Torrent{},
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	chatID := int64(1)
	h.registerLiveView(chatID, &LiveView{
		ChatID:      chatID,
		MessageID:   100,
		ViewType:    ViewDetail,
		TorrentHash: "missing",
	})

	err := h.refreshLiveView(context.Background(), h.liveViews[chatID])
	if err != nil {
		t.Fatalf("expected no error for missing torrent (deregistered), got: %v", err)
	}

	// Should have deregistered the view.
	h.liveViewsMu.Lock()
	_, ok := h.liveViews[chatID]
	h.liveViewsMu.Unlock()
	if ok {
		t.Fatal("expected live view to be deregistered on missing torrent")
	}
}

func TestRefreshLiveView_DefaultViewType(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	lv := &LiveView{
		ChatID:    1,
		MessageID: 100,
		ViewType:  ViewType("unknown"),
	}

	err := h.refreshLiveView(context.Background(), lv)
	if err != nil {
		t.Fatalf("unexpected error for unknown view type: %v", err)
	}
}

func TestRefreshViews_MultipleViews(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{
		torrents: []qbt.Torrent{
			{Hash: "a", Name: "A", Progress: 0.5, State: "downloading"},
			{Hash: "b", Name: "B", Progress: 1.0, State: "uploading"},
		},
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	// Register two views.
	h.registerLiveView(1, &LiveView{
		ChatID:    1,
		MessageID: 100,
		ViewType:  ViewList,
		Filter:    qbt.FilterAll,
		Page:      1,
	})
	h.registerLiveView(2, &LiveView{
		ChatID:      2,
		MessageID:   200,
		ViewType:    ViewDetail,
		TorrentHash: "a",
	})

	h.refreshViews(context.Background())

	// Both should have been refreshed.
	if !sender.hasRequest() {
		t.Fatal("expected Request calls for both views")
	}
}

// ---------------------------------------------------------------------------
// Mock sender helpers
// ---------------------------------------------------------------------------

func (m *mockSender) hasRequest() bool {
	for _, msg := range m.sentMessages {
		if _, ok := msg.(tgbotapi.EditMessageTextConfig); ok {
			return true
		}
	}
	return false
}

func (m *mockSender) reset() {
	m.sentMessages = nil
	m.requestErr = nil
}

func TestHandleSearchCommand_WithQuery(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{
		searchJobID: 123,
		searchResults: []qbt.SearchResult{
			{FileName: "Ubuntu 24.04", FileSize: 1024, NbSeeders: 10, FileURL: "magnet:?xt=urn:btih:abc"},
		},
		searchTotal: 1,
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	msg := &tgbotapi.Message{
		Chat: &tgbotapi.Chat{ID: 1},
		From: &tgbotapi.User{ID: 1},
		Text: "/search ubuntu",
		Entities: []tgbotapi.MessageEntity{
			{Type: "bot_command", Offset: 0, Length: 7},
		},
	}
	h.handleSearchCommand(context.Background(), msg)

	if !sender.hasText("Searching for 'ubuntu'") {
		t.Fatalf("expected searching message, got: %v", sender.sentTexts())
	}
}

func TestHandleSearchCommand_WithoutQuery(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	msg := &tgbotapi.Message{
		Chat: &tgbotapi.Chat{ID: 1},
		From: &tgbotapi.User{ID: 1},
		Text: "/search",
		Entities: []tgbotapi.MessageEntity{
			{Type: "bot_command", Offset: 0, Length: 7},
		},
	}
	h.handleSearchCommand(context.Background(), msg)

	if !sender.hasText("What to search for?") {
		t.Fatalf("expected prompt message, got: %v", sender.sentTexts())
	}

	prompt := h.takeSearchPrompt(1)
	if prompt == nil {
		t.Fatal("expected search prompt to be stored")
	}
}

func TestHandleSearchPromptReply(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{
		searchJobID: 456,
		searchResults: []qbt.SearchResult{
			{FileName: "Debian ISO", FileSize: 2048, NbSeeders: 5, FileURL: "magnet:?xt=urn:btih:def"},
		},
		searchTotal: 1,
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	msg := &tgbotapi.Message{
		Chat: &tgbotapi.Chat{ID: 1},
		From: &tgbotapi.User{ID: 1},
		Text: "debian",
	}
	h.handleSearchPromptReply(context.Background(), msg)

	if !sender.hasText("Searching for 'debian'") {
		t.Fatalf("expected searching message, got: %v", sender.sentTexts())
	}
}

func TestHandleSearchPromptReply_Empty(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	msg := &tgbotapi.Message{
		Chat: &tgbotapi.Chat{ID: 1},
		From: &tgbotapi.User{ID: 1},
		Text: "",
	}
	h.handleSearchPromptReply(context.Background(), msg)

	if !sender.hasText("Usage: /search <query>") {
		t.Fatalf("expected usage message, got: %v", sender.sentTexts())
	}
}

func TestSortSearchResults_BySeeders(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	results := []qbt.SearchResult{
		{FileName: "A", NbSeeders: 5},
		{FileName: "B", NbSeeders: 20},
		{FileName: "C", NbSeeders: 10},
	}

	h.sortSearchResults(results, "seeders", false)

	if results[0].FileName != "B" || results[1].FileName != "C" || results[2].FileName != "A" {
		t.Fatalf("expected descending seeders sort, got: %v", results)
	}
}

func TestSortSearchResults_BySeedersAsc(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	results := []qbt.SearchResult{
		{FileName: "A", NbSeeders: 5},
		{FileName: "B", NbSeeders: 20},
		{FileName: "C", NbSeeders: 10},
	}

	h.sortSearchResults(results, "seeders", true)

	if results[0].FileName != "A" || results[1].FileName != "C" || results[2].FileName != "B" {
		t.Fatalf("expected ascending seeders sort, got: %v", results)
	}
}

func TestSortSearchResults_BySize(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	results := []qbt.SearchResult{
		{FileName: "A", FileSize: 100},
		{FileName: "B", FileSize: 500},
		{FileName: "C", FileSize: 200},
	}

	h.sortSearchResults(results, "size", false)

	if results[0].FileName != "B" || results[1].FileName != "C" || results[2].FileName != "A" {
		t.Fatalf("expected descending size sort, got: %v", results)
	}
}

func TestSortSearchResults_ByDate(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	results := []qbt.SearchResult{
		{FileName: "A", PubDate: 100},
		{FileName: "B", PubDate: 300},
		{FileName: "C", PubDate: 200},
	}

	h.sortSearchResults(results, "date", false)

	if results[0].FileName != "B" || results[1].FileName != "C" || results[2].FileName != "A" {
		t.Fatalf("expected descending date sort, got: %v", results)
	}
}

func TestStoreTakeGetSearch(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	state := &SearchState{ChatID: 1, Query: "test", JobID: 42}
	h.storeSearch(1, state)

	got := h.getSearch(1)
	if got == nil || got.JobID != 42 {
		t.Fatalf("expected stored search, got: %v", got)
	}

	taken := h.takeSearch(1)
	if taken == nil || taken.JobID != 42 {
		t.Fatalf("expected taken search, got: %v", taken)
	}

	if h.getSearch(1) != nil {
		t.Fatal("expected search to be removed after take")
	}
}

func TestStoreTakeSearchPrompt(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	prompt := &SearchPrompt{ChatID: 1}
	h.storeSearchPrompt(1, prompt)

	taken := h.takeSearchPrompt(1)
	if taken == nil {
		t.Fatal("expected taken prompt")
	}

	if h.takeSearchPrompt(1) != nil {
		t.Fatal("expected prompt to be removed after take")
	}
}

func TestSendSearchResultsPage(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	results := []qbt.SearchResult{
		{FileName: "Ubuntu 24.04", FileSize: 1024, NbSeeders: 10, FileURL: "magnet:?xt=urn:btih:abc"},
	}
	state := &SearchState{
		ChatID:    1,
		Query:     "ubuntu",
		JobID:     123,
		Results:   results,
		Total:     1,
		SortField: "seeders",
		SortAsc:   false,
	}

	msgID := h.sendSearchResultsPage(1, state, 1, 0)
	if msgID == 0 {
		t.Fatal("expected non-zero message ID")
	}

	if !sender.hasText("Search: ubuntu") {
		t.Fatalf("expected search results text, got: %v", sender.sentTexts())
	}
}

func TestSendSearchResultsPage_EmptyResults(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	state := &SearchState{
		ChatID:    1,
		Query:     "ubuntu",
		JobID:     123,
		Results:   []qbt.SearchResult{},
		Total:     0,
		SortField: "seeders",
		SortAsc:   false,
	}

	msgID := h.sendSearchResultsPage(1, state, 1, 0)
	if msgID == 0 {
		t.Fatal("expected non-zero message ID")
	}

	if !sender.hasText("No torrents found") {
		t.Fatalf("expected no results text, got: %v", sender.sentTexts())
	}
}

func TestPollSearchResults(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{
		searchJobID: 789,
		searchResults: []qbt.SearchResult{
			{FileName: "Ubuntu 24.04", FileSize: 1024, NbSeeders: 10, FileURL: "magnet:?xt=urn:btih:abc"},
		},
		searchTotal: 1,
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	h.pollSearchResults(ctx, 1, "ubuntu", 0)

	time.Sleep(100 * time.Millisecond)

	if !sender.hasText("Search: ubuntu") && !sender.hasText("Searching for 'ubuntu'") {
		t.Fatalf("expected search results or initial message, got: %v", sender.sentTexts())
	}
}

func TestPollSearchResults_NoResults(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{
		searchJobID:   789,
		searchResults: []qbt.SearchResult{},
		searchTotal:   0,
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	h.pollSearchResults(ctx, 1, "nothing", 0)

	time.Sleep(100 * time.Millisecond)

	if !sender.hasText("No torrents found for 'nothing'") && !sender.hasText("Searching for 'nothing'") {
		t.Fatalf("expected no results message, got: %v", sender.sentTexts())
	}
}

func TestPollSearchResults_StartError(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{
		searchErr: errors.New("search unavailable"),
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	h.pollSearchResults(ctx, 1, "ubuntu", 0)

	time.Sleep(100 * time.Millisecond)

	if !sender.hasText("Search unavailable") && !sender.hasText("Searching for 'ubuntu'") {
		t.Fatalf("expected unavailable message, got: %v", sender.sentTexts())
	}
}

func TestPollSearchResults_ErrorStatus(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{
		searchJobID:  789,
		searchStatus: "Error",
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	h.pollSearchResults(ctx, 1, "ubuntu", 0)

	time.Sleep(100 * time.Millisecond)

	if !sender.hasText("Search failed or returned no results") {
		t.Fatalf("expected failure message, got: %v", sender.sentTexts())
	}
	if !qbtClient.deleteSearchCalled {
		t.Fatal("expected DeleteSearch to be called")
	}
}

func TestPollSearchResults_NoResultsStatus(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{
		searchJobID:  789,
		searchStatus: "NoResults",
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	h.pollSearchResults(ctx, 1, "nothing", 0)

	time.Sleep(100 * time.Millisecond)

	if !sender.hasText("Search failed or returned no results") {
		t.Fatalf("expected failure message, got: %v", sender.sentTexts())
	}
	if !qbtClient.deleteSearchCalled {
		t.Fatal("expected DeleteSearch to be called")
	}
}

func TestPollSearchResults_Timeout(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{
		searchJobID:  789,
		searchStatus: "Running", // never completes
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	// Very short deadline to trigger timeout quickly.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	h.pollSearchResults(ctx, 1, "ubuntu", 0)

	time.Sleep(200 * time.Millisecond)

	if !sender.hasText("Search timed out") {
		t.Fatalf("expected timeout message, got: %v", sender.sentTexts())
	}
	// Cleanup runs in background goroutine — wait briefly.
	time.Sleep(50 * time.Millisecond)
	if !qbtClient.stopSearchCalled {
		t.Fatal("expected StopSearch to be called on timeout")
	}
	if !qbtClient.deleteSearchCalled {
		t.Fatal("expected DeleteSearch to be called on timeout")
	}
}

func TestPollSearchResults_ContextCancelled(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{
		searchJobID:  789,
		searchStatus: "Running",
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediate cancellation

	h.pollSearchResults(ctx, 1, "ubuntu", 0)

	time.Sleep(100 * time.Millisecond)

	// Context cancellation (not timeout) should NOT send a message.
	if sender.hasText("Search timed out") {
		t.Fatalf("should not send timeout on context cancel, got: %v", sender.sentTexts())
	}
	// Cleanup runs in background goroutine — wait briefly.
	time.Sleep(50 * time.Millisecond)
	if !qbtClient.stopSearchCalled {
		t.Fatal("expected StopSearch to be called on cancel")
	}
	if !qbtClient.deleteSearchCalled {
		t.Fatal("expected DeleteSearch to be called on cancel")
	}
}

func TestSearchGoroutineDedup(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{
		searchJobID:  789,
		searchStatus: "Running", // keeps first search alive
		searchResults: []qbt.SearchResult{
			{FileName: "Result", FileSize: 100, NbSeeders: 5, FileURL: "magnet:?xt=urn:btih:xyz"},
		},
		searchTotal: 1,
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	// Launch first search via direct cancel injection (simulates launchSearch).
	ctx1, cancel1 := context.WithTimeout(context.Background(), searchTimeout+5*time.Second)
	h.searchMu.Lock()
	h.searchCancels[1] = cancel1
	h.searchMu.Unlock()
	go func() {
		defer cancel1()
		h.pollSearchResults(ctx1, 1, "first", 0)
	}()

	time.Sleep(100 * time.Millisecond)

	// Verify first search is still registered.
	h.searchMu.Lock()
	_, hasCancel := h.searchCancels[1]
	h.searchMu.Unlock()
	if !hasCancel {
		t.Fatal("expected first search to be registered")
	}

	// Launch second search — should cancel the first via handleSearchCommand dedup.
	cmdMsg := tgbotapi.Update{
		Message: &tgbotapi.Message{
			Chat: &tgbotapi.Chat{ID: 1},
			From: &tgbotapi.User{ID: 1},
			Text: "/search second",
			Entities: []tgbotapi.MessageEntity{
				{Type: "bot_command", Offset: 0, Length: 7},
			},
		},
	}
	h.HandleUpdate(context.Background(), cmdMsg)

	time.Sleep(200 * time.Millisecond)

	// First search should have been canceled.
	select {
	case <-ctx1.Done():
		// Expected — first search was canceled.
	default:
		t.Fatal("expected first search context to be canceled")
	}
}

func TestPollSearchResults_Timeout_WithEditMsgID(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{
		searchJobID:  789,
		searchStatus: "Running",
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	h.pollSearchResults(ctx, 1, "ubuntu", 42) // replyMsgID=42 exercises edit path

	time.Sleep(200 * time.Millisecond)

	// With non-zero replyMsgID, the timeout message is sent via editMessageText (Request), not Send.
	if !sender.hasRequest() {
		t.Fatal("expected editMessageText call when replyMsgID is non-zero")
	}
}

func TestEditOrReply(t *testing.T) {
	auth := NewAuthorizer([]int64{1})

	t.Run("edits when replyMsgID non-zero", func(t *testing.T) {
		sender := &mockSender{}
		h := New(context.Background(), sender, &mockQBTClient{}, auth, HandlerOptions{BotToken: "test-token"})
		h.editOrReply(1, 42, "test edit")
		if !sender.hasRequest() {
			t.Fatal("expected editMessageText when replyMsgID != 0")
		}
	})

	t.Run("sends new message when replyMsgID zero", func(t *testing.T) {
		sender := &mockSender{}
		h := New(context.Background(), sender, &mockQBTClient{}, auth, HandlerOptions{BotToken: "test-token"})
		h.editOrReply(1, 0, "test send")
		if sender.hasRequest() {
			t.Fatal("expected new message send when replyMsgID == 0, not edit")
		}
		if !sender.hasText("test send") {
			t.Fatalf("expected new message, got: %v", sender.sentTexts())
		}
	})

	t.Run("falls back to new message on edit failure", func(t *testing.T) {
		sender := &mockSender{requestErr: errors.New("edit failed")}
		h := New(context.Background(), sender, &mockQBTClient{}, auth, HandlerOptions{BotToken: "test-token"})
		h.editOrReply(1, 42, "fallback test")
		// editMessageText was attempted (hasRequest) but failed, so replyText should also have been called.
		if !sender.hasRequest() {
			t.Fatal("expected edit attempt")
		}
		if !sender.hasText("fallback test") {
			t.Fatalf("expected fallback new message after edit failure, got: %v", sender.sentTexts())
		}
	})
}

// ############################################################################
// LiveView lifecycle tests (REQ-8, REQ-9)
// ############################################################################

func TestRegisterLiveView_SetsTimestamp(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	chatID := int64(1)
	before := time.Now()
	h.registerLiveView(chatID, &LiveView{
		ChatID:    chatID,
		MessageID: 100,
		ViewType:  ViewList,
		Filter:    qbt.FilterAll,
		Page:      1,
	})
	after := time.Now()

	h.liveViewsMu.Lock()
	lv := h.liveViews[chatID]
	h.liveViewsMu.Unlock()

	if lv == nil {
		t.Fatal("expected live view to be registered")
	}
	if lv.RegisteredAt.Before(before) || lv.RegisteredAt.After(after) {
		t.Fatalf("RegisteredAt=%v, want between %v and %v", lv.RegisteredAt, before, after)
	}

	// Re-register should update timestamp.
	time.Sleep(10 * time.Millisecond)
	h.registerLiveView(chatID, &LiveView{
		ChatID:      chatID,
		MessageID:   200,
		ViewType:    ViewDetail,
		TorrentHash: "abc",
	})
	h.liveViewsMu.Lock()
	newLV := h.liveViews[chatID]
	h.liveViewsMu.Unlock()

	if !newLV.RegisteredAt.After(lv.RegisteredAt) {
		t.Fatal("re-registration should set a newer RegisteredAt")
	}
}

func TestRefreshLiveView_DeadlineExpired(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{
		torrents: []qbt.Torrent{
			{Hash: "a", Name: "A", Progress: 0.5, State: "downloading"},
		},
	}
	auth := NewAuthorizer([]int64{1})
	// Use a zero ViewTTL so every view is instantly expired.
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{
		BotToken: "test-token",
		ViewTTL:  0, // defaults to 5min, but we set RegisteredAt far in the past
	})

	chatID := int64(1)
	lv := &LiveView{
		ChatID:    chatID,
		MessageID: 100,
		ViewType:  ViewList,
		Filter:    qbt.FilterAll,
		Page:      1,
	}
	// Simulate a view registered long ago.
	lv.RegisteredAt = time.Now().Add(-1 * time.Hour)
	h.liveViewsMu.Lock()
	h.liveViews[chatID] = lv
	h.liveViewsMu.Unlock()

	err := h.refreshLiveView(context.Background(), lv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// View should be deregistered.
	h.liveViewsMu.Lock()
	_, ok := h.liveViews[chatID]
	h.liveViewsMu.Unlock()
	if ok {
		t.Fatal("expected view to be deregistered after deadline expiry")
	}
	// No API call should have been made (deadline fires before rendering).
	if sender.hasRequest() {
		t.Fatal("expected no Request call for expired view")
	}
}

func TestRefreshLiveView_CooldownSkip(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{
		torrents: []qbt.Torrent{
			{Hash: "a", Name: "A", Progress: 0.5, State: "downloading"},
		},
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

	chatID := int64(1)
	lv := &LiveView{
		ChatID:        chatID,
		MessageID:     100,
		ViewType:      ViewList,
		Filter:        qbt.FilterAll,
		Page:          1,
		RegisteredAt:  time.Now(),
		NextRefreshAt: time.Now().Add(5 * time.Minute), // cooldown active
	}
	h.liveViewsMu.Lock()
	h.liveViews[chatID] = lv
	h.liveViewsMu.Unlock()

	err := h.refreshLiveView(context.Background(), lv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No API call — cooldown skipped the refresh.
	if sender.hasRequest() {
		t.Fatal("expected no Request call during cooldown")
	}
	// View is still registered (cooldown does not deregister).
	h.liveViewsMu.Lock()
	_, ok := h.liveViews[chatID]
	h.liveViewsMu.Unlock()
	if !ok {
		t.Fatal("expected view to remain registered during cooldown")
	}
}

func TestRefreshLiveView_ErrorDiscrimination(t *testing.T) {
	t.Run("429 sets NextRefreshAt and keeps view", func(t *testing.T) {
		sender := &mockSender{
			requestErr: tgbotapi.Error{
				Code:    429,
				Message: "Too Many Requests: retry after 30",
				ResponseParameters: tgbotapi.ResponseParameters{
					RetryAfter: 30,
				},
			},
		}
		qbtClient := &mockQBTClient{
			torrents: []qbt.Torrent{
				{Hash: "a", Name: "A", Progress: 0.5, State: "downloading"},
			},
		}
		auth := NewAuthorizer([]int64{1})
		h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

		chatID := int64(1)
		before := time.Now()
		lv := &LiveView{
			ChatID:       chatID,
			MessageID:    100,
			ViewType:     ViewList,
			Filter:       qbt.FilterAll,
			Page:         1,
			RegisteredAt: time.Now(),
		}
		h.liveViewsMu.Lock()
		h.liveViews[chatID] = lv
		h.liveViewsMu.Unlock()

		err := h.refreshLiveView(context.Background(), lv)
		if err == nil {
			t.Fatal("expected error from 429")
		}

		if lv.NextRefreshAt.Before(before.Add(25 * time.Second)) {
			t.Fatalf("expected NextRefreshAt to be set ~30s in future, got %v (before=%v)", lv.NextRefreshAt, before)
		}
		if lv.ErrorCount != 0 {
			t.Fatalf("expected ErrorCount=0 after 429, got %d", lv.ErrorCount)
		}
		// View stays registered.
		h.liveViewsMu.Lock()
		_, ok := h.liveViews[chatID]
		h.liveViewsMu.Unlock()
		if !ok {
			t.Fatal("expected view to remain registered after 429")
		}
	})

	t.Run("404 deregisters immediately", func(t *testing.T) {
		sender := &mockSender{
			requestErr: errors.New("message to edit not found"),
		}
		qbtClient := &mockQBTClient{
			torrents: []qbt.Torrent{
				{Hash: "a", Name: "A", Progress: 0.5, State: "downloading"},
			},
		}
		auth := NewAuthorizer([]int64{1})
		h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

		chatID := int64(1)
		lv := &LiveView{
			ChatID:       chatID,
			MessageID:    100,
			ViewType:     ViewList,
			Filter:       qbt.FilterAll,
			Page:         1,
			RegisteredAt: time.Now(),
		}
		h.liveViewsMu.Lock()
		h.liveViews[chatID] = lv
		h.liveViewsMu.Unlock()

		err := h.refreshLiveView(context.Background(), lv)
		if err == nil {
			t.Fatal("expected error from 404")
		}

		h.liveViewsMu.Lock()
		_, ok := h.liveViews[chatID]
		h.liveViewsMu.Unlock()
		if ok {
			t.Fatal("expected view to be deregistered after 404")
		}
	})

	t.Run("3 consecutive errors deregister", func(t *testing.T) {
		sender := &mockSender{
			requestErr: errors.New("network timeout"),
		}
		qbtClient := &mockQBTClient{
			torrents: []qbt.Torrent{
				{Hash: "a", Name: "A", Progress: 0.5, State: "downloading"},
			},
		}
		auth := NewAuthorizer([]int64{1})
		h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token"})

		chatID := int64(1)
		lv := &LiveView{
			ChatID:       chatID,
			MessageID:    100,
			ViewType:     ViewList,
			Filter:       qbt.FilterAll,
			Page:         1,
			RegisteredAt: time.Now(),
		}
		h.liveViewsMu.Lock()
		h.liveViews[chatID] = lv
		h.liveViewsMu.Unlock()

		// Error 1 — still registered.
		_ = h.refreshLiveView(context.Background(), lv)
		if lv.ErrorCount != 1 {
			t.Fatalf("expected ErrorCount=1, got %d", lv.ErrorCount)
		}

		// Error 2 — still registered.
		_ = h.refreshLiveView(context.Background(), lv)
		if lv.ErrorCount != 2 {
			t.Fatalf("expected ErrorCount=2, got %d", lv.ErrorCount)
		}

		// Error 3 — deregistered.
		_ = h.refreshLiveView(context.Background(), lv)
		h.liveViewsMu.Lock()
		_, ok := h.liveViews[chatID]
		h.liveViewsMu.Unlock()
		if ok {
			t.Fatal("expected view to be deregistered after 3 consecutive errors")
		}
		if lv.ErrorCount != 3 {
			t.Fatalf("expected ErrorCount=3, got %d", lv.ErrorCount)
		}
	})
}

func TestRefreshViews_ConcurrencyLimit(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{
		torrents: []qbt.Torrent{
			{Hash: "a", Name: "A", Progress: 0.5, State: "downloading"},
		},
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, HandlerOptions{BotToken: "test-token", ViewTTL: 1 * time.Hour})

	// Register 10 views (more than maxConcurrentRefreshes=5).
	for i := int64(1); i <= 10; i++ {
		h.registerLiveView(i, &LiveView{
			ChatID:    i,
			MessageID: int(i * 100),
			ViewType:  ViewList,
			Filter:    qbt.FilterAll,
			Page:      1,
		})
	}

	// refreshViews should complete without panicking or deadlocking.
	h.refreshViews(context.Background())

	// All 10 views should have been processed.
	h.liveViewsMu.Lock()
	remaining := len(h.liveViews)
	h.liveViewsMu.Unlock()
	if remaining != 10 {
		t.Fatalf("expected 10 views registered, got %d", remaining)
	}
	// Verify sender was called (hash-based change detection means all views
	// produce different content on the first refresh).
	if !sender.hasRequest() {
		t.Fatal("expected Request calls for concurrent views")
	}
}
