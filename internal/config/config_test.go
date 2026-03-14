package config

import (
	"testing"
	"time"
)

// setRequiredEnv sets the minimum required environment variables for a
// successful Load call. Tests that want to override a specific field can call
// t.Setenv again after this helper.
func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("TELEGRAM_ALLOWED_USERS", "111,222")
	t.Setenv("QBITTORRENT_URL", "http://localhost:8080")
	t.Setenv("QBITTORRENT_USERNAME", "admin")
	t.Setenv("QBITTORRENT_PASSWORD", "secret")
}

func TestLoad_AllRequiredFields(t *testing.T) {
	setRequiredEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if cfg.TelegramToken != "test-token" {
		t.Errorf("TelegramToken = %q, want %q", cfg.TelegramToken, "test-token")
	}
	if len(cfg.AllowedUsers) != 2 {
		t.Fatalf("AllowedUsers length = %d, want 2", len(cfg.AllowedUsers))
	}
	if cfg.AllowedUsers[0] != 111 || cfg.AllowedUsers[1] != 222 {
		t.Errorf("AllowedUsers = %v, want [111 222]", cfg.AllowedUsers)
	}
	if cfg.QBTBaseURL != "http://localhost:8080" {
		t.Errorf("QBTBaseURL = %q, want %q", cfg.QBTBaseURL, "http://localhost:8080")
	}
	if cfg.QBTUsername != "admin" {
		t.Errorf("QBTUsername = %q, want %q", cfg.QBTUsername, "admin")
	}
	if cfg.QBTPassword != "secret" {
		t.Errorf("QBTPassword = %q, want %q", cfg.QBTPassword, "secret")
	}
	if cfg.PollInterval != 30*time.Second {
		t.Errorf("PollInterval = %v, want 30s", cfg.PollInterval)
	}
}

func TestLoad_MissingTelegramToken(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("TELEGRAM_BOT_TOKEN", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing TELEGRAM_BOT_TOKEN, got nil")
	}
}

func TestLoad_MissingAllowedUsers(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("TELEGRAM_ALLOWED_USERS", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing TELEGRAM_ALLOWED_USERS, got nil")
	}
}

func TestLoad_InvalidUserID(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("TELEGRAM_ALLOWED_USERS", "111,notanumber,333")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for non-numeric user ID, got nil")
	}
}

func TestLoad_MissingQBTURL(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("QBITTORRENT_URL", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing QBITTORRENT_URL, got nil")
	}
}

func TestLoad_CustomPollInterval(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("POLL_INTERVAL", "2m")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.PollInterval != 2*time.Minute {
		t.Errorf("PollInterval = %v, want 2m", cfg.PollInterval)
	}
}

func TestLoad_DefaultPollInterval(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("POLL_INTERVAL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.PollInterval != 30*time.Second {
		t.Errorf("PollInterval = %v, want 30s (default)", cfg.PollInterval)
	}
}

func TestLoad_EmptyAllowedUsers(t *testing.T) {
	setRequiredEnv(t)
	// Whitespace-only / comma-only value should result in zero valid IDs.
	t.Setenv("TELEGRAM_ALLOWED_USERS", "  ,  ,  ")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for empty TELEGRAM_ALLOWED_USERS, got nil")
	}
}
