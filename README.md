# tt-bot

A stateless Telegram bot for managing qBittorrent downloads.

## Features

- **Add torrents** — send magnet links or `.torrent` files
- **Category selection** — pick a qBittorrent category via inline keyboard before adding
- **List torrents** — paginated views of all or active downloads
- **Completion notifications** — get notified when a download finishes
- **Access control** — whitelist Telegram users by ID

## Quick Start

### Prerequisites

- Go 1.22+ (for local development)
- Docker & Docker Compose (for production or integration tests)
- A Telegram bot token from [@BotFather](https://t.me/BotFather)
- A running qBittorrent instance with WebUI enabled

### Configuration

Copy `.env.example` to `.env` and fill in your values:

```bash
cp .env.example .env
```

| Variable | Required | Description |
|----------|----------|-------------|
| `TELEGRAM_BOT_TOKEN` | Yes | Bot token from @BotFather |
| `TELEGRAM_ALLOWED_USERS` | Yes | Comma-separated Telegram user IDs |
| `QBITTORRENT_URL` | Yes | WebUI URL (e.g. `http://localhost:8080`) |
| `QBITTORRENT_USERNAME` | Yes | WebUI username |
| `QBITTORRENT_PASSWORD` | Yes | WebUI password |
| `POLL_INTERVAL` | No | How often to check for completed downloads (default: `30s`) |

### Run with Docker Compose

```bash
docker compose up --build
```

This starts both qBittorrent and the bot. The bot connects to qBittorrent via Docker's internal network.

### Run locally

```bash
# Export env vars
set -a && source .env && set +a

# Run
go run ./cmd/bot/
```

## Bot Commands

| Command | Description |
|---------|-------------|
| `/list` | List all torrents (paginated) |
| `/active` | List active downloads (paginated) |
| `/help` | Show help message |

You can also send:
- A **magnet link** — the bot prompts you to pick a category, then adds it
- A **.torrent file** — same flow as magnet links

## Development

```bash
make build          # Build
make lint           # Lint with golangci-lint
make test           # Unit tests with coverage
make test-integration  # Integration + E2E tests against real qBittorrent in Docker
make gate-all       # Full quality gate: build → lint → test
```

### Test Coverage

| Package | Coverage |
|---------|----------|
| config | 91.3% |
| formatter | 94.8% |
| poller | 88.2% |
| bot | 81.5% |
| qbt | 77.6% |

### Architecture

```
cmd/bot/main.go        Entry point — wires everything, runs Telegram long-polling
internal/config/       Environment variable loading and validation
internal/qbt/          qBittorrent Web API v2 client (interface + HTTP implementation)
internal/bot/          Telegram update dispatcher, auth, callbacks, pending state
internal/formatter/    Message formatting within Telegram limits, pagination keyboards
internal/poller/       Background goroutine detecting completed downloads
```

Key design decisions:
- **Stateless** — all state is in-memory and lost on restart (by design)
- **Interface-driven** — `qbt.Client`, `bot.Sender`, `poller.Notifier` for testability
- **Telegram-safe** — messages stay under 4096 chars, callback data under 64 bytes

## License

MIT
