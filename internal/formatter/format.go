// Package formatter provides functions for building Telegram-safe messages
// and inline keyboard representations from qBittorrent torrent data.
// It does not import telegram-bot-api; callers convert the returned types
// to the Telegram library's own structures.
package formatter

import (
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/home/tt-bot/internal/qbt"
)

const (
	// MaxMessageLength is the maximum number of characters in a Telegram message.
	MaxMessageLength = 4096
	// MaxCallbackData is the maximum number of bytes in Telegram callback data.
	MaxCallbackData = 64
	// TorrentsPerPage is the number of torrents shown per page in the list view.
	TorrentsPerPage = 5

	maxNameLength = 40
)

// Button represents an inline keyboard button.
type Button struct {
	Text         string
	CallbackData string
}

// ButtonRow is a row of buttons in an inline keyboard.
type ButtonRow []Button

// Keyboard is a collection of button rows forming an inline keyboard.
type Keyboard []ButtonRow

// stateLabels maps raw qBittorrent state strings to human-readable labels with emoji prefixes.
var stateLabels = map[string]string{
	"error":              "❌ Error",
	"missingFiles":       "⚠️ Missing Files",
	"uploading":          "🌱 Seeding",
	"pausedUP":           "⏸️ Paused (Seeding)",
	"stoppedUP":          "⏸️ Stopped (Seeding)",
	"queuedUP":           "🕐 Queued (Seeding)",
	"stalledUP":          "🌱 Seeding (stalled)",
	"checkingUP":         "🔍 Checking",
	"forcedUP":           "⏫ Force Seeding",
	"allocating":         "💾 Allocating",
	"downloading":        "⬇️ Downloading",
	"metaDL":             "🔎 Fetching Metadata",
	"pausedDL":           "⏸️ Paused (Downloading)",
	"stoppedDL":          "⏸️ Stopped (Downloading)",
	"queuedDL":           "🕐 Queued (Downloading)",
	"stalledDL":          "⬇️ Downloading (stalled)",
	"checkingDL":         "🔍 Checking",
	"forcedDL":           "⏬ Force Downloading",
	"checkingResumeData": "🔍 Checking",
	"moving":             "📦 Moving",
	"unknown":            "❓ Unknown",
}

// FormatState maps a raw qBittorrent state string to a human-readable label
// with an emoji prefix. If the state is not recognized, it returns "❓ <state>".
// It never returns an empty string and never panics.
func FormatState(state string) string {
	if label, ok := stateLabels[state]; ok {
		return label
	}
	return "❓ " + state
}

// FormatSpeed formats a bytes-per-second value into a human-readable speed string.
// Values below 1 KB/s are shown as "X B/s", below 1 MB/s as "X.X KB/s",
// and anything larger as "X.X MB/s".
func FormatSpeed(bytesPerSec int64) string {
	const kb = 1024
	const mb = 1024 * 1024

	switch {
	case bytesPerSec < kb:
		return fmt.Sprintf("%d B/s", bytesPerSec)
	case bytesPerSec < mb:
		return fmt.Sprintf("%.1f KB/s", float64(bytesPerSec)/kb)
	default:
		return fmt.Sprintf("%.1f MB/s", float64(bytesPerSec)/mb)
	}
}

// FormatProgress returns a 10-character progress bar followed by the integer
// percentage. For example: "██████░░░░ 60%".
func FormatProgress(progress float64) string {
	const barLen = 10
	const filled = '█'
	const empty = '░'

	// Clamp progress to [0, 1].
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}

	filledCount := int(math.Round(progress * barLen))
	bar := strings.Repeat(string(filled), filledCount) +
		strings.Repeat(string(empty), barLen-filledCount)

	pct := int(math.Round(progress * 100))
	return fmt.Sprintf("%s %d%%", bar, pct)
}

// truncateName shortens a torrent name to at most maxNameLength characters,
// appending "..." if truncation occurred.
func truncateName(name string) string {
	runes := []rune(name)
	if len(runes) <= maxNameLength {
		return name
	}
	return string(runes[:maxNameLength-3]) + "..."
}

