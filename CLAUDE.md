# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

tt-bot is a stateless Go Telegram bot for managing qBittorrent downloads. Whitelisted Telegram users can add torrents (magnet links and .torrent files), pick categories via inline keyboards, list torrents with pagination, and receive completion notifications.

## Build & Test Commands

```bash
# Build
go build ./...

# Lint
golangci-lint run

# Unit tests (mocked dependencies, no Docker needed)
go test ./... -short -cover

# Single test
go test ./internal/qbt/ -run TestLogin -short -v

# Integration tests (requires qBittorrent running via docker-compose.test.yml)
docker compose -f docker-compose.test.yml up -d qbittorrent
./scripts/wait-for-qbt.sh
go test ./... -tags=integration -run Integration -v
docker compose -f docker-compose.test.yml down

# E2E flow tests
docker compose -f docker-compose.test.yml up --build --abort-on-container-exit
docker compose -f docker-compose.test.yml down

# All quality gates
make gate-all
```

## Architecture

```
cmd/bot/main.go          → wires config, qbt client, bot handler, poller; long-polling mode
internal/config/          → env-var loading (TELEGRAM_BOT_TOKEN, TELEGRAM_ALLOWED_USERS, QBITTORRENT_*)
internal/qbt/             → qBittorrent Web API v2 client (interface in client.go, HTTP impl in http.go)
internal/bot/             → Telegram update dispatcher, callback handler, auth whitelist, Sender interface
internal/formatter/       → message formatting respecting Telegram 4096-char limit, pagination keyboards
internal/poller/          → background goroutine polling for completed torrents, sends notifications
```

**Key interfaces** (all in `internal/`): `qbt.Client`, `bot.Sender`, `poller.Notifier` — mock these in unit tests.

**Telegram constraints**: messages max 4096 UTF-8 chars, callback data max 64 bytes. Callback encoding uses short prefixes: `cat:<name>`, `pg:all:<page>`, `pg:act:<page>`.

**Stateless design**: pending torrent state (between magnet/file submission and category selection) lives in an in-memory map keyed by user ID with 5-minute expiry. Completion poller tracks known hashes in memory. All state is lost on restart — this is intentional.

**Auth flow**: qBittorrent uses SID cookie auth with automatic re-login on 403. Telegram users are whitelisted by numeric ID via `TELEGRAM_ALLOWED_USERS` env var.

## Test Organization

- `_test.go` files without build tags: unit tests, run with `-short`, use `httptest.NewServer` or mock interfaces
- `//go:build integration` tagged files: require real qBittorrent via Docker, test actual API calls and bot flows
