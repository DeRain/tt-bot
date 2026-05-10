# Repository Guidelines

## Project

tt-bot is a stateless Go Telegram bot for managing qBittorrent downloads. Whitelisted Telegram users can add torrents (magnet links and .torrent files), pick categories via inline keyboards, list torrents with pagination, and receive completion notifications.

## Project Structure & Module Organization

`cmd/bot/main.go` is the entrypoint. Core packages live under `internal/`: `bot/` for Telegram handlers and callbacks, `qbt/` for the qBittorrent Web API client, `config/` for env loading, `formatter/` for Telegram-safe output, and `poller/` for completion notifications. Repository docs live in `docs/`; feature work is tracked in `docs/features/<feature-id>/` with `spec.md`, `design.md`, `plan.md`, `traceability.md`, and `verification.md`. Docker and local orchestration files are in the repo root, and helper scripts are in `scripts/`.

## Architecture Overview

The bot is intentionally stateless. Runtime wiring happens in `cmd/bot/main.go`; request handling stays in `internal/bot/`, qBittorrent API access stays in `internal/qbt/`, and completion polling stays in `internal/poller/`. Prefer extending existing interfaces before introducing cross-package coupling. Keep Telegram-specific limits in mind: formatted messages must stay within Telegram size limits, and callback payloads should remain compact.

```
cmd/bot/main.go          → wires config, qbt client, bot handler, poller; long-polling mode
internal/config/          → env-var loading
internal/qbt/             → qBittorrent Web API v2 client (interface in client.go, HTTP impl in http.go)
internal/bot/             → Telegram update dispatcher, callback handler, auth whitelist, Sender interface
internal/formatter/       → message formatting respecting Telegram 4096-char limit, pagination keyboards
internal/poller/          → background goroutine polling for completed torrents
```

**Key interfaces**: `qbt.Client`, `bot.Sender`, `poller.Notifier` — mock these in unit tests.

**Telegram constraints**: messages max 4096 UTF-8 chars, callback data max 64 bytes. Callback encoding uses short prefixes: `cat:<name>`, `pg:all:<page>`, `pg:act:<page>`.

**Design**: stateless, pending torrents in in-memory map with 5-min expiry, completion poller tracks known hashes in memory.

**Auth**: qBittorrent uses SID cookie auth with auto-re-login on 403. Telegram users whitelisted by numeric ID.

## Model Routing

This project uses role-based delegation to optimize cost and quality. The orchestrator (main session) plans and reviews; implementation subagents execute code changes; lightweight subagents handle verification.

### Claude Code Routing

| Role | Model | Scope |
|------|-------|-------|
| Orchestration | Opus | Docs (`.md`), plans, git ops, `.claude/` config, review |
| Implementation | Sonnet (subagents) | `*.go`, `*.yaml`, `Dockerfile`, `docker-compose*.yml`, `Makefile`, `*.toml`, `*.json`, `.env*` |
| Gate checks | Haiku (subagents) | Lint, build, coverage verification |

Dispatch implementation subagents with `model: sonnet`. Dispatch gate checks with `model: haiku`.

### OpenCode Routing (DeepSeek)

| Role | Agent | DeepSeek Model | Cost/1M tokens |
|------|-------|---------------|----------------|
| Orchestration | **build** (primary) | `deepseek/deepseek-v4-pro` | $0.435 / $0.87 |
| Implementation | **@implementer** (subagent) | `deepseek/deepseek-v4-flash` | $0.14 / $0.28 |
| Code review | **@reviewer** (subagent) | `deepseek/deepseek-v4-pro` | $0.435 / $0.87 |
| Exploration | **explore** (built-in) | `deepseek/deepseek-v4-flash` | $0.14 / $0.28 |

Implementation uses `deepseek-v4-flash` because tasks follow a detailed plan (`docs/features/<feature-id>/plan.md`) with explicit `TASK-*` steps — the model just needs to execute, not design. This saves ~3x in cost vs v4-pro.

To configure in `opencode.json`:
```json
{
  "model": "deepseek/deepseek-v4-pro",
  "agent": {
    "implementer": { "mode": "subagent", "model": "deepseek/deepseek-v4-flash" },
    "reviewer": { "mode": "subagent", "model": "deepseek/deepseek-v4-pro" }
  }
}
```

