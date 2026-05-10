---
title: "Architecture Governance (arch-go)"
feature_id: "arch-go"
status: implemented
owner: arch-go
source_files: []
last_updated: 2026-05-10
---

# Architecture Governance (arch-go) — Specification

## Overview

Integrate the [arch-go](https://github.com/arch-go/arch-go) architecture governance tool into the tt-bot project to encode the current layered dependency structure as machine-checkable rules. This prevents accidental introduction of circular dependencies, layer violations, or unauthorized coupling during future development.

## Problem Statement

The project has a clean layered architecture (cmd → internal/bot → internal/formatter, internal/poller → internal/qbt; config and qbt are stdlib-only leaves), but this structure exists only in documentation and convention. Nothing prevents a developer from accidentally importing a package against the intended layering — importing `cmd` from `internal`, importing `config` from `qbt`, or creating a circular dependency. Without automated enforcement, architecture drift accumulates silently.

## Goals

- Encode the current 6-package dependency structure as rules in `arch-go.yml`
- Run architecture checks via `make arch-check`, integrated into `make gate-all`
- Update quality gate documentation to include architecture verification
- Use descriptive rules only — match what the codebase IS, not an idealized future state

## Non-Goals

- Prescriptive rules (content rules, function rules, naming rules) — follow-up feature
- Programmatic `go test` integration — CLI-only for minimal dependency footprint
- Refactoring existing code to satisfy stricter rules
- CI workflow changes (already covered by `gate-all` in local workflows)
- Package coverage threshold below 90%

## Scope

This feature covers: creating `arch-go.yml` with 6 dependency rules, adding `arch-check` to the Makefile, wiring it into `gate-all`, and updating `docs/gates.md`, `docs/pr-checklist.md`, and the plan template. Out of scope: go.mod dependency addition, programmatic tests, content/function/naming rules.

## Requirements

- **REQ-1**: An `arch-go.yml` configuration file MUST exist at the repository root encoding the current 6-package dependency structure as dependency rules.
- **REQ-2**: A `make arch-check` target MUST run arch-go against the project and exit 0 when all rules pass.
- **REQ-3**: `make gate-all` MUST include `arch-check` in its target chain so architecture violations block the quality gate.
- **REQ-4**: Quality gate documentation (`docs/gates.md`, `docs/pr-checklist.md`, `docs/features/_templates/plan.md`) MUST reference architecture checks.
- **REQ-5**: The rules MUST be descriptive (match current state), not prescriptive. No content, function, or naming rules.
- **REQ-6**: `docker-compose.test.yml` MUST pass Telegram environment variables to the integration test container so E2E tests can register bot commands against the real Telegram API.

## Acceptance Criteria

- **AC-1.1**: `arch-go.yml` contains 6 `dependenciesRules` entries covering all packages: `**.cmd.**`, `**.internal.qbt`, `**.internal.config`, `**.internal.formatter`, `**.internal.poller`, `**.internal.bot`.
- **AC-1.2**: `arch-go describe` lists each of the 6 dependency rules with correct package patterns and constraints.
- **AC-1.3**: `arch-go` (or equivalent `go run`) exits 0 with 100% compliance and >=90% coverage against the current codebase.
- **AC-2.1**: `make arch-check` executes without requiring pre-installed tooling and exits 0.
- **AC-3.1**: `make gate-all` dependency chain includes `arch-check` (runs after lint, before completion).
- **AC-3.2**: `make gate-all` exits 0 with no architecture violations.
- **AC-4.1**: `docs/gates.md` Gate 4 (Implementation Gate) lists `make arch-check` as a pass criterion.
- **AC-4.2**: `docs/pr-checklist.md` Verification Evidence section includes an architecture rules checkbox.
- **AC-4.3**: `docs/features/_templates/plan.md` quality gates section references `make arch-check`.
- **AC-5.1**: `arch-go.yml` contains no `contentsRules`, `functionsRules`, or `namingRules` sections.
- **AC-6.1**: `docker-compose.test.yml` integration-tests service includes `TELEGRAM_BOT_TOKEN` and `TELEGRAM_ALLOWED_USERS` environment variables.

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
grep -c "^- \*\*REQ-" docs/features/arch-go/spec.md  # count requirements
grep -c "^- \*\*AC-"  docs/features/arch-go/spec.md  # count acceptance criteria
grep -c "TODO:"        docs/features/arch-go/spec.md  # should be 0 for approved
```

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| arch-go version breakage on `@latest` | Low | Medium | Pin version in Makefile if flaky; `go run` always fetches latest patch |
| Future package additions violate coverage threshold | Low | Low | Set coverage to 90%; new leaf packages covered by `**.internal.<name>` rules in follow-up |

## Open Questions

<!-- None -->
