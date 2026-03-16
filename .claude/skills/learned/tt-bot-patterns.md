---
name: tt-bot-patterns
description: "Coding patterns extracted from tt-bot — Go Telegram bot for qBittorrent management"
user-invocable: false
origin: local-git-analysis
version: 1.0.0
analyzed_commits: 49
---

# tt-bot Patterns

**Extracted:** 2026-03-16
**Repository:** DeRain/tt-bot (Go Telegram bot for qBittorrent)

## Commit Conventions

Strict **conventional commits** with optional feature scope:

| Type | Usage | Frequency |
|------|-------|-----------|
| `feat` | New features | 35% |
| `chore` | Maintenance, tooling, config | 20% |
| `fix` | Bug fixes | 16% |
| `docs` | Documentation | 12% |
| `test` | Test-only changes | 4% |

Feature commits use scoped format: `feat(<feature-id>): <description>`

Examples:
```
feat(torrent-files): view and manage torrent file download priorities
fix: exclude paused/stopped uploads from /uploading list
test: add coverage for stoppedUP/stoppedDL labels
chore: migrate golangci-lint v2, fix all lint issues
```

## Code Architecture

```
cmd/bot/main.go              # Entrypoint: wires config, client, handler, poller
internal/
  config/config.go           # Env-var loading and validation
  qbt/
    client.go                # Interface: qbt.Client
    http.go                  # HTTP implementation of qbt.Client
    types.go                 # Shared types (TorrentInfo, TorrentFile, etc.)
    http_test.go             # Unit tests (httptest mocks)
    http_integration_test.go # Integration tests (real qBittorrent)
  bot/
    handler.go               # Telegram update dispatcher, main handler
    callback.go              # Inline keyboard callback routing
    auth.go                  # Whitelist-based user authorization
    sender.go                # Sender interface for testability
    commands.go              # Bot command registration
    *_test.go                # Unit tests
    e2e_test.go              # End-to-end flow tests
  formatter/
    format.go                # Message formatting, pagination keyboards
    format_test.go           # Format tests
  poller/
    poller.go                # Background completion polling
    poller_test.go           # Poller tests
```

## Key Co-Change Patterns

Files that always change together (implement feature → update tests):

| Primary Change | Always Co-Changes With |
|---------------|----------------------|
| `bot/callback.go` | `bot/handler.go`, `bot/callback_test.go`, `bot/handler_test.go` |
| `formatter/format.go` | `formatter/format_test.go` |
| `qbt/http.go` | `qbt/http_test.go`, `qbt/client.go` |
| `qbt/client.go` (interface) | `qbt/http.go` (impl), `qbt/http_test.go` |

**Rule:** Never change an implementation file without updating its test file in the same commit.

## Feature Development Workflow

Every feature follows the docs-first pipeline:

1. **Docs first**: `docs/features/<feature-id>/` with spec, design, plan, traceability, verification
2. **Interface first**: Add methods to `qbt.Client` interface
3. **Implementation**: HTTP client methods in `qbt/http.go`
4. **Bot wiring**: Callback routing in `callback.go`, handler dispatch in `handler.go`
5. **Formatting**: Display logic in `formatter/format.go`
6. **Tests**: Unit → Integration → E2E in that order
7. **Verification**: Update traceability.md and verification.md

## Testing Patterns

### Test Organization
- `*_test.go` (no build tag): unit tests, run with `-short`
- `//go:build integration`: integration tests requiring Docker
- `e2e_test.go`: end-to-end bot flow tests

### Test Naming
- Unit: `TestFunctionName` or `TestFunctionName_scenario`
- Integration: same naming, gated by build tag
- E2E: `TestE2E_FlowDescription`

### Mock Strategy
- **Mock**: `qbt.Client` interface, `bot.Sender` interface
- **Never mock**: qBittorrent HTTP responses for integration tests
- **httptest**: Used for unit-level qbt client tests only

## Interface-Driven Design

All external dependencies are behind interfaces:
- `qbt.Client` — qBittorrent operations
- `bot.Sender` — Telegram message sending
- `poller.Notifier` — completion notification delivery

New features extend these interfaces first, then implement.

## Callback Data Encoding

Telegram limits callback_data to 64 bytes. This project uses short prefixes:
- `cat:<name>`, `pg:all:<N>`, `pg:act:<N>`, `sel:<hash>`, `rm:<hash>`, `noop`

Always truncate user-provided strings at UTF-8 boundaries.

## Quality Gates

Every commit passes through: `make gate-all` (build + lint + unit) then `make test-integration` (Docker-based integration + E2E tests). Integration tests are mandatory and never deferred.
