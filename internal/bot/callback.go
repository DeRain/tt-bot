package bot

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/home/tt-bot/internal/formatter"
	"github.com/home/tt-bot/internal/qbt"
)

// listTorrentsForFilter fetches torrents from qBittorrent, applying client-side
// post-filtering for virtual filters like FilterDownloading and FilterUploading.
func (h *Handler) listTorrentsForFilter(ctx context.Context, filter qbt.TorrentFilter) ([]qbt.Torrent, error) {
	apiFilter := filter
	if filter == qbt.FilterDownloading || filter == qbt.FilterUploading {
		apiFilter = qbt.FilterAll
	}
	all, err := h.qbt.ListTorrents(ctx, qbt.ListOptions{Filter: apiFilter})
	if err != nil {
		return nil, err
	}
	switch filter {
	case qbt.FilterDownloading:
		filtered := make([]qbt.Torrent, 0, len(all))
		for _, t := range all {
			// Progress is set to exactly 1.0 by qBittorrent on completion; direct equality is safe here.
			if t.Progress < 1.0 {
				filtered = append(filtered, t)
			}
		}
		return filtered, nil
	case qbt.FilterUploading:
		filtered := make([]qbt.Torrent, 0, len(all))
		for _, t := range all {
			// Progress is set to exactly 1.0 by qBittorrent on completion; direct equality is safe here.
			// Exclude paused and stopped uploads so only active seeders are shown.
			if t.Progress == 1.0 && t.State != "pausedUP" && t.State != "stoppedUP" {
				filtered = append(filtered, t)
			}
		}
		return filtered, nil
	default:
		return all, nil
	}
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
	case "u":
		return qbt.FilterUploading, true
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
	case "u":
		return "up"
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
	case qbt.FilterUploading:
		return "u"
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

	case strings.HasPrefix(data, "pg:fl:"):
		h.handleFilesPageNavCallback(ctx, cq, strings.TrimPrefix(data, "pg:fl:"))

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

	case strings.HasPrefix(data, "pg:up:"):
		page, err := strconv.Atoi(strings.TrimPrefix(data, "pg:up:"))
		if err != nil {
			h.answerCallback(cq.ID, "Invalid page.")
			return
		}
		h.handlePaginationCallback(ctx, cq, qbt.FilterUploading, "up", page)

	case strings.HasPrefix(data, "sel:"):
		h.handleSelectCallback(ctx, cq, strings.TrimPrefix(data, "sel:"))

	case strings.HasPrefix(data, "pa:"):
		h.handlePauseCallback(ctx, cq, strings.TrimPrefix(data, "pa:"))

	case strings.HasPrefix(data, "re:"):
		h.handleResumeCallback(ctx, cq, strings.TrimPrefix(data, "re:"))

	case strings.HasPrefix(data, "bk:fl:"):
		h.handleBackFromFilesCallback(ctx, cq, strings.TrimPrefix(data, "bk:fl:"))

	case strings.HasPrefix(data, "bk:"):
		h.handleBackCallback(ctx, cq, strings.TrimPrefix(data, "bk:"))

	case strings.HasPrefix(data, "rm:"):
		h.handleRemoveConfirmCallback(ctx, cq, strings.TrimPrefix(data, "rm:"))

	case strings.HasPrefix(data, "rd:"):
		h.handleRemoveDeleteCallback(ctx, cq, strings.TrimPrefix(data, "rd:"), false)

	case strings.HasPrefix(data, "rf:"):
		h.handleRemoveDeleteCallback(ctx, cq, strings.TrimPrefix(data, "rf:"), true)

	case strings.HasPrefix(data, "rc:"):
		h.handleRemoveCancelCallback(ctx, cq, strings.TrimPrefix(data, "rc:"))

	case strings.HasPrefix(data, "fl:"):
		h.handleFilesPageCallback(ctx, cq, strings.TrimPrefix(data, "fl:"))

	case strings.HasPrefix(data, "fs:"):
		h.handleFileSelectCallback(ctx, cq, strings.TrimPrefix(data, "fs:"))

	case strings.HasPrefix(data, "fp:"):
		h.handleFilePriorityCallback(ctx, cq, strings.TrimPrefix(data, "fp:"))

	case strings.HasPrefix(data, "sr:") || strings.HasPrefix(data, "sp:") ||
		strings.HasPrefix(data, "sx:") || strings.HasPrefix(data, "ss:") ||
		strings.HasPrefix(data, "sc:") || strings.HasPrefix(data, "sb:") ||
		strings.HasPrefix(data, "dp:"):
		h.handleSearchCallback(ctx, cq, data)

	case data == "noop":
		h.answerCallback(cq.ID, "")

	default:
		h.answerCallback(cq.ID, "Unknown action.")
	}
}

