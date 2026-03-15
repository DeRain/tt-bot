# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

tt-bot is a stateless Go Telegram bot for managing qBittorrent downloads. Whitelisted Telegram users can add torrents (magnet links and .torrent files), pick categories via inline keyboards, list torrents with pagination, and receive completion notifications.

## Model Routing Enforcement (MANDATORY)

Opus MUST NOT write implementation code directly. All implementation file edits MUST be dispatched to Sonnet agents via the `model: sonnet` parameter.

### Role Separation

| Role | Model | Scope |
|------|-------|-------|
| Orchestration | Opus | Docs (`.md`), plans, memory, git ops, `.claude/` config, CLAUDE.md, review |
| Implementation | Sonnet (agents) | All code and config files |
| Gate checks | Haiku (agents) | Lint, build verification, coverage checks |

### Implementation Files (Sonnet only)

`*.go`, `*.yaml`, `*.yml`, `Dockerfile`, `docker-compose*.yml`, `Makefile`, `*.toml`, `*.json` (non-claude config), `.env*`

### Orchestration Files (Opus ok)

`*.md`, `.claude/` directory config, `CLAUDE.md`

### Enforcement

- Before editing any implementation file, Opus MUST dispatch a Sonnet agent instead
- Opus may only edit implementation files for trivial 1-line fixes after a Sonnet agent has already done the main work
- Violating this rule wastes tokens at 5x cost and contradicts the project's model routing strategy

## Pre-Commit Quality Gate Checklist (MANDATORY)

**See `docs/gates.md` for full gate definitions, harness commands, and the Iterative Harness Loop Protocol.**

**After Sonnet implementation agent returns, Opus MUST NOT immediately commit. These steps are REQUIRED:**

1. Opus dispatches `go-reviewer` agent for code review
2. Opus independently runs `make test-integration` to verify
3. Address any code review findings (dispatch Sonnet if needed)
4. **ONLY THEN**: commit, push, create PR

**Steps 1-3 are NOT optional. The Sonnet agent's claim of "all tests pass" must be independently verified.**

## Sonnet Implementation Agent Prompts (MANDATORY)

Every Sonnet implementation agent prompt MUST include the gate requirements from @docs/gates.md — specifically the "Iterative Harness Loop Protocol" steps. Omitting gate requirements from the agent prompt is a violation of this project's workflow.

## Docs-First Feature Workflow (MANDATORY)

This repository uses **requirements-traceability** for all non-trivial work. Every feature requirement and acceptance criterion is traced through: specification → design → plan → implementation → verification.

### Rules

1. All non-trivial work MUST begin in `docs/features/<feature-id>/`.
2. `spec.md`, `design.md`, and `plan.md` are **validation gates** — they must exist and pass before implementation.
3. Implementation MUST NOT begin until:
   - `spec.md` defines requirements (REQ-*) and acceptance criteria (AC-*)
   - `design.md` maps every REQ-* to design items (DES-*)
   - `plan.md` maps every DES-* to tasks (TASK-*) with verification targets
4. Every code change MUST reference related REQ-* and TASK-* identifiers.
5. Every acceptance criterion MUST be validated in `verification.md` before a feature is complete.
6. Untraced code changes are considered **out-of-scope** until justified and documented.
7. Templates for all artifacts are in `docs/features/_templates/`.

### Feature Directory Structure

```
docs/features/<feature-id>/
  spec.md           → requirements and acceptance criteria
  design.md         → architecture decisions mapped to requirements
  plan.md           → ordered tasks mapped to design and requirements
  traceability.md   → bidirectional traceability matrix
  verification.md   → test/check results mapped to acceptance criteria
  decisions.md      → assumptions, choices, deferred work
```

### Identifier Conventions

| Type | Format | Example |
|------|--------|---------|
| Requirement | REQ-N | REQ-1 |
| Acceptance criterion | AC-req.N | AC-1.2 |
| Design item | DES-N | DES-3 |
| Plan task | TASK-N | TASK-5 |
| Automated test | TEST-N | TEST-2 |
| Manual check | CHECK-N | CHECK-1 |

### Quality Gates, ECC Compliance, and Agent Model Routing

All gate definitions, harness commands, ECC component mappings, and agent model routing are in:

@docs/gates.md

All ECC components operating in this repo MUST follow the docs-first workflow.

### Existing Features

| Feature ID | Description | Docs |
|------------|-------------|------|
| auth | User authorization via Telegram ID whitelist | `docs/features/auth/` |
| add-torrent | Add torrents via magnet links and .torrent files | `docs/features/add-torrent/` |
| list-torrents | List all/active torrents with pagination | `docs/features/list-torrents/` |
| completion-notifications | Background polling and completion alerts | `docs/features/completion-notifications/` |
| config | Environment variable loading and validation | `docs/features/config/` |
| set-commands | Register bot commands with Telegram on startup | `docs/features/set-commands/` |
| downloading-list | List non-completed torrents (paused + active) | `docs/features/downloading-list/` |

### PR and Commit Conventions

@docs/pr-checklist.md

## Build & Test Commands

| Command | Description |
|---------|-------------|
| `make build` | `go build ./...` |
| `make lint` | `golangci-lint run` |
| `make test` | Unit tests with coverage (`go test ./... -short -cover`) |
| `make test-integration` | Integration + E2E tests in Docker (spins up qBittorrent, runs all `Integration\|E2E` tests, tears down) |
| `make gate-all` | Full quality gate: build → lint → unit tests |
| `make clean` | Remove coverage.out and bot binary |

**Integration tests are MANDATORY.** Always run `make test-integration` before marking any AC as PASS or any feature as complete. Unit tests with mocks cannot catch real API contract issues — endpoint renames, response format changes, and auth behavior differences are invisible to httptest-based tests. This was learned the hard way: qBittorrent v5 renamed `/pause` → `/stop` and `/resume` → `/start`, and only `make test-integration` caught the 404s.

```bash
# Run a single test
go test ./internal/qbt/ -run TestLogin -short -v

# Run production stack
docker compose up --build
```

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `TELEGRAM_BOT_TOKEN` | Yes | Bot token from @BotFather |
| `TELEGRAM_ALLOWED_USERS` | Yes | Comma-separated Telegram user IDs (int64) |
| `QBITTORRENT_URL` | Yes | WebUI URL (overridden in docker-compose.yml) |
| `QBITTORRENT_USERNAME` | Yes | WebUI username |
| `QBITTORRENT_PASSWORD` | Yes | WebUI password |
| `POLL_INTERVAL` | No | Completion poll interval (default: `30s`) |

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
