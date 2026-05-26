# Repository Guidelines

**Generated:** 2026-05-23
**Commit:** 5278f56
**Branch:** main

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

## Build, Test, and Development Commands

| Command | Description |
|---------|-------------|
| `make build` | `go build ./...` |
| `make lint` | `golangci-lint run` |
| `make test` | Unit tests with coverage (`go test ./... -short -cover`) |
| `make test-integration` | Integration + E2E tests in Docker (spins up qBittorrent, runs all `Integration\|E2E` tests, tears down) |
| `make arch-check` | Validate architecture dependency rules (`arch-go.yml`) |
| `make gate-all` | Full quality gate: build → lint → test → arch-check |
| `make clean` | Remove coverage.out and bot binary |

For local services, use `docker compose up --build` to run the bot with qBittorrent. For focused test runs, use commands such as `go test ./internal/qbt -run TestLogin -short -v`.

**Integration tests are MANDATORY.** Always run `make test-integration` before marking any AC as PASS or any feature as complete. Unit tests with mocks cannot catch real API contract issues — endpoint renames, response format changes, and auth behavior differences are invisible to httptest-based tests. This was learned the hard way: qBittorrent v5 renamed `/pause` → `/stop` and `/resume` → `/start`, and only `make test-integration` caught the 404s.

**Integration/E2E tests MUST be run via Docker (`make test-integration`), NOT locally.** The test environment spins up qBittorrent + Jackett via `docker-compose.test.yml` and runs all `//go:build integration` tests. Running `go test -tags=integration` locally will fail because there is no qBittorrent instance at `localhost:18080`. Always use the full Docker command: `make test-integration`.

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
| `VIEW_REFRESH_INTERVAL` | No | Live view auto-refresh interval (default: `5s`) |

Copy `.env.example` to `.env` and keep secrets out of git.

## Coding Style

Follow standard Go formatting with tabs, `gofmt`, and `goimports`; do not hand-format files. Keep package names short and lowercase. Exported identifiers use `CamelCase`; unexported helpers use `camelCase`. Prefer table-driven tests and small interfaces such as `qbt.Client` or `bot.Sender`. Name tests in `_test.go` files with clear behavior-focused names like `TestHandleListUnauthorized`.

## Testing Guidelines

Unit tests run with `go test ./... -short`. Integration tests (`make test-integration`) are **mandatory** before marking any feature complete — mocks cannot catch API contract changes.

**Test types:** `_test.go` (unit, `-short`), `//go:build integration` (Docker+qBittorrent), `e2e_test.go` (bot flow).

**Co-change pattern:** Never change implementation without updating tests:

| Primary Change | Must Co-Change |
|---------------|----------------|
| `bot/callback.go` | `bot/callback_test.go`, `bot/handler_test.go` |
| `formatter/format.go` | `formatter/format_test.go` |
| `qbt/http.go` | `qbt/http_test.go`, `qbt/client.go` |
| `qbt/client.go` (interface) | `qbt/http.go` (impl), `qbt/http_test.go` |

**Callback data:** max 64 bytes. Prefixes: `cat:<name>`, `pg:all:<N>`, `pg:act:<N>`, `sel:<hash>`, `rm:<hash>`, `noop`. Truncate at valid UTF-8 boundaries.

**qBittorrent v5+:** use `/torrents/stop` and `/torrents/start`; v4 `/pause` and `/resume` return 404.

### Mutation Testing Rules

- `make mutation-test-pr` runs on every PR with **100% efficacy threshold**. Any surviving mutant blocks the CI check.
- **New code MUST NOT use `gomutants:disable` comments.** Fix all mutants with real tests.
- **Touching old code that has existing `gomutants:disable` comments REQUIRES fixing the underlying mutant** — remove the suppression and add a test.
- **Equivalent mutants** (behaviorally identical to original) should be fixed by **restructuring the code** to eliminate the untestable branch, not by suppressing the mutant.
- Run `make mutation-test-pr` locally before pushing to confirm 0 lived mutants.

## Interface-Driven Design

All external dependencies are behind interfaces. When adding new qBittorrent API calls:
1. Add method to `qbt.Client` interface in `internal/qbt/client.go`
2. Implement in `internal/qbt/http.go`
3. Wire into bot handler via `internal/bot/callback.go` or `internal/bot/handler.go`

## Feature Documentation

Features are documented in `docs/features/<feature-id>/` with:
- `spec.md` — requirements
- `design.md` — architecture decisions
- `plan.md` — implementation tasks
- `traceability.md` — evidence mapping
- `verification.md` — test results

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

See `docs/features/<feature-id>/` for each feature's spec, design, plan, traceability, and verification. Active features include: auth, add-torrent, list-torrents, completion-notifications, downloading-list, uploading-list, torrent-files, torrent-detail-extra, torrent-remove, torrent-control, status-emojis.

## Commit & Pull Request Guidelines

Format: `feat(<feature-id>): TASK-N description` or `fix(<feature-id>): TASK-N description`. Reference related `REQ-*`, `TASK-*`, and acceptance criteria in the PR description. PRs should include verification results for `make build`, `make lint`, `make test`, and, when applicable, `make test-integration`, plus updates to `traceability.md` and `verification.md`. Open pull requests against `main`.

See `docs/pr-checklist.md` for the full PR template and validation rules.

## Branching & Review

Use `main` as the base branch for PRs. Keep changes scoped to one feature or fix. Reviewers should be able to trace each code change back to the relevant feature docs.

## Security

Never commit real bot tokens, Telegram user IDs that are not already intended for the repo, or qBittorrent credentials. If configuration changes affect deployment behavior, update `README.md` and the relevant feature docs in the same change.

<!-- code-intel:start -->
# Code Intelligence

LSP: gopls (Go). Key tools: `lsp_find_references`, `lsp_goto_definition`, `lsp_symbols`, `lsp_diagnostics`.

CLI fallbacks: `gopls references/implementation/definition file:line:col`, `sg -p 'pattern' -l go`, `rg 'pattern'`, `fd 'pattern'`.

Pre-commit: `lsp_diagnostics` → `make gate-all` → `make test-integration` (API changes).
<!-- code-intel:end -->
