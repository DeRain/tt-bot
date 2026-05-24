---
title: "Mutation Testing Integration"
feature_id: "mutation-testing"
status: in-review
owner: TODO
source_files: []
last_updated: 2026-05-24
---

# Mutation Testing Integration — Specification

## Overview

Integrate gomutants mutation testing into the tt-bot quality pipeline to catch weak tests that code coverage misses, providing a precise quality signal for every PR.

## Problem Statement

The project enforces high code coverage (>=80% per package), but coverage is a weak quality signal — 100% coverage can pass while important code paths go untested. Mutation testing fills this gap by injecting faults and checking whether tests detect them. Without it, the team ships with false confidence: a test may assert nothing while covering everything. Adding gomutants as a parallel CI step (Phase 1: informational, Phase 2: blocking) surfaces mutation efficacy per package, raises the bar for test quality, and prevents weakening of the test suite.

## Goals

- Add mutation testing as an automated PR gate (informational in Phase 1, blocking in Phase 2)
- Define and enforce per-package mutation efficacy thresholds
- Provide structured JSON reports for surviving mutant review
- Support equivalent mutant suppression with mandatory reason and expiry
- Establish baseline efficacy measurements for all target packages

## Non-Goals

- Running mutation tests on integration or E2E tests
- Running mutation tests on `cmd/bot` (thin wiring only, no domain logic)
- Building or maintaining stateful test infrastructure (gomutants runs stateless on unit tests)
- Replacing coverage thresholds with mutation scores (both signals used)
- Performance optimization of the mutation run beyond package-level parallelization
- Third-party CI service integration beyond GitHub Actions

## Scope

This feature covers: adding gomutants as a dev dependency, creating the orchestration to run per-package mutation tests, adding a `make mutation-test` target, adding a mutation job to the CI workflow, generating structured JSON survivorship reports, implementing `// gomutants:disable` suppression directives, documenting per-package floors, and establishing the baseline run. It does not cover redesigning tests to improve scores — floors are set to current baseline minus a safety margin.

## Requirements

- **REQ-1**: Mutation tests MUST run on every PR. Phase 1 (informational) posts efficacy metrics without blocking. Phase 2 (blocking) fails the PR if thresholds are unmet.
- **REQ-2**: PR-scoped mutation efficacy — the proportion of mutants on lines changed by the PR that are killed — MUST be >= 0.80 in Phase 2.
- **REQ-3**: Full-tree mutation efficacy MUST meet per-package floors: `qbt` >= 0.60, `bot` >= 0.60, `formatter` >= 0.70, `poller` >= 0.45, `config` >= 0.55.
- **REQ-4**: Surviving mutants MUST be reviewable via a structured JSON report containing file path, line number, mutator type, and surrounding source context.
- **REQ-5**: Equivalent mutants (semantically identical to original) MUST be suppressable via `// gomutants:disable` directives. Each directive MUST include a mandatory reason string and an expiry date.
- **REQ-6**: The mutation CI job MUST run in parallel to the integration test job. A mutation failure MUST NOT block the integration job, and vice versa.
- **REQ-7**: `make test` MUST NOT invoke gomutants. A separate `make mutation-test` target MUST be added to the Makefile.
- **REQ-8**: The addition of the mutation testing target MUST NOT break existing tests, coverage thresholds (>=80%), or architecture rules (arch-check passes).

## Acceptance Criteria

- **AC-1.1**: A PR against `main` triggers a CI workflow step that invokes `make mutation-test`.
- **AC-1.2** (Phase 1): The mutation step posts a PR comment with efficacy metrics but the PR can merge even if thresholds are unmet.
- **AC-1.3** (Phase 2): The mutation step fails the CI check when efficacy thresholds are unmet.
- **AC-2.1**: PR-scoped efficacy >= 0.80 produces a passing CI result (Phase 2).
- **AC-2.2**: PR-scoped efficacy < 0.80 produces a failing CI result (Phase 2).
- **AC-3.1**: Every package listed in REQ-3 achieves at least its floor efficacy when `make mutation-test` runs against the full tree.
- **AC-3.2**: The mutation run output includes a per-package efficacy breakdown.
- **AC-4.1**: The JSON report file contains an array of survivor entries, each with `file`, `line`, `mutator`, and `context` fields.
- **AC-4.2**: The JSON report is valid JSON and can be parsed with `jq` or equivalent tooling.
- **AC-5.1**: Inserting `// gomutants:disable reason="equivalent mutant" expires="2027-01-01"` before a mutant location suppresses it from the survivor count.
- **AC-5.2**: A `// gomutants:disable` directive missing `reason` or `expires` produces a non-fatal warning during the mutation run.
- **AC-5.3**: An expired `// gomutants:disable` directive (expires in the past) is reported as a warning during the mutation run.
- **AC-6.1**: The CI workflow definition has two parallel jobs (`integration` and `mutation`) at the same level, both depending on `gate` but not on each other.
- **AC-6.2**: A mutation step failure does not cancel or skip the integration job.
- **AC-7.1**: Running `make test` completes without executing gomutants.
- **AC-7.2**: Running `make mutation-test` executes gomutants against at least the packages listed in REQ-3.
- **AC-8.1**: `go test ./... -short -cover` reports >=80% coverage after all mutation-related files are added.
- **AC-8.2**: `make arch-check` exits 0 after all mutation-related files are added.

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
grep -c "^- \*\*REQ-" docs/features/mutation-testing/spec.md  # count requirements
grep -c "^- \*\*AC-"  docs/features/mutation-testing/spec.md  # count acceptance criteria
grep -c "TODO:"        docs/features/mutation-testing/spec.md  # should be 0 for approved
```

## Risks

- **HIGH**: gomutants is a young tool (v0.3.0, single maintainer) — if upstream development stalls or bugs appear, the feature may need to migrate to an alternative (go-mutesting) or maintain a fork. Mitigated by keeping the integration layer thin (one shell script, one Make target) and using standard Go build constraints.
- **MEDIUM**: Equivalent mutants (mutations that produce identical behavior) lower efficacy scores and require manual review. Mitigated by the suppression directive mechanism with expiry, ensuring survivors are reviewed periodically.
- **MEDIUM**: Mutation testing multiplies CI runtime (each mutation = one build + one test run). Full-tree runs could take 10x longer than unit tests. Mitigated by restricting per-package runs (not whole-tree), running mutations in parallel with integration tests, and exploring incremental (PR-scoped) mutation in Phase 2.
- **LOW**: Developers may ignore Phase 1 informational output. Mitigated by a clear Phase 1 to Phase 2 migration plan advertised in advance, and a PR comment summarizing delta against baseline.

## Open Questions

1. Should per-package floors be defined in a config file or hardcoded in the Makefile and script? OPEN — a config file is preferred for discoverability but adds parsing complexity.
2. Should the Phase 1 to Phase 2 transition be time-based (e.g., after 1 month) or metric-based (e.g., after 3 consecutive stable baseline runs)? OPEN — time-based is simpler to communicate.
3. Should PR-scoped efficacy use `git diff` to select only mutants on changed lines, or is whole-package efficacy sufficient? OPEN — PR-scoped is harder to implement but gives faster, more relevant feedback.
4. Is the Go 1.26.1 toolchain fully compatible with gomutants' build system? OPEN — needs a spike to verify `go vet`, build tag, and module interaction.