func (h *Handler) handleSearchCallback(ctx context.Context, cq *tgbotapi.CallbackQuery, data string) {
	switch {
	case strings.HasPrefix(data, "sr:"):
		h.handleSearchSelectCallback(ctx, cq, strings.TrimPrefix(data, "sr:"))
	case strings.HasPrefix(data, "sp:"):
		h.handleSearchPageCallback(ctx, cq, strings.TrimPrefix(data, "sp:"))
	case strings.HasPrefix(data, "sx:"):
		h.handleSearchCancelCallback(ctx, cq, strings.TrimPrefix(data, "sx:"))
	case strings.HasPrefix(data, "ss:"):
		h.handleSearchSortCallback(ctx, cq, strings.TrimPrefix(data, "ss:"))
	case strings.HasPrefix(data, "sc:"):
		h.handleSearchConfirmCallback(ctx, cq, strings.TrimPrefix(data, "sc:"))
	case strings.HasPrefix(data, "sb:"):
		h.handleSearchBackCallback(ctx, cq, strings.TrimPrefix(data, "sb:"))
	case strings.HasPrefix(data, "dp:"):
		h.handleDescriptionPageCallback(ctx, cq, strings.TrimPrefix(data, "dp:"))
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
		_ = h.editMessageText(chatID, cq.Message.MessageID,
			fmt.Sprintf("Failed to add torrent: %v", addErr), nil)
		return
	}

	h.answerCallback(cq.ID, "Torrent added!")

	confirmText := "Torrent added!"
	if category != "" {
		confirmText = fmt.Sprintf("Torrent added to %s!", category)
	}
	_ = h.editMessageText(chatID, cq.Message.MessageID, confirmText, nil)
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
	_ = h.editMessageText(chatID, messageID, text, &tgKB)

	h.liveViewsMu.Lock()
	if lv, ok := h.liveViews[chatID]; ok && lv.MessageID == messageID {
		lv.Page = page
		lv.Filter = filter
		lv.FilterChar = filterToChar(filter)
		lv.LastContentHash = ""      // force refresh on next tick
		lv.RegisteredAt = time.Now() // reset TTL deadline
	}
	h.liveViewsMu.Unlock()
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
		return "", 0, "", errors.New("invalid callback format")
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

	if _, ok := filterCharToFilter(filterChar); !ok {
		h.answerCallback(cq.ID, "Invalid filter.")
		return
	}

	all, err := h.listTorrentsForFilter(ctx, qbt.FilterAll)
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
	_ = h.editMessageText(cq.Message.Chat.ID, cq.Message.MessageID, text, &kb)

	h.registerLiveView(cq.Message.Chat.ID, &LiveView{
		ChatID:      cq.Message.Chat.ID,
		MessageID:   cq.Message.MessageID,
		ViewType:    ViewDetail,
		TorrentHash: hash,
		FilterChar:  filterChar,
		Page:        page,
	})
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

	if _, ok := filterCharToFilter(filterChar); !ok {
		h.answerCallback(cq.ID, "Invalid filter.")
		return
	}

	// Fetch current state before the action so we can detect when it changes.
	preAll, preErr := h.listTorrentsForFilter(ctx, qbt.FilterAll)
	var oldState string
	if preErr == nil {
		if t, found := findTorrentByHash(preAll, hash); found {
			oldState = t.State
		}
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

	actionText := "Paused"
	if !pause {
		actionText = "Resumed"
	}

	// Poll until qBittorrent reflects the state change.
	torrent, changed := h.awaitStateChange(ctx, hash, oldState)
	if !changed {
		// Timeout or context canceled — still show the action confirmation.
		h.answerCallback(cq.ID, actionText)
		if torrent.Hash != "" {
			text := formatter.FormatTorrentDetail(torrent)
			kb := toTGKeyboard(formatter.TorrentDetailKeyboard(hash, filterChar, page, torrent.State))
			_ = h.editMessageText(cq.Message.Chat.ID, cq.Message.MessageID, text, &kb)

			h.registerLiveView(cq.Message.Chat.ID, &LiveView{
				ChatID:      cq.Message.Chat.ID,
				MessageID:   cq.Message.MessageID,
				ViewType:    ViewDetail,
				TorrentHash: hash,
				FilterChar:  filterChar,
				Page:        page,
			})
		}
		return
	}

	text := formatter.FormatTorrentDetail(torrent)
	kb := toTGKeyboard(formatter.TorrentDetailKeyboard(hash, filterChar, page, torrent.State))

	h.answerCallback(cq.ID, actionText)
	_ = h.editMessageText(cq.Message.Chat.ID, cq.Message.MessageID, text, &kb)

	h.registerLiveView(cq.Message.Chat.ID, &LiveView{
		ChatID:      cq.Message.Chat.ID,
		MessageID:   cq.Message.MessageID,
		ViewType:    ViewDetail,
		TorrentHash: hash,
		FilterChar:  filterChar,
		Page:        page,
	})
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
	_ = h.editMessageText(cq.Message.Chat.ID, cq.Message.MessageID, text, &tgKB)

	h.registerLiveView(cq.Message.Chat.ID, &LiveView{
		ChatID:     cq.Message.Chat.ID,
		MessageID:  cq.Message.MessageID,
		ViewType:   ViewList,
		Filter:     filter,
		FilterChar: filterChar,
		Page:       page,
	})
}

// It parses rm:<filterChar>:<page>:<hash>, fetches the torrent name, and renders
// the confirmation message with RemoveConfirmKeyboard. No qBittorrent mutation occurs.
func (h *Handler) handleRemoveConfirmCallback(ctx context.Context, cq *tgbotapi.CallbackQuery, data string) {
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

	all, err := h.listTorrentsForFilter(ctx, filter)
	if err != nil {
		h.answerCallback(cq.ID, fmt.Sprintf("Error: %v", err))
		return
	}

	torrent, found := findTorrentByHash(all, hash)
	if !found {
		// Torrent disappeared between list view and clicking Remove; go back to list.
		filterPrefix := filterCharToPrefix(filterChar)
		text, kb, listErr := h.renderTorrentListPage(ctx, filter, filterPrefix, page)
		if listErr != nil {
			h.answerCallback(cq.ID, "Torrent not found.")
			return
		}
		tgKB := toTGKeyboard(kb)
		h.answerCallback(cq.ID, "Torrent not found.")
		_ = h.editMessageText(cq.Message.Chat.ID, cq.Message.MessageID, text, &tgKB)
		return
	}

	text := formatter.FormatRemoveConfirmation(torrent.Name)
	kb := toTGKeyboard(formatter.RemoveConfirmKeyboard(hash, filterChar, page))

	h.answerCallback(cq.ID, "")
	_ = h.editMessageText(cq.Message.Chat.ID, cq.Message.MessageID, text, &kb)
	// Confirmation view is not auto-refreshed — deregister any live view.
	h.deregisterLiveView(cq.Message.Chat.ID, cq.Message.MessageID)
}

// handleRemoveDeleteCallback handles both rd: (deleteFiles=false) and rf: (deleteFiles=true).
// It calls DeleteTorrents then navigates back to the torrent list at the original filter/page.
func (h *Handler) handleRemoveDeleteCallback(ctx context.Context, cq *tgbotapi.CallbackQuery, data string, deleteFiles bool) {
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

	if err := h.qbt.DeleteTorrents(ctx, []string{hash}, deleteFiles); err != nil {
		h.answerCallback(cq.ID, fmt.Sprintf("Error: %v", err))
		return
	}

	filterPrefix := filterCharToPrefix(filterChar)
	text, kb, listErr := h.renderTorrentListPage(ctx, filter, filterPrefix, page)
	if listErr != nil {
		h.answerCallback(cq.ID, "Removed.")
		_ = h.editMessageText(cq.Message.Chat.ID, cq.Message.MessageID, "Removed.", nil)
		h.deregisterLiveView(cq.Message.Chat.ID, cq.Message.MessageID)
		return
	}

	tgKB := toTGKeyboard(kb)
	h.answerCallback(cq.ID, "Removed.")
	_ = h.editMessageText(cq.Message.Chat.ID, cq.Message.MessageID, text, &tgKB)

	h.registerLiveView(cq.Message.Chat.ID, &LiveView{
		ChatID:     cq.Message.Chat.ID,
		MessageID:  cq.Message.MessageID,
		ViewType:   ViewList,
		Filter:     filter,
		FilterChar: filterChar,
		Page:       page,
	})
}

// isValidFilePriority reports whether p is one of the four defined qBittorrent
// per-file priority values (skip, normal, high, maximum).
func isValidFilePriority(p int) bool {
	switch qbt.FilePriority(p) {
	case qbt.FilePrioritySkip, qbt.FilePriorityNormal,
		qbt.FilePriorityHigh, qbt.FilePriorityMaximum:
		return true
	}
	return false
}

// parseFileSelectCallback parses fs:<hash>:<fileIndex>:<filePage>:<filterChar>:<listPage>.
// Returns an error if the format is invalid or fileIndex is negative.
//
//nolint:gocritic // callback data parser with multiple extracted fields
func parseFileSelectCallback(data string) (hash string, fileIndex, filePage int, filterChar string, listPage int, err error) {
	parts := strings.SplitN(data, ":", 5)
	if len(parts) != 5 {
		return "", 0, 0, "", 0, errors.New("invalid fs: format")
	}
	hash = parts[0]
	fileIndex, err = strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, 0, "", 0, fmt.Errorf("invalid fileIndex: %w", err)
	}
	if fileIndex < 0 {
		return "", 0, 0, "", 0, errors.New("fileIndex must be non-negative")
	}
	filePage, err = strconv.Atoi(parts[2])
	if err != nil {
		return "", 0, 0, "", 0, fmt.Errorf("invalid filePage: %w", err)
	}
	filterChar = parts[3]
	listPage, err = strconv.Atoi(parts[4])
	if err != nil {
		return "", 0, 0, "", 0, fmt.Errorf("invalid listPage: %w", err)
	}
	return hash, fileIndex, filePage, filterChar, listPage, nil
}

