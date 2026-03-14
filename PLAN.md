# Plan: tt-bot — Telegram Bot for qBittorrent

## Context

Empty project directory. Need to scaffold a Go project from scratch: a stateless Telegram bot that lets whitelisted users manage torrents via qBittorrent Web API v2. Must include Docker infrastructure and quality gates for automated feedback loops.

## Tech Stack

- **Language**: Go 1.22+ (latest stable)
- **Telegram**: `github.com/go-telegram-bot-api/telegram-bot-api/v5` (6k+ stars, most mature)
- **qBittorrent**: Custom thin HTTP client (existing Go libs are outdated/unmaintained)
- **Docker**: `lscr.io/linuxserver/qbittorrent:latest` for qBittorrent runtime
- **Lint**: golangci-lint

## Project Structure

```
tt-bot/
├── cmd/bot/main.go                    # Entry point
├── internal/
│   ├── config/config.go               # Env-based config
│   ├── config/config_test.go
│   ├── qbt/
│   │   ├── client.go                  # Interface definition
│   │   ├── types.go                   # Torrent, Category, ListOptions
│   │   ├── http.go                    # HTTP implementation (auth, multipart, re-auth on 403)
│   │   ├── http_test.go              # Unit tests with httptest
│   │   └── http_integration_test.go  # Integration tests (build tag)
│   ├── bot/
│   │   ├── handler.go                # Update dispatcher (commands, messages, documents)
│   │   ├── handler_test.go
│   │   ├── callback.go               # Callback query handler (category select, pagination)
│   │   ├── callback_test.go
│   │   ├── auth.go                   # Whitelist authorizer
│   │   ├── auth_test.go
│   │   ├── sender.go                 # Sender interface (wraps bot.Send for testability)
│   │   └── e2e_test.go              # Flow tests (build tag: integration)
│   ├── formatter/
│   │   ├── format.go                 # Message formatting + pagination keyboards
│   │   └── format_test.go
│   └── poller/
│       ├── poller.go                 # Completion notification goroutine
│       └── poller_test.go
├── Dockerfile                         # Multi-stage Go build
├── docker-compose.yml                 # Production: bot + qbittorrent
├── docker-compose.test.yml           # Test: qbittorrent + integration runner
├── testdata/qbt-config/qBittorrent.conf  # Pre-configured test credentials
├── scripts/wait-for-qbt.sh          # Health check script
├── Makefile                          # Quality gate targets
├── .golangci.yml                     # Linter config
├── .gitignore
├── .env.example
└── go.mod
```

## Implementation Phases

### Phase 1: Scaffold + Config ✅ (partial — scaffolding done)
- `go mod init`, `.gitignore`, `.env.example`, `Makefile`, `.golangci.yml` — DONE
- `internal/config/config.go` — load from env vars:
  - `TELEGRAM_BOT_TOKEN`, `TELEGRAM_ALLOWED_USERS` (comma-separated int64 IDs)
  - `QBITTORRENT_URL`, `QBITTORRENT_USERNAME`, `QBITTORRENT_PASSWORD`
  - `POLL_INTERVAL` (default `30s`)
- Unit tests for config parsing/validation
- **Gate**: `go build ./...` passes

### Phase 2: qBittorrent Client
- `internal/qbt/client.go` — interface:
  - `Login(ctx) error`
  - `AddMagnet(ctx, magnet, category) error`
  - `AddTorrentFile(ctx, filename, data io.Reader, category) error`
  - `ListTorrents(ctx, ListOptions) ([]Torrent, error)`
  - `Categories(ctx) ([]Category, error)`
- `internal/qbt/types.go` — Torrent, Category, TorrentFilter, ListOptions
- `internal/qbt/http.go` — implementation:
  - SID cookie-based auth, auto re-auth on 403
  - Multipart form for torrent uploads
  - `sync.Mutex` on re-auth to prevent thundering herd
- Unit tests against `httptest.NewServer`
- **Gate**: build + lint + unit tests pass

### Phase 3: Docker Test Infrastructure
- `docker-compose.test.yml` with qBittorrent + healthcheck
- `testdata/qbt-config/qBittorrent.conf` with known credentials (`admin`/`testpass`) and `AuthSubnetWhitelistEnabled=true` as fallback
- `scripts/wait-for-qbt.sh` — polls `/api/v2/app/version`
- `internal/qbt/http_integration_test.go` (build tag `integration`):
  - Login succeeds
  - Add magnet + list returns the torrent
  - Categories returns data
- **Gate**: integration tests pass against real qBittorrent

