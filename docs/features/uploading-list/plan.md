---
title: "Uploading Torrents List — Plan"
feature_id: "uploading-list"
status: draft
depends_on_design: "docs/features/uploading-list/design.md"
last_updated: 2026-03-15
---

# Uploading Torrents List — Plan

## Overview

Implement in 5 tasks: filter constant, filter char/prefix mappings, pagination callback routing, command dispatch and client-side filtering logic, and tests. Tasks are ordered by dependency.

## Preconditions

- Existing list-torrents, downloading-list, and torrent-control features are implemented and passing
- `make gate-all` passes on current main

## Task Sequence

- **TASK-1**: Add `FilterUploading` constant to `internal/qbt/types.go`
  - Derived from: DES-1
  - Implements: REQ-1, REQ-2
  - Impacts: `internal/qbt/types.go`
  - Verification: TEST-1
  - Gate: 4

- **TASK-2**: Add filter char mappings (`u` ↔ `FilterUploading` ↔ `up`) in `internal/bot/callback.go`
  - Derived from: DES-4
  - Implements: REQ-3, REQ-4
  - Impacts: `internal/bot/callback.go`
  - Verification: TEST-2
  - Gate: 4

- **TASK-3**: Add `pg:up:` pagination case in `handleCallback()` in `internal/bot/callback.go`
  - Derived from: DES-4
  - Implements: REQ-3
  - Impacts: `internal/bot/callback.go`
  - Verification: TEST-3
  - Gate: 4

- **TASK-4**: Add client-side post-filtering in `renderTorrentListPage()` and `/uploading` command dispatch in `internal/bot/handler.go`; register command in `internal/bot/commands.go`
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

## Verification Targets

| ID | Description | Type |
|----|-------------|------|
| TEST-1 | `FilterUploading` constant value is distinct from existing filter constants | unit |
| TEST-2 | `filterCharToFilter("u")` returns `FilterUploading`; `filterToChar(FilterUploading)` returns `"u"`; `filterCharToPrefix("u")` returns `"up"` | unit |
| TEST-3 | `handleCallback()` routes `pg:up:<page>` to `sendTorrentPage` with `FilterUploading` | unit |
| TEST-4 | `renderTorrentListPage()` with `FilterUploading` returns only torrents with `Progress == 1.0`; returns empty/no-message when none present | unit |
| TEST-5 | `/uploading` command triggers `sendTorrentPage` with `FilterUploading`; command appears in `BotCommands` | unit |
| CHECK-1 | `make test-integration` passes — uploading filter verified against real qBittorrent instance | integration |

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
- `internal/bot/callback_test.go` — unit tests for char/prefix mappings and routing
- `internal/bot/handler_test.go` — unit tests for filtering and command dispatch

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
