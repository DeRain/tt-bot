# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

tt-bot is a stateless Go Telegram bot for managing qBittorrent downloads. Whitelisted Telegram users can add torrents (magnet links and .torrent files), pick categories via inline keyboards, list torrents with pagination, and receive completion notifications.

## Build & Test Commands

<!-- AUTO-GENERATED from Makefile -->
| Command | Description |
|---------|-------------|
| `make build` | `go build ./...` |
| `make lint` | `golangci-lint run` |
| `make test` | Unit tests with coverage (`go test ./... -short -cover`) |
| `make test-integration` | Integration + E2E tests in Docker (spins up qBittorrent, runs all `Integration\|E2E` tests, tears down) |
| `make gate-all` | Full quality gate: build → lint → unit tests |
| `make clean` | Remove coverage.out and bot binary |
<!-- END AUTO-GENERATED -->

```bash
# Run a single test
go test ./internal/qbt/ -run TestLogin -short -v

# Run production stack
docker compose up --build
```

## Environment Variables

<!-- AUTO-GENERATED from .env.example -->
| Variable | Required | Description |
|----------|----------|-------------|
| `TELEGRAM_BOT_TOKEN` | Yes | Bot token from @BotFather |
| `TELEGRAM_ALLOWED_USERS` | Yes | Comma-separated Telegram user IDs (int64) |
| `QBITTORRENT_URL` | Yes | WebUI URL (overridden in docker-compose.yml) |
| `QBITTORRENT_USERNAME` | Yes | WebUI username |
| `QBITTORRENT_PASSWORD` | Yes | WebUI password |
| `POLL_INTERVAL` | No | Completion poll interval (default: `30s`) |
<!-- END AUTO-GENERATED -->

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


## RAG and Search Tools

Use **claude-context** MCP server (`@zilliz/claude-context-mcp`) as the semantic search tool for the codebase. Fall back to built-in tools (Grep, Glob, Read) when claude-context returns insufficient results.

### claude-context Availability

- **On session start**, if claude-context is unavailable, attempt to reconnect via `/mcp` before proceeding.
- **During workload**, if a claude-context call fails with a connection error, reconnect via `/mcp` and retry once. If reconnection fails, fall back to built-in tools (Grep, Glob, Read) and notify the user.
- **Dependencies**: Requires Milvus (`docker compose up milvus-standalone`) and Ollama running locally.

### claude-context Tools

| Tool | Purpose |
|------|---------|
| `search_code` | Semantic search using natural language queries. Requires absolute path. Supports `extensionFilter` and `limit` params. |
| `index_codebase` | Index a codebase directory for semantic search. Uses AST splitter by default. Only needed if search fails due to unindexed codebase. |
| `get_indexing_status` | Check indexing progress for a codebase directory. |
| `clear_index` | Clear the search index for a codebase directory. |

### Re-indexing-

Uses incremental indexing (Merkle trees) — only changed files are re-processed. Re-index (`index_codebase`) after significant code changes (new modules, large refactors, branch switches).

### Tool Selection Guide

| Task | Primary Tool | Fallback |
|------|-------------|----------|
| Find code related to a concept | `search_code` | Grep |
| Find a symbol definition | `search_code` | Grep |
| Find all references to a symbol | `search_code` | Grep |
| Explore a module's structure | `search_code` | Glob + Read |
| Find files by name/pattern | Glob | — |
| Exact text/regex search | Grep | `search_code` |
| Read full file content | Read | — |
| Run linting | `golangci-lint run` via Bash | — |

### Fallback Chain

1. **Always try `search_code` first** for code discovery, symbol lookup, and understanding code structure.
2. **Fall back to Grep** for exact text/regex matches or when `search_code` returns insufficient results.
3. **Fall back to Glob** for file name/pattern discovery.
4. **Fall back to Read** for full file content when needed.

### Token-Efficiency Rules

1. Use `search_code` as the primary discovery tool — it returns relevant chunks without reading entire files.
2. Use Grep/Glob when you need exact text matches or file patterns.
3. Read full file bodies only when search results are insufficient.
4. Avoid repeated scans of unchanged areas.
5. Batch parallelizable reads/searches — do not serialize independent commands.
6. If a command fails, diagnose once, pivot strategy, continue — no blind retry loops.

### Anti-Patterns to Avoid

1. Reading entire files without searching first — use `search_code` to find relevant chunks.
2. Running independent commands sequentially when they can be parallelized.
3. Repeating failed commands without changing inputs or approach.
4. Skipping `search_code` and jumping straight to Grep/Glob/Read for discovery tasks.
5. Never trust a single empty result — verify with Grep before concluding something has no references.
