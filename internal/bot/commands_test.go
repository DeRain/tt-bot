package bot

import (
	"errors"
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// TEST-1 + TEST-2: RegisterCommands builds correct config and calls Request once.
func TestRegisterCommands_BuildsCorrectConfig(t *testing.T) {
	sender := &mockSender{}

	err := RegisterCommands(sender)
	if err != nil {
		t.Fatalf("RegisterCommands() error = %v", err)
	}

	// TEST-2: exactly one Request call.
	if len(sender.sentMessages) != 1 {
		t.Fatalf("expected 1 Request call, got %d", len(sender.sentMessages))
	}

	cfg, ok := sender.sentMessages[0].(tgbotapi.SetMyCommandsConfig)
	if !ok {
		t.Fatalf("expected SetMyCommandsConfig, got %T", sender.sentMessages[0])
	}

	if len(cfg.Commands) != len(BotCommands) {
		t.Fatalf("expected %d commands, got %d", len(BotCommands), len(cfg.Commands))
	}

	for i, cmd := range cfg.Commands {
		if cmd.Command != BotCommands[i].Command {
			t.Errorf("command[%d] = %q, want %q", i, cmd.Command, BotCommands[i].Command)
		}
		if cmd.Description != BotCommands[i].Description {
			t.Errorf("description[%d] = %q, want %q", i, cmd.Description, BotCommands[i].Description)
		}
	}
}

// TEST-3: RegisterCommands returns error on API failure without panicking.
func TestRegisterCommands_FailOpen(t *testing.T) {
	sender := &mockSender{
		requestErr: errors.New("telegram API unavailable"),
	}

	err := RegisterCommands(sender)

	if err == nil {
		t.Fatal("expected error when API fails, got nil")
	}
	if !strings.Contains(err.Error(), "telegram API unavailable") {
		t.Errorf("expected wrapped error, got: %v", err)
	}
}

// TEST-4: HelpText is generated from BotCommands slice.
func TestHelpText_GeneratedFromBotCommands(t *testing.T) {
	text := HelpText()

	for _, cmd := range BotCommands {
		if !strings.Contains(text, "/"+cmd.Command) {
			t.Errorf("help text missing command /%s", cmd.Command)
		}
		if !strings.Contains(text, cmd.Description) {
			t.Errorf("help text missing description %q", cmd.Description)
		}
	}
}

// Verify expected commands are registered.
func TestBotCommands_ContainsExpectedCommands(t *testing.T) {
	expected := []string{"list", "active", "help"}
	if len(BotCommands) != len(expected) {
		t.Fatalf("expected %d commands, got %d", len(expected), len(BotCommands))
	}
	for i, name := range expected {
		if BotCommands[i].Command != name {
			t.Errorf("BotCommands[%d].Command = %q, want %q", i, BotCommands[i].Command, name)
		}
	}
}
