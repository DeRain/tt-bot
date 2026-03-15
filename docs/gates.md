# Quality Gates

This document defines the validation gates that govern the docs-first feature workflow. Each gate is a prerequisite for the next phase of work. Gates are designed to be checkable by both humans and automated harness loops.

## Gate Sequence

```
Spec Gate → Design Gate → Plan Gate → Implementation Gate → Verification Gate
```

No gate may be skipped. Implementation MUST NOT begin until Gates 1-3 pass.

## Gate 1: Spec Gate

**Location**: `docs/features/<feature-id>/spec.md`

**Pass criteria:**
- [ ] All requirements have stable IDs (REQ-N)
- [ ] All acceptance criteria have stable IDs (AC-N.M) and are testable
- [ ] Scope and non-goals are defined
- [ ] No unresolved open questions block implementation
- [ ] At least one AC exists per REQ

**Harness commands:**
```bash
FEATURE="<feature-id>"
# Requirements exist
test $(grep -c "^\- \*\*REQ-" docs/features/$FEATURE/spec.md) -gt 0
# ACs exist
test $(grep -c "^\- \*\*AC-" docs/features/$FEATURE/spec.md) -gt 0
# No blocking TODOs
test $(grep -c "TODO:" docs/features/$FEATURE/spec.md) -eq 0
```

## Gate 2: Design Gate

**Location**: `docs/features/<feature-id>/design.md`

**Pass criteria:**
- [ ] Every REQ-* from spec is addressed by at least one DES-*
- [ ] Every AC-* from spec is covered by at least one DES-*
- [ ] Risks and tradeoffs are documented
- [ ] Requirement mapping table is complete
- [ ] No DES-* exists without a linked REQ-*

**Harness commands:**
```bash
FEATURE="<feature-id>"
SPEC="docs/features/$FEATURE/spec.md"
DESIGN="docs/features/$FEATURE/design.md"
# All spec REQs appear in design
spec_reqs=$(grep -oP 'REQ-\d+' "$SPEC" | sort -u)
design_reqs=$(grep -oP 'REQ-\d+' "$DESIGN" | sort -u)
missing=$(comm -23 <(echo "$spec_reqs") <(echo "$design_reqs"))
test -z "$missing" || echo "FAIL: missing REQs in design: $missing"
```

## Gate 3: Plan Gate

**Location**: `docs/features/<feature-id>/plan.md`

**Pass criteria:**
- [ ] Every DES-* from design is addressed by at least one TASK-*
- [ ] Every TASK-* references at least one REQ-*
- [ ] Every TASK-* has a verification target (TEST-* or CHECK-*)
- [ ] Task sequencing respects dependencies
- [ ] Affected files are listed

**Harness commands:**
```bash
FEATURE="<feature-id>"
PLAN="docs/features/$FEATURE/plan.md"
# Every TASK has a Verification line
tasks=$(grep -c "^\- \*\*TASK-" "$PLAN")
verifications=$(grep -c "Verification:" "$PLAN")
test "$tasks" -eq "$verifications" || echo "FAIL: $tasks tasks but $verifications verifications"
```

## Gate 4: Implementation Gate

**Triggered after**: code changes are complete.

**Pass criteria:**
- [ ] `make gate-all` passes (build + lint + unit tests)
- [ ] `make test-integration` passes (integration + E2E tests against real services)
- [ ] Changed files map to TASK-* and REQ-* in plan
- [ ] No untraced code changes exist
- [ ] Implementation evidence recorded in traceability.md
- [ ] No regressions in existing tests

**Harness commands:**
```bash
make gate-all
make test-integration
# Verify traceability is updated
FEATURE="<feature-id>"
test $(grep -c "| TODO |" docs/features/$FEATURE/traceability.md) -eq 0
```

## Gate 5: Verification Gate

**Triggered after**: implementation gate passes.

**Pass criteria:**
- [ ] Every AC-* has at least one TEST-* or CHECK-*
- [ ] All unit tests pass (`go test ./... -short`)
- [ ] All integration + E2E tests pass (`make test-integration`) — **MANDATORY, never skip or defer**
- [ ] All manual checks recorded with evidence
- [ ] No AC-* has Result = TODO or FAIL in verification.md
- [ ] Coverage meets 80% threshold

> **CRITICAL**: Unit tests with mocks cannot catch real API contract issues (e.g. renamed endpoints, changed response formats). An AC is NOT verified until integration tests confirm it against a real service. Never mark an AC as PASS based on unit tests alone when an integration test exists for it. See `decisions.md` in torrent-control for the qBittorrent v5 endpoint rename incident.

**Harness commands:**
```bash
FEATURE="<feature-id>"
# Unit tests pass with coverage
go test ./... -short -cover -coverprofile=coverage.out
# Integration + E2E tests against real services — MANDATORY
make test-integration
# No unverified ACs (must be 0 BEFORE marking feature complete)
test $(grep -c "| TODO |" docs/features/$FEATURE/verification.md) -eq 0
```

