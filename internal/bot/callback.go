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

	offset := (page - 1) * formatter.TorrentsPerPage
	torrents, err := h.qbt.ListTorrents(ctx, qbt.ListOptions{
		Filter: filter,
		Limit:  formatter.TorrentsPerPage,
		Offset: offset,
	})
	if err != nil {
		h.answerCallback(cq.ID, fmt.Sprintf("Error: %v", err))
		return
	}

	all, err := h.qbt.ListTorrents(ctx, qbt.ListOptions{Filter: filter})
	if err != nil {
		h.answerCallback(cq.ID, fmt.Sprintf("Error: %v", err))
		return
	}

	totalPages := formatter.TotalPages(len(all), formatter.TorrentsPerPage)
	text := formatter.FormatTorrentList(torrents, page, totalPages)

	kb := toTGKeyboard(formatter.PaginationKeyboard(page, totalPages, filterPrefix))

	h.answerCallback(cq.ID, "")
	h.editMessageText(chatID, messageID, text, &kb)
}
