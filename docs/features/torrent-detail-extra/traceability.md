---
title: "Extended Torrent Detail Info — Traceability Matrix"
feature_id: "torrent-detail-extra"
status: draft
last_updated: 2026-03-15
---

# Extended Torrent Detail Info — Traceability Matrix

## Forward Traceability (Requirement → Verification)

| Requirement | Acceptance Criteria | Design | Plan Tasks | Implementation Evidence | Verification | Status |
|-------------|---------------------|--------|------------|-------------------------|--------------|--------|
| REQ-1 | AC-1.1, AC-1.2, AC-1.3 | DES-1, DES-2 | TASK-2, TASK-3 | `internal/formatter/formatter.go` (`FormatTorrentDetail`) | TEST-2, TEST-3 | TODO |
| REQ-2 | AC-2.1, AC-2.2, AC-2.3 | DES-1, DES-2 | TASK-2, TASK-3 | `internal/formatter/formatter.go` (`FormatTorrentDetail`) | TEST-2, TEST-3 | TODO |
| REQ-3 | AC-3.1, AC-3.2, AC-3.3 | DES-1, DES-3 | TASK-1, TASK-4 | `internal/qbt/client.go` (`Torrent`) | TEST-1, TEST-4 | TODO |

## Backward Traceability (Code → Requirement)

| Source File | Functions/Types | Traces To | Via |
|-------------|----------------|-----------|-----|
| `internal/qbt/client.go` | `Torrent` (fields `Uploaded`, `Ratio`) | REQ-3 | TASK-1, DES-1, DES-3 |
| `internal/formatter/formatter.go` | `FormatTorrentDetail` | REQ-1, REQ-2 | TASK-2, DES-2 |
| `internal/formatter/formatter_test.go` | `TestFormatTorrentDetail` | REQ-1, REQ-2 | TASK-3, DES-2 |
| `internal/qbt/http_integration_test.go` | integration assertions | REQ-3 | TASK-4, DES-3 |

## Coverage Summary

| Metric | Count | Covered | Gaps |
|--------|-------|---------|------|
| Requirements | 3 | 0 | 3 |
| Acceptance Criteria | 9 | 0 | 9 |
| Design Items | 3 | 0 | 3 |
| Plan Tasks | 4 | 0 | 4 |
| Verification Items | 4 | 0 | 4 |

## Rules

- No REQ-* may exist without at least one linked DES-*.
- No DES-* may exist without at least one linked TASK-*.
- No TASK-* may exist without at least one linked verification item.
- No AC-* may remain unverified.
- Status values: Complete | Partial | Blocked | Missing | N/A

## Harness Validation

```bash
# Count untraced requirements (should be 0)
grep "| TODO |" docs/features/torrent-detail-extra/traceability.md | wc -l

# Count missing verification (should be 0)
grep "| Missing |" docs/features/torrent-detail-extra/traceability.md | wc -l
```
