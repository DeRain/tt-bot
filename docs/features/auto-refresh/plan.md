---
title: "Auto-Refresh Views — Plan"
feature_id: "auto-refresh"
status: draft
depends_on_design: "docs/features/auto-refresh/design.md"
last_updated: 2026-05-12
---

# Auto-Refresh Views — Plan

## Overview

Implementation follows a bottom-up approach: config layer first (VIEW_REFRESH_INTERVAL), then handler layer (LiveView struct, liveViews map, refresh goroutine), then callback wiring (registration/deregistration at lifecycle points), then main.go integration. Tests are written alongside each task. Final verification via `make gate-all` and `make test-integration`.

## Preconditions

- Existing `list-torrents`, `downloading-list`, `uploading-list`, and `torrent-control` features are implemented and passing
- `qbt.Client` interface exists with `ListTorrents` method
- `formatter` package has `FormatTorrentList`, `FormatTorrentDetail`, and keyboard builders
- Callback routing infrastructure exists in `bot/callback.go`
- `config` package has pattern for env-var loading with defaults
- `bot.Sender` interface exists for testability

## Task Sequence

### Phase 1: Config

- **TASK-1**: Add `VIEW_REFRESH_INTERVAL` config
  - Derived from: DES-4
  - Implements: REQ-5
  - Impacts: `internal/config/config.go`, `internal/config/config_test.go`
  - Verification: TEST-1 (unit tests for default, valid, and invalid values)
  - Gate: 4

### Phase 2: Handler — LiveView and Refresh

- **TASK-2**: Add `LiveView` type, `liveViews` map, and mutex to `Handler`
  - Derived from: DES-1, DES-2
  - Implements: REQ-1, REQ-2, REQ-4
  - Impacts: `internal/bot/handler.go`
  - Verification: TEST-2 (unit tests for register/deregister, single-view enforcement)
  - Gate: 4
  - Depends on: TASK-1

- **TASK-3**: Implement auto-refresh goroutine and helpers
  - Derived from: DES-3, DES-4
  - Implements: REQ-1, REQ-2, REQ-3, REQ-6, REQ-7
  - Impacts: `internal/bot/handler.go`
  - Verification: TEST-3 (unit tests for hash comparison, refresh logic, error handling)
  - Gate: 4
  - Depends on: TASK-2

### Phase 3: Callback Wiring

- **TASK-4**: Wire registration and deregistration into callback handlers
  - Derived from: DES-5
  - Implements: REQ-4, REQ-6
  - Impacts: `internal/bot/callback.go`, `internal/bot/handler.go`
  - Verification: TEST-4 (unit tests verify views are registered/deregistered at correct points)
  - Gate: 4
  - Depends on: TASK-2, TASK-3

### Phase 4: Main Integration

- **TASK-5**: Wire `viewRefreshInterval` into `main.go`
  - Derived from: DES-4
  - Implements: REQ-5
  - Impacts: `cmd/bot/main.go`, `internal/bot/handler.go` (constructor signature)
  - Verification: TEST-5 (integration test verifies goroutine starts with config)
  - Gate: 4
  - Depends on: TASK-1, TASK-3

### Phase 5: Test Coverage

- **TASK-6**: Add comprehensive tests for all changes
  - Derived from: DES-1 through DES-6
  - Implements: REQ-1 through REQ-7
  - Impacts: `internal/config/config_test.go`, `internal/bot/handler_test.go`, `internal/bot/callback_test.go`, `internal/bot/e2e_test.go`
  - Verification: TEST-6 (coverage at or above 80% on changed files)
  - Gate: 4
  - Depends on: TASK-1 through TASK-5

### Phase 6: Quality Gates

- **TASK-7**: Run full quality gate verification
  - Derived from: DES-1 through DES-6
  - Implements: REQ-1 through REQ-7
  - Impacts: all test files
  - Verification: `make gate-all` passes, `make test-integration` passes
  - Gate: 4, 5
  - Depends on: TASK-1 through TASK-6

## Dependencies

```
TASK-1 ──┬─ TASK-2 ── TASK-3 ──┬─ TASK-4 ──┬─ TASK-6 ── TASK-7
         └─ TASK-5 ─────────────┘           │
                                            └─ (TASK-5 also feeds TASK-6)
```

