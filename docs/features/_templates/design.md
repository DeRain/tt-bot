---
title: "<Feature Title> — Design"
feature_id: "<feature-id>"
status: draft | in-review | approved | implemented
depends_on_spec: "docs/features/<feature-id>/spec.md"
last_updated: YYYY-MM-DD
---

# <Feature Title> — Design

## Overview

<!-- High-level design approach. Reference the spec. -->

## Architecture

<!-- System components, their responsibilities, and interactions. -->

## Data Flow

<!-- Step-by-step flow from trigger to outcome. Numbered list. -->

## Interfaces

<!-- Public APIs, function signatures, interface types. -->

## Data/Storage Impact

<!-- Schema changes, new storage, migration needs. "None" if stateless. -->

## Error Handling

<!-- How errors propagate, what the user sees, what gets logged. -->

## Security Considerations

<!-- Auth, secrets, input validation, information disclosure. -->

## Performance Considerations

<!-- Latency, throughput, resource usage. -->

## Tradeoffs

<!-- Decisions with alternatives considered and rationale. -->

## Risks

<!-- Design-level risks beyond what spec covers. -->

## Design Items

<!-- Each design element gets a stable ID with explicit requirement mapping. -->

- **DES-1**: <design decision or architecture element>
  - Satisfies: REQ-1
  - Covers: AC-1.1, AC-1.2

- **DES-2**: <design decision or architecture element>
  - Satisfies: REQ-2
  - Covers: AC-2.1

## Quality Gates

### Gate 2: Design Gate

This design passes when:
- [ ] Every REQ-* from spec.md is addressed by at least one DES-*
- [ ] Every AC-* from spec.md is covered by at least one DES-*
- [ ] Risks and tradeoffs are documented
- [ ] No DES-* exists without a linked REQ-*

**Harness check command:**
```bash
# Verify design-to-spec coverage
spec_reqs=$(grep -oP 'REQ-\d+' docs/features/<feature-id>/spec.md | sort -u)
design_reqs=$(grep -oP 'REQ-\d+' docs/features/<feature-id>/design.md | sort -u)
comm -23 <(echo "$spec_reqs") <(echo "$design_reqs")  # should be empty
```

## Requirement Mapping

| Design Item | Satisfies | Covers |
|-------------|-----------|--------|
| DES-1 | REQ-1 | AC-1.1, AC-1.2 |
| DES-2 | REQ-2 | AC-2.1 |
