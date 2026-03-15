package bot

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/home/tt-bot/internal/formatter"
	"github.com/home/tt-bot/internal/qbt"
)

const (
	// pendingTTL is the maximum age of a pending torrent before it is evicted.
	pendingTTL = 5 * time.Minute
	// cleanupInterval is how often the pending-map cleanup goroutine runs.
	cleanupInterval = 1 * time.Minute
	// actionStateDelay is how long to wait after a pause/resume action
	// for qBittorrent to update the torrent state before re-fetching.
	actionStateDelay = 500 * time.Millisecond
)

// PendingTorrent holds a torrent that the user has sent but has not yet been
// assigned a category. It is stored in the Handler's pending map keyed by
// chat ID and expires after pendingTTL.
type PendingTorrent struct {
	// MagnetLink is set when the user sends a magnet URI.
	MagnetLink string
	// FileData contains the raw .torrent file bytes when the user uploads a file.
	FileData []byte
	// FileName is the original filename of the uploaded .torrent file.
	FileName string
	// CreatedAt records when the entry was created for TTL eviction.
	CreatedAt time.Time
}

// Handler dispatches incoming Telegram updates to the appropriate handler
// functions. It owns the per-chat pending torrent state.
type Handler struct {
	sender     Sender
	qbt        qbt.Client
	auth       *Authorizer
	token      string
	httpClient *http.Client
	pending    map[int64]*PendingTorrent
	mu         sync.Mutex
}

// New constructs a Handler and starts the background cleanup goroutine that
// evicts pending torrent entries older than pendingTTL.
// botToken is required to construct the file-download URL for .torrent uploads.
// ctx controls the lifetime of the background cleanup goroutine.
func New(ctx context.Context, sender Sender, qbtClient qbt.Client, auth *Authorizer, botToken string) *Handler {
	h := &Handler{
		sender:     sender,
		qbt:        qbtClient,
		auth:       auth,
		token:      botToken,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		pending:    make(map[int64]*PendingTorrent),
	}
	go h.runCleanup(ctx)
	return h
}

// runCleanup periodically evicts expired pending torrent entries.
// It returns when ctx is canceled.
func (h *Handler) runCleanup(ctx context.Context) {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.evictExpired()
		}
	}
}

// evictExpired removes all pending entries older than pendingTTL.
func (h *Handler) evictExpired() {
	h.mu.Lock()
	defer h.mu.Unlock()
	cutoff := time.Now().Add(-pendingTTL)
	for chatID, pt := range h.pending {
		if pt.CreatedAt.Before(cutoff) {
			delete(h.pending, chatID)
		}
	}
}

// HandleUpdate is the main entry point for incoming Telegram updates.
// It routes callback queries and messages to the appropriate sub-handler.
func (h *Handler) HandleUpdate(ctx context.Context, update tgbotapi.Update) {
	if update.CallbackQuery != nil {
		h.handleCallback(ctx, update.CallbackQuery)
		return
	}

	if update.Message == nil {
		return
	}

	msg := update.Message

	// Authorisation check. msg.From is nil for channel posts.
	if msg.From == nil || !h.auth.IsAllowed(msg.From.ID) {
		h.replyText(msg.Chat.ID, "Access denied.")
		return
	}

	// Command dispatch.
	if msg.IsCommand() {
		h.handleCommand(ctx, msg)
		return
	}

	// Magnet link.
	if strings.Contains(msg.Text, "magnet:?") {
		h.handleMagnet(ctx, msg)
		return
	}

	// .torrent document.
	if msg.Document != nil && strings.HasSuffix(strings.ToLower(msg.Document.FileName), ".torrent") {
		h.handleTorrentFile(ctx, msg)
		return
	}

	// Unknown message — silently ignore to avoid spamming the user.
}

// handleCommand dispatches bot commands (/start, /help, /list, /active, /downloading).
func (h *Handler) handleCommand(ctx context.Context, msg *tgbotapi.Message) {
	switch msg.Command() {
	case "start", "help":
		h.replyText(msg.Chat.ID, HelpText())

	case "list":
		h.sendTorrentPage(ctx, msg.Chat.ID, qbt.FilterAll, 1)

	case "active":
		h.sendTorrentPage(ctx, msg.Chat.ID, qbt.FilterActive, 1)

	case "downloading":
		h.sendTorrentPage(ctx, msg.Chat.ID, qbt.FilterDownloading, 1)

	case "uploading":
		h.sendTorrentPage(ctx, msg.Chat.ID, qbt.FilterUploading, 1)
	}
}

// handleMagnet extracts the first magnet URI from the message text, stores it
// as a pending torrent, fetches available categories, and shows the category
// selection keyboard.
func (h *Handler) handleMagnet(ctx context.Context, msg *tgbotapi.Message) {
	// Extract the magnet URI (everything from "magnet:?" to the next whitespace).
	text := msg.Text
	start := strings.Index(text, "magnet:?")
	if start == -1 {
		return
	}
	end := strings.IndexAny(text[start:], " \t\n\r")
	var magnet string
	if end == -1 {
		magnet = text[start:]
	} else {
		magnet = text[start : start+end]
	}

	h.storePending(msg.Chat.ID, &PendingTorrent{
		MagnetLink: magnet,
		CreatedAt:  time.Now(),
	})

	h.sendCategoryKeyboard(ctx, msg.Chat.ID, "Select category for this torrent:")
}

