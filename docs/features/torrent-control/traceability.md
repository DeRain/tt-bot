---
title: "Torrent Control — Traceability Matrix"
feature_id: "torrent-control"
status: complete
last_updated: 2026-03-15
---

# Torrent Control — Traceability Matrix

## Forward Traceability (Requirement → Verification)

| Requirement | Acceptance Criteria | Design | Plan Tasks | Implementation Evidence | Verification | Status |
|-------------|---------------------|--------|------------|-------------------------|--------------|--------|
| REQ-1 | AC-1.1, AC-1.2 | DES-6, DES-7 | TASK-8, TASK-13 | `formatter.TorrentSelectionKeyboard`, `handler.renderTorrentListPage` | TEST-8, TEST-13, E2E-1 | Complete |
| REQ-2 | AC-2.1, AC-2.2 | DES-2, DES-3 | TASK-4, TASK-6, TASK-10 | `formatter.FormatSize`, `formatter.FormatTorrentDetail`, `callback.handleSelectCallback` | TEST-4, TEST-6, TEST-10, E2E-1 | Complete |
| REQ-3 | AC-3.1, AC-3.2 | DES-4, DES-5, DES-8 | TASK-5, TASK-7, TASK-10 | `formatter.IsPaused`, `formatter.TorrentDetailKeyboard`, `callback.handleSelectCallback` | TEST-5, TEST-7, TEST-10 | Complete |
| REQ-4 | AC-4.1, AC-4.2 | DES-4, DES-5, DES-8 | TASK-5, TASK-7, TASK-10 | `formatter.IsPaused`, `formatter.TorrentDetailKeyboard`, `callback.handleSelectCallback` | TEST-5, TEST-7, TEST-10 | Complete |
| REQ-5 | AC-5.1, AC-5.2 | DES-1, DES-9 | TASK-2, TASK-11 | `HTTPClient.PauseTorrents`, `callback.handlePauseCallback` | TEST-2, TEST-11, ITEST-1, E2E-2 | Complete |
| REQ-6 | AC-6.1, AC-6.2 | DES-1, DES-9 | TASK-2, TASK-11 | `HTTPClient.ResumeTorrents`, `callback.handleResumeCallback` | TEST-2, TEST-11, ITEST-1, E2E-2 | Complete |
| REQ-7 | AC-7.1 | DES-4, DES-10 | TASK-7, TASK-12 | `formatter.TorrentDetailKeyboard`, `callback.handleBackCallback` | TEST-7, TEST-12, E2E-2 | Complete |
| REQ-8 | AC-8.1 | DES-4, DES-6, DES-11 | TASK-7, TASK-8, TASK-9 | `formatter.TorrentDetailKeyboard`, `formatter.TorrentSelectionKeyboard`, `callback.filterCharToFilter` | TEST-7, TEST-8, TEST-9 | Complete |
| REQ-9 | AC-2.2 | DES-2 | TASK-6 | `formatter.FormatTorrentDetail` (length guard) | TEST-6 | Complete |
| REQ-10 | AC-10.1, AC-10.2 | DES-1 | TASK-1, TASK-2, TASK-3 | `qbt.Client` interface, `HTTPClient.PauseTorrents/ResumeTorrents`, mock updates | TEST-1, TEST-2, TEST-3, ITEST-1 | Complete |

## Backward Traceability (Code → Requirement)

| Source File | Functions/Types | Traces To | Via |
|-------------|----------------|-----------|-----|
| `internal/qbt/client.go` | `PauseTorrents`, `ResumeTorrents` | REQ-10 | TASK-1, DES-1 |
| `internal/qbt/http.go` | `HTTPClient.PauseTorrents`, `HTTPClient.ResumeTorrents` | REQ-5, REQ-6, REQ-10 | TASK-2, DES-1 |
| `internal/qbt/http_integration_test.go` | `TestIntegration_PauseAndResumeTorrent` | REQ-5, REQ-6, REQ-10 | ITEST-1 |
| `internal/formatter/format.go` | `FormatSize` | REQ-2 | TASK-4, DES-3 |
| `internal/formatter/format.go` | `IsPaused` | REQ-3, REQ-4 | TASK-5, DES-5 |
| `internal/formatter/format.go` | `FormatTorrentDetail` | REQ-2, REQ-9 | TASK-6, DES-2 |
| `internal/formatter/format.go` | `TorrentDetailKeyboard` | REQ-3, REQ-4, REQ-7, REQ-8 | TASK-7, DES-4 |
| `internal/formatter/format.go` | `TorrentSelectionKeyboard` | REQ-1, REQ-8 | TASK-8, DES-6 |
| `internal/bot/callback.go` | `filterCharToFilter`, `filterCharToPrefix`, `filterToChar` | REQ-8 | TASK-9, DES-11 |
| `internal/bot/callback.go` | `handleSelectCallback` | REQ-2, REQ-3, REQ-4 | TASK-10, DES-8 |
| `internal/bot/callback.go` | `handlePauseCallback`, `handleResumeCallback`, `handleTorrentAction` | REQ-5, REQ-6 | TASK-11, DES-9 |
| `internal/bot/callback.go` | `handleBackCallback` | REQ-7 | TASK-12, DES-10 |
| `internal/bot/callback.go` | `renderTorrentListPage`, `parseControlCallback`, `findTorrentByHash` | REQ-1 | TASK-13, DES-7 |
| `internal/bot/handler.go` | `sendTorrentPage` (refactored) | REQ-1 | TASK-13, DES-7 |
| `internal/bot/e2e_test.go` | `TestE2E_SelectTorrentShowsDetail` | REQ-1, REQ-2 | E2E-1 |
| `internal/bot/e2e_test.go` | `TestE2E_PauseResumeTorrent` | REQ-5, REQ-6, REQ-7 | E2E-2 |
| `internal/bot/handler_test.go` | `mockQBTClient.PauseTorrents/ResumeTorrents` | REQ-10 | TASK-3 |
| `internal/poller/poller_test.go` | `mockQBT.PauseTorrents/ResumeTorrents` | REQ-10 | TASK-3 |
| `internal/bot/handler_extra_test.go` | `errorQBTClient.PauseTorrents/ResumeTorrents` | REQ-10 | TASK-3 |

## Coverage Summary

| Metric | Count | Covered | Gaps |
|--------|-------|---------|------|
| Requirements | 10 | 10 | 0 |
| Acceptance Criteria | 17 | 17 | 0 |
| Design Items | 11 | 11 | 0 |
| Plan Tasks | 14 | 14 | 0 |
| Unit Tests | 13 | 13 (all PASS) | 0 |
| Integration Tests | 1 | 1 (PASS) | 0 |
| E2E Tests | 2 | 2 (PASS) | 0 |

## Rules

- No REQ-* may exist without at least one linked DES-*.
- No DES-* may exist without at least one linked TASK-*.
- No TASK-* may exist without at least one linked verification item.
- No AC-* may remain unverified.
- Status values: Complete | Partial | Blocked | Missing | N/A

## Harness Validation

```bash
# Count untraced requirements (should be 0)
grep "| Complete |" docs/features/torrent-control/traceability.md | wc -l  # should be 10

# Count missing verification (should be 0)
grep "| Missing |" docs/features/torrent-control/traceability.md | wc -l  # should be 0

# Run integration tests to close remaining gaps
make test-integration
```