### Cross-Tool Equivalence

| Concern | Claude Code | OpenCode |
|---------|------------|----------|
| Orchestration | Opus | build primary (v4-pro) |
| Implementation | Sonnet subagent | @implementer (v4-flash) |
| Code review | Opus | @reviewer (v4-pro) |
| Gate checks | Haiku subagent | explore (v4-flash) |

## Build, Test, and Development Commands

| Command | Description |
|---------|-------------|
| `make build` | `go build ./...` |
| `make lint` | `golangci-lint run` |
| `make test` | Unit tests with coverage (`go test ./... -short -cover`) |
| `make test-integration` | Integration + E2E tests in Docker (spins up qBittorrent, runs all `Integration\|E2E` tests, tears down) |
| `make gate-all` | Full quality gate: build → lint → unit tests |
| `make clean` | Remove coverage.out and bot binary |

For local services, use `docker compose up --build` to run the bot with qBittorrent. For focused test runs, use commands such as `go test ./internal/qbt -run TestLogin -short -v`.

**Integration tests are MANDATORY.** Always run `make test-integration` before marking any AC as PASS or any feature as complete. Unit tests with mocks cannot catch real API contract issues — endpoint renames, response format changes, and auth behavior differences are invisible to httptest-based tests. This was learned the hard way: qBittorrent v5 renamed `/pause` → `/stop` and `/resume` → `/start`, and only `make test-integration` caught the 404s.

## Pre-Commit Quality Gate (MANDATORY)

**See `docs/gates.md` for full gate definitions and the Iterative Harness Loop Protocol.**

After implementation is complete, before committing:

1. Run `make gate-all` (build + lint + unit tests)
2. Run `make test-integration` (Docker-based integration + E2E tests)
3. Review changes for traceability (every change maps to a TASK-*/REQ-*)
4. Address any code review findings
5. **ONLY THEN**: commit, push, create PR

Never trust an implementation subagent's claim of "all tests pass" — independently verify.

## Protected Files

The following files define structural contracts and must NOT be modified without explicit operator approval:

| File | Reason |
|------|--------|
| `arch-go.yml` | Architecture dependency rules — defines the layered dependency graph |
| `.gitignore` | Build and security boundary |
| `docker-compose.test.yml` | CI test environment contract |

To change a protected file: create a feature doc (`docs/features/<id>/spec.md`), design the change, get operator review, then implement. Never modify these files as a side effect of another task.

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `TELEGRAM_BOT_TOKEN` | Yes | Bot token from @BotFather |
| `TELEGRAM_ALLOWED_USERS` | Yes | Comma-separated Telegram user IDs (int64) |
| `QBITTORRENT_URL` | Yes | WebUI URL (overridden in docker-compose.yml) |
| `QBITTORRENT_USERNAME` | Yes | WebUI username |
| `QBITTORRENT_PASSWORD` | Yes | WebUI password |
| `POLL_INTERVAL` | No | Completion poll interval (default: `30s`) |

Copy `.env.example` to `.env` and keep secrets out of git.

## Coding Style & Naming Conventions

Follow standard Go formatting with tabs, `gofmt`, and `goimports`; do not hand-format files. Keep package names short and lowercase. Exported identifiers use `CamelCase`; unexported helpers use `camelCase`. Prefer table-driven tests and small interfaces such as `qbt.Client` or `bot.Sender`. Name tests in `_test.go` files with clear behavior-focused names like `TestHandleListUnauthorized`.

## Testing Guidelines

Unit tests sit next to implementation files and should run with `go test ./... -short`. Integration coverage uses Docker and should validate real API behavior, not only mocks. Treat `make test-integration` as required before considering feature work complete. Keep coverage at or above the repository's 80% expectation.

When changing qBittorrent control flows, auth, or callback behavior, add or update integration coverage instead of relying only on mocked HTTP tests. Preserve the existing package-local test layout: `internal/<pkg>/*_test.go`.

### Test Organization
- `_test.go` without build tags: unit tests, run with `-short`, use `httptest.NewServer` or mock interfaces
- `//go:build integration` tagged files: require real qBittorrent via Docker
- `e2e_test.go`: end-to-end bot flow tests

### Co-Change Pattern

Never commit an implementation change without updating the corresponding test file:

