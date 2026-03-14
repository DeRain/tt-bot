// Package bot implements the Telegram bot update handler, callback dispatcher,
// and whitelist-based authorization for tt-bot.
package bot

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/home/tt-bot/internal/formatter"
)

// Sender abstracts Telegram message sending for testability.
// The production implementation wraps *tgbotapi.BotAPI.
type Sender interface {
	// Send transmits any Chattable (message, edit, callback answer, etc.) to Telegram.
	Send(msg tgbotapi.Chattable) (tgbotapi.Message, error)
	// GetFile retrieves file metadata from Telegram so the caller can download the
	// actual bytes via the file path returned in tgbotapi.File.FilePath.
	GetFile(config tgbotapi.FileConfig) (tgbotapi.File, error)
}

// toTGKeyboard converts a formatter.Keyboard into a tgbotapi.InlineKeyboardMarkup.
// Each formatter.ButtonRow becomes one row in the Telegram inline keyboard.
func toTGKeyboard(kb formatter.Keyboard) tgbotapi.InlineKeyboardMarkup {
	rows := make([][]tgbotapi.InlineKeyboardButton, 0, len(kb))
	for _, row := range kb {
		tgRow := make([]tgbotapi.InlineKeyboardButton, 0, len(row))
		for _, btn := range row {
			data := btn.CallbackData
			tgRow = append(tgRow, tgbotapi.NewInlineKeyboardButtonData(btn.Text, data))
		}
		rows = append(rows, tgRow)
	}
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}
