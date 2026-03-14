package bot

import (
	"bytes"
	"context"
	"fmt"
	"io"
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
)

// helpText is sent in response to /start and /help.
const helpText = `tt-bot — qBittorrent Telegram controller

Commands:
  /list   — list all torrents (paginated)
  /active — list active torrents (paginated)
  /help   — show this message

You can also send:
  • A magnet link (magnet:?…) — prompts for category then adds it
  • A .torrent file — prompts for category then adds it`

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
	sender  Sender
	qbt     qbt.Client
	auth    *Authorizer
	token   string
	pending map[int64]*PendingTorrent
	mu      sync.Mutex
}

// New constructs a Handler and starts the background cleanup goroutine that
// evicts pending torrent entries older than pendingTTL.
// botToken is required to construct the file-download URL for .torrent uploads.
func New(sender Sender, qbtClient qbt.Client, auth *Authorizer, botToken string) *Handler {
	h := &Handler{
		sender:  sender,
		qbt:     qbtClient,
		auth:    auth,
		token:   botToken,
		pending: make(map[int64]*PendingTorrent),
	}
	go h.runCleanup()
	return h
}

// runCleanup periodically evicts expired pending torrent entries.
func (h *Handler) runCleanup() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for range ticker.C {
		h.evictExpired()
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

	// Authorisation check.
	if !h.auth.IsAllowed(msg.From.ID) {
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

// handleCommand dispatches bot commands (/start, /help, /list, /active).
func (h *Handler) handleCommand(ctx context.Context, msg *tgbotapi.Message) {
	switch msg.Command() {
	case "start", "help":
		h.replyText(msg.Chat.ID, helpText)

	case "list":
		h.sendTorrentPage(ctx, msg.Chat.ID, qbt.FilterAll, 1)

	case "active":
		h.sendTorrentPage(ctx, msg.Chat.ID, qbt.FilterActive, 1)
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
		h.replyText(msg.Chat.ID, fmt.Sprintf("Failed to download file: %v", err))
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
func (h *Handler) downloadFile(ctx context.Context, filePath string) ([]byte, error) {
	url := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", h.token, filePath)
	return downloadFileURL(ctx, url)
}

// downloadFileURL fetches raw bytes from url. It is a package-level function
// so that tests can call it directly with a local httptest server URL.
func downloadFileURL(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	return data, nil
}

// sendTorrentPage fetches one page of torrents and sends it to the chat.
// On success the message includes a pagination keyboard.
func (h *Handler) sendTorrentPage(ctx context.Context, chatID int64, filter qbt.TorrentFilter, page int) {
	// Fetch the page of torrents.
	offset := (page - 1) * formatter.TorrentsPerPage
	torrents, err := h.qbt.ListTorrents(ctx, qbt.ListOptions{
		Filter: filter,
		Limit:  formatter.TorrentsPerPage,
		Offset: offset,
	})
	if err != nil {
		h.replyText(chatID, fmt.Sprintf("Error fetching torrents: %v", err))
		return
	}

	// Determine total pages by fetching all (no limit) — count only.
	// We need total count; fetch all with no limit and count.
	all, err := h.qbt.ListTorrents(ctx, qbt.ListOptions{Filter: filter})
	if err != nil {
		h.replyText(chatID, fmt.Sprintf("Error fetching torrent count: %v", err))
		return
	}

	totalPages := formatter.TotalPages(len(all), formatter.TorrentsPerPage)
	text := formatter.FormatTorrentList(torrents, page, totalPages)

	filterPrefix := "all"
	if filter == qbt.FilterActive {
		filterPrefix = "act"
	}

	kb := formatter.PaginationKeyboard(page, totalPages, filterPrefix)
	replyMsg := tgbotapi.NewMessage(chatID, text)
	replyMsg.ReplyMarkup = toTGKeyboard(kb)

	if _, err := h.sender.Send(replyMsg); err != nil {
		// Best-effort; log via stderr is out of scope for this package.
		_ = err
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
		_ = err
	}
}

// replyText sends a plain-text message to chatID.
func (h *Handler) replyText(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := h.sender.Send(msg); err != nil {
		_ = err
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
func (h *Handler) editMessageText(chatID int64, messageID int, text string, kb *tgbotapi.InlineKeyboardMarkup) {
	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	if kb != nil {
		edit.ReplyMarkup = kb
	}
	if _, err := h.sender.Send(edit); err != nil {
		_ = err
	}
}

// answerCallback dismisses the loading spinner on a callback query button.
func (h *Handler) answerCallback(callbackID string, text string) {
	answer := tgbotapi.NewCallback(callbackID, text)
	if _, err := h.sender.Send(answer); err != nil {
		_ = err
	}
}

// bytes.NewReader helper — exposed for callback.go use within the package.
func newBytesReader(data []byte) io.Reader {
	return bytes.NewReader(data)
}