// parseFilePriorityCallback parses fp:<hash>:<fileIndex>:<priority>:<filePage>:<filterChar>:<listPage>.
// Returns an error if the format is invalid or fileIndex is negative.
//
//nolint:gocritic // callback data parser with multiple extracted fields
func parseFilePriorityCallback(data string) (hash string, fileIndex, priority, filePage int, filterChar string, listPage int, err error) {
	parts := strings.SplitN(data, ":", 6)
	if len(parts) != 6 {
		return "", 0, 0, 0, "", 0, errors.New("invalid fp: format")
	}
	hash = parts[0]
	fileIndex, err = strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, 0, 0, "", 0, fmt.Errorf("invalid fileIndex: %w", err)
	}
	if fileIndex < 0 {
		return "", 0, 0, 0, "", 0, errors.New("fileIndex must be non-negative")
	}
	priority, err = strconv.Atoi(parts[2])
	if err != nil {
		return "", 0, 0, 0, "", 0, fmt.Errorf("invalid priority: %w", err)
	}
	filePage, err = strconv.Atoi(parts[3])
	if err != nil {
		return "", 0, 0, 0, "", 0, fmt.Errorf("invalid filePage: %w", err)
	}
	filterChar = parts[4]
	listPage, err = strconv.Atoi(parts[5])
	if err != nil {
		return "", 0, 0, 0, "", 0, fmt.Errorf("invalid listPage: %w", err)
	}
	return hash, fileIndex, priority, filePage, filterChar, listPage, nil
}