Recommended execution order:
`TASK-1 → TASK-2 → TASK-3 → TASK-4 + TASK-5 (parallel) → TASK-6 → TASK-7`

TASK-4 and TASK-5 can be done in parallel since they touch different files with no dependency between them.

## Affected Files

| File | Action | Reason |
|------|--------|--------|
| `internal/config/config.go` | Modify | Add `ViewRefreshInterval()` accessor |
| `internal/config/config_test.go` | Modify | Add tests for new config function |
| `internal/bot/handler.go` | Modify | Add LiveView type, liveViews map, register/deregister helpers, runAutoRefresh goroutine, refreshViews, refreshLiveView, contentHash, updated constructor |
| `internal/bot/handler_test.go` | Modify | Add unit tests for view registration, refresh logic, hash comparison, error handling |
| `internal/bot/callback.go` | Modify | Add register/deregister calls at lifecycle points (sendTorrentPage, handleSelect, handleBack, handlePause, handleResume, command handlers) |
| `internal/bot/callback_test.go` | Modify | Add tests verifying views are registered/deregistered at correct points |
| `cmd/bot/main.go` | Modify | Read `viewRefreshInterval` from config, pass to `bot.New()`, start goroutine |
| `internal/bot/e2e_test.go` | Modify | Add E2E test for auto-refresh flow (if feasible in test environment) |

## Rollout Notes

- Default 5-second interval is safe for single-user deployments. Multi-user operators should increase `VIEW_REFRESH_INTERVAL` to reduce qBittorrent load.
- The feature is self-contained: if `VIEW_REFRESH_INTERVAL` is not set, auto-refresh still runs at 5s default. No opt-in required.
- After bot restart, all views are deregistered. Users must re-send commands to re-register. This is documented and acceptable per DES-6.
- The `liveViews` map has no size cap but is bounded by the number of distinct chats actively using the bot. For typical usage (single-digit concurrent users), memory usage is negligible.

## Blockers

None.

## Quality Gates

### Gate 3: Plan Gate

This plan passes when:
- [ ] Every TASK-* maps to at least one DES-* and REQ-*
- [ ] Task sequencing is coherent (dependencies respected)
- [ ] Every TASK-* has a verification target (TEST-*)
- [ ] No TASK-* exists without implementation evidence path
- [ ] Affected files are listed

**Harness check command:**
```bash
# Verify plan-to-design coverage
design_items=$(grep -oP 'DES-\d+' docs/features/auto-refresh/design.md | sort -u)
plan_items=$(grep -oP 'DES-\d+' docs/features/auto-refresh/plan.md | sort -u)
comm -23 <(echo "$design_items") <(echo "$plan_items")  # should be empty

# Verify every task has verification
grep "^- \*\*TASK-" docs/features/auto-refresh/plan.md | wc -l  # task count
grep "Verification:" docs/features/auto-refresh/plan.md | wc -l  # should match
```

### Iterative Harness Loop Protocol

When executing this plan via an agent harness loop:
1. Execute tasks in dependency order
2. After each TASK-*, run its verification target
3. If verification fails: fix, re-verify, max 3 retries
4. After all tasks: run `make gate-all` (includes `make arch-check`) as the implementation gate
5. Update traceability.md with implementation evidence
6. Run verification.md checks

## Task-to-Requirement Mapping

| Task | Derived From | Implements | Impacts |
|------|-------------|------------|---------|
| TASK-1 | DES-4 | REQ-5 | config.go, config_test.go |
| TASK-2 | DES-1, DES-2 | REQ-1, REQ-2, REQ-4 | handler.go |
| TASK-3 | DES-3, DES-4 | REQ-1, REQ-2, REQ-3, REQ-6, REQ-7 | handler.go |
| TASK-4 | DES-5 | REQ-4, REQ-6 | callback.go, handler.go |
| TASK-5 | DES-4 | REQ-5 | main.go, handler.go |
| TASK-6 | DES-1 through DES-6 | REQ-1 through REQ-7 | all test files |
| TASK-7 | DES-1 through DES-6 | REQ-1 through REQ-7 | all files |
