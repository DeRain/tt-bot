package bot

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// CommandDef defines a single bot command for registration and help text.
type CommandDef struct {
	Command     string
	Description string
}

// BotCommands is the single source of truth for all bot commands.
// Used for both Telegram setMyCommands registration and help text generation.
var BotCommands = []CommandDef{
	{Command: "list", Description: "List all torrents (paginated)"},
	{Command: "active", Description: "List active downloads (paginated)"},
	{Command: "help", Description: "Show help message"},
}

// RegisterCommands registers bot commands with the Telegram API via setMyCommands.
// Returns an error if the API call fails; callers should treat this as non-fatal.
func RegisterCommands(sender Sender) error {
	cmds := make([]tgbotapi.BotCommand, len(BotCommands))
	for i, c := range BotCommands {
		cmds[i] = tgbotapi.BotCommand{
			Command:     c.Command,
			Description: c.Description,
		}
	}

	cfg := tgbotapi.NewSetMyCommands(cmds...)
	if _, err := sender.Request(cfg); err != nil {
		return fmt.Errorf("set my commands: %w", err)
	}

	return nil
}

// HelpText generates help text from BotCommands, ensuring a single source of truth.
func HelpText() string {
	var b strings.Builder
	b.WriteString("tt-bot — qBittorrent Telegram controller\n\nCommands:\n")
	for _, cmd := range BotCommands {
		fmt.Fprintf(&b, "  /%s — %s\n", cmd.Command, cmd.Description)
	}
	b.WriteString("\nYou can also send:\n")
	b.WriteString("  • A magnet link (magnet:?…) — prompts for category then adds it\n")
	b.WriteString("  • A .torrent file — prompts for category then adds it")
	return b.String()
}