// FormatTorrentList formats a paginated list of torrents into a single
// Telegram-safe message string. The output is guaranteed to stay under
// MaxMessageLength (4096) characters.
//
// Page and totalPages are 1-based.
func FormatTorrentList(torrents []qbt.Torrent, page, totalPages int) string {
	if len(torrents) == 0 {
		return "No torrents found."
	}

	header := fmt.Sprintf("Torrents (page %d/%d)\n", page, totalPages)
	var sb strings.Builder
	sb.WriteString(header)

	for _, t := range torrents {
		name := truncateName(t.Name)
		progress := FormatProgress(t.Progress)
		dl := FormatSpeed(t.DLSpeed)
		up := FormatSpeed(t.UPSpeed)

		entry := fmt.Sprintf(
			"📥 %s\n   %s | ↓%s ↑%s | %s\n",
			name,
			progress,
			dl,
			up,
			FormatState(t.State),
		)

		// Guard against exceeding the Telegram message limit.
		if sb.Len()+len(entry) > MaxMessageLength-1 {
			break
		}
		sb.WriteString(entry)
	}

	return sb.String()
}

// TotalPages computes the total number of pages required to display totalItems
// items given perPage items per page. Returns 1 if totalItems is zero.
func TotalPages(totalItems, perPage int) int {
	if totalItems <= 0 || perPage <= 0 {
		return 1
	}
	return (totalItems + perPage - 1) / perPage
}

// PaginationKeyboard builds an inline keyboard row with Prev / current-page /
// Next buttons. FilterPrefix must be "all" or "act".
//
// The Prev button is omitted when currentPage == 1; the Next button is omitted
// when currentPage == totalPages. The center button has callback data "noop".
func PaginationKeyboard(currentPage, totalPages int, filterPrefix string) Keyboard {
	var row ButtonRow

	if currentPage > 1 {
		row = append(row, Button{
			Text:         "<< Prev",
			CallbackData: fmt.Sprintf("pg:%s:%d", filterPrefix, currentPage-1),
		})
	}

	row = append(row, Button{
		Text:         fmt.Sprintf("Page %d/%d", currentPage, totalPages),
		CallbackData: "noop",
	})

	if currentPage < totalPages {
		row = append(row, Button{
			Text:         "Next >>",
			CallbackData: fmt.Sprintf("pg:%s:%d", filterPrefix, currentPage+1),
		})
	}

	return Keyboard{row}
}

// FormatSize formats a byte count into a human-readable size string.
// Values below 1 KB are shown as "X B", below 1 MB as "X.X KB",
// below 1 GB as "X.X MB", below 1 TB as "X.X GB", and anything larger as "X.X TB".
func FormatSize(b int64) string {
	const (
		kb = 1024
		mb = 1024 * 1024
		gb = 1024 * 1024 * 1024
		tb = 1024 * 1024 * 1024 * 1024
	)

	switch {
	case b < kb:
		return fmt.Sprintf("%d B", b)
	case b < mb:
		return fmt.Sprintf("%.1f KB", float64(b)/kb)
	case b < gb:
		return fmt.Sprintf("%.1f MB", float64(b)/mb)
	case b < tb:
		return fmt.Sprintf("%.1f GB", float64(b)/gb)
	default:
		return fmt.Sprintf("%.1f TB", float64(b)/tb)
	}
}

// FormatTorrentDetail renders a single torrent's full metadata as a
// Telegram-safe message string.
//
//nolint:gocritic // t is passed by value intentionally; changing to pointer would require interface changes
func FormatTorrentDetail(t qbt.Torrent) string {
	cat := t.Category
	if cat == "" {
		cat = "none"
	}

	name := t.Name
	// Truncate extremely long names to stay under message limit.
	nameRunes := []rune(name)
	if len(nameRunes) > 200 {
		name = string(nameRunes[:197]) + "..."
	}

	return fmt.Sprintf(
		"📥 %s\n\nSize: %s\nProgress: %s\nDownload: %s\nUpload: %s\nUploaded: %s\nRatio: %.2f\nState: %s\nCategory: %s",
		name,
		FormatSize(t.Size),
		FormatProgress(t.Progress),
		FormatSpeed(t.DLSpeed),
		FormatSpeed(t.UPSpeed),
		FormatSize(t.Uploaded),
		t.Ratio,
		FormatState(t.State),
		cat,
	)
}

