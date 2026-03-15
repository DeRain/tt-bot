---
title: "Torrent Control — Plan"
feature_id: "torrent-control"
status: draft
depends_on_design: "docs/features/torrent-control/design.md"
last_updated: 2026-03-15
---

# Torrent Control — Plan

## Overview

Implementation follows a bottom-up approach: qbt client layer first (pause/resume API), then formatter layer (detail view, keyboards, helpers), then bot handler layer (callback routing, integration). Tests are written alongside each task (TDD). Final verification via `make gate-all`.

## Preconditions

- Existing `list-torrents` feature is implemented and passing
- `qbt.Client` interface exists with `ListTorrents` method
- Callback routing infrastructure exists in `bot/callback.go`
- `formatter` package has `FormatTorrentList`, `PaginationKeyboard`

## Task Sequence

### Phase 1: qbt Client — Pause/Resume API

- **TASK-1**: Add `PauseTorrents` and `ResumeTorrents` to `qbt.Client` interface
  - Derived from: DES-1
  - Implements: REQ-10
  - Impacts: `internal/qbt/client.go`
  - Verification: TEST-1
  - Gate: 4

- **TASK-2**: Implement `PauseTorrents` and `ResumeTorrents` in `HTTPClient`
  - Derived from: DES-1
  - Implements: REQ-5, REQ-6, REQ-10
  - Impacts: `internal/qbt/http.go`
  - Verification: TEST-2
  - Gate: 4
  - Depends on: TASK-1

- **TASK-3**: Update `mockQBTClient` in bot test files
  - Derived from: DES-1
  - Implements: REQ-10
  - Impacts: `internal/bot/handler_test.go`
  - Verification: TEST-3
  - Gate: 4
  - Depends on: TASK-1

### Phase 2: Formatter — Detail View and Selection Keyboard

- **TASK-4**: Add `FormatSize` helper
  - Derived from: DES-3
  - Implements: REQ-2
  - Impacts: `internal/formatter/format.go`
  - Verification: TEST-4
  - Gate: 4

- **TASK-5**: Add `IsPaused` helper
  - Derived from: DES-5
  - Implements: REQ-3, REQ-4
  - Impacts: `internal/formatter/format.go`
  - Verification: TEST-5
  - Gate: 4

- **TASK-6**: Add `FormatTorrentDetail` function
  - Derived from: DES-2
  - Implements: REQ-2, REQ-9
  - Impacts: `internal/formatter/format.go`
  - Verification: TEST-6
  - Gate: 4
  - Depends on: TASK-4

- **TASK-7**: Add `TorrentDetailKeyboard` function
  - Derived from: DES-4
  - Implements: REQ-3, REQ-4, REQ-7, REQ-8
  - Impacts: `internal/formatter/format.go`
  - Verification: TEST-7
  - Gate: 4
  - Depends on: TASK-5

- **TASK-8**: Add `TorrentSelectionKeyboard` function
  - Derived from: DES-6
  - Implements: REQ-1, REQ-8
  - Impacts: `internal/formatter/format.go`
  - Verification: TEST-8
  - Gate: 4

### Phase 3: Bot Handler — Callback Routing and Handlers

- **TASK-9**: Add filter character mapping helpers
  - Derived from: DES-11
  - Implements: REQ-8
  - Impacts: `internal/bot/callback.go`
  - Verification: TEST-9
  - Gate: 4

- **TASK-10**: Add `handleSelectCallback` handler
  - Derived from: DES-8
  - Implements: REQ-2, REQ-3, REQ-4
  - Impacts: `internal/bot/callback.go`
  - Verification: TEST-10
  - Gate: 4
  - Depends on: TASK-6, TASK-7, TASK-9

- **TASK-11**: Add `handlePauseCallback` and `handleResumeCallback` handlers
  - Derived from: DES-9
  - Implements: REQ-5, REQ-6
  - Impacts: `internal/bot/callback.go`
  - Verification: TEST-11
  - Gate: 4
  - Depends on: TASK-2, TASK-3, TASK-10

