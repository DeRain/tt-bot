---
title: "Human-Readable Statuses with Emojis — Plan"
feature_id: status-emojis
status: draft
depends_on_design: "docs/features/status-emojis/design.md"
last_updated: 2026-03-15
---

# Human-Readable Statuses with Emojis — Plan

## Overview

Three sequential tasks: write tests first (TDD), add the mapping and function, then wire it into the existing format functions. The entire change is confined to `internal/formatter`.

## Preconditions

- `docs/features/status-emojis/spec.md` approved (Gate 1 passed).
- `docs/features/status-emojis/design.md` approved (Gate 2 passed).
- `make gate-all` passes on the current branch before changes begin.

## Task Sequence

- **TASK-1**: Write unit tests for `FormatState` covering all 19 documented states, the empty-string fallback, and an unrecognised state fallback.
  - Derived from: DES-1, DES-2
  - Implements: REQ-1, REQ-2, REQ-4
  - Impacts: `internal/formatter/formatter_test.go`
  - Verification: TEST-1
  - Gate: 4

- **TASK-2**: Write unit tests for the updated `FormatTorrentList` and `FormatTorrentDetail` asserting that mapped labels appear in output and raw state strings do not.
  - Derived from: DES-3
  - Implements: REQ-3
  - Impacts: `internal/formatter/formatter_test.go`
  - Verification: TEST-2
  - Gate: 4
  - Depends on: TASK-1 (tests can be authored in the same pass, but must be red before TASK-3)

- **TASK-3**: Add `stateLabels` map and `FormatState` function to `internal/formatter/formatter.go`.
  - Derived from: DES-1, DES-2
  - Implements: REQ-1, REQ-2, REQ-4
  - Impacts: `internal/formatter/formatter.go`
  - Verification: TEST-1 (green)
  - Gate: 4
  - Depends on: TASK-1

- **TASK-4**: Update `FormatTorrentList` and `FormatTorrentDetail` to call `FormatState(t.State)` instead of `t.State`.
  - Derived from: DES-3
  - Implements: REQ-3
  - Impacts: `internal/formatter/formatter.go`
  - Verification: TEST-2 (green)
  - Gate: 4
  - Depends on: TASK-2, TASK-3

- **TASK-5**: Run `make gate-all` and confirm all tests pass; update `traceability.md` and `verification.md`.
  - Derived from: DES-1, DES-2, DES-3
  - Implements: REQ-1, REQ-2, REQ-3, REQ-4
  - Impacts: `docs/features/status-emojis/traceability.md`, `docs/features/status-emojis/verification.md`
  - Verification: CHECK-1
  - Gate: 4–5
  - Depends on: TASK-4

## Dependencies

```
TASK-1 → TASK-3 → TASK-4 → TASK-5
TASK-2 → TASK-4
```

TASK-1 and TASK-2 may be authored together (both are test-only files, no production code changes). TASK-3 must not begin until TASK-1 tests are confirmed red.

## Affected Files

| File | Change |
|------|--------|
| `internal/formatter/formatter.go` | Add `stateLabels` map and `FormatState` func; update `FormatTorrentList` and `FormatTorrentDetail` |
| `internal/formatter/formatter_test.go` | Add `TestFormatState` table-driven tests; extend existing list/detail tests |
| `docs/features/status-emojis/traceability.md` | Created/updated post-implementation |
| `docs/features/status-emojis/verification.md` | Created/updated post-implementation |

## Rollout Notes

No environment variables, feature flags, database migrations, or deployment steps are required. The change is a pure in-process formatter update; deploy by rebuilding and restarting the bot container.

## Quality Gates

### Gate 3: Plan Gate

This plan passes when:
- [x] Every TASK-* maps to at least one DES-* and REQ-*
- [x] Task sequencing is coherent (dependencies respected)
- [x] Every TASK-* has a verification target (TEST-* or CHECK-*)
- [x] No TASK-* exists without implementation evidence path
- [x] Affected files are listed

**Harness check command:**
```bash
# Verify plan-to-design coverage
design_items=$(grep -oP 'DES-\d+' docs/features/status-emojis/design.md | sort -u)
plan_items=$(grep -oP 'DES-\d+' docs/features/status-emojis/plan.md | sort -u)
comm -23 <(echo "$design_items") <(echo "$plan_items")  # should be empty

# Verify every task has verification
grep "^- \*\*TASK-" docs/features/status-emojis/plan.md | wc -l  # expect 5
grep "Verification:" docs/features/status-emojis/plan.md | wc -l  # expect 5
```

### Iterative Harness Loop Protocol

When executing this plan via an agent harness loop:
1. Execute tasks in dependency order (TASK-1 → TASK-2 → TASK-3 → TASK-4 → TASK-5).
2. After TASK-1 and TASK-2: run `go test ./internal/formatter/... -short -v -run TestFormatState` — expect FAIL (red).
3. After TASK-3: re-run — expect PASS for `TestFormatState` (green).
4. After TASK-4: run `go test ./internal/formatter/... -short -v` — expect all tests green.
5. After TASK-5: run `make gate-all` — expect clean build, lint, and test pass.
6. If any step fails: fix, re-verify, max 3 retries, then escalate.

## Verification Steps

- **TEST-1**: `TestFormatState` — table-driven test covering all 19 states, empty input, and unrecognised input. Asserts correct emoji prefix and label text. Verifies AC-1.1, AC-1.2, AC-1.3, AC-2.1, AC-2.2, AC-4.1, AC-4.2.
- **TEST-2**: Updated `TestFormatTorrentList` and `TestFormatTorrentDetail` — assert mapped label present and raw state string absent. Verifies AC-3.1, AC-3.2.
- **CHECK-1**: `make gate-all` exits 0 with no lint warnings. Verifies overall build health.

Full evidence recorded in `docs/features/status-emojis/verification.md`.

## Blockers

None.
