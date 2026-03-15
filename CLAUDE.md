# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

tt-bot is a stateless Go Telegram bot for managing qBittorrent downloads. Whitelisted Telegram users can add torrents (magnet links and .torrent files), pick categories via inline keyboards, list torrents with pagination, and receive completion notifications.

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

### Quality Gates

See `docs/gates.md` for full gate definitions and harness commands.

| Gate | When | Key Check |
|------|------|-----------|
| 1. Spec | Before design | REQs and ACs are clear and testable |
| 2. Design | Before planning | Every REQ mapped to DES |
| 3. Plan | Before implementation | Every TASK maps to DES + REQ + verification |
| 4. Implementation | After coding | `make gate-all` passes, traceability updated |
| 5. Verification | Before completion | Every AC validated, no gaps |

### Harness Iterative Loop Support

Feature plans are designed for agent harness execution:
- Tasks are ordered with explicit dependencies
- Each task has a verification target (TEST-* or CHECK-*)
- Gate check commands are embedded in templates for automated validation
- Recovery protocol: fix → re-verify → max 3 retries → escalate
- See `docs/gates.md` "Iterative Harness Loop Protocol" for the full agent execution model

### ECC Compliance

All Everything Claude Code (ECC) components operating in this repo MUST follow the docs-first workflow.

#### ECC Commands → Gate Mapping

| Command | Gate | Traceability Rule |
|---------|------|-------------------|
| `/plan` | 1-3 | MUST check for existing `docs/features/<feature-id>/`. New plans follow `plan.md` template with TASK-* → DES-* → REQ-*. |
| `/go-test` | 4-5 | MUST reference TEST-* from `verification.md`. New tests MUST map to AC-*. Write tests first. |
| `/tdd` | 4-5 | Scaffold interfaces, tests FIRST, minimal impl. MUST reference TEST-* and AC-*. |
| `/go-build` | 4 recovery | Fix build/vet/lint errors. Uses `go-build-resolver` agent. Follow `docs/gates.md` recovery protocol. |
| `/go-review` | 4 | MUST verify code changes trace to TASK-* and REQ-*. Flag untraced changes. Uses `go-reviewer` agent. |
| `/code-review` | 4 | General quality review. MUST check traceability of changes. |
| `/verify` | 5 | Run verification loop. MUST check all AC-* have TEST-*/CHECK-* results. |
| `/eval` | 5 | Evaluate implementation against acceptance criteria from spec.md. |
| `/test-coverage` | 5 | Verify 80%+ coverage threshold. Report per-package coverage. |
| `/checkpoint` | any | Save verification state mid-harness loop for recovery. |
| `/learn-eval` | post | Extract reusable patterns after feature completion. |
| `/update-docs` | post | Sync documentation with code changes. Update traceability docs. |
| `/update-codemaps` | post | Update codebase maps after structural changes. |
| `/orchestrate` | 1-5 | Multi-agent coordination across gates. |
| `/multi-plan` | 1-3 | Decompose feature into parallel tasks across agents. |
| `/multi-execute` | 4 | Execute parallel TASK-* items via multiple agents. |

#### ECC Agents → Gate Mapping

| Agent | Gate | When to Use |
|-------|------|-------------|
| `planner` | 1-3 | Creating spec/design/plan for new features |
| `architect` | 2 | System design decisions, architecture for design.md |
| `tdd-guide` | 4-5 | Enforcing write-tests-first during implementation |
| `code-reviewer` | 4 | Post-implementation quality + traceability review |
| `security-reviewer` | 4 | Security-relevant requirement verification |
| `go-reviewer` | 4 | Go-specific idiomatic patterns, concurrency, error handling |
| `go-build-resolver` | 4 recovery | Fix Go build/vet/lint with minimal changes |
| `build-error-resolver` | 4 recovery | Fix general build errors |
| `refactor-cleaner` | maintenance | Dead code cleanup between features |
| `doc-updater` | post | Sync traceability docs after changes |

#### ECC Skills (Reference)

| Skill | Purpose in Workflow |
|-------|---------------------|
| `golang-patterns` | Idiomatic Go reference when implementing TASK-* items |
| `golang-testing` | Table-driven tests, subtests, benchmarks when writing TEST-* items |
| `tdd-workflow` | TDD methodology: RED → GREEN → REFACTOR cycle |
| `security-review` | Security checklist for Gate 4 |
| `verification-loop` | Continuous verification system for Gate 5 |
| `eval-harness` | Evaluation framework for harness loop quality |
| `search-first` | Research existing solutions before implementing TASK-* |
| `docker-patterns` | Docker/Compose patterns (used by this project) |
| `deployment-patterns` | CI/CD, rollout, health checks |
| `autonomous-loops` | Harness loop architecture patterns |
| `continuous-learning-v2` | Extract instincts from sessions for future reuse |
| `strategic-compact` | Context window management in long harness sessions |

#### ECC Rules

| Rule Set | Applied |
|----------|---------|
| `common/coding-style` | Yes — immutability, file organization |
| `common/git-workflow` | Yes — commit format, PR process |
| `common/testing` | Yes — TDD, 80% coverage |
| `common/performance` | Yes — model routing, context management |
| `common/patterns` | Yes — design patterns, skeleton projects |
| `common/hooks` | Yes — hook architecture, TodoWrite |
| `common/agents` | Yes — agent delegation rules |
| `common/security` | Yes — mandatory security checks |
| `golang/` | Yes — Go-specific rules |

#### Gate Execution Flow

```
/plan + planner + architect      → Gates 1-3 (spec, design, plan)
/go-test + /tdd + tdd-guide      → Gate 4 (write tests for AC-*)
/multi-execute + go-build-resolver → Gate 4 (implement TASK-*, fix build)
/go-review + code-reviewer        → Gate 4 (review traceability)
/verify + /eval + /test-coverage  → Gate 5 (validate all AC-*)
/checkpoint                       → Save state between gates
/learn-eval + /update-docs        → Post-gate (extract patterns, sync docs)
```

### Existing Features

| Feature ID | Description | Docs |
|------------|-------------|------|
| auth | User authorization via Telegram ID whitelist | `docs/features/auth/` |
| add-torrent | Add torrents via magnet links and .torrent files | `docs/features/add-torrent/` |
| list-torrents | List all/active torrents with pagination | `docs/features/list-torrents/` |
| completion-notifications | Background polling and completion alerts | `docs/features/completion-notifications/` |
| config | Environment variable loading and validation | `docs/features/config/` |

### PR and Commit Conventions

See `docs/pr-checklist.md` for the full PR template and commit message guidance.

- PRs MUST include: Feature ID, REQ-* implemented, TASK-* completed, AC-* covered, verification evidence.
- Commits SHOULD include feature ID and task ID: `feat(auth): TASK-2 implement whitelist authorizer`
- Untraced changes MUST be justified in the PR description.

## Build & Test Commands

| Command | Description |
|---------|-------------|
| `make build` | `go build ./...` |
| `make lint` | `golangci-lint run` |
| `make test` | Unit tests with coverage (`go test ./... -short -cover`) |
| `make test-integration` | Integration + E2E tests in Docker (spins up qBittorrent, runs all `Integration\|E2E` tests, tears down) |
| `make gate-all` | Full quality gate: build → lint → unit tests |
| `make clean` | Remove coverage.out and bot binary |

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
