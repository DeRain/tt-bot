---
title: "Stop and Remove Torrent Actions — Plan"
feature_id: torrent-remove
status: draft
depends_on_design: "docs/features/torrent-remove/design.md"
last_updated: 2026-03-15
---

# Stop and Remove Torrent Actions — Plan

## Overview

Implementation follows the same bottom-up layering used by `torrent-control`: qbt client first, formatter second, bot callback handlers third. Each layer is independently testable, so unit tests can be written alongside each task before wiring them together. Integration verification uses `make test-integration` with a real qBittorrent instance to confirm actual file-deletion behavior.

## Preconditions

- `torrent-control` feature is implemented and all its tests pass
- `qbt.Client` interface exists with `PauseTorrents`, `ResumeTorrents`, `ListTorrents`
- `formatter.TorrentDetailKeyboard` exists with its current two-row layout
- Callback dispatcher in `internal/bot/callback.go` routes `pa:`, `re:`, `bk:`, `sel:` prefixes
- `make gate-all` passes on the current main branch

## Task Sequence

### Phase 1: qbt Client — DeleteTorrents API

- **TASK-1**: Add `DeleteTorrents(ctx context.Context, hashes []string, deleteFiles bool) error` to the `qbt.Client` interface
  - Derived from: DES-1
  - Implements: REQ-3, REQ-4
  - Impacts: `internal/qbt/client.go`
  - Verification: TEST-1 (interface compile check; mock update compiles)
  - Gate: 4

- **TASK-2**: Implement `DeleteTorrents` on `HTTPClient`; POST `hashes=<pipe-sep>&deleteFiles=<bool>` to `/api/v2/torrents/delete` via `doWithAuth`
  - Derived from: DES-2
  - Implements: REQ-3, REQ-4
  - Impacts: `internal/qbt/http.go`
  - Verification: TEST-2 (unit test with `httptest.NewServer` verifying form fields and response handling), CHECK-1 (integration: torrent is absent from list after delete; files present/absent per `deleteFiles` flag)
  - Gate: 4
  - Depends on: TASK-1

- **TASK-3**: Update `mockQBTClient` in bot test files to implement `DeleteTorrents`
  - Derived from: DES-1
  - Implements: REQ-3, REQ-4
  - Impacts: `internal/bot/handler_test.go` (or shared mock file if extracted)
  - Verification: TEST-3 (`make build` passes with no "does not implement" errors)
  - Gate: 4
  - Depends on: TASK-1

### Phase 2: Formatter — Confirmation View and Updated Detail Keyboard

- **TASK-4**: Extend `TorrentDetailKeyboard` to add a Remove row (`🗑 Remove` with `rm:` callback) between the pause/resume row and the back row; update existing unit tests that assert row count
  - Derived from: DES-3
  - Implements: REQ-1
  - Impacts: `internal/formatter/format.go`, `internal/formatter/format_test.go`
  - Verification: TEST-4 (table-driven tests assert: Remove button present in all states, callback data fits 64 bytes at worst case, row count is now 3)
  - Gate: 4

- **TASK-5**: Add `FormatRemoveConfirmation(torrentName string) string` to formatter
  - Derived from: DES-4
  - Implements: REQ-2
  - Impacts: `internal/formatter/format.go`
  - Verification: TEST-5 (unit test asserts output contains torrent name and confirmation prompt text)
  - Gate: 4

- **TASK-6**: Add `RemoveConfirmKeyboard(hash, filterChar string, page int) Keyboard` to formatter; three rows: "Remove torrent only" (`rd:`), "Remove with files" (`rf:`), "Cancel" (`rc:`)
  - Derived from: DES-4
  - Implements: REQ-2, REQ-6
  - Impacts: `internal/formatter/format.go`
  - Verification: TEST-6 (table-driven tests assert: three rows, callback prefixes correct, all callbacks fit 64 bytes at page=99 with 40-char hash)
  - Gate: 4

### Phase 3: Bot Callback Handlers

- **TASK-7**: Add `handleRemoveConfirmCallback` in `internal/bot/callback.go`; parse `rm:<f>:<page>:<hash>`, fetch torrent by hash, render confirmation view, edit message; no mutation
  - Derived from: DES-5
  - Implements: REQ-2
  - Impacts: `internal/bot/callback.go`
  - Verification: TEST-7 (unit test: message edited with confirmation view; no qbt mutating calls made; torrent-not-found path answers callback and navigates to list)
  - Gate: 4
  - Depends on: TASK-5, TASK-6

- **TASK-8**: Add `handleRemoveDeleteCallback` in `internal/bot/callback.go`; handles `rd:` (`deleteFiles=false`) and `rf:` (`deleteFiles=true`); calls `DeleteTorrents`, then navigates to list view
  - Derived from: DES-6
  - Implements: REQ-3, REQ-4, REQ-5
  - Impacts: `internal/bot/callback.go`
  - Verification: TEST-8 (unit tests: `DeleteTorrents` called with correct `deleteFiles` bool; message edited to list view on success; error path answers callback with error text; empty list after deletion shows empty-list message)
  - Gate: 4
  - Depends on: TASK-2, TASK-3, TASK-7

