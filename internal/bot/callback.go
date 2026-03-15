package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/home/tt-bot/internal/formatter"
	"github.com/home/tt-bot/internal/qbt"
)

// listTorrentsForFilter fetches torrents from qBittorrent, applying client-side
// post-filtering for virtual filters like FilterDownloading.
func (h *Handler) listTorrentsForFilter(ctx context.Context, filter qbt.TorrentFilter) ([]qbt.Torrent, error) {
	apiFilter := filter
	if filter == qbt.FilterDownloading {
		apiFilter = qbt.FilterAll
	}
	all, err := h.qbt.ListTorrents(ctx, qbt.ListOptions{Filter: apiFilter})
	if err != nil {
		return nil, err
	}
	if filter == qbt.FilterDownloading {
		filtered := make([]qbt.Torrent, 0, len(all))
		for _, t := range all {
			if t.Progress < 1.0 {
				filtered = append(filtered, t)
			}
		}
		return filtered, nil
	}
	return all, nil
}

// filterCharToFilter converts a single-character filter code to a TorrentFilter.
func filterCharToFilter(char string) (qbt.TorrentFilter, bool) {
	switch char {
	case "a":
		return qbt.FilterAll, true
	case "c":
		return qbt.FilterActive, true
	case "d":
		return qbt.FilterDownloading, true
	default:
		return "", false
	}
}

// filterCharToPrefix converts a single-character filter code to the pagination prefix.
func filterCharToPrefix(char string) string {
	switch char {
	case "a":
		return "all"
	case "c":
		return "act"
	case "d":
		return "dw"
	default:
		return "all"
	}
}

// filterToChar converts a TorrentFilter to a single-character code for callbacks.
func filterToChar(filter qbt.TorrentFilter) string {
	switch filter {
	case qbt.FilterActive:
		return "c"
	case qbt.FilterDownloading:
		return "d"
	default:
		return "a"
	}
}

// handleCallback processes all incoming callback queries.
// It parses the callback data prefix and delegates to the appropriate action.
func (h *Handler) handleCallback(ctx context.Context, cq *tgbotapi.CallbackQuery) {
	data := cq.Data

	switch {
	case strings.HasPrefix(data, "cat:"):
		h.handleCategoryCallback(ctx, cq, strings.TrimPrefix(data, "cat:"))

	case strings.HasPrefix(data, "pg:all:"):
		page, err := strconv.Atoi(strings.TrimPrefix(data, "pg:all:"))
		if err != nil {
			h.answerCallback(cq.ID, "Invalid page.")
			return
		}
		h.handlePaginationCallback(ctx, cq, qbt.FilterAll, "all", page)

	case strings.HasPrefix(data, "pg:act:"):
		page, err := strconv.Atoi(strings.TrimPrefix(data, "pg:act:"))
		if err != nil {
			h.answerCallback(cq.ID, "Invalid page.")
			return
		}
		h.handlePaginationCallback(ctx, cq, qbt.FilterActive, "act", page)

	case strings.HasPrefix(data, "pg:dw:"):
		page, err := strconv.Atoi(strings.TrimPrefix(data, "pg:dw:"))
		if err != nil {
			h.answerCallback(cq.ID, "Invalid page.")
			return
		}
		h.handlePaginationCallback(ctx, cq, qbt.FilterDownloading, "dw", page)

	case strings.HasPrefix(data, "sel:"):
		h.handleSelectCallback(ctx, cq, strings.TrimPrefix(data, "sel:"))

	case strings.HasPrefix(data, "pa:"):
		h.handlePauseCallback(ctx, cq, strings.TrimPrefix(data, "pa:"))

	case strings.HasPrefix(data, "re:"):
		h.handleResumeCallback(ctx, cq, strings.TrimPrefix(data, "re:"))

	case strings.HasPrefix(data, "bk:"):
		h.handleBackCallback(ctx, cq, strings.TrimPrefix(data, "bk:"))

	case data == "noop":
		// Page indicator button — dismiss spinner, do nothing.
		h.answerCallback(cq.ID, "")

	default:
		h.answerCallback(cq.ID, "Unknown action.")
	}
}

