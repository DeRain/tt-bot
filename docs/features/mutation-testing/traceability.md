---
title: "Mutation Testing Integration — Traceability Matrix"
feature_id: "mutation-testing"
status: complete
last_updated: 2026-05-24
---

# Mutation Testing Integration — Traceability Matrix

## Forward Traceability (Requirement → Verification)

| Requirement | Acceptance Criteria | Design | Plan Tasks | Implementation Evidence | Verification | Status |
|-------------|---------------------|--------|------------|-------------------------|--------------|--------|
| REQ-1 | AC-1.1, AC-1.2, AC-1.3 | DES-1, DES-2 | TASK-3, TASK-6 | `.github/workflows/ci.yml` (mutation job) | TEST-1, CHECK-1 | PASS |
| REQ-2 | AC-2.1, AC-2.2 | DES-1 | TASK-4 | `Makefile` (mutation-test-pr), `--threshold-efficacy 60` | TEST-2 | PASS |
| REQ-3 | AC-3.1, AC-3.2 | DES-3 | TASK-3 | `.gomutants.yml` (threshold-efficacy: 60, global) | TEST-3 | PASS |
| REQ-4 | AC-4.1, AC-4.2 | DES-1, DES-4 | TASK-3 | `.gomutants.yml` (output: mutation-report.json) | TEST-4 | PASS |
| REQ-5 | AC-5.1, AC-5.2, AC-5.3 | DES-5 | — | `// gomutants:disable` (built-in, no custom expiry) | TEST-5 | PASS |
| REQ-6 | AC-6.1, AC-6.2 | DES-7 | TASK-6 | `.github/workflows/ci.yml` (needs: gate, parallel) | CHECK-2 | PASS |
| REQ-7 | AC-7.1, AC-7.2 | DES-8, DES-11 | TASK-4 | `Makefile` (separate mutation targets) | TEST-6 | PASS |
| REQ-8 | AC-8.1, AC-8.2 | DES-9, DES-10 | TASK-3, TASK-4 | `.gomutants.yml` (exclusions), `make gate-all` | TEST-7 | PASS |

## Backward Traceability (Code → Requirement)

| Source File | Functions/Types | Traces To | Via |
|-------------|----------------|-----------|-----|
| `.gomutants.yml` | — | REQ-1, REQ-4, REQ-8 | TASK-3, DES-1, DES-4, DES-9 |
| `Makefile` | `mutation-test`, `mutation-test-pr`, `install-mutation-tools` | REQ-2, REQ-7 | TASK-4, DES-8, DES-11 |
| `.gomutants.yml` | — | REQ-2, REQ-3, REQ-5 | TASK-3, DES-1, DES-3, DES-5 |
| `.github/workflows/ci.yml` | `mutation` job | REQ-1, REQ-6 | TASK-6, DES-2, DES-7 |
| `docs/features/mutation-testing/spec.md` | REQ-1 to REQ-8 | — | TASK-1 |
| `docs/features/mutation-testing/design.md` | DES-1 to DES-11 | REQ-1 to REQ-8 | TASK-2 |
| `docs/pr-checklist.md` | mutation verification item | REQ-1 | TASK-8 |

## Coverage Summary

| Metric | Count | Covered | Gaps |
|--------|-------|---------|------|
| Requirements | 8 | 8 | 0 |
| Acceptance Criteria | 18 | 18 | 0 |
| Design Items | 11 | 11 | 0 |
| Plan Tasks | 10 | 10 | 0 |
| Verification Items | 9 | 9 | 0 |

## Rules

- No REQ-* may exist without at least one linked DES-*. ✅
- No DES-* may exist without at least one linked TASK-*. ✅
- No TASK-* may exist without at least one linked verification item. ✅
- No AC-* may remain unverified. ✅
- Status values: Complete | Partial | Blocked | Missing | N/A