- **TASK-9**: Add `handleRemoveCancelCallback` in `internal/bot/callback.go`; parse `rc:<f>:<page>:<hash>`, re-fetch torrent, render detail view, edit message; no mutation
  - Derived from: DES-7
  - Implements: REQ-6
  - Impacts: `internal/bot/callback.go`
  - Verification: TEST-9 (unit test: message edited to detail view; no qbt mutating calls made; torrent-not-found path navigates to list)
  - Gate: 4
  - Depends on: TASK-4, TASK-7

- **TASK-10**: Register `rm:`, `rd:`, `rf:`, `rc:` in the callback dispatcher switch statement in `internal/bot/callback.go`
  - Derived from: DES-8
  - Implements: REQ-1, REQ-2, REQ-3, REQ-4, REQ-5, REQ-6
  - Impacts: `internal/bot/callback.go`
  - Verification: TEST-10 (integration: end-to-end callback routing verified; `make gate-all` passes)
  - Gate: 4
  - Depends on: TASK-7, TASK-8, TASK-9

### Phase 4: Verification

- **TASK-11**: Run full gate verification: `make gate-all` + `make test-integration`; update `traceability.md` and `verification.md`
  - Derived from: DES-1, DES-2, DES-3, DES-4, DES-5, DES-6, DES-7, DES-8
  - Implements: REQ-1, REQ-2, REQ-3, REQ-4, REQ-5, REQ-6
  - Impacts: `docs/features/torrent-remove/traceability.md`, `docs/features/torrent-remove/verification.md`
  - Verification: `make gate-all` exits 0; `make test-integration` exits 0; all AC-* have PASS results in verification.md
  - Gate: 4, 5
  - Depends on: TASK-1 through TASK-10

## Dependencies

```
TASK-1 ─┬─ TASK-2 ──────────────────────────────┐
        └─ TASK-3 ──────────────────────────────┤
                                                 │
TASK-4 ─────────────────────────────────────────┤
                                                 │
TASK-5 ─┬─ TASK-7 ──┬─ TASK-8 (needs 2,3) ──┐  │
TASK-6 ─┘           └─ TASK-9 (needs 4) ────┤  │
                                             ├──┴─ TASK-10 ── TASK-11
                                             │
                        (TASK-8 needs 2,3) ──┘
```

Recommended execution order:
`TASK-1 → TASK-3 → TASK-2 → TASK-4 → TASK-5 → TASK-6 → TASK-7 → TASK-9 → TASK-8 → TASK-10 → TASK-11`

## Affected Files

| File | Action |
|------|--------|
| `internal/qbt/client.go` | Modify — add `DeleteTorrents` to interface |
| `internal/qbt/http.go` | Modify — implement `DeleteTorrents` on `HTTPClient` |
| `internal/qbt/http_test.go` | Modify — add `DeleteTorrents` unit tests |
| `internal/formatter/format.go` | Modify — extend `TorrentDetailKeyboard`; add `FormatRemoveConfirmation`, `RemoveConfirmKeyboard` |
| `internal/formatter/format_test.go` | Modify — update row-count assertions; add tests for new functions |
| `internal/bot/callback.go` | Modify — add 3 new handlers; extend dispatcher switch |
| `internal/bot/callback_test.go` | Modify — add tests for `rm:`, `rd:`, `rf:`, `rc:` handlers |
| `internal/bot/handler_test.go` | Modify — update mock to implement `DeleteTorrents` |
| `docs/features/torrent-remove/traceability.md` | Create — bidirectional traceability matrix |
| `docs/features/torrent-remove/verification.md` | Create — AC-* verification results |

## Rollout Notes

- No database migrations required (stateless feature).
- No environment variable changes.
- Docker image rebuild required for deployment.
- Build for `linux/amd64` and push to both registries (local + Docker Hub) per release process.
- The updated `TorrentDetailKeyboard` is backward-compatible at the call-site level (signature unchanged), but displays a new row — acceptable as a UI enhancement.

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
design_items=$(grep -oP 'DES-\d+' docs/features/torrent-remove/design.md | sort -u)
plan_items=$(grep -oP 'DES-\d+' docs/features/torrent-remove/plan.md | sort -u)
comm -23 <(echo "$design_items") <(echo "$plan_items")  # should be empty

# Verify every task has verification
grep "^- \*\*TASK-" docs/features/torrent-remove/plan.md | wc -l  # task count
grep "Verification:" docs/features/torrent-remove/plan.md | wc -l  # should match
```

### Iterative Harness Loop Protocol

When executing this plan via an agent harness loop:
1. Execute tasks in dependency order
2. After each TASK-*, run its verification target
3. If verification fails: fix, re-verify, max 3 retries
4. After all tasks: run `make gate-all` as the implementation gate
5. Run `make test-integration` — mandatory before marking any AC as PASS
6. Update `traceability.md` with implementation evidence
7. Run `verification.md` checks

## Verification Steps

See `verification.md` (to be created at TASK-11) for the full AC-* verification matrix. Post-implementation summary:

1. `make gate-all` passes (build + lint + unit tests)
2. `make test-integration` passes (confirms real qBittorrent delete behavior, file presence/absence)
3. All TEST-* pass with 80%+ coverage across affected packages
4. All AC-* in `spec.md` have PASS results in `verification.md`

## Blockers

None.