// TorrentDetailKeyboard builds an inline keyboard for the torrent detail view.
// Row 1: Pause and Start buttons side by side (always both visible).
// Row 2: Files button.
// Row 3: Remove button.
// Row 4: Back to list button.
func TorrentDetailKeyboard(hash, filterChar string, page int, _ string) Keyboard {
	pauseBtn := Button{
		Text:         "⏸ Pause",
		CallbackData: fmt.Sprintf("pa:%s:%d:%s", filterChar, page, hash),
	}
	startBtn := Button{
		Text:         "▶️ Start",
		CallbackData: fmt.Sprintf("re:%s:%d:%s", filterChar, page, hash),
	}
	// fl:<filterChar>:<listPage>:<hash>
	filesBtn := Button{
		Text:         "📁 Files",
		CallbackData: fmt.Sprintf("fl:%s:%d:%s", filterChar, page, hash),
	}
	removeBtn := Button{
		Text:         "🗑 Remove",
		CallbackData: fmt.Sprintf("rm:%s:%d:%s", filterChar, page, hash),
	}
	backBtn := Button{
		Text:         "⬅️ Back to list",
		CallbackData: fmt.Sprintf("bk:%s:%d", filterChar, page),
	}

	return Keyboard{
		ButtonRow{pauseBtn, startBtn},
		ButtonRow{filesBtn},
		ButtonRow{removeBtn},
		ButtonRow{backBtn},
	}
}

// FormatRemoveConfirmation builds a confirmation prompt for torrent removal.
// It includes the torrent name so the user can confirm they are removing the right torrent.
func FormatRemoveConfirmation(torrentName string) string {
	return fmt.Sprintf("Remove torrent?\n\n%s\n\nChoose an action:", torrentName)
}

// RemoveConfirmKeyboard builds the confirmation keyboard shown after the Remove button is pressed.
// Row 1: Remove torrent only (rd:).
// Row 2: Remove with files (rf:).
// Row 3: Cancel (rc:).
func RemoveConfirmKeyboard(hash, filterChar string, page int) Keyboard {
	removeOnlyBtn := Button{
		Text:         "🗑 Remove torrent only",
		CallbackData: fmt.Sprintf("rd:%s:%d:%s", filterChar, page, hash),
	}
	removeFilesBtn := Button{
		Text:         "🗑 Remove with files",
		CallbackData: fmt.Sprintf("rf:%s:%d:%s", filterChar, page, hash),
	}
	cancelBtn := Button{
		Text:         "❌ Cancel",
		CallbackData: fmt.Sprintf("rc:%s:%d:%s", filterChar, page, hash),
	}

	return Keyboard{
		ButtonRow{removeOnlyBtn},
		ButtonRow{removeFilesBtn},
		ButtonRow{cancelBtn},
	}
}

// TorrentSelectionKeyboard builds a keyboard with one button per torrent,
// allowing the user to select a torrent from the list view.
func TorrentSelectionKeyboard(torrents []qbt.Torrent, filterChar string, page int) Keyboard {
	if len(torrents) == 0 {
		return nil
	}

	kb := make(Keyboard, 0, len(torrents))
	for i, t := range torrents {
		label := fmt.Sprintf("%d. %s", i+1, truncateName(t.Name))
		data := fmt.Sprintf("sel:%s:%d:%s", filterChar, page, t.Hash)
		kb = append(kb, ButtonRow{Button{Text: label, CallbackData: data}})
	}
	return kb
}

// FilesPerPage is the number of files shown per page in the file list view.
const FilesPerPage = 5

// truncateFileName returns only the last path component of name (after the
// final '/'), truncated to maxNameLength runes with a trailing '…' if needed.
func truncateFileName(name string) string {
	// Take only the last path component.
	if idx := strings.LastIndexByte(name, '/'); idx >= 0 {
		name = name[idx+1:]
	}
	runes := []rune(name)
	if len(runes) <= maxNameLength {
		return name
	}
	// Truncate and append single-rune ellipsis (…).
	return string(runes[:maxNameLength-1]) + "…"
}