// parseFilesOpenCallback parses fl:<filterChar>:<listPage>:<hash>.
func parseFilesOpenCallback(data string) (filterChar string, listPage int, hash string, err error) {
	parts := strings.SplitN(data, ":", 3)
	if len(parts) != 3 {
		return "", 0, "", errors.New("invalid fl: format")
	}
	filterChar = parts[0]
	listPage, err = strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, "", fmt.Errorf("invalid listPage: %w", err)
	}
	hash = parts[2]
	return filterChar, listPage, hash, nil
}

// parseFilesNavCallback parses pg:fl:<hash>:<filePage>:<filterChar>:<listPage>.
func parseFilesNavCallback(data string) (hash string, filePage int, filterChar string, listPage int, err error) {
	parts := strings.SplitN(data, ":", 4)
	if len(parts) != 4 {
		return "", 0, "", 0, errors.New("invalid pg:fl: format")
	}
	hash = parts[0]
	filePage, err = strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, "", 0, fmt.Errorf("invalid filePage: %w", err)
	}
	filterChar = parts[2]
	listPage, err = strconv.Atoi(parts[3])
	if err != nil {
		return "", 0, "", 0, fmt.Errorf("invalid listPage: %w", err)
	}
	return hash, filePage, filterChar, listPage, nil
}

// parseBackFromFilesCallback parses bk:fl:<filterChar>:<listPage>:<hash>.
func parseBackFromFilesCallback(data string) (filterChar string, listPage int, hash string, err error) {
	parts := strings.SplitN(data, ":", 3)
	if len(parts) != 3 {
		return "", 0, "", errors.New("invalid bk:fl: format")
	}
	filterChar = parts[0]
	listPage, err = strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, "", fmt.Errorf("invalid listPage: %w", err)
	}
	hash = parts[2]
	return filterChar, listPage, hash, nil
}

// renderFilesPage fetches files for hash and builds the message text and keyboard
// for the given filePage. It is shared by handleFilesPageCallback,
// handleFilesPageNavCallback, and handleFilePriorityCallback.
func (h *Handler) renderFilesPage(
	ctx context.Context,
	hash string,
	torrentName string,
	fps formatter.FilesPageState,
) (string, formatter.Keyboard, error) {
	files, err := h.qbt.ListFiles(ctx, hash)
	if err != nil {
		return "", nil, err
	}

	totalFilePages := formatter.TotalPages(len(files), formatter.FilesPerPage)
	if fps.FilePage < 1 {
		fps.FilePage = 1
	}
	if fps.FilePage > totalFilePages {
		fps.FilePage = totalFilePages
	}

	offset := (fps.FilePage - 1) * formatter.FilesPerPage
	end := offset + formatter.FilesPerPage
	if end > len(files) {
		end = len(files)
	}
	var pageFiles []qbt.TorrentFile
	if offset < len(files) {
		pageFiles = files[offset:end]
	}

	text := formatter.FormatFileList(torrentName, pageFiles, fps.FilePage, totalFilePages)
	kb := formatter.FileListKeyboard(pageFiles, hash, offset, totalFilePages, fps)
	return text, kb, nil
}

