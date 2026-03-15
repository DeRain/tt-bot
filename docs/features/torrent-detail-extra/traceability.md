---
title: "Extended Torrent Detail Info — Traceability Matrix"
feature_id: "torrent-detail-extra"
status: complete
last_updated: 2026-03-15
---

# Extended Torrent Detail Info — Traceability Matrix

## Forward Traceability (Requirement → Verification)

| Requirement | Acceptance Criteria | Design | Plan Tasks | Implementation Evidence | Verification | Status |
|-------------|---------------------|--------|------------|-------------------------|--------------|--------|
| REQ-1 | AC-1.1, AC-1.2, AC-1.3 | DES-1, DES-2 | TASK-2, TASK-3 | `internal/formatter/format.go` (`FormatTorrentDetail`) | TEST-2, TEST-3 | Complete |
| REQ-2 | AC-2.1, AC-2.2, AC-2.3 | DES-1, DES-2 | TASK-2, TASK-3 | `internal/formatter/format.go` (`FormatTorrentDetail`) | TEST-2, TEST-3 | Complete |
| REQ-3 | AC-3.1, AC-3.2, AC-3.3 | DES-1, DES-3 | TASK-1, TASK-4 | `internal/qbt/types.go` (`Torrent`) | TEST-1, TEST-4 | Complete |

## Backward Traceability (Code → Requirement)

| Source File | Functions/Types | Traces To | Via |
|-------------|----------------|-----------|-----|
| `internal/qbt/types.go` | `Torrent` (fields `Uploaded`, `Ratio`) | REQ-3 | TASK-1, DES-1, DES-3 |
| `internal/formatter/format.go` | `FormatTorrentDetail` | REQ-1, REQ-2 | TASK-2, DES-2 |
| `internal/formatter/format_test.go` | `TestFormatTorrentDetail_UploadedAndRatio_NonZero`, `TestFormatTorrentDetail_UploadedAndRatio_Zero` | REQ-1, REQ-2 | TASK-3, DES-2 |
| `internal/qbt/http_integration_test.go` | integration assertions | REQ-3 | TASK-4, DES-3 |

## Coverage Summary

| Metric | Count | Covered | Gaps |
|--------|-------|---------|------|
| Requirements | 3 | 3 | 0 |
| Acceptance Criteria | 9 | 9 | 0 |
| Design Items | 3 | 3 | 0 |
| Plan Tasks | 4 | 4 | 0 |
| Verification Items | 4 | 4 | 0 |

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