// handleCategoryCallback looks up the pending torrent for the chat, adds it to
// qBittorrent with the chosen category, and edits the message to confirm.
func (h *Handler) handleCategoryCallback(ctx context.Context, cq *tgbotapi.CallbackQuery, category string) {
	chatID := cq.Message.Chat.ID
	pt := h.takePending(chatID)
	if pt == nil {
		h.answerCallback(cq.ID, "No pending torrent. Please resend the magnet link or file.")
		return
	}

	var addErr error
	if pt.MagnetLink != "" {
		addErr = h.qbt.AddMagnet(ctx, pt.MagnetLink, category)
	} else {
		addErr = h.qbt.AddTorrentFile(ctx, pt.FileName, newBytesReader(pt.FileData), category)
	}

	if addErr != nil {
		h.answerCallback(cq.ID, fmt.Sprintf("Error: %v", addErr))
		// Edit message to show the error so the user sees it even after the
		// spinner disappears.
		h.editMessageText(chatID, cq.Message.MessageID,
			fmt.Sprintf("Failed to add torrent: %v", addErr), nil)
		return
	}

	h.answerCallback(cq.ID, "Torrent added!")

	confirmText := "Torrent added!"
	if category != "" {
		confirmText = fmt.Sprintf("Torrent added to %s!", category)
	}
	h.editMessageText(chatID, cq.Message.MessageID, confirmText, nil)
}

// handlePaginationCallback fetches the requested page of torrents and edits the
// existing message in place.
func (h *Handler) handlePaginationCallback(
	ctx context.Context,
	cq *tgbotapi.CallbackQuery,
	filter qbt.TorrentFilter,
	filterPrefix string,
	page int,
) {
	chatID := cq.Message.Chat.ID
	messageID := cq.Message.MessageID

	text, kb, err := h.renderTorrentListPage(ctx, filter, filterPrefix, page)
	if err != nil {
		h.answerCallback(cq.ID, fmt.Sprintf("Error: %v", err))
		return
	}

	tgKB := toTGKeyboard(kb)
	h.answerCallback(cq.ID, "")
	h.editMessageText(chatID, messageID, text, &tgKB)
}

// renderTorrentListPage fetches torrents and builds the list text and combined
// keyboard (pagination + selection). This is shared by sendTorrentPage,
// handlePaginationCallback, and handleBackCallback.
func (h *Handler) renderTorrentListPage(
	ctx context.Context,
	filter qbt.TorrentFilter,
	filterPrefix string,
	page int,
) (string, formatter.Keyboard, error) {
	all, err := h.listTorrentsForFilter(ctx, filter)
	if err != nil {
		return "", nil, err
	}

	totalPages := formatter.TotalPages(len(all), formatter.TorrentsPerPage)
	offset := (page - 1) * formatter.TorrentsPerPage
	end := offset + formatter.TorrentsPerPage
	if end > len(all) {
		end = len(all)
	}
	var torrents []qbt.Torrent
	if offset < len(all) {
		torrents = all[offset:end]
	}
	text := formatter.FormatTorrentList(torrents, page, totalPages)

	paginationKB := formatter.PaginationKeyboard(page, totalPages, filterPrefix)
	selectionKB := formatter.TorrentSelectionKeyboard(torrents, filterToChar(filter), page)

	// Combine: pagination row(s) first, then selection row(s).
	combined := make(formatter.Keyboard, 0, len(paginationKB)+len(selectionKB))
	combined = append(combined, paginationKB...)
	combined = append(combined, selectionKB...)

	return text, combined, nil
}

// parseControlCallback parses callback data in the format "<filterChar>:<page>:<hash>".
// Returns the filter char, page number, and torrent hash.
func parseControlCallback(data string) (filterChar string, page int, hash string, err error) {
	parts := strings.SplitN(data, ":", 3)
	if len(parts) != 3 {
		return "", 0, "", fmt.Errorf("invalid callback format")
	}
	filterChar = parts[0]
	page, err = strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, "", fmt.Errorf("invalid page: %w", err)
	}
	hash = parts[2]
	return filterChar, page, hash, nil
}

// findTorrentByHash searches for a torrent with the given hash in the list.
func findTorrentByHash(torrents []qbt.Torrent, hash string) (qbt.Torrent, bool) {
	for _, t := range torrents {
		if t.Hash == hash {
			return t, true
		}
	}
	return qbt.Torrent{}, false
}