// handleFilesPageCallback handles fl:<filterChar>:<listPage>:<hash> — opens the
// first page of a torrent's file list from the detail view.
func (h *Handler) handleFilesPageCallback(ctx context.Context, cq *tgbotapi.CallbackQuery, data string) {
	filterChar, listPage, hash, err := parseFilesOpenCallback(data)
	if err != nil {
		h.answerCallback(cq.ID, "Invalid action.")
		return
	}

	// Fetch torrent name for the header.
	filter, ok := filterCharToFilter(filterChar)
	if !ok {
		h.answerCallback(cq.ID, "Invalid filter.")
		return
	}
	all, listErr := h.listTorrentsForFilter(ctx, filter)
	torrentName := hash
	if listErr == nil {
		if t, found := findTorrentByHash(all, hash); found {
			torrentName = t.Name
		}
	}

	text, kb, err := h.renderFilesPage(ctx, hash, torrentName, formatter.FilesPageState{FilePage: 1, FilterChar: filterChar, ListPage: listPage})
	if err != nil {
		h.answerCallback(cq.ID, "Failed to load files.")
		return
	}

	tgKB := toTGKeyboard(kb)
	h.answerCallback(cq.ID, "")
	_ = h.editMessageText(cq.Message.Chat.ID, cq.Message.MessageID, text, &tgKB)
}

// handleFilesPageNavCallback handles pg:fl:<hash>:<filePage>:<filterChar>:<listPage> —
// navigates between file list pages.
func (h *Handler) handleFilesPageNavCallback(ctx context.Context, cq *tgbotapi.CallbackQuery, data string) {
	hash, filePage, filterChar, listPage, err := parseFilesNavCallback(data)
	if err != nil {
		h.answerCallback(cq.ID, "Invalid action.")
		return
	}

	// Fetch torrent name for the header.
	filter, ok := filterCharToFilter(filterChar)
	if !ok {
		h.answerCallback(cq.ID, "Invalid filter.")
		return
	}
	all, listErr := h.listTorrentsForFilter(ctx, filter)
	torrentName := hash
	if listErr == nil {
		if t, found := findTorrentByHash(all, hash); found {
			torrentName = t.Name
		}
	}

	fps := formatter.FilesPageState{
		FilePage:   filePage,
		FilterChar: filterChar,
		ListPage:   listPage,
	}
	text, kb, err := h.renderFilesPage(ctx, hash, torrentName, fps)
	if err != nil {
		h.answerCallback(cq.ID, "Failed to load files.")
		return
	}

	tgKB := toTGKeyboard(kb)
	h.answerCallback(cq.ID, "")
	_ = h.editMessageText(cq.Message.Chat.ID, cq.Message.MessageID, text, &tgKB)
}

// handleBackFromFilesCallback handles bk:fl:<filterChar>:<listPage>:<hash> — returns
// from the file list view to the torrent detail view.
func (h *Handler) handleBackFromFilesCallback(ctx context.Context, cq *tgbotapi.CallbackQuery, data string) {
	filterChar, listPage, hash, err := parseBackFromFilesCallback(data)
	if err != nil {
		h.answerCallback(cq.ID, "Invalid action.")
		return
	}

	if _, ok := filterCharToFilter(filterChar); !ok {
		h.answerCallback(cq.ID, "Invalid filter.")
		return
	}

	all, err := h.listTorrentsForFilter(ctx, qbt.FilterAll)
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
	kb := toTGKeyboard(formatter.TorrentDetailKeyboard(hash, filterChar, listPage, torrent.State))

	h.answerCallback(cq.ID, "")
	_ = h.editMessageText(cq.Message.Chat.ID, cq.Message.MessageID, text, &kb)
}

// handleFileSelectCallback handles fs: callbacks — showing the priority selector for a file.
// Format: fs:<hash>:<fileIndex>:<filePage>:<filterChar>:<listPage>.
func (h *Handler) handleFileSelectCallback(ctx context.Context, cq *tgbotapi.CallbackQuery, data string) {
	hash, fileIndex, filePage, filterChar, listPage, err := parseFileSelectCallback(data)
	if err != nil {
		h.answerCallback(cq.ID, "Invalid action.")
		return
	}

	// Fetch current priority for this file to mark it in the keyboard.
	files, listErr := h.qbt.ListFiles(ctx, hash)
	var currentPriority qbt.FilePriority
	if listErr == nil {
		for _, f := range files {
			if f.Index == fileIndex {
				currentPriority = f.Priority
				break
			}
		}
	}

	fps := formatter.FilesPageState{FilePage: filePage, FilterChar: filterChar, ListPage: listPage}
	kb := toTGKeyboard(formatter.PriorityKeyboard(hash, fileIndex, currentPriority, fps))
	h.answerCallback(cq.ID, "")
	_ = h.editMessageText(cq.Message.Chat.ID, cq.Message.MessageID, "Select priority:", &kb)
}