### Phase 4: Message Formatter
- Telegram limits: 4096 chars/message, 64 bytes/callback data
- Torrent list: 5 per page, names truncated to 40 chars, progress/speed/state per line
- `CategoryKeyboard()` — inline keyboard, one button per category, callback `cat:<name>` (truncated to 58 bytes)
- `PaginationKeyboard()` — `[<< Prev] [Page X/Y] [Next >>]`, callback `pg:all:<N>` / `pg:act:<N>`
- Unit tests assert worst-case output stays under 4096 chars
- **Gate**: unit tests pass

### Phase 5: Bot Handlers
- `internal/bot/auth.go` — set-based whitelist lookup
- `internal/bot/sender.go` — `Sender` interface wrapping `bot.Send` (testable)
- `internal/bot/handler.go` — dispatches updates:
  - `/start`, `/help` — welcome text
  - `/list` — all torrents page 1
  - `/active` — active torrents page 1
  - Text containing `magnet:?` — store in pending map, show category keyboard
  - Document with `.torrent` — download file bytes, store pending, show category keyboard
  - Unauthorized users get "Access denied" reply
- `internal/bot/callback.go`:
  - `cat:<name>` — add pending torrent with category, confirm
  - `pg:all:<N>` / `pg:act:<N>` — edit message with requested page
- Pending torrent state: in-memory `map[int64]*PendingTorrent` (per user), 5min expiry cleanup
- Unit tests with mock qbt.Client and mock Sender
- **Gate**: unit tests pass

### Phase 6: Completion Poller
- `internal/poller/poller.go`:
  - On startup: fetch all torrents, seed `knownHashes` with already-completed ones (no spurious notifications)
  - Every `PollInterval`: fetch torrents, detect newly completed (`progress >= 1.0` and not in knownHashes), notify all allowed users
  - Prune hashes for deleted torrents
- `Notifier` interface — implemented by bot sender
- Unit tests: verify no notification on first poll, notification on new completion, no duplicates
- **Gate**: unit tests pass

### Phase 7: Entry Point + Production Docker
- `cmd/bot/main.go`: wire config → qbt client → bot → poller, graceful shutdown via SIGINT/SIGTERM
- Long-polling mode (not webhooks)
- `Dockerfile` — multi-stage: `golang:1.22-alpine` builder → `alpine:3.19` runtime
- `docker-compose.yml` — bot + qbittorrent services
- `.env.example` with all required vars
- **Gate**: `docker-compose up` starts both services, bot logs "connected"

### Phase 8: E2E Tests & CI
- Makefile and `.golangci.yml` already created in Phase 1
- `internal/bot/e2e_test.go` (build tag `integration`):
  - Unauthorized user rejected
  - Add magnet with category selection flow (mock Telegram, real qBittorrent)
  - List pagination returns correct pages
- **Gate**: `make gate-all` green

## Key Design Decisions

| Decision | Choice | Why |
|----------|--------|-----|
| Telegram polling vs webhooks | Long-polling | Simpler, no TLS/domain needed, works behind NAT |
| qBittorrent client | Custom | Existing Go libs are outdated; API surface is small (~5 endpoints) |
| Pending torrent state | In-memory map | Stateless requirement; lost on restart is acceptable |
| Completion detection | `progress >= 1.0` | More reliable than checking `state` which has multiple completed states |
| Callback data encoding | Short prefixes (`cat:`, `pg:`) | Fits within 64-byte Telegram limit |
| Test qBittorrent auth | Pre-configured conf file | Avoids random password problem with linuxserver image |

## Verification

1. **Unit tests**: `go test ./... -short -cover` — target 80%+ coverage
2. **Integration tests**: `make test-integration` — real qBittorrent API calls
3. **E2E flow tests**: `make test-e2e` — simulated bot flows against real qBittorrent
4. **Manual smoke test**: `docker-compose up`, send `/list` and magnet link to bot via Telegram
5. **Lint**: `golangci-lint run` clean

## Model Routing

| Phase | Model | Why |
|-------|-------|-----|
| All implementation (1→8) | **Sonnet** | Detailed plan, standard Go patterns, no ambiguity |
| Fallback | **Opus** | Only if integration test design or Docker orchestration hits unexpected complexity |

## Implementation Notes for Sonnet

- Implement phases sequentially (1→8), running quality gates after each
- Use interfaces everywhere for testability (qbt.Client, bot.Sender, poller.Notifier)
- All public functions get doc comments
- No external dependencies beyond `telegram-bot-api/v5` — stdlib for HTTP, JSON, multipart
- Immutable config (value type, not pointer)
- Context propagation on all I/O methods