// PriorityLabel returns the human-readable label for a file priority.
// Unknown values (e.g., the sentinel 4 for "mixed") are returned as "Mixed".
func PriorityLabel(p qbt.FilePriority) string {
	switch p {
	case qbt.FilePrioritySkip:
		return "Skip"
	case qbt.FilePriorityNormal:
		return "Normal"
	case qbt.FilePriorityHigh:
		return "High"
	case qbt.FilePriorityMaximum:
		return "Max"
	default:
		return "Mixed"
	}
}

// FormatFileList formats a paginated list of torrent files into a single
// Telegram-safe message string (≤4096 chars).
//
// TorrentName is shown as a header. Page and totalPages are 1-based.
func FormatFileList(torrentName string, files []qbt.TorrentFile, page, totalPages int) string {
	var sb strings.Builder

	// Header: torrent name (truncated) + page indicator.
	header := "📁 Files: " + truncateName(torrentName)
	if totalPages > 1 {
		header += fmt.Sprintf(" (Page %d/%d)", page, totalPages)
	}
	sb.WriteString(header)
	sb.WriteByte('\n')

	for _, f := range files {
		name := truncateFileName(f.Name)
		size := FormatSize(f.Size)
		progress := FormatProgress(f.Progress)
		priority := PriorityLabel(f.Priority)

		entry := fmt.Sprintf(
			"\n%s\n   %s | %s | %s\n",
			name,
			size,
			progress,
			priority,
		)

		if sb.Len()+len(entry) > MaxMessageLength-1 {
			break
		}
		sb.WriteString(entry)
	}

	return sb.String()
}

// FilesPageState holds pagination and navigation state for a file list page.
// Used to reduce parameter counts in keyboard-building functions.
type FilesPageState struct {
	FilePage   int
	FilterChar string
	ListPage   int
}

// FileListKeyboard builds the inline keyboard for the file list view.
//
// Each visible file gets one tap button (fs: prefix). Pagination buttons use
// pg:fl: prefix. A Back button uses bk:fl: to return to the torrent detail view.
//
// fileIndexOffset is the zero-based index of the first file in the current page
// relative to the full file list (i.e. (filePage-1)*FilesPerPage).
func FileListKeyboard(
	files []qbt.TorrentFile,
	hash string,
	fileIndexOffset int,
	totalFilePages int,
	fps FilesPageState,
) Keyboard {
	kb := make(Keyboard, 0, len(files)+3)

	for i, f := range files {
		fileIdx := fileIndexOffset + i
		name := truncateFileName(f.Name)
		label := fmt.Sprintf("%d. %s [%s]", fileIdx+1, name, PriorityLabel(f.Priority))
		// fs:<hash>:<fileIndex>:<filePage>:<filterChar>:<listPage>
		data := fmt.Sprintf("fs:%s:%d:%d:%s:%d", hash, fileIdx, fps.FilePage, fps.FilterChar, fps.ListPage)
		kb = append(kb, ButtonRow{Button{Text: label, CallbackData: data}})
	}

	// Pagination row (only when multiple pages exist).
	if totalFilePages > 1 {
		var pagRow ButtonRow
		if fps.FilePage > 1 {
			// pg:fl:<hash>:<filePage>:<filterChar>:<listPage>
			pagRow = append(pagRow, Button{
				Text:         "<< Prev",
				CallbackData: fmt.Sprintf("pg:fl:%s:%d:%s:%d", hash, fps.FilePage-1, fps.FilterChar, fps.ListPage),
			})
		}
		pagRow = append(pagRow, Button{
			Text:         fmt.Sprintf("Page %d/%d", fps.FilePage, totalFilePages),
			CallbackData: "noop",
		})
		if fps.FilePage < totalFilePages {
			pagRow = append(pagRow, Button{
				Text:         "Next >>",
				CallbackData: fmt.Sprintf("pg:fl:%s:%d:%s:%d", hash, fps.FilePage+1, fps.FilterChar, fps.ListPage),
			})
		}
		kb = append(kb, pagRow)
	}

	// Back button: bk:fl:<filterChar>:<listPage>:<hash>
	kb = append(kb, ButtonRow{Button{
		Text:         "⬅️ Back",
		CallbackData: fmt.Sprintf("bk:fl:%s:%d:%s", fps.FilterChar, fps.ListPage, hash),
	}})

	return kb
}

