// Package formatter provides functions for building Telegram-safe messages
// and inline keyboard representations from qBittorrent torrent data.
// It does not import telegram-bot-api; callers convert the returned types
// to the Telegram library's own structures.
package formatter

import (
	"fmt"
	"math"
	"strings"
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
// page and totalPages are 1-based.
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
			t.State,
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
// Next buttons. filterPrefix must be "all" or "act".
//
// The Prev button is omitted when currentPage == 1; the Next button is omitted
// when currentPage == totalPages. The centre button has callback data "noop".
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

// IsPaused returns true if the torrent state represents a paused condition.
func IsPaused(state string) bool {
	return state == "pausedDL" || state == "pausedUP"
}

// FormatTorrentDetail renders a single torrent's full metadata as a
// Telegram-safe message string.
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
		"📥 %s\n\nSize: %s\nProgress: %s\nDownload: %s\nUpload: %s\nState: %s\nCategory: %s",
		name,
		FormatSize(t.Size),
		FormatProgress(t.Progress),
		FormatSpeed(t.DLSpeed),
		FormatSpeed(t.UPSpeed),
		t.State,
		cat,
	)
}

// TorrentDetailKeyboard builds an inline keyboard for the torrent detail view.
// Row 1: Pause or Resume button (based on torrent state).
// Row 2: Back to list button.
func TorrentDetailKeyboard(hash, filterChar string, page int, state string) Keyboard {
	var actionBtn Button
	if IsPaused(state) {
		actionBtn = Button{
			Text:         "▶️ Resume",
			CallbackData: fmt.Sprintf("re:%s:%d:%s", filterChar, page, hash),
		}
	} else {
		actionBtn = Button{
			Text:         "⏸ Pause",
			CallbackData: fmt.Sprintf("pa:%s:%d:%s", filterChar, page, hash),
		}
	}

	backBtn := Button{
		Text:         "⬅️ Back to list",
		CallbackData: fmt.Sprintf("bk:%s:%d", filterChar, page),
	}

	return Keyboard{
		ButtonRow{actionBtn},
		ButtonRow{backBtn},
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
			for len(data) > 0 && !utf8.Valid([]byte(data)) {
				data = data[:len(data)-1]
			}
		}
		kb = append(kb, ButtonRow{
			Button{Text: cat.Name, CallbackData: data},
		})
	}
	return kb
}