// handleFilePriorityCallback handles fp: callbacks — setting a file's download priority.
// Format: fp:<hash>:<fileIndex>:<priority>:<filePage>:<filterChar>:<listPage>.
func (h *Handler) handleFilePriorityCallback(ctx context.Context, cq *tgbotapi.CallbackQuery, data string) {
	hash, fileIndex, priority, filePage, filterChar, listPage, err := parseFilePriorityCallback(data)
	if err != nil {
		h.answerCallback(cq.ID, "Invalid action.")
		return
	}
	if !isValidFilePriority(priority) {
		h.answerCallback(cq.ID, "Invalid priority.")
		return
	}
	if err := h.qbt.SetFilePriority(ctx, hash, []int{fileIndex}, qbt.FilePriority(priority)); err != nil {
		h.answerCallback(cq.ID, fmt.Sprintf("Failed to set priority: %v", err))
		return
	}

	// Re-fetch the file list to show the updated priority.
	filter, ok := filterCharToFilter(filterChar)
	if !ok {
		h.answerCallback(cq.ID, "Priority updated.")
		return
	}
	all, listErr := h.listTorrentsForFilter(ctx, filter)
	torrentName := hash
	if listErr == nil {
		if t, found := findTorrentByHash(all, hash); found {
			torrentName = t.Name
		}
	}

	fps := formatter.FilesPageState{
		FilePage:   filePage,
		FilterChar: filterChar,
		ListPage:   listPage,
	}
	text, kb, renderErr := h.renderFilesPage(ctx, hash, torrentName, fps)
	if renderErr != nil {
		h.answerCallback(cq.ID, "Priority updated.")
		return
	}

	tgKB := toTGKeyboard(kb)
	h.answerCallback(cq.ID, "Priority updated.")
	_ = h.editMessageText(cq.Message.Chat.ID, cq.Message.MessageID, text, &tgKB)
}

// handleRemoveCancelCallback handles rc: by returning to the torrent detail view.
// It parses rc:<filterChar>:<page>:<hash>, re-fetches the torrent, and renders the detail view.
func (h *Handler) handleRemoveCancelCallback(ctx context.Context, cq *tgbotapi.CallbackQuery, data string) {
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

	all, err := h.listTorrentsForFilter(ctx, qbt.FilterAll)
	if err != nil {
		h.answerCallback(cq.ID, fmt.Sprintf("Error: %v", err))
		return
	}

	torrent, found := findTorrentByHash(all, hash)
	if !found {
		// Torrent disappeared; navigate to list.
		filterPrefix := filterCharToPrefix(filterChar)
		text, kb, listErr := h.renderTorrentListPage(ctx, filter, filterPrefix, page)
		if listErr != nil {
			h.answerCallback(cq.ID, "Torrent not found.")
			return
		}
		tgKB := toTGKeyboard(kb)
		h.answerCallback(cq.ID, "Torrent not found.")
		_ = h.editMessageText(cq.Message.Chat.ID, cq.Message.MessageID, text, &tgKB)

		h.registerLiveView(cq.Message.Chat.ID, &LiveView{
			ChatID:     cq.Message.Chat.ID,
			MessageID:  cq.Message.MessageID,
			ViewType:   ViewList,
			Filter:     filter,
			FilterChar: filterChar,
			Page:       page,
		})
		return
	}

	text := formatter.FormatTorrentDetail(torrent)
	kb := toTGKeyboard(formatter.TorrentDetailKeyboard(hash, filterChar, page, torrent.State))

	h.answerCallback(cq.ID, "")
	_ = h.editMessageText(cq.Message.Chat.ID, cq.Message.MessageID, text, &kb)

	h.registerLiveView(cq.Message.Chat.ID, &LiveView{
		ChatID:      cq.Message.Chat.ID,
		MessageID:   cq.Message.MessageID,
		ViewType:    ViewDetail,
		TorrentHash: hash,
		FilterChar:  filterChar,
		Page:        page,
	})
}

func (h *Handler) handleSearchSelectCallback(ctx context.Context, cq *tgbotapi.CallbackQuery, data string) {
	parts := strings.SplitN(data, ":", 2)
	if len(parts) != 2 {
		h.answerCallback(cq.ID, "Invalid action.")
		return
	}
	jobID, err := strconv.Atoi(parts[0])
	if err != nil {
		h.answerCallback(cq.ID, "Invalid action.")
		return
	}
	idx, err := strconv.Atoi(parts[1])
	if err != nil {
		h.answerCallback(cq.ID, "Invalid action.")
		return
	}

	state := h.getSearch(cq.Message.Chat.ID)
	if state == nil || state.JobID != jobID {
		h.answerCallback(cq.ID, "Search expired. Please try again.")
		return
	}

	if idx < 0 || idx >= len(state.Results) {
		h.answerCallback(cq.ID, "Invalid result.")
		return
	}

	result := state.Results[idx]
	// Reset description state on new selection.
	state.DescriptionText = ""
	state.DescriptionPages = 0
	state.SelectedIdx = idx

	text := formatter.FormatSearchConfirm(result, "", 0, 0)
	page := idx/formatter.SearchResultsPerPage + 1
	kb := toTGKeyboard(formatter.SearchConfirmKeyboard(jobID, idx, page))

	h.answerCallback(cq.ID, "")
	_ = h.editMessageText(cq.Message.Chat.ID, cq.Message.MessageID, text, &kb)

	// Async: fetch description text if DescrLink is available.
	if result.DescrLink != "" {
		//nolint:gosec // async goroutine with Background context is intentional
		go h.fetchAndUpdateDescription(cq.Message.Chat.ID, cq.Message.MessageID, state, result, idx, page)
	}
}

