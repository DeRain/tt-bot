---
title: "GitHub Actions CI Workflow — Traceability Matrix"
feature_id: "ci"
status: complete
last_updated: 2026-05-10
---

# GitHub Actions CI Workflow — Traceability Matrix

## Forward Traceability (Requirement → Verification)

| Requirement | Acceptance Criteria | Design | Plan Tasks | Implementation Evidence | Verification | Status |
|-------------|---------------------|--------|------------|-------------------------|--------------|--------|
| REQ-1 | AC-1.1, AC-1.2 | DES-1 | TASK-1 | `.github/workflows/ci.yml` (`on.push`, `on.pull_request`) | CHECK-4, CHECK-5 | Complete |
| REQ-2 | AC-2.1, AC-2.2 | DES-2 | TASK-1 | `.github/workflows/ci.yml` (`jobs.gate`) | CHECK-1 | Complete |
| REQ-3 | AC-3.1, AC-3.2 | DES-3, DES-4 | TASK-2 | `.github/workflows/ci.yml` (`jobs.integration`, services, env vars) | CHECK-2 | Complete |
| REQ-4 | AC-4.1 | DES-6 | TASK-2 | `.github/workflows/ci.yml` (`needs: gate`) | CHECK-2 | Complete |
| REQ-5 | AC-5.1 | DES-5 | TASK-2 | `.github/workflows/ci.yml` (`${{ secrets.TELEGRAM_BOT_TOKEN }}`) | CHECK-2 | Complete |
| REQ-6 | AC-6.1, AC-6.2 | DES-1 | TASK-3, TASK-4 | `README.md` (CI section), `act push` output | CHECK-1, CHECK-2, CHECK-4 | Complete |

## Backward Traceability (Code → Requirement)

| Source File | Key Sections | Traces To | Via |
|-------------|-------------|-----------|-----|
| `.github/workflows/ci.yml` | `on.push`, `on.pull_request` | REQ-1 | TASK-1, DES-1 |
| `.github/workflows/ci.yml` | `jobs.gate` steps (build, lint, test, arch-check) | REQ-2 | TASK-1, DES-2 |
| `.github/workflows/ci.yml` | `jobs.integration`, services.qbittorrent, env vars | REQ-3 | TASK-2, DES-3, DES-4 |
| `.github/workflows/ci.yml` | `needs: gate` on integration job | REQ-4 | TASK-2, DES-6 |
| `.github/workflows/ci.yml` | `${{ secrets.TELEGRAM_BOT_TOKEN }}`, `${{ secrets.TELEGRAM_ALLOWED_USERS }}` | REQ-5 | TASK-2, DES-5 |
| `README.md` | CI section | REQ-6 | TASK-3, DES-1 |

## Coverage Summary

| Metric | Count | Covered | Gaps |
|--------|-------|---------|------|
| Requirements | 6 | 6 | 0 |
| Acceptance Criteria | 8 | 8 | 0 |
| Design Items | 6 | 6 | 0 |
| Plan Tasks | 4 | 4 | 0 |
| Verification Items | 5 | 5 | 0 |

## Rules

- No REQ-* may exist without at least one linked DES-*.
- No DES-* may exist without at least one linked TASK-*.
- No TASK-* may exist without at least one linked verification item.
- No AC-* may remain unverified.
- Status values: Complete | Partial | Blocked | Missing | N/A

## Harness Validation

```bash
# Count untraced requirements (should be 0)
grep "| TODO |" docs/features/ci/traceability.md | wc -l

# Count missing verification (should be 0)
grep "| Missing |" docs/features/ci/traceability.md | wc -l
```