// handleSelectCallback displays a torrent detail view when a user selects a
// torrent from the list.
func (h *Handler) handleSelectCallback(ctx context.Context, cq *tgbotapi.CallbackQuery, data string) {
	filterChar, page, hash, err := parseControlCallback(data)
	if err != nil {
		h.answerCallback(cq.ID, "Invalid selection.")
		return
	}

	filter, ok := filterCharToFilter(filterChar)
	if !ok {
		h.answerCallback(cq.ID, "Invalid filter.")
		return
	}

	all, err := h.listTorrentsForFilter(ctx, filter)
	if err != nil {
		h.answerCallback(cq.ID, fmt.Sprintf("Error: %v", err))
		return
	}

	torrent, found := findTorrentByHash(all, hash)
	if !found {
		h.answerCallback(cq.ID, "Torrent not found.")
		return
	}

	text := formatter.FormatTorrentDetail(torrent)
	kb := toTGKeyboard(formatter.TorrentDetailKeyboard(hash, filterChar, page, torrent.State))

	h.answerCallback(cq.ID, "")
	h.editMessageText(cq.Message.Chat.ID, cq.Message.MessageID, text, &kb)
}

// handlePauseCallback pauses a torrent and refreshes the detail view.
func (h *Handler) handlePauseCallback(ctx context.Context, cq *tgbotapi.CallbackQuery, data string) {
	h.handleTorrentAction(ctx, cq, data, true)
}

// handleResumeCallback resumes a torrent and refreshes the detail view.
func (h *Handler) handleResumeCallback(ctx context.Context, cq *tgbotapi.CallbackQuery, data string) {
	h.handleTorrentAction(ctx, cq, data, false)
}

// handleTorrentAction is the shared logic for pause and resume callbacks.
func (h *Handler) handleTorrentAction(ctx context.Context, cq *tgbotapi.CallbackQuery, data string, pause bool) {
	filterChar, page, hash, err := parseControlCallback(data)
	if err != nil {
		h.answerCallback(cq.ID, "Invalid action.")
		return
	}

	filter, ok := filterCharToFilter(filterChar)
	if !ok {
		h.answerCallback(cq.ID, "Invalid filter.")
		return
	}

	if pause {
		err = h.qbt.PauseTorrents(ctx, []string{hash})
	} else {
		err = h.qbt.ResumeTorrents(ctx, []string{hash})
	}
	if err != nil {
		h.answerCallback(cq.ID, fmt.Sprintf("Error: %v", err))
		return
	}

	// Re-fetch the torrent to display the updated state.
	all, listErr := h.listTorrentsForFilter(ctx, filter)
	if listErr != nil {
		actionText := "Paused"
		if !pause {
			actionText = "Resumed"
		}
		h.answerCallback(cq.ID, actionText)
		return
	}

	torrent, found := findTorrentByHash(all, hash)
	if !found {
		h.answerCallback(cq.ID, "Torrent not found after action.")
		return
	}

	text := formatter.FormatTorrentDetail(torrent)
	kb := toTGKeyboard(formatter.TorrentDetailKeyboard(hash, filterChar, page, torrent.State))

	actionText := "Paused"
	if !pause {
		actionText = "Resumed"
	}
	h.answerCallback(cq.ID, actionText)
	h.editMessageText(cq.Message.Chat.ID, cq.Message.MessageID, text, &kb)
}

// handleBackCallback returns from the detail view to the torrent list.
func (h *Handler) handleBackCallback(ctx context.Context, cq *tgbotapi.CallbackQuery, data string) {
	parts := strings.SplitN(data, ":", 2)
	if len(parts) != 2 {
		h.answerCallback(cq.ID, "Invalid navigation.")
		return
	}

	filterChar := parts[0]
	page, err := strconv.Atoi(parts[1])
	if err != nil {
		h.answerCallback(cq.ID, "Invalid page.")
		return
	}

	filter, ok := filterCharToFilter(filterChar)
	if !ok {
		h.answerCallback(cq.ID, "Invalid filter.")
		return
	}

	filterPrefix := filterCharToPrefix(filterChar)
	text, kb, listErr := h.renderTorrentListPage(ctx, filter, filterPrefix, page)
	if listErr != nil {
		h.answerCallback(cq.ID, fmt.Sprintf("Error: %v", listErr))
		return
	}

	tgKB := toTGKeyboard(kb)
	h.answerCallback(cq.ID, "")
	h.editMessageText(cq.Message.Chat.ID, cq.Message.MessageID, text, &tgKB)
}
