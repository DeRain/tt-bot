---
title: "Downloading Torrents List — Plan"
feature_id: "downloading-list"
status: implemented
depends_on_design: "docs/features/downloading-list/design.md"
last_updated: 2026-03-15
---

# Downloading Torrents List — Plan

## Overview

Implement in 5 tasks: filter constant, client-side filtering logic, command registration, callback routing, and tests. Tasks are ordered by dependency.

## Preconditions

- Existing list-torrents and torrent-control features are implemented and passing
- `make gate-all` passes on current main

## Task Sequence

- **TASK-1**: Add `FilterDownloading` constant to `internal/qbt/types.go`
  - Derived from: DES-1
  - Implements: REQ-1, REQ-2
  - Impacts: `internal/qbt/types.go`
  - Verification: TEST-1
  - Gate: 4

- **TASK-2**: Add filter char mappings (`d` ↔ `FilterDownloading` ↔ `dw`) in `internal/bot/callback.go`
  - Derived from: DES-4
  - Implements: REQ-3, REQ-4
  - Impacts: `internal/bot/callback.go`
  - Verification: TEST-2
  - Gate: 4

- **TASK-3**: Add `pg:dw:` pagination case in `handleCallback()` in `internal/bot/callback.go`
  - Derived from: DES-4
  - Implements: REQ-3
  - Impacts: `internal/bot/callback.go`
  - Verification: TEST-3
  - Gate: 4

- **TASK-4**: Add client-side post-filtering in `renderTorrentListPage()` and `/downloading` command dispatch in `internal/bot/handler.go`
  - Derived from: DES-2, DES-3
  - Implements: REQ-1, REQ-2, REQ-5
  - Impacts: `internal/bot/handler.go`, `internal/bot/commands.go`
  - Verification: TEST-4, TEST-5
  - Gate: 4

- **TASK-5**: Write unit tests and integration tests
  - Derived from: DES-1, DES-2, DES-3, DES-4
  - Implements: REQ-1 through REQ-5
  - Impacts: `internal/bot/*_test.go`, `internal/qbt/*_test.go`
  - Verification: TEST-1 through TEST-5, `make gate-all`, `make test-integration`
  - Gate: 4, 5

## Dependencies

```
TASK-1 → TASK-2 → TASK-3
TASK-1 → TASK-4
TASK-2, TASK-3, TASK-4 → TASK-5
```

## Affected Files

- `internal/qbt/types.go` — new filter constant
- `internal/bot/callback.go` — filter char mappings, pagination case
- `internal/bot/handler.go` — command dispatch, client-side filtering
- `internal/bot/commands.go` — command registration
- `internal/bot/callback_test.go` — unit tests for mappings
- `internal/bot/handler_test.go` — unit tests for filtering and command

## Rollout Notes

No migration needed. Command registration happens automatically on startup via `setMyCommands`.

## Quality Gates

### Gate 3: Plan Gate

This plan passes when:
- [x] Every TASK-* maps to at least one DES-* and REQ-*
- [x] Task sequencing is coherent (dependencies respected)
- [x] Every TASK-* has a verification target (TEST-* or CHECK-*)
- [x] No TASK-* exists without implementation evidence path
- [x] Affected files are listed

### Iterative Harness Loop Protocol

1. Execute TASK-1 through TASK-4
2. After each task, run `make build && make lint`
3. Execute TASK-5 (tests)
4. Run `make gate-all` and `make test-integration`
5. Update traceability and verification docs

## Blockers

None.
