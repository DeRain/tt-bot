//go:build integration

package bot

import (
	"os"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// telegramBot creates a *tgbotapi.BotAPI from TELEGRAM_BOT_TOKEN env var.
// Skips the test if the token is not set.
func telegramBot(t *testing.T) *tgbotapi.BotAPI {
	t.Helper()
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		t.Skip("TELEGRAM_BOT_TOKEN not set, skipping integration test")
	}
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		t.Fatalf("failed to create BotAPI: %v", err)
	}
	return bot
}

// TEST-5: Integration test that calls setMyCommands against real Telegram API.
func TestRegisterCommands_Integration(t *testing.T) {
	botAPI := telegramBot(t)

	err := RegisterCommands(botAPI)
	if err != nil {
		t.Fatalf("RegisterCommands() error = %v", err)
	}

	// Verify by calling getMyCommands to confirm they were set.
	cmds, err := botAPI.GetMyCommands()
	if err != nil {
		t.Fatalf("GetMyCommands() error = %v", err)
	}

	if len(cmds) != len(BotCommands) {
		t.Fatalf("expected %d commands registered, got %d", len(BotCommands), len(cmds))
	}

	// Compare as sets — Telegram does not guarantee order.
	registered := make(map[string]string, len(cmds))
	for _, cmd := range cmds {
		registered[cmd.Command] = cmd.Description
	}

	for _, want := range BotCommands {
		desc, ok := registered[want.Command]
		if !ok {
			t.Errorf("command %q not found in registered commands", want.Command)
			continue
		}
		if desc != want.Description {
			t.Errorf("command %q description = %q, want %q", want.Command, desc, want.Description)
		}
	}
}
