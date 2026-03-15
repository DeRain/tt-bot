---
title: "<Feature Title> — Verification"
feature_id: "<feature-id>"
status: draft | in-progress | verified
last_updated: YYYY-MM-DD
---

# <Feature Title> — Verification

## Validation Strategy

<!-- Overall approach: automated tests, manual checks, or both. -->

## Automated Tests

- **TEST-1**: <test description>
  - Validates: AC-1.1
  - Covers: REQ-1
  - Evidence: <test file, function name, command to run>
  - Pass criteria: <what "pass" means>

- **TEST-2**: <test description>
  - Validates: AC-2.1
  - Covers: REQ-2
  - Evidence: <test file, function name, command to run>
  - Pass criteria: <what "pass" means>

## Manual Checks

- **CHECK-1**: <manual validation step>
  - Validates: AC-1.2
  - Covers: REQ-1
  - Evidence: <screenshots, logs, reviewer signoff>
  - Pass criteria: <what "pass" means>

## Acceptance Criteria Results

| AC | Validation | Result | Evidence |
|----|-----------|--------|----------|
| AC-1.1 | TEST-1 | TODO | — |
| AC-1.2 | CHECK-1 | TODO | — |
| AC-2.1 | TEST-2 | TODO | — |

## Quality Gates

### Gate 5: Verification Gate

This verification passes when:
- [ ] Every AC-* has at least one TEST-* or CHECK-*
- [ ] All automated tests pass (`make test`)
- [ ] All manual checks are recorded with evidence
- [ ] No AC-* has Result = TODO or FAIL
- [ ] Gaps are explicitly documented (not silently omitted)

**Harness check commands:**
```bash
# Run unit tests for this feature's packages
go test ./internal/<pkg>/... -short -v -cover

# Count unverified ACs (should be 0)
grep "| TODO |" docs/features/<feature-id>/verification.md | wc -l

# Integration tests (if applicable)
make test-integration
```

## Traceability Coverage

<!-- Summary: X of Y requirements verified, Z acceptance criteria validated. -->

## Exceptions / Unresolved Gaps

<!-- If any AC cannot be verified, explain why and how it will be addressed. -->