//nolint:gocritic // result is passed by value intentionally
func (h *Handler) fetchAndUpdateDescription(chatID int64, messageID int, state *SearchState, result qbt.SearchResult, resultIdx, listPage int) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fetcher := newDescriptionFetcher()
	desc := fetcher.fetch(ctx, result.DescrLink)
	if desc == "" {
		return
	}

	// Re-validate search state and selected result index.
	s := h.getSearch(chatID)
	if s == nil || s.JobID != state.JobID || s.SelectedIdx != resultIdx {
		return
	}

	// Store full description and compute pagination.
	pageSize := formatter.DescriptionPageSize(formatter.FormatSearchConfirmBase(result), result.DescrLink)
	pages := formatter.SplitDescription(desc, pageSize)
	totalPages := len(pages)
	if totalPages == 0 {
		return
	}
	s.DescriptionText = desc
	s.DescriptionPages = totalPages

	// Show page 1 with pagination keyboard if multi-page.
	updatedText := formatter.FormatSearchConfirm(result, desc, 1, totalPages)
	kb := toTGKeyboard(formatter.SearchConfirmKeyboardWithDesc(s.JobID, resultIdx, listPage, 1, totalPages))
	_ = h.editMessageText(chatID, messageID, updatedText, &kb)
}

func (h *Handler) handleDescriptionPageCallback(ctx context.Context, cq *tgbotapi.CallbackQuery, data string) {
	// Format: <jobID>:<idx>:<page>
	parts := strings.SplitN(data, ":", 3)
	if len(parts) != 3 {
		h.answerCallback(cq.ID, "Invalid action.")
		return
	}
	jobID, err := strconv.Atoi(parts[0])
	if err != nil {
		h.answerCallback(cq.ID, "Invalid action.")
		return
	}
	idx, err := strconv.Atoi(parts[1])
	if err != nil {
		h.answerCallback(cq.ID, "Invalid action.")
		return
	}
	page, err := strconv.Atoi(parts[2])
	if err != nil || page < 1 {
		h.answerCallback(cq.ID, "Invalid action.")
		return
	}

	state := h.getSearch(cq.Message.Chat.ID)
	if state == nil || state.JobID != jobID || state.DescriptionText == "" {
		h.answerCallback(cq.ID, "Search expired.")
		return
	}
	if page > state.DescriptionPages {
		h.answerCallback(cq.ID, "Invalid page.")
		return
	}

	result := state.Results[idx]
	listPage := idx/formatter.SearchResultsPerPage + 1
	text := formatter.FormatSearchConfirm(result, state.DescriptionText, page, state.DescriptionPages)
	kb := toTGKeyboard(formatter.SearchConfirmKeyboardWithDesc(jobID, idx, listPage, page, state.DescriptionPages))

	h.answerCallback(cq.ID, "")
	_ = h.editMessageText(cq.Message.Chat.ID, cq.Message.MessageID, text, &kb)
}

func (h *Handler) handleSearchPageCallback(ctx context.Context, cq *tgbotapi.CallbackQuery, data string) {
	parts := strings.SplitN(data, ":", 2)
	if len(parts) != 2 {
		h.answerCallback(cq.ID, "Invalid action.")
		return
	}
	jobID, err := strconv.Atoi(parts[0])
	if err != nil {
		h.answerCallback(cq.ID, "Invalid action.")
		return
	}
	page, err := strconv.Atoi(parts[1])
	if err != nil {
		h.answerCallback(cq.ID, "Invalid action.")
		return
	}

	state := h.getSearch(cq.Message.Chat.ID)
	if state == nil || state.JobID != jobID {
		h.answerCallback(cq.ID, "Search expired. Please try again.")
		return
	}

	h.sendSearchResultsPage(cq.Message.Chat.ID, state, page, state.MessageID)
	h.answerCallback(cq.ID, "")
}

func (h *Handler) handleSearchCancelCallback(ctx context.Context, cq *tgbotapi.CallbackQuery, data string) {
	jobID, err := strconv.Atoi(data)
	if err != nil {
		h.answerCallback(cq.ID, "Invalid action.")
		return
	}

	// Cancel the background polling goroutine before taking state.
	h.searchMu.Lock()
	if cancel, ok := h.searchCancels[cq.Message.Chat.ID]; ok {
		cancel()
		delete(h.searchCancels, cq.Message.Chat.ID)
	}
	h.searchMu.Unlock()

	state := h.takeSearch(cq.Message.Chat.ID)
	if state == nil || state.JobID != jobID {
		h.answerCallback(cq.ID, "Search expired.")
		return
	}

	if err := h.qbt.StopSearch(ctx, jobID); err != nil {
		log.Printf("bot: stop search %d: %v", jobID, err)
	}
	if err := h.qbt.DeleteSearch(ctx, jobID); err != nil {
		log.Printf("bot: delete search %d: %v", jobID, err)
	}
	_ = h.editMessageText(cq.Message.Chat.ID, cq.Message.MessageID, "Search canceled.", nil)
	h.answerCallback(cq.ID, "")
}

