---
title: "<Feature Title>"
feature_id: "<feature-id>"
status: draft | in-review | approved | implemented
owner: TODO
source_files: []
last_updated: YYYY-MM-DD
---

# <Feature Title> — Specification

## Overview

<!-- 1-2 sentence summary of the feature -->

## Problem Statement

<!-- What problem does this feature solve? Why is it needed? -->

## Goals

<!-- Bulleted list of what this feature achieves -->

## Non-Goals

<!-- Explicitly out of scope -->

## Scope

<!-- Boundary of this feature — what it covers and where it stops -->

## Requirements

<!-- Each requirement gets a stable ID. Requirements must be implementation-agnostic where possible. -->

- **REQ-1**: <clear requirement statement>
- **REQ-2**: <clear requirement statement>

## Acceptance Criteria

<!-- Each AC must be testable and observable. Linked to a parent REQ. -->

- **AC-1.1**: <testable criterion for REQ-1>
- **AC-1.2**: <testable criterion for REQ-1>
- **AC-2.1**: <testable criterion for REQ-2>

## Quality Gates

### Gate 1: Spec Gate

This spec passes when:
- [ ] All requirements are clear and unambiguous
- [ ] All acceptance criteria are testable
- [ ] Scope and non-goals are defined
- [ ] No unresolved open questions block implementation
- [ ] At least one AC exists per requirement

**Harness check command:**
```bash
# Verify spec completeness (used by iterative harness loops)
grep -c "^- \*\*REQ-" docs/features/<feature-id>/spec.md  # count requirements
grep -c "^- \*\*AC-"  docs/features/<feature-id>/spec.md  # count acceptance criteria
grep -c "TODO:"        docs/features/<feature-id>/spec.md  # should be 0 for approved
```

## Risks

<!-- What could go wrong? How is each risk mitigated? -->

## Open Questions

<!-- Unresolved items that may affect requirements. Mark each OPEN or RESOLVED. -->