- **TASK-12**: Add `handleBackCallback` handler
  - Derived from: DES-10
  - Implements: REQ-7
  - Impacts: `internal/bot/callback.go`
  - Verification: TEST-12
  - Gate: 4
  - Depends on: TASK-8, TASK-9

- **TASK-13**: Update callback dispatcher and integrate selection keyboard into list views
  - Derived from: DES-7, DES-8, DES-9, DES-10
  - Implements: REQ-1
  - Impacts: `internal/bot/callback.go`, `internal/bot/handler.go`
  - Verification: TEST-13
  - Gate: 4
  - Depends on: TASK-8, TASK-9, TASK-10, TASK-11, TASK-12

### Phase 4: Test Verification

- **TASK-14**: Run full gate verification
  - Derived from: DES-1 through DES-11
  - Implements: REQ-1 through REQ-10
  - Impacts: all test files
  - Verification: `make gate-all` passes, 80%+ coverage
  - Gate: 4, 5
  - Depends on: TASK-1 through TASK-13

## Dependencies

```
TASK-1 ─┬─ TASK-2
        └─ TASK-3

TASK-4 ── TASK-6 ──┐
TASK-5 ── TASK-7 ──┤
                   ├─ TASK-10 ──┐
TASK-8 ────────────┤            ├─ TASK-13 ── TASK-14
TASK-9 ────────────┤            │
                   ├─ TASK-12 ──┤
TASK-2, TASK-3 ────┴─ TASK-11 ──┘
```

Recommended execution order:
`TASK-1 → TASK-3 → TASK-2 → TASK-4 → TASK-5 → TASK-8 → TASK-9 → TASK-6 → TASK-7 → TASK-10 → TASK-11 → TASK-12 → TASK-13 → TASK-14`

## Affected Files

| File | Action |
|------|--------|
| `internal/qbt/client.go` | Modify — add 2 methods to interface |
| `internal/qbt/http.go` | Modify — add 2 method implementations |
| `internal/qbt/http_test.go` | Modify — add pause/resume tests |
| `internal/formatter/format.go` | Modify — add 5 new functions |
| `internal/formatter/format_test.go` | Modify — add tests for new functions |
| `internal/bot/callback.go` | Modify — add 4 handlers, filter helpers, dispatcher cases |
| `internal/bot/callback_test.go` | Modify — add tests for new handlers |
| `internal/bot/handler.go` | Modify — extract shared helper, add selection keyboard |
| `internal/bot/handler_test.go` | Modify — update mock, add selection keyboard tests |

## Rollout Notes

- No database migrations required (stateless feature).
- No environment variable changes.
- Docker image rebuild required for deployment.
- Build for `linux/amd64` and push to both registries (local + Docker Hub) per release process.

## Quality Gates

### Gate 3: Plan Gate

This plan passes when:
- [x] Every TASK-* maps to at least one DES-* and REQ-*
- [x] Task sequencing is coherent (dependencies respected)
- [x] Every TASK-* has a verification target (TEST-* or CHECK-*)
- [x] No TASK-* exists without implementation evidence path
- [x] Affected files are listed

**Harness check command:**
```bash
# Verify plan-to-design coverage
design_items=$(grep -oP 'DES-\d+' docs/features/torrent-control/design.md | sort -u)
plan_items=$(grep -oP 'DES-\d+' docs/features/torrent-control/plan.md | sort -u)
comm -23 <(echo "$design_items") <(echo "$plan_items")  # should be empty

# Verify every task has verification
grep "^- \*\*TASK-" docs/features/torrent-control/plan.md | wc -l  # task count
grep "Verification:" docs/features/torrent-control/plan.md | wc -l  # should match
```

### Iterative Harness Loop Protocol

When executing this plan via an agent harness loop:
1. Execute tasks in dependency order
2. After each TASK-*, run its verification target
3. If verification fails: fix, re-verify, max 3 retries
4. After all tasks: run `make gate-all` as the implementation gate
5. Update traceability.md with implementation evidence
6. Run verification.md checks

## Verification Steps

See `verification.md` for full verification matrix. Post-implementation:
1. `make gate-all` passes
2. All TEST-* pass with 80%+ coverage
3. All AC-* validated in verification.md

## Blockers

None.
