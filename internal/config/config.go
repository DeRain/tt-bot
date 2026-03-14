// Package config loads and validates bot configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all runtime configuration for tt-bot. It is an immutable value
// type; callers should pass it by value to prevent accidental mutation.
type Config struct {
	// TelegramToken is the Telegram Bot API token (TELEGRAM_BOT_TOKEN).
	TelegramToken string

	// AllowedUsers is the whitelist of Telegram user IDs permitted to use the
	// bot (TELEGRAM_ALLOWED_USERS, comma-separated int64 values).
	AllowedUsers []int64

	// QBTBaseURL is the base URL of the qBittorrent Web API (QBITTORRENT_URL).
	QBTBaseURL string

	// QBTUsername is the qBittorrent Web UI username (QBITTORRENT_USERNAME).
	QBTUsername string

	// QBTPassword is the qBittorrent Web UI password (QBITTORRENT_PASSWORD).
	QBTPassword string

	// PollInterval controls how often the completion poller checks for finished
	// torrents (POLL_INTERVAL, default "30s").
	PollInterval time.Duration
}

// Load reads configuration from environment variables, validates all required
// fields, and returns an immutable Config value. It returns a descriptive error
// if any required variable is missing or malformed.
func Load() (Config, error) {
	token, err := requireEnv("TELEGRAM_BOT_TOKEN")
	if err != nil {
		return Config{}, err
	}

	allowedRaw, err := requireEnv("TELEGRAM_ALLOWED_USERS")
	if err != nil {
		return Config{}, err
	}

	allowedUsers, err := parseAllowedUsers(allowedRaw)
	if err != nil {
		return Config{}, err
	}

	qbtURL, err := requireEnv("QBITTORRENT_URL")
	if err != nil {
		return Config{}, err
	}

	qbtUsername, err := requireEnv("QBITTORRENT_USERNAME")
	if err != nil {
		return Config{}, err
	}

	qbtPassword, err := requireEnv("QBITTORRENT_PASSWORD")
	if err != nil {
		return Config{}, err
	}

	pollInterval, err := parsePollInterval(os.Getenv("POLL_INTERVAL"))
	if err != nil {
		return Config{}, err
	}

	return Config{
		TelegramToken: token,
		AllowedUsers:  allowedUsers,
		QBTBaseURL:    qbtURL,
		QBTUsername:   qbtUsername,
		QBTPassword:   qbtPassword,
		PollInterval:  pollInterval,
	}, nil
}

// requireEnv returns the value of an environment variable or an error if the
// variable is unset or empty.
func requireEnv(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("required environment variable %s is not set", key)
	}
	return v, nil
}

// parseAllowedUsers splits a comma-separated string of Telegram user IDs and
// converts each token to int64. It returns an error if the string is empty,
// contains only whitespace, or any token cannot be parsed as an integer.
func parseAllowedUsers(raw string) ([]int64, error) {
	parts := strings.Split(raw, ",")
	ids := make([]int64, 0, len(parts))

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid user ID %q in TELEGRAM_ALLOWED_USERS: must be an integer", p)
		}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("TELEGRAM_ALLOWED_USERS must contain at least one valid user ID")
	}

	return ids, nil
}

// parsePollInterval parses a duration string for POLL_INTERVAL. An empty
// string returns the default of 30 seconds.
func parsePollInterval(raw string) (time.Duration, error) {
	const defaultInterval = 30 * time.Second

	if raw == "" {
		return defaultInterval, nil
	}

	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid POLL_INTERVAL %q: %w", raw, err)
	}

	return d, nil
}
