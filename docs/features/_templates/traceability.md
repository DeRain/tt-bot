---
title: "<Feature Title> — Traceability Matrix"
feature_id: "<feature-id>"
status: draft | complete
last_updated: YYYY-MM-DD
---

# <Feature Title> — Traceability Matrix

## Forward Traceability (Requirement → Verification)

| Requirement | Acceptance Criteria | Design | Plan Tasks | Implementation Evidence | Verification | Status |
|-------------|---------------------|--------|------------|-------------------------|--------------|--------|
| REQ-1 | AC-1.1, AC-1.2 | DES-1 | TASK-1 | `file.go` (`functionName`) | TEST-1 | TODO |
| REQ-2 | AC-2.1 | DES-2 | TASK-2 | `file.go` (`functionName`) | TEST-2 | TODO |

## Backward Traceability (Code → Requirement)

| Source File | Functions/Types | Traces To | Via |
|-------------|----------------|-----------|-----|
| `internal/pkg/file.go` | `FunctionName` | REQ-1 | TASK-1, DES-1 |

## Coverage Summary

| Metric | Count | Covered | Gaps |
|--------|-------|---------|------|
| Requirements | 0 | 0 | 0 |
| Acceptance Criteria | 0 | 0 | 0 |
| Design Items | 0 | 0 | 0 |
| Plan Tasks | 0 | 0 | 0 |
| Verification Items | 0 | 0 | 0 |

## Rules

- No REQ-* may exist without at least one linked DES-*.
- No DES-* may exist without at least one linked TASK-*.
- No TASK-* may exist without at least one linked verification item.
- No AC-* may remain unverified.
- Status values: Complete | Partial | Blocked | Missing | N/A

## Harness Validation

```bash
# Count untraced requirements (should be 0)
grep "| TODO |" docs/features/<feature-id>/traceability.md | wc -l

# Count missing verification (should be 0)
grep "| Missing |" docs/features/<feature-id>/traceability.md | wc -l
```
