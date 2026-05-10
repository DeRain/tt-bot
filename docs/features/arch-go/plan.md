---
title: "Architecture Governance (arch-go) — Plan"
feature_id: "arch-go"
status: implemented
depends_on_design: "docs/features/arch-go/design.md"
last_updated: 2026-05-10
---

# Architecture Governance (arch-go) — Plan

## Overview

Four implementation tasks: create the `arch-go.yml` configuration file (TASK-1), add the `arch-check` Makefile target and wire into `gate-all` (TASK-2), update quality gate documentation (TASK-3 through TASK-5), and run full verification (TASK-6). All tasks are independent of each other except TASK-6 which depends on TASK-1 and TASK-2.

## Preconditions

- `arch-go` CLI available via `go run` (no pre-install required)
- Current codebase passes `make build`, `make lint`, `make test`
- Module path is `github.com/home/tt-bot`

## Task Sequence

- **TASK-1**: Create `arch-go.yml` at repository root with 6 dependency rules and threshold configuration
  - Derived from: DES-1
  - Implements: REQ-1, REQ-5
  - Impacts: `arch-go.yml` (new)
  - Verification: `go run github.com/arch-go/arch-go/v2@latest` exits 0; `arch-go describe` lists all 6 rules
  - Gate: Gate 4 (Implementation Gate)

- **TASK-2**: Add `arch-check` target to Makefile and wire into `gate-all` dependency chain
  - Derived from: DES-2, DES-3
  - Implements: REQ-2, REQ-3
  - Impacts: `Makefile` (modify)
  - Verification: `make arch-check` exits 0; `make gate-all` exits 0
  - Gate: Gate 4 (Implementation Gate)

- **TASK-3**: Update `docs/gates.md` Gate 4 (Implementation Gate) criteria to include architecture check
  - Derived from: DES-4
  - Implements: REQ-4
  - Impacts: `docs/gates.md` (modify)
  - Verification: `grep "arch-check" docs/gates.md` returns the updated line
  - Gate: Gate 4 (Implementation Gate)

- **TASK-4**: Update `docs/pr-checklist.md` Verification Evidence section with architecture rules checkbox
  - Derived from: DES-4
  - Implements: REQ-4
  - Impacts: `docs/pr-checklist.md` (modify)
  - Verification: `grep "arch-check" docs/pr-checklist.md` returns the updated line
  - Gate: Gate 4 (Implementation Gate)

- **TASK-5**: Update `docs/features/_templates/plan.md` quality gates section to reference architecture check
  - Derived from: DES-4
  - Implements: REQ-4
  - Impacts: `docs/features/_templates/plan.md` (modify)
  - Verification: `grep "arch-check" docs/features/_templates/plan.md` returns the updated line
  - Gate: Gate 4 (Implementation Gate)

- **TASK-7**: Pass TELEGRAM_BOT_TOKEN and TELEGRAM_ALLOWED_USERS to Docker test container via docker-compose.test.yml
  - Derived from: DES-5
  - Implements: REQ-6
  - Impacts: `docker-compose.test.yml` (modify)
  - Verification: `grep "TELEGRAM_BOT_TOKEN" docker-compose.test.yml` returns the line
  - Gate: Gate 4 (Implementation Gate)

- **TASK-6**: Run full verification: `make gate-all`, record results in traceability.md and verification.md
  - Derived from: DES-1, DES-2, DES-3, DES-4
  - Implements: REQ-1, REQ-2, REQ-3, REQ-4, REQ-5
  - Impacts: `docs/features/arch-go/traceability.md` (update), `docs/features/arch-go/verification.md` (update)
  - Verification: `make gate-all` exits 0; all AC-* have PASS results in verification.md
  - Gate: Gate 5 (Verification Gate)

## Dependencies

```
TASK-1 ──┐
          ├── TASK-6
TASK-2 ──┤
TASK-7 ──┘

TASK-3 ── (independent)
TASK-4 ── (independent)
TASK-5 ── (independent)
```

TASK-1, TASK-2, TASK-3, TASK-4, TASK-5, TASK-7 can run in parallel. TASK-6 requires TASK-1, TASK-2, and TASK-7.

## Affected Files

| File | Action | Justification |
|------|--------|---------------|
| `arch-go.yml` | CREATE | Architecture rules configuration |
| `Makefile` | UPDATE | Add `arch-check` target, wire into `gate-all` |
| `docs/gates.md` | UPDATE | Gate 4 criteria includes arch-check |
| `docs/pr-checklist.md` | UPDATE | Verification evidence includes architecture rules |
| `docs/features/_templates/plan.md` | UPDATE | Quality gates section references arch-check |
| `docs/features/arch-go/spec.md` | CREATE | Feature specification |
| `docs/features/arch-go/design.md` | CREATE | Feature design |
| `docs/features/arch-go/plan.md` | CREATE | Feature plan (this file) |
| `docs/features/arch-go/traceability.md` | CREATE | Traceability matrix |
| `docs/features/arch-go/verification.md` | CREATE | Verification results |

## Rollout Notes

- No deployment impact — `arch-go.yml` is a dev-only artifact
- No runtime dependency — `go run` fetches on demand
- No go.mod changes — zero impact on the binary

## Quality Gates

### Gate 3: Plan Gate

This plan passes when:
- [ ] Every TASK-* maps to at least one DES-* and REQ-*
- [ ] Task sequencing is coherent (dependencies respected)
- [ ] Every TASK-* has a verification target (TEST-* or CHECK-*)
- [ ] No TASK-* exists without implementation evidence path
- [ ] Affected files are listed

**Harness check command:**
```bash
design_items=$(grep -oP 'DES-\d+' docs/features/arch-go/design.md | sort -u)
plan_items=$(grep -oP 'DES-\d+' docs/features/arch-go/plan.md | sort -u)
comm -23 <(echo "$design_items") <(echo "$plan_items")  # should be empty

grep "^- \*\*TASK-" docs/features/arch-go/plan.md | wc -l  # task count
grep "Verification:" docs/features/arch-go/plan.md | wc -l  # should match
```

### Iterative Harness Loop Protocol

When executing this plan via an agent harness loop:
1. Execute tasks in dependency order
2. After each TASK-*, run its verification target
3. If verification fails: fix, re-verify, max 3 retries
4. After all tasks: run `make gate-all` as the implementation gate
5. Update traceability.md with implementation evidence
6. Run verification.md checks

## Verification Steps

Reference `verification.md` for per-AC results and `traceability.md` for the full requirement-to-code mapping.

## Blockers

None.