| Primary Change | Must Co-Change |
|---------------|----------------|
| `bot/callback.go` | `bot/callback_test.go`, `bot/handler_test.go` |
| `formatter/format.go` | `formatter/format_test.go` |
| `qbt/http.go` | `qbt/http_test.go`, `qbt/client.go` |
| `qbt/client.go` (interface) | `qbt/http.go` (impl), `qbt/http_test.go` |

### Callback Data Encoding

Telegram limits callback_data to 64 bytes. Use short colon-delimited prefixes:
- `cat:<name>`, `pg:all:<N>`, `pg:act:<N>`, `sel:<hash>`, `rm:<hash>`, `noop`

Always truncate user-provided strings at valid UTF-8 boundaries, not raw byte offsets.

### qBittorrent v5+ Endpoints

qBittorrent v5+ renamed endpoints. Always use the current names:

| Action | v4 (old, 404 on v5) | v5+ (current) |
|--------|---------------------|---------------|
| Pause/Stop | `/api/v2/torrents/pause` | `/api/v2/torrents/stop` |
| Resume/Start | `/api/v2/torrents/resume` | `/api/v2/torrents/start` |

## Interface-Driven Design

All external dependencies are behind interfaces. When adding new qBittorrent API calls:
1. Add method to `qbt.Client` interface in `internal/qbt/client.go`
2. Implement in `internal/qbt/http.go`
3. Wire into bot handler via `internal/bot/callback.go` or `internal/bot/handler.go`

## Docs-First Workflow

Non-trivial changes should start in `docs/features/<feature-id>/`. Define requirements in `spec.md`, map them in `design.md`, break them into `TASK-*` items in `plan.md`, then record implementation evidence in `traceability.md` and results in `verification.md`. Keep identifiers stable and reference them in code changes, commits, and PR descriptions where practical.

### Identifier Conventions

| Type | Format | Example |
|------|--------|---------|
| Requirement | REQ-N | REQ-1 |
| Acceptance criterion | AC-N.M | AC-1.2 |
| Design item | DES-N | DES-3 |
| Plan task | TASK-N | TASK-5 |
| Test | TEST-N | TEST-2 |
| Manual check | CHECK-N | CHECK-1 |

### Existing Features

| Feature ID | Description | Docs |
|------------|-------------|------|
| auth | User authorization via Telegram ID whitelist | `docs/features/auth/` |
| add-torrent | Add torrents via magnet links and .torrent files | `docs/features/add-torrent/` |
| list-torrents | List all/active torrents with pagination | `docs/features/list-torrents/` |
| completion-notifications | Background polling and completion alerts | `docs/features/completion-notifications/` |
| config | Environment variable loading and validation | `docs/features/config/` |
| set-commands | Register bot commands with Telegram on startup | `docs/features/set-commands/` |
| downloading-list | List non-completed torrents | `docs/features/downloading-list/` |
| uploading-list | List seeding/uploading torrents | `docs/features/uploading-list/` |
| torrent-files | View and manage torrent file priorities | `docs/features/torrent-files/` |
| torrent-detail-extra | Extended torrent info (ETA, ratio, seeds) | `docs/features/torrent-detail-extra/` |
| torrent-remove | Safe torrent removal with confirmation | `docs/features/torrent-remove/` |
| torrent-control | Start/stop/force-start/recheck operations | `docs/features/torrent-control/` |
| status-emojis | Per-state emoji indicators in lists | `docs/features/status-emojis/` |

## Commit & Pull Request Guidelines

Format: `feat(<feature-id>): TASK-N description` or `fix(<feature-id>): TASK-N description`. Reference related `REQ-*`, `TASK-*`, and acceptance criteria in the PR description. PRs should include verification results for `make build`, `make lint`, `make test`, and, when applicable, `make test-integration`, plus updates to `traceability.md` and `verification.md`. Open pull requests against `main`.

See `docs/pr-checklist.md` for the full PR template and validation rules.

## Branching & Review

Use `main` as the base branch for PRs. Keep changes scoped to one feature or fix. Reviewers should be able to trace each code change back to the relevant feature docs.

## Security & Configuration Tips

Never commit real bot tokens, Telegram user IDs that are not already intended for the repo, or qBittorrent credentials. If configuration changes affect deployment behavior, update `README.md` and the relevant feature docs in the same change.