## Iterative Harness Loop Protocol

When an agent harness executes a feature plan:

1. **Pre-flight**: Verify Gates 1-3 pass before writing any code.
2. **Execute tasks** in dependency order from plan.md.
3. **After each TASK-***: run its verification target. If fail → fix → retry (max 3).
4. **After all tasks**: run Gate 4 — `make gate-all` AND `make test-integration`.
5. **Update traceability.md** with implementation evidence.
6. **Run Gate 5** — unit tests, integration tests, verify all ACs are PASS (not TODO).
7. **Report**: produce summary of gate results, gaps, and next actions.

> **Never defer integration tests.** If Docker is available, run `make test-integration` in the same session. Integration tests catch real API issues that unit tests with mocks cannot (endpoint renames, response format changes, auth behavior). An AC marked "PASS (unit), TODO (integration)" is NOT verified.

### Recovery Protocol

- **Build failure**: invoke `/go-build` skill (uses `go-build-resolver` agent internally). Fix in-place, re-run gate.
- **Test failure**: invoke `/go-test` skill (TDD for Go: table-driven tests, 80%+ coverage). Fix implementation, not tests (unless tests are wrong).
- **Lint failure**: fix lint issues, re-run `make lint`.
- **Code review**: invoke `/go-review` skill (uses `go-reviewer` agent internally) after implementation.
- **Max retries exceeded**: stop, report blocker, request human intervention.

### ECC Components per Gate

#### Gates 1-3 (Spec → Design → Plan)

| Component | Type | Purpose |
|-----------|------|---------|
| `/plan` | Command | Create spec/design/plan with traceability IDs |
| `/multi-plan` | Command | Decompose into parallel tasks across agents |
| `planner` | Agent | Feature implementation planning |
| `architect` | Agent | System design decisions for design.md |
| `search-first` | Skill | Research existing solutions before new work |

#### Gate 4 (Implementation)

| Component | Type | Purpose |
|-----------|------|---------|
| `/go-test` | Command | Go TDD: table-driven tests first, 80%+ coverage |
| `/tdd` | Command | Generic TDD: scaffold → test → implement |
| `/go-build` | Command | Fix Go build/vet/lint errors |
| `/go-review` | Command | Go code review with traceability check |
| `/code-review` | Command | General quality + security review |
| `/multi-execute` | Command | Execute parallel TASK-* via agents |
| `/checkpoint` | Command | Save state mid-harness for recovery |
| `tdd-guide` | Agent | Enforce write-tests-first methodology |
| `code-reviewer` | Agent | Quality + traceability review |
| `security-reviewer` | Agent | Security requirement verification |
| `go-reviewer` | Agent | Go-specific idiomatic review |
| `go-build-resolver` | Agent | Fix Go build errors minimally |
| `build-error-resolver` | Agent | Fix general build errors |
| `golang-patterns` | Skill | Idiomatic Go reference |
| `golang-testing` | Skill | Table-driven tests, subtests, benchmarks |
| `tdd-workflow` | Skill | TDD methodology: RED → GREEN → REFACTOR |
| `security-review` | Skill | Security checklist |
| `docker-patterns` | Skill | Docker/Compose patterns |

#### Gate 5 (Verification)

| Component | Type | Purpose |
|-----------|------|---------|
| `/verify` | Command | Run verification loop against AC-* |
| `/eval` | Command | Evaluate against acceptance criteria |
| `/test-coverage` | Command | Verify 80%+ coverage threshold |
| `verification-loop` | Skill | Continuous verification system |
| `eval-harness` | Skill | Evaluation framework for harness quality |

#### Post-Gate (Learning + Docs)

| Component | Type | Purpose |
|-----------|------|---------|
| `/learn-eval` | Command | Extract reusable patterns from session |
| `/update-docs` | Command | Sync traceability docs with code |
| `/update-codemaps` | Command | Update codebase maps |
| `doc-updater` | Agent | Documentation sync |
| `refactor-cleaner` | Agent | Dead code cleanup between features |
| `continuous-learning-v2` | Skill | Instinct-based session learning |
| `strategic-compact` | Skill | Context management in long sessions |

#### Harness Loop Support

| Component | Type | Purpose |
|-----------|------|---------|
| `/orchestrate` | Command | Multi-agent coordination across gates |
| `autonomous-loops` | Skill | Harness loop architecture patterns |
| `eval-harness` | Skill | Eval-driven development framework |
| `iterative-retrieval` | Skill | Progressive context refinement for subagents |
| `deployment-patterns` | Skill | CI/CD, rollout, health checks |

### Agent Model Routing

| Phase | Recommended Model | Reason |
|-------|-------------------|--------|
| Spec/Design/Plan review | Opus | Deep reasoning for requirements analysis |
| Implementation tasks | Sonnet | Fast, accurate coding |
| Gate checks | Haiku | Lightweight verification |
| Debugging failures | Sonnet | Good debugging with speed |