// PriorityKeyboard builds the inline keyboard for priority selection of a single file.
//
// It produces 4 priority option buttons (current priority marked with a ✓ prefix)
// and a Back button that returns to the file list page the user came from.
func PriorityKeyboard(
	hash string,
	fileIndex int,
	currentPriority qbt.FilePriority,
	fps FilesPageState,
) Keyboard {
	priorities := []qbt.FilePriority{
		qbt.FilePrioritySkip,
		qbt.FilePriorityNormal,
		qbt.FilePriorityHigh,
		qbt.FilePriorityMaximum,
	}

	kb := make(Keyboard, 0, len(priorities)+1)

	for _, p := range priorities {
		label := PriorityLabel(p)
		if p == currentPriority {
			label = "✓ " + label
		}
		// fp:<hash>:<fileIndex>:<priority>:<filePage>:<filterChar>:<listPage>
		data := fmt.Sprintf("fp:%s:%d:%d:%d:%s:%d", hash, fileIndex, int(p), fps.FilePage, fps.FilterChar, fps.ListPage)
		kb = append(kb, ButtonRow{Button{Text: label, CallbackData: data}})
	}

	// Back: return to the file list page (pg:fl: reuses the file list render).
	kb = append(kb, ButtonRow{Button{
		Text:         "⬅️ Back to files",
		CallbackData: fmt.Sprintf("pg:fl:%s:%d:%s:%d", hash, fps.FilePage, fps.FilterChar, fps.ListPage),
	}})

	return kb
}

const SearchResultsPerPage = 8

// ListPageFromIndex computes the 1-based page number for a search result at the given index.
func ListPageFromIndex(idx int) int {
	return idx/SearchResultsPerPage + 1
}

type SearchSortInfo struct {
	Field string
	Asc   bool
}

func FormatSearchResults(results []qbt.SearchResult, query string, page, totalPages int, sort SearchSortInfo) string {
	if len(results) == 0 {
		return fmt.Sprintf("No torrents found for '%s'.", query)
	}

	header := fmt.Sprintf("Search: %s (page %d/%d)\nsort: %s\n\n", query, page, totalPages, sort.Field)
	var sb strings.Builder
	sb.WriteString(header)

	for i, r := range results {
		name := truncateName(r.FileName)
		size := FormatSize(r.FileSize)
		date := time.Unix(r.PubDate, 0).Format("2006-01-02")

		entry := fmt.Sprintf(
			"%d. %s\n   %s | S:%d L:%d | %s\n\n",
			i+1,
			name,
			size,
			r.NbSeeders,
			r.NbLeechers,
			date,
		)

		if sb.Len()+len(entry) > MaxMessageLength-1 {
			break
		}
		sb.WriteString(entry)
	}

	return sb.String()
}

func SearchResultKeyboard(results []qbt.SearchResult, jobID, page int) Keyboard {
	if len(results) == 0 {
		return nil
	}

	// Compute global result indices so sr: callbacks reference the correct
	// position in the full (unsliced) results array.
	startIdx := (page - 1) * SearchResultsPerPage

	kb := make(Keyboard, 0, len(results))
	for i := range results {
		label := fmt.Sprintf("%d. %s", startIdx+i+1, truncateName(results[i].FileName))
		data := fmt.Sprintf("sr:%d:%d", jobID, startIdx+i)
		kb = append(kb, ButtonRow{Button{Text: label, CallbackData: data}})
	}
	return kb
}

