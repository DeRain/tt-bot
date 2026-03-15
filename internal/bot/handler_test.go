package bot

import (
	"context"
	"io"
	"strings"
	"testing"

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
}

func (m *mockSender) Send(msg tgbotapi.Chattable) (tgbotapi.Message, error) {
	m.sentMessages = append(m.sentMessages, msg)
	return tgbotapi.Message{}, nil
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
	loginErr      error
	magnets       []string
	files         []string
	torrents      []qbt.Torrent
	categories    []qbt.Category
	addMagnetErr  error
	pausedHashes  []string
	resumedHashes []string
	pauseErr      error
	resumeErr     error
	deleteErr     error
	deletedHashes []string
	deletedFiles  bool

	// torrentFiles maps torrent hash to the list of files returned by ListFiles.
	torrentFiles       map[string][]qbt.TorrentFile
	listFilesErr       error
	setFilePriorityErr error
	// setPriorityRecords tracks calls made to SetFilePriority.
	setPriorityRecords []setPriorityCall
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
	m.files = append(m.files, filename)
	return nil
}

func (m *mockQBTClient) ListTorrents(_ context.Context, opts qbt.ListOptions) ([]qbt.Torrent, error) {
	torrents := m.torrents

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
	return nil
}

func (m *mockQBTClient) ResumeTorrents(_ context.Context, hashes []string) error {
	if m.resumeErr != nil {
		return m.resumeErr
	}
	m.resumedHashes = append(m.resumedHashes, hashes...)
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

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestHandler_UnauthorizedUser(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{}
	auth := NewAuthorizer([]int64{999}) // user 1 is NOT allowed
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

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
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

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
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

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
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

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
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

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
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

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
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

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
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

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
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

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
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

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
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	update := newCommandUpdate(1, 1, "uploading")
	h.HandleUpdate(context.Background(), update)

	if !sender.hasText("No torrents found") {
		t.Fatalf("expected 'No torrents found', got: %v", sender.sentTexts())
	}
}

func TestHandler_UploadingCommand_PausedUP_Appears(t *testing.T) {
	sender := &mockSender{}
	qbtClient := &mockQBTClient{
		torrents: []qbt.Torrent{
			{Hash: "h1", Name: "Paused Seed", Progress: 1.0, State: "pausedUP"},
		},
	}
	auth := NewAuthorizer([]int64{1})
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

	update := newCommandUpdate(1, 1, "uploading")
	h.HandleUpdate(context.Background(), update)

	if !sender.hasText("Paused Seed") {
		t.Fatalf("expected paused completed torrent in response, got: %v", sender.sentTexts())
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
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

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
	h := New(context.Background(), sender, qbtClient, auth, "test-token")

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