// handleTorrentFile downloads the .torrent file attached to the message, stores
// it as a pending torrent, fetches categories, and shows the category keyboard.
func (h *Handler) handleTorrentFile(ctx context.Context, msg *tgbotapi.Message) {
	doc := msg.Document

	fileInfo, err := h.sender.GetFile(tgbotapi.FileConfig{FileID: doc.FileID})
	if err != nil {
		h.replyText(msg.Chat.ID, fmt.Sprintf("Failed to get file info: %v", err))
		return
	}

	data, err := h.downloadFile(ctx, fileInfo.FilePath)
	if err != nil {
		log.Printf("bot: download file %s: %v", doc.FileName, err)
		h.replyText(msg.Chat.ID, "Failed to download file. Please try again.")
		return
	}

	h.storePending(msg.Chat.ID, &PendingTorrent{
		FileData:  data,
		FileName:  doc.FileName,
		CreatedAt: time.Now(),
	})

	h.sendCategoryKeyboard(ctx, msg.Chat.ID, "Select category for this torrent:")
}

// downloadFile fetches the file from the Telegram CDN using the bot token.
// Errors are sanitized to avoid leaking the bot token (which appears in the URL).
func (h *Handler) downloadFile(ctx context.Context, filePath string) ([]byte, error) {
	url := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", h.token, filePath)
	data, err := downloadFileURL(ctx, h.httpClient, url)
	if err != nil {
		// Sanitize: don't propagate URL (contains bot token) in error
		return nil, fmt.Errorf("failed to download file %s", filePath)
	}
	return data, nil
}

// downloadFileURL fetches raw bytes from url using the provided client.
// It is a package-level function so that tests can call it directly with
// a local httptest server URL.
func downloadFileURL(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http get: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	return data, nil
}

// sendTorrentPage fetches torrents and sends the requested page to the chat.
// A single API call fetches all matching torrents; paging is done in Go.
func (h *Handler) sendTorrentPage(ctx context.Context, chatID int64, filter qbt.TorrentFilter, page int) {
	var filterPrefix string
	switch filter {
	case qbt.FilterActive:
		filterPrefix = "act"
	case qbt.FilterDownloading:
		filterPrefix = "dw"
	case qbt.FilterUploading:
		filterPrefix = "up"
	default:
		filterPrefix = "all"
	}

	text, kb, err := h.renderTorrentListPage(ctx, filter, filterPrefix, page)
	if err != nil {
		h.replyText(chatID, fmt.Sprintf("Error fetching torrents: %v", err))
		return
	}

	replyMsg := tgbotapi.NewMessage(chatID, text)
	replyMsg.ReplyMarkup = toTGKeyboard(kb)

	if _, err := h.sender.Send(replyMsg); err != nil {
		log.Printf("bot: send error: %v", err)
	}
}

// sendCategoryKeyboard fetches the current qBittorrent categories and sends an
// inline keyboard asking the user to choose one.
func (h *Handler) sendCategoryKeyboard(ctx context.Context, chatID int64, prompt string) {
	cats, err := h.qbt.Categories(ctx)
	if err != nil {
		h.replyText(chatID, fmt.Sprintf("Failed to fetch categories: %v", err))
		return
	}

	kb := formatter.CategoryKeyboard(cats)
	msg := tgbotapi.NewMessage(chatID, prompt)
	msg.ReplyMarkup = toTGKeyboard(kb)

	if _, err := h.sender.Send(msg); err != nil {
		log.Printf("bot: send error: %v", err)
	}
}

// replyText sends a plain-text message to chatID.
func (h *Handler) replyText(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := h.sender.Send(msg); err != nil {
		log.Printf("bot: send error: %v", err)
	}
}

// storePending stores pt under chatID, replacing any existing entry.
func (h *Handler) storePending(chatID int64, pt *PendingTorrent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pending[chatID] = pt
}

// takePending retrieves and removes the pending torrent for chatID.
// Returns nil if no entry exists.
func (h *Handler) takePending(chatID int64) *PendingTorrent {
	h.mu.Lock()
	defer h.mu.Unlock()
	pt, ok := h.pending[chatID]
	if !ok {
		return nil
	}
	delete(h.pending, chatID)
	return pt
}

// editMessageText replaces the text of an existing inline message.
// Uses Request instead of Send because Telegram returns bool, not Message.
func (h *Handler) editMessageText(chatID int64, messageID int, text string, kb *tgbotapi.InlineKeyboardMarkup) {
	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	if kb != nil {
		edit.ReplyMarkup = kb
	}
	if _, err := h.sender.Request(edit); err != nil {
		if !strings.Contains(err.Error(), "message is not modified") {
			log.Printf("bot: edit message error: %v", err)
		}
	}
}

// answerCallback dismisses the loading spinner on a callback query button.
// Uses Request instead of Send because Telegram returns bool, not Message.
func (h *Handler) answerCallback(callbackID string, text string) {
	answer := tgbotapi.NewCallback(callbackID, text)
	if _, err := h.sender.Request(answer); err != nil {
		log.Printf("bot: answer callback error: %v", err)
	}
}

// bytes.NewReader helper — exposed for callback.go use within the package.
func newBytesReader(data []byte) io.Reader {
	return bytes.NewReader(data)
}
