---
title: "Uploading Torrents List — Traceability Matrix"
feature_id: "uploading-list"
status: complete
last_updated: 2026-03-15
---

# Uploading Torrents List — Traceability Matrix

## Forward Traceability (Requirement → Verification)

| Requirement | Acceptance Criteria | Design | Plan Tasks | Implementation Evidence | Verification | Status |
|-------------|---------------------|--------|------------|------------------------|--------------|--------|
| REQ-1 | AC-1.1, AC-1.2 | DES-1, DES-2 | TASK-1, TASK-4, TASK-5 | `FilterUploading` in `qbt/types.go`; `listTorrentsForFilter` post-filter in `callback.go` | TEST-1, TEST-4 | PASS |
| REQ-2 | AC-2.1, AC-2.2 | DES-1, DES-2 | TASK-1, TASK-4, TASK-5 | `Progress == 1.0` filter includes `pausedUP` and `uploading`/`stalledUP` | TEST-4 | PASS |
| REQ-3 | AC-3.1, AC-3.2 | DES-4 | TASK-2, TASK-3, TASK-5 | `pg:up:` case in `handleCallback`; `filterCharToPrefix("u") == "up"` | TEST-2, TEST-3 | PASS |
| REQ-4 | AC-4.1, AC-4.2 | DES-4 | TASK-2, TASK-3, TASK-5 | `filterCharToFilter("u")` returns `FilterUploading`; sel/pa/re/bk all work via existing flow | TEST-2, TEST-3 | PASS |
| REQ-5 | AC-5.1 | DES-3 | TASK-4, TASK-5 | `"uploading"` entry in `BotCommands`; `case "uploading"` in `handleCommand` | TEST-5 | PASS |

## Backward Traceability (Code → Requirement)

| File | Symbol / Change | Requirement | Task |
|------|----------------|-------------|------|
| `internal/qbt/types.go` | `FilterUploading TorrentFilter = "uploading"` | REQ-1, REQ-2 | TASK-1 |
| `internal/bot/callback.go` | `filterCharToFilter` case `"u"` | REQ-3, REQ-4 | TASK-2 |
| `internal/bot/callback.go` | `filterCharToPrefix` case `"u"` → `"up"` | REQ-3, REQ-4 | TASK-2 |
| `internal/bot/callback.go` | `filterToChar` case `FilterUploading` → `"u"` | REQ-3, REQ-4 | TASK-2 |
| `internal/bot/callback.go` | `case strings.HasPrefix(data, "pg:up:")` in `handleCallback` | REQ-3 | TASK-3 |
| `internal/bot/callback.go` | `listTorrentsForFilter` `FilterUploading` post-filter (`Progress == 1.0`) | REQ-1, REQ-2 | TASK-4 |
| `internal/bot/handler.go` | `case qbt.FilterUploading: filterPrefix = "up"` in `sendTorrentPage` | REQ-3 | TASK-4 |
| `internal/bot/handler.go` | `case "uploading":` dispatch in `handleCommand` | REQ-5 | TASK-4 |
| `internal/bot/commands.go` | `{Command: "uploading", …}` in `BotCommands` | REQ-5 | TASK-4 |
| `internal/bot/callback_test.go` | TEST-2, TEST-3 unit tests | REQ-3, REQ-4 | TASK-5 |
| `internal/bot/handler_test.go` | TEST-4, TEST-5 unit tests | REQ-1, REQ-2, REQ-5 | TASK-5 |
| `internal/bot/e2e_test.go` | `TestE2E_UploadingCommandReturnsValidResponse`, `TestE2E_UploadingPaginationCallback` | REQ-1–REQ-5 | TASK-5 |

## Coverage Summary

| Category | Count |
|----------|-------|
| Requirements (REQ-*) | 5 |
| Acceptance Criteria (AC-*) | 9 |
| Design Items (DES-*) | 4 |
| Plan Tasks (TASK-*) | 5 |
| Automated Tests (TEST-*) | 5 |
| Manual Checks (CHECK-*) | 1 |
| Requirements fully covered by DES | 5 / 5 |
| ACs with at least one TEST or CHECK | 9 / 9 |
| TASKs with verification target | 5 / 5 |

## Rules

1. Every REQ-* must map to at least one DES-*.
2. Every DES-* must map to at least one TASK-*.
3. Every TASK-* must have at least one verification target (TEST-* or CHECK-*).
4. Every AC-* must appear in the Acceptance Criteria Results table in `verification.md`.
5. No implementation file may be added or changed without a corresponding TASK-* and REQ-*.
6. Backward traceability must be filled in after implementation.

## Harness Validation

```bash
# Confirm spec, design, plan, traceability, and verification docs exist
ls docs/features/uploading-list/

# Check Gate 4 (build + lint + unit tests)
make gate-all

# Check Gate 5 (integration tests against real qBittorrent)
make test-integration

# Run only uploading-list–related unit tests
go test ./internal/bot/... ./internal/qbt/... -run "Uploading|uploading|FilterUp" -short -v
```
