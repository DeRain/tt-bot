---
title: "Architecture Governance (arch-go) — Traceability Matrix"
feature_id: "arch-go"
status: complete
last_updated: 2026-05-10
---

# Architecture Governance (arch-go) — Traceability Matrix

## Forward Traceability (Requirement → Verification)

| Requirement | Acceptance Criteria | Design | Plan Tasks | Implementation Evidence | Verification | Status |
|-------------|---------------------|--------|------------|-------------------------|--------------|--------|
| REQ-1 | AC-1.1, AC-1.2, AC-1.3, AC-5.1 | DES-1 | TASK-1 | `arch-go.yml` (6 dependency rules) | CHECK-1 | PASS |
| REQ-2 | AC-2.1 | DES-2 | TASK-2 | `Makefile:6-8` (`arch-check` target) | CHECK-2 | PASS |
| REQ-3 | AC-3.1, AC-3.2 | DES-3 | TASK-2 | `Makefile:18` (`gate-all: build lint test arch-check`) | CHECK-3 | PASS |
| REQ-4 | AC-4.1, AC-4.2, AC-4.3 | DES-4 | TASK-3, TASK-4, TASK-5 | `docs/gates.md`, `docs/pr-checklist.md`, `docs/features/_templates/plan.md` | CHECK-4 | PASS |
| REQ-5 | AC-5.1 | DES-1 | TASK-1 | `arch-go.yml` (no content/function/naming rules) | CHECK-1 | PASS |
| REQ-6 | AC-6.1 | DES-5 | TASK-7 | `docker-compose.test.yml` (TELEGRAM env vars) | CHECK-5 | PASS |

## Backward Traceability (Code → Requirement)

| Source File | Functions/Types | Traces To | Via |
|-------------|----------------|-----------|-----|
| `arch-go.yml` | 6 `dependenciesRules` entries | REQ-1, REQ-5 | TASK-1, DES-1 |
| `Makefile:6-8` | `arch-check` target | REQ-2 | TASK-2, DES-2 |
| `Makefile:18` | `gate-all` dependency chain | REQ-3 | TASK-2, DES-3 |
| `docs/gates.md` | Gate 4 pass criteria, harness commands | REQ-4 | TASK-3, DES-4 |
| `docs/pr-checklist.md` | Verification evidence, validation rules | REQ-4 | TASK-4, DES-4 |
| `docs/features/_templates/plan.md` | Harness loop protocol | REQ-4 | TASK-5, DES-4 |
| `docker-compose.test.yml` | TELEGRAM_BOT_TOKEN, TELEGRAM_ALLOWED_USERS | REQ-6 | TASK-7, DES-5 |

## Coverage Summary

| Metric | Count | Covered | Gaps |
|--------|-------|---------|------|
| Requirements | 6 | 6 | 0 |
| Acceptance Criteria | 11 | 11 | 0 |
| Design Items | 5 | 5 | 0 |
| Plan Tasks | 7 | 7 | 0 |
| Verification Items | 5 | 5 | 0 |

## Rules

- No REQ-* may exist without at least one linked DES-*.
- No DES-* may exist without at least one linked TASK-*.
- No TASK-* may exist without at least one linked verification item.
- No AC-* may remain unverified.
- Status values: Complete | Partial | Blocked | Missing | N/A

## Harness Validation

```bash
grep "| TODO |" docs/features/arch-go/traceability.md | wc -l   # should be 0
grep "| Missing |" docs/features/arch-go/traceability.md | wc -l # should be 0
```
