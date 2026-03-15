---
title: "Torrent File Management — Plan"
feature_id: "torrent-files"
status: draft
depends_on_design: "docs/features/torrent-files/design.md"
last_updated: 2026-03-15
---

# Torrent File Management — Plan

## Overview

Implement in 8 ordered tasks: data types, client interface, HTTP implementation, formatter, keyboard builders, bot callback routing, detail keyboard extension, and tests. Tasks are sequenced so that each layer (data → client → formatter → bot) is complete before the layer above is implemented.

## Preconditions

- `make gate-all` passes on current `main`.
- The torrent detail view (handler + keyboard) is implemented (`internal/bot/handler.go`).
- Existing `qbt.Client` interface and HTTP implementation patterns are understood.

## Task Sequence

- **TASK-1**: Add `TorrentFile` struct and `FilePriority` constants to `internal/qbt/types.go`
  - Derived from: DES-1
  - Implements: REQ-1, REQ-2, REQ-6
  - Impacts: `internal/qbt/types.go`
  - Verification: TEST-1 (unit test: `FilePriority` constants have correct integer values; `TorrentFile` JSON unmarshals correctly from a sample API response)
  - Gate: 4

- **TASK-2**: Add `ListFiles` and `SetFilePriority` to the `qbt.Client` interface in `internal/qbt/client.go`
  - Derived from: DES-1
  - Implements: REQ-1, REQ-4
  - Impacts: `internal/qbt/client.go`
  - Verification: CHECK-1 (compilation passes; existing mock/stub in tests satisfies updated interface)
  - Gate: 4

- **TASK-3**: Implement `ListFiles` and `SetFilePriority` HTTP methods in `internal/qbt/http.go`
  - Derived from: DES-2
  - Implements: REQ-1, REQ-4
  - Impacts: `internal/qbt/http.go`
  - Verification: TEST-2 (unit test using `httptest.NewServer`: `ListFiles` parses a JSON response correctly; `SetFilePriority` sends correct form fields and handles 200 OK)
  - Gate: 4

- **TASK-4**: Implement `PriorityLabel` and `FormatFileList` in `internal/formatter/format.go`
  - Derived from: DES-3
  - Implements: REQ-2, REQ-3, REQ-6
  - Impacts: `internal/formatter/format.go`
  - Verification: TEST-3 (unit tests: all four priority labels correct; file name truncation at 40 chars; progress bar format; size formatting; page header present; message length ≤ 4096 chars for a 5-file page)
  - Gate: 4

- **TASK-5**: Implement `FileListKeyboard` and `PriorityKeyboard` in `internal/formatter/format.go`
  - Derived from: DES-3, DES-4
  - Implements: REQ-3, REQ-4, REQ-5
  - Impacts: `internal/formatter/format.go`
  - Verification: TEST-4 (unit tests: `FileListKeyboard` produces correct `pg:fl:` prev/next callbacks, correct `fs:` per-file callbacks, `bk:fl:` back callback, all callback data ≤ 64 bytes; `PriorityKeyboard` produces four priority buttons with checkmark on current, correct `fp:` callbacks, correct back callback)
  - Gate: 4

- **TASK-6**: Add "Files" button to the torrent detail inline keyboard in `internal/bot/handler.go`
  - Derived from: DES-5
  - Implements: REQ-5
  - Impacts: `internal/bot/handler.go`
  - Verification: TEST-5 (unit test: detail keyboard for a torrent with a valid hash contains a button whose callback data starts with `fl:`)
  - Gate: 4

- **TASK-7**: Route `fl:`, `pg:fl:`, `fs:`, `fp:`, and `bk:fl:` callbacks in `internal/bot/callback.go`
  - Derived from: DES-4, DES-6
  - Implements: REQ-1, REQ-3, REQ-4, REQ-5
  - Impacts: `internal/bot/callback.go`
  - Verification: TEST-6 (unit tests using mock `qbt.Client`: `fl:` callback triggers `ListFiles` and edits message; `pg:fl:` callback changes page; `fs:` callback shows priority keyboard; `fp:` callback calls `SetFilePriority` and re-renders file list; `bk:fl:` callback re-renders torrent detail)
  - Gate: 4

