package bot

import (
	"context"
	"fmt"
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
			if t.Progress == 1.0 {
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

	if _, ok := filterCharToFilter(filterChar); !ok {
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

	select {
	case <-time.After(actionStateDelay):
	case <-ctx.Done():
		h.answerCallback(cq.ID, "Canceled.")
		return
	}

	// Re-fetch the torrent to display the updated state.
	all, listErr := h.listTorrentsForFilter(ctx, qbt.FilterAll)
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

// handleRemoveConfirmCallback shows the confirmation view when the Remove button is pressed.
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
		h.editMessageText(cq.Message.Chat.ID, cq.Message.MessageID, text, &tgKB)
		return
	}

	text := formatter.FormatRemoveConfirmation(torrent.Name)
	kb := toTGKeyboard(formatter.RemoveConfirmKeyboard(hash, filterChar, page))

	h.answerCallback(cq.ID, "")
	h.editMessageText(cq.Message.Chat.ID, cq.Message.MessageID, text, &kb)
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
		h.editMessageText(cq.Message.Chat.ID, cq.Message.MessageID, "Removed.", nil)
		return
	}

	tgKB := toTGKeyboard(kb)
	h.answerCallback(cq.ID, "Removed.")
	h.editMessageText(cq.Message.Chat.ID, cq.Message.MessageID, text, &tgKB)
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
func parseFileSelectCallback(data string) (hash string, fileIndex, filePage int, filterChar string, listPage int, err error) {
	parts := strings.SplitN(data, ":", 5)
	if len(parts) != 5 {
		return "", 0, 0, "", 0, fmt.Errorf("invalid fs: format")
	}
	hash = parts[0]
	fileIndex, err = strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, 0, "", 0, fmt.Errorf("invalid fileIndex: %w", err)
	}
	if fileIndex < 0 {
		return "", 0, 0, "", 0, fmt.Errorf("fileIndex must be non-negative")
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
func parseFilePriorityCallback(data string) (hash string, fileIndex, priority, filePage int, filterChar string, listPage int, err error) {
	parts := strings.SplitN(data, ":", 6)
	if len(parts) != 6 {
		return "", 0, 0, 0, "", 0, fmt.Errorf("invalid fp: format")
	}
	hash = parts[0]
	fileIndex, err = strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, 0, 0, "", 0, fmt.Errorf("invalid fileIndex: %w", err)
	}
	if fileIndex < 0 {
		return "", 0, 0, 0, "", 0, fmt.Errorf("fileIndex must be non-negative")
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
		return "", 0, "", fmt.Errorf("invalid fl: format")
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
		return "", 0, "", 0, fmt.Errorf("invalid pg:fl: format")
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
		return "", 0, "", fmt.Errorf("invalid bk:fl: format")
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
	filePage int,
	filterChar string,
	listPage int,
) (string, formatter.Keyboard, error) {
	files, err := h.qbt.ListFiles(ctx, hash)
	if err != nil {
		return "", nil, err
	}

	totalFilePages := formatter.TotalPages(len(files), formatter.FilesPerPage)
	if filePage < 1 {
		filePage = 1
	}
	if filePage > totalFilePages {
		filePage = totalFilePages
	}

	offset := (filePage - 1) * formatter.FilesPerPage
	end := offset + formatter.FilesPerPage
	if end > len(files) {
		end = len(files)
	}
	var pageFiles []qbt.TorrentFile
	if offset < len(files) {
		pageFiles = files[offset:end]
	}

	text := formatter.FormatFileList(torrentName, pageFiles, filePage, totalFilePages)
	kb := formatter.FileListKeyboard(pageFiles, hash, offset, filePage, totalFilePages, filterChar, listPage)
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

	text, kb, err := h.renderFilesPage(ctx, hash, torrentName, 1, filterChar, listPage)
	if err != nil {
		h.answerCallback(cq.ID, "Failed to load files.")
		return
	}

	tgKB := toTGKeyboard(kb)
	h.answerCallback(cq.ID, "")
	h.editMessageText(cq.Message.Chat.ID, cq.Message.MessageID, text, &tgKB)
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

	text, kb, err := h.renderFilesPage(ctx, hash, torrentName, filePage, filterChar, listPage)
	if err != nil {
		h.answerCallback(cq.ID, "Failed to load files.")
		return
	}

	tgKB := toTGKeyboard(kb)
	h.answerCallback(cq.ID, "")
	h.editMessageText(cq.Message.Chat.ID, cq.Message.MessageID, text, &tgKB)
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
	h.editMessageText(cq.Message.Chat.ID, cq.Message.MessageID, text, &kb)
}

// handleFileSelectCallback handles fs: callbacks — showing the priority selector for a file.
// Format: fs:<hash>:<fileIndex>:<filePage>:<filterChar>:<listPage>
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

	kb := toTGKeyboard(formatter.PriorityKeyboard(hash, fileIndex, currentPriority, filePage, filterChar, listPage))
	h.answerCallback(cq.ID, "")
	h.editMessageText(cq.Message.Chat.ID, cq.Message.MessageID, "Select priority:", &kb)
}

// handleFilePriorityCallback handles fp: callbacks — setting a file's download priority.
// Format: fp:<hash>:<fileIndex>:<priority>:<filePage>:<filterChar>:<listPage>
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

	text, kb, renderErr := h.renderFilesPage(ctx, hash, torrentName, filePage, filterChar, listPage)
	if renderErr != nil {
		h.answerCallback(cq.ID, "Priority updated.")
		return
	}

	tgKB := toTGKeyboard(kb)
	h.answerCallback(cq.ID, "Priority updated.")
	h.editMessageText(cq.Message.Chat.ID, cq.Message.MessageID, text, &tgKB)
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
		h.editMessageText(cq.Message.Chat.ID, cq.Message.MessageID, text, &tgKB)
		return
	}

	text := formatter.FormatTorrentDetail(torrent)
	kb := toTGKeyboard(formatter.TorrentDetailKeyboard(hash, filterChar, page, torrent.State))

	h.answerCallback(cq.ID, "")
	h.editMessageText(cq.Message.Chat.ID, cq.Message.MessageID, text, &kb)
}
