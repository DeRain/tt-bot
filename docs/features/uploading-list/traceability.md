---
title: "Uploading Torrents List — Traceability Matrix"
feature_id: "uploading-list"
status: draft
last_updated: 2026-03-15
---

# Uploading Torrents List — Traceability Matrix

## Forward Traceability (Requirement → Verification)

| Requirement | Acceptance Criteria | Design | Plan Tasks | Implementation Evidence | Verification | Status |
|-------------|---------------------|--------|------------|------------------------|--------------|--------|
| REQ-1 | AC-1.1, AC-1.2 | DES-1, DES-2 | TASK-1, TASK-4, TASK-5 | TODO | TEST-1, TEST-4 | TODO |
| REQ-2 | AC-2.1, AC-2.2 | DES-1, DES-2 | TASK-1, TASK-4, TASK-5 | TODO | TEST-4 | TODO |
| REQ-3 | AC-3.1, AC-3.2 | DES-4 | TASK-2, TASK-3, TASK-5 | TODO | TEST-2, TEST-3 | TODO |
| REQ-4 | AC-4.1, AC-4.2 | DES-4 | TASK-2, TASK-3, TASK-5 | TODO | TEST-2, TEST-3 | TODO |
| REQ-5 | AC-5.1 | DES-3 | TASK-4, TASK-5 | TODO | TEST-5 | TODO |

## Backward Traceability (Code → Requirement)

| File | Symbol / Change | Requirement | Task |
|------|----------------|-------------|------|
| — | — (not yet implemented) | — | — |

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
