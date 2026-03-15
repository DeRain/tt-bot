---
title: "<Feature Title> — Plan"
feature_id: "<feature-id>"
status: draft | in-review | approved | implemented
depends_on_design: "docs/features/<feature-id>/design.md"
last_updated: YYYY-MM-DD
---

# <Feature Title> — Plan

## Overview

<!-- Summarize the implementation approach and sequencing strategy. -->

## Preconditions

<!-- What must be true before this plan can execute? -->

## Task Sequence

<!-- Each task gets a stable ID. Tasks are ordered and actionable. -->

- **TASK-1**: <implementation task>
  - Derived from: DES-1
  - Implements: REQ-1
  - Impacts: <files/modules/components>
  - Verification: TEST-1, CHECK-1
  - Gate: <which gate this task must pass>

- **TASK-2**: <implementation task>
  - Derived from: DES-2
  - Implements: REQ-2
  - Impacts: <files/modules/components>
  - Verification: TEST-2
  - Gate: <which gate this task must pass>

## Dependencies

<!-- Task dependency graph. Which tasks block others? -->

## Affected Files

<!-- Complete list of files created or modified. -->

## Rollout Notes

<!-- Deployment considerations, feature flags, migration steps. -->

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
# Verify plan-to-design coverage
design_items=$(grep -oP 'DES-\d+' docs/features/<feature-id>/design.md | sort -u)
plan_items=$(grep -oP 'DES-\d+' docs/features/<feature-id>/plan.md | sort -u)
comm -23 <(echo "$design_items") <(echo "$plan_items")  # should be empty

# Verify every task has verification
grep "^- \*\*TASK-" docs/features/<feature-id>/plan.md | wc -l  # task count
grep "Verification:" docs/features/<feature-id>/plan.md | wc -l  # should match
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

<!-- Post-implementation verification summary. References verification.md. -->

## Blockers

<!-- Known blockers. Empty if none. -->
