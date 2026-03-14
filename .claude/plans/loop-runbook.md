# Loop Runbook: tt-bot Sequential Implementation

## Pattern: Sequential with auto-commit
## Model: Sonnet (all phases), Opus fallback on blockers
## Mode: Safe (full quality gates)

## Stop Condition
All 8 phases complete AND `make gate-all` passes AND integration tests pass.

## Phase Sequence

| # | Phase | Gate | Commit message prefix |
|---|-------|------|-----------------------|
| 1 | Config package | `go build ./...` | `feat: add config package` |
| 2 | qBittorrent client | `make gate-all` | `feat: add qbt client` |
| 3 | Docker test infra | Integration tests pass | `chore: add docker test infrastructure` |
| 4 | Message formatter | `make gate-all` | `feat: add message formatter` |
| 5 | Bot handlers | `make gate-all` | `feat: add bot handlers` |
| 6 | Completion poller | `make gate-all` | `feat: add completion poller` |
| 7 | Entry point + Docker | `go build ./...` + compose validates | `feat: add entrypoint and docker` |
| 8 | E2E tests + CI | `make gate-all` + integration | `feat: add e2e tests` |

## Gate Protocol (per phase)

1. Sonnet agent implements phase in worktree or main
2. Run `go build ./...` — must pass
3. Run `golangci-lint run` — must pass
4. Run `go test ./... -short -cover` — must pass, target 80%+
5. If phase 3+: run integration tests against Docker qBittorrent
6. On pass: auto-commit with conventional message
7. On fail: fix in same phase, re-run gates (max 3 retries)
8. After 3 failures: escalate to Opus or pause for user

## Recovery Protocol

- Build failure → invoke go-build-resolver agent
- Test failure → read error, fix implementation (not tests unless tests are wrong)
- Lint failure → auto-fix with `golangci-lint run --fix`, re-run
- Docker failure → check compose logs, fix config

## Current State

- [x] Phase 0: Scaffolding (go.mod, .gitignore, .env.example, Makefile, .golangci.yml)
- [x] Phase 1: Config package (91.3% coverage)
- [x] Phase 2: qBittorrent client (72.1% coverage)
- [x] Phase 3: Docker test infrastructure (4 integration tests pass)
- [x] Phase 4: Message formatter (96.4% coverage)
- [x] Phase 5: Bot handlers (80.0% coverage)
- [x] Phase 6: Completion poller (87.0% coverage)
- [x] Phase 7: Entry point + production Docker
- [x] Phase 8: E2E tests (5 E2E + 4 integration = 9 tests against real qBittorrent)