func (h *Handler) handleSearchSortCallback(ctx context.Context, cq *tgbotapi.CallbackQuery, data string) {
	parts := strings.SplitN(data, ":", 2)
	if len(parts) != 2 {
		h.answerCallback(cq.ID, "Invalid action.")
		return
	}
	jobID, err := strconv.Atoi(parts[0])
	if err != nil {
		h.answerCallback(cq.ID, "Invalid action.")
		return
	}
	field := parts[1]

	state := h.getSearch(cq.Message.Chat.ID)
	if state == nil || state.JobID != jobID {
		h.answerCallback(cq.ID, "Search expired. Please try again.")
		return
	}

	if state.SortField == field {
		state.SortAsc = !state.SortAsc
	} else {
		state.SortField = field
		state.SortAsc = false
	}

	h.sortSearchResults(state.Results, state.SortField, state.SortAsc)
	h.sendSearchResultsPage(cq.Message.Chat.ID, state, 1, state.MessageID)
	h.answerCallback(cq.ID, "")
}

func (h *Handler) handleSearchConfirmCallback(ctx context.Context, cq *tgbotapi.CallbackQuery, data string) {
	parts := strings.SplitN(data, ":", 2)
	if len(parts) != 2 {
		h.answerCallback(cq.ID, "Invalid action.")
		return
	}
	jobID, err := strconv.Atoi(parts[0])
	if err != nil {
		h.answerCallback(cq.ID, "Invalid action.")
		return
	}
	idx, err := strconv.Atoi(parts[1])
	if err != nil {
		h.answerCallback(cq.ID, "Invalid action.")
		return
	}

	state := h.getSearch(cq.Message.Chat.ID)
	if state == nil || state.JobID != jobID {
		h.answerCallback(cq.ID, "Search expired. Please try again.")
		return
	}

	if idx < 0 || idx >= len(state.Results) {
		h.answerCallback(cq.ID, "Invalid result.")
		return
	}

	result := state.Results[idx]

	if strings.HasPrefix(result.FileURL, "magnet:?") {
		// Magnet link — existing flow unchanged.
		h.storePending(cq.Message.Chat.ID, &PendingTorrent{
			MagnetLink: result.FileURL,
			CreatedAt:  time.Now(),
		})
		h.sendCategoryKeyboard(ctx, cq.Message.Chat.ID, "Select category for this torrent:")
		h.answerCallback(cq.ID, "")
		return
	}

	//nolint:nestif // URL scheme check is a simple 2-condition filter
	if strings.HasPrefix(result.FileURL, "http://") || strings.HasPrefix(result.FileURL, "https://") {
		// Download .torrent file from HTTP URL, then proceed through the
		// same category → AddTorrentFile pipeline used for Telegram-uploaded files.
		data, err := downloadUserTorrentFn(ctx, newDownloadClient(), result.FileURL)
		if err != nil {
			var magnetErr *MagnetRedirectError
			if errors.As(err, &magnetErr) {
				h.storePending(cq.Message.Chat.ID, &PendingTorrent{
					MagnetLink: magnetErr.URI,
					CreatedAt:  time.Now(),
				})
				h.sendCategoryKeyboard(ctx, cq.Message.Chat.ID, "Select category for this torrent:")
				h.answerCallback(cq.ID, "")
				return
			}
			h.answerCallback(cq.ID, "")
			_ = h.editMessageText(cq.Message.Chat.ID, cq.Message.MessageID,
				fmt.Sprintf("Failed to download torrent: %v", err), nil)
			return
		}

		fileName := result.FileName
		if !strings.HasSuffix(strings.ToLower(fileName), ".torrent") {
			fileName += ".torrent"
		}

		h.storePending(cq.Message.Chat.ID, &PendingTorrent{
			FileData:  data,
			FileName:  fileName,
			CreatedAt: time.Now(),
		})
		h.sendCategoryKeyboard(ctx, cq.Message.Chat.ID, "Select category for this torrent:")
		h.answerCallback(cq.ID, "")
		return
	}

	// Neither magnet nor HTTP URL — still an error.
	h.answerCallback(cq.ID, "")
	_ = h.editMessageText(cq.Message.Chat.ID, cq.Message.MessageID,
		"This result doesn't have a valid download link. Try another result.", nil)
}

func (h *Handler) handleSearchBackCallback(ctx context.Context, cq *tgbotapi.CallbackQuery, data string) {
	parts := strings.SplitN(data, ":", 2)
	if len(parts) != 2 {
		h.answerCallback(cq.ID, "Invalid action.")
		return
	}
	jobID, err := strconv.Atoi(parts[0])
	if err != nil {
		h.answerCallback(cq.ID, "Invalid action.")
		return
	}
	page, err := strconv.Atoi(parts[1])
	if err != nil {
		h.answerCallback(cq.ID, "Invalid action.")
		return
	}

	state := h.getSearch(cq.Message.Chat.ID)
	if state == nil || state.JobID != jobID {
		h.answerCallback(cq.ID, "Search expired. Please try again.")
		return
	}

	h.sendSearchResultsPage(cq.Message.Chat.ID, state, page, state.MessageID)
	h.answerCallback(cq.ID, "")
}