- **TASK-8**: Write integration and E2E tests
  - Derived from: DES-1, DES-2, DES-3, DES-4, DES-5, DES-6
  - Implements: REQ-1 through REQ-6
  - Impacts: `internal/qbt/http_integration_test.go`, `internal/bot/e2e_test.go`
  - Verification: TEST-7 (`ListFiles` integration test against real qBittorrent: verifies JSON contract, non-empty response for a known torrent); TEST-8 (`SetFilePriority` integration test: sets priority and verifies it is reflected by a follow-up `ListFiles` call); `make test-integration` passes
  - Gate: 4, 5

## Dependencies

```
TASK-1 → TASK-2 → TASK-3
TASK-1 → TASK-4
TASK-4 → TASK-5
TASK-3, TASK-5 → TASK-7
TASK-5 → TASK-6
TASK-6, TASK-7 → TASK-8
```

## Affected Files

| File | Change |
|------|--------|
| `internal/qbt/types.go` | Add `TorrentFile` struct, `FilePriority` type and constants |
| `internal/qbt/client.go` | Add `ListFiles` and `SetFilePriority` to `Client` interface |
| `internal/qbt/http.go` | Implement `ListFiles` and `SetFilePriority` HTTP methods |
| `internal/formatter/format.go` | Add `PriorityLabel`, `FormatFileList`, `FileListKeyboard`, `PriorityKeyboard` |
| `internal/bot/handler.go` | Add "Files" button to detail keyboard builder |
| `internal/bot/callback.go` | Route `fl:`, `pg:fl:`, `fs:`, `fp:`, `bk:fl:` callbacks |
| `internal/qbt/http_test.go` | Unit tests for `ListFiles` and `SetFilePriority` (TASK-3) |
| `internal/formatter/format_test.go` | Unit tests for formatter and keyboard functions (TASK-4, TASK-5) |
| `internal/bot/handler_test.go` | Unit test for "Files" button presence (TASK-6) |
| `internal/bot/callback_test.go` | Unit tests for new callback routes (TASK-7) |
| `internal/qbt/http_integration_test.go` | Integration tests for `ListFiles` / `SetFilePriority` (TASK-8) |
| `internal/bot/e2e_test.go` | E2E test for file list and priority change flow (TASK-8) |

## Rollout Notes

No migration needed. The "Files" button appears automatically on all torrent detail views once TASK-6 is deployed. No feature flag is required — the feature is safe to enable immediately because it only adds a new button and new callback paths; existing functionality is unaffected.

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
design_items=$(grep -oP 'DES-\d+' docs/features/torrent-files/design.md | sort -u)
plan_items=$(grep -oP 'DES-\d+' docs/features/torrent-files/plan.md | sort -u)
comm -23 <(echo "$design_items") <(echo "$plan_items")  # should be empty

grep "^- \*\*TASK-" docs/features/torrent-files/plan.md | wc -l    # should be 8
grep "Verification:" docs/features/torrent-files/plan.md | wc -l   # should match (8)
```

### Iterative Harness Loop Protocol

1. Execute TASK-1 through TASK-7 in dependency order.
2. After each task, run `make build && make lint` — fix any failures before proceeding (max 3 retries per task).
3. After TASK-3, TASK-4, TASK-5, TASK-6, TASK-7: run the corresponding unit tests listed in Verification.
4. Execute TASK-8 (integration + E2E tests).
5. Run `make gate-all` — must pass before Gate 4 is considered met.
6. Run `make test-integration` — must pass before Gate 5 is considered met.
7. Update `traceability.md` and `verification.md` with evidence for each AC-*.

## Verification Steps

See `docs/features/torrent-files/verification.md` (to be created during TASK-8).

Summary of what must pass:
- TEST-1: `TorrentFile` JSON round-trip and `FilePriority` constant values.
- TEST-2: `ListFiles` and `SetFilePriority` unit tests via `httptest`.
- TEST-3: `FormatFileList` output format, truncation, length constraints.
- TEST-4: `FileListKeyboard` and `PriorityKeyboard` callback data correctness and byte-length constraints.
- TEST-5: Detail keyboard contains `fl:` button.
- TEST-6: All five new callback routes exercise correct client methods.
- TEST-7: `ListFiles` integration test against real qBittorrent.
- TEST-8: `SetFilePriority` integration test with observable state change.
- `make gate-all` passes.
- `make test-integration` passes.

## Blockers

None.