func SearchPaginationKeyboard(jobID, currentPage, totalPages int) Keyboard {
	if totalPages <= 1 {
		return nil
	}

	var rows = make(Keyboard, 0, 2)

	var pagRow ButtonRow
	if currentPage > 1 {
		pagRow = append(pagRow, Button{
			Text:         "<< Prev",
			CallbackData: fmt.Sprintf("sp:%d:%d", jobID, currentPage-1),
		})
	}

	pagRow = append(pagRow, Button{
		Text:         fmt.Sprintf("Page %d/%d", currentPage, totalPages),
		CallbackData: "noop",
	})

	if currentPage < totalPages {
		pagRow = append(pagRow, Button{
			Text:         "Next >>",
			CallbackData: fmt.Sprintf("sp:%d:%d", jobID, currentPage+1),
		})
	}
	rows = append(rows,
		pagRow,
		ButtonRow{
			Button{Text: "Sort: Seeders", CallbackData: fmt.Sprintf("ss:%d:seeders", jobID)},
			Button{Text: "Sort: Size", CallbackData: fmt.Sprintf("ss:%d:size", jobID)},
			Button{Text: "Sort: Date", CallbackData: fmt.Sprintf("ss:%d:date", jobID)},
		})

	return rows
}

// FormatSearchConfirmBase returns the base message (without description) for computing page sizes.
//
//nolint:gocritic // result is passed by value intentionally
func FormatSearchConfirmBase(result qbt.SearchResult) string {
	msg := buildBaseMessage(result)
	return appendMoreInfoLink(msg, result.DescrLink)
}

// FormatSearchConfirm builds a confirmation message for a selected search result.
// If description and totalPages are non-zero, show "Description (page N/M):\n" with paginated text.
// When result.DescrLink is non-empty, a "More info:" link line is always included.
//
//nolint:gocritic // result is passed by value intentionally; changing to pointer would require interface changes
func FormatSearchConfirm(result qbt.SearchResult, description string, page, totalPages int) string {
	msg := buildBaseMessage(result)
	msg = appendMoreInfoLink(msg, result.DescrLink)
	msg = appendDescriptionPaginated(msg, description, page, totalPages)
	return msg
}

// DescriptionPageSize returns the max chars available for description text on one page.
func DescriptionPageSize(baseMsg, descrLink string) int {
	msg := baseMsg
	if descrLink != "" {
		msg += "\n\nMore info: " + descrLink
	}
	return MaxMessageLength - 1 - len(msg) - 32 // margin for "Description:\n" or "Description (page N/M):\n"
}

// SplitDescription splits text into pages that fit within maxPerPage bytes.
// Truncation is UTF-8 safe. Empty pages are skipped to prevent infinite loops.
func SplitDescription(text string, maxPerPage int) []string {
	if maxPerPage < 1 {
		return nil
	}
	var pages []string
	remaining := text
	for remaining != "" {
		end := min(maxPerPage, len(remaining))
		page := remaining[:end]
		for !utf8.Valid([]byte(page)) {
			page = page[:len(page)-1]
		}
		if page == "" {
			// Multi-byte character at start consumed the whole window; skip 1 byte.
			remaining = remaining[1:]
			continue
		}
		pages = append(pages, page)
		remaining = remaining[len(page):]
	}
	return pages
}

func appendDescriptionPaginated(msg, description string, page, totalPages int) string {
	if description == "" {
		return msg
	}
	if page < 1 {
		return msg
	}
	if page > totalPages {
		return msg
	}

	// Compute page text: slice the full description for the current page.
	pageSize := DescriptionPageSize(msg, "")
	if min(pageSize, 1) != 1 {
		return msg
	}
	pageText := descriptionPage(description, page, pageSize)

	var label string
	if totalPages == 1 {
		label = "\n\nDescription:\n"
	} else {
		label = fmt.Sprintf("\n\nDescription (page %d/%d):\n", page, totalPages)
	}
	return msg + label + pageText
}

func descriptionPage(text string, page, pageSize int) string {
	start := (page - 1) * pageSize
	// Align to next rune start if we landed in the middle of a multi-byte char.
	for {
		if min(start, len(text)) == len(text) || utf8.RuneStart(text[start]) {
			break
		}
		start++
	}
	limit := len(text)
	if min(start, limit) == limit {
		return ""
	}
	end := min(start+pageSize, len(text))
	result := text[start:end]
	// Trim trailing incomplete rune.
	for !utf8.Valid([]byte(result)) {
		result = result[:len(result)-1]
	}
	return result
}

//nolint:gocritic // result is passed by value intentionally; matching FormatSearchConfirm convention
func buildBaseMessage(result qbt.SearchResult) string {
	return fmt.Sprintf(
		"Add this torrent?\n\n%s\nSize: %s\nSeeders: %d\nLeechers: %d",
		result.FileName,
		FormatSize(result.FileSize),
		result.NbSeeders,
		result.NbLeechers,
	)
}

func appendMoreInfoLink(msg, descrLink string) string {
	if descrLink == "" {
		return msg
	}
	descLine := "\n\nMore info: " + descrLink
	if min(len(msg)+len(descLine), MaxMessageLength) == len(msg)+len(descLine) {
		return msg + descLine
	}
	// Truncate link to fit after "More info: " prefix and "..." suffix.
	linkMax := MaxMessageLength - len(msg) - len("\n\nMore info: ") - 3
	end := min(linkMax, len(descrLink))
	if min(end, 1) != 1 {
		return msg
	}
	truncated := descrLink[:end]
	for !utf8.Valid([]byte(truncated)) {
		truncated = truncated[:len(truncated)-1]
	}
	return msg + "\n\nMore info: " + truncated + "..."
}

// SearchCancelKeyboard builds a single-row keyboard with a Close button
// that cancels and deletes the active search job.
func SearchCancelKeyboard(jobID int) Keyboard {
	return Keyboard{
		ButtonRow{
			Button{Text: "❌ Close", CallbackData: fmt.Sprintf("sx:%d", jobID)},
		},
	}
}

// SearchConfirmKeyboard builds the confirmation keyboard for adding a search result.
func SearchConfirmKeyboard(jobID, resultIdx, page int) Keyboard {
	return Keyboard{
		ButtonRow{
			Button{Text: "Add this torrent", CallbackData: fmt.Sprintf("sc:%d:%d", jobID, resultIdx)},
		},
		ButtonRow{
			Button{Text: "Back to results", CallbackData: fmt.Sprintf("sb:%d:%d", jobID, page)},
		},
	}
}

// SearchConfirmKeyboardWithDesc builds the confirmation keyboard with optional
// description pagination buttons.
func SearchConfirmKeyboardWithDesc(jobID, resultIdx, listPage, descPage, descTotal int) Keyboard {
	kb := SearchConfirmKeyboard(jobID, resultIdx, listPage)
	btnRow := ButtonRow{}
	if descPage > 1 {
		btnRow = append(btnRow, Button{
			Text:         "▲ Prev page",
			CallbackData: fmt.Sprintf("dp:%d:%d:%d", jobID, resultIdx, descPage-1),
		})
	}
	if descPage < descTotal {
		btnRow = append(btnRow, Button{
			Text:         "▼ Next page",
			CallbackData: fmt.Sprintf("dp:%d:%d:%d", jobID, resultIdx, descPage+1),
		})
	}
	if len(btnRow) > 0 {
		kb = append(kb, btnRow)
	}
	return kb
}

// CategoryKeyboard builds an inline keyboard with one button per category.
// Each button's callback data is "cat:<name>" truncated to MaxCallbackData bytes.
//
// If categories is empty, a single "No category" button with callback "cat:" is
// returned so the caller always has at least one option.
func CategoryKeyboard(categories []qbt.Category) Keyboard {
	if len(categories) == 0 {
		return Keyboard{
			ButtonRow{
				Button{Text: "No category", CallbackData: "cat:"},
			},
		}
	}

	kb := make(Keyboard, 0, len(categories))
	const prefix = "cat:"
	for _, cat := range categories {
		data := prefix + cat.Name
		// Truncate to MaxCallbackData bytes (not runes) as Telegram enforces byte length.
		// Back off to a valid UTF-8 boundary to avoid splitting a multi-byte sequence.
		if len(data) > MaxCallbackData {
			data = data[:MaxCallbackData]
			for !utf8.Valid([]byte(data)) {
				data = data[:len(data)-1]
			}
		}
		kb = append(kb, ButtonRow{
			Button{Text: cat.Name, CallbackData: data},
		})
	}
	return kb
}
