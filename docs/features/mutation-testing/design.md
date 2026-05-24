---
title: "Mutation Testing Integration — Design"
feature_id: "mutation-testing"
status: in-review
depends_on_spec: "docs/features/mutation-testing/spec.md"
last_updated: 2026-05-24
---

# Mutation Testing Integration — Design

## Overview

Integrate [gomutants](https://github.com/szhekpisov/gomutants) as a mutation testing tool into the tt-bot quality pipeline. The spec (REQ-1 through REQ-8) calls for mutation testing as a PR gate. The implementation uses a grandfathering model: existing code runs against a 0% efficacy threshold (baseline measurement only, never blocks), while new or changed code in PRs must kill 100% of its mutants. A unified `make gate-all` target includes the PR-scoped mutation run, and a parallel CI job enforces it.

## Architecture

### Grandfathering Model

Rather than a phased rollout (informational then blocking), the implementation distinguishes between two run modes:

- **Full-tree baseline** (`make mutation-test`): runs with `--threshold-efficacy 0` across all internal packages. This never fails. It measures the existing mutation landscape and surfaces survivors for manual review. Existing code is grandfathered in — no minimum efficacy is required.
- **PR-scoped gate** (`make mutation-test-pr`): runs with `--threshold-efficacy 100` and `--changed-since origin/main`, scoped to lines changed in the current branch. New or modified code must kill all its mutants. Any survivor causes exit code 10 and blocks the CI check.

This avoids the complexity of two distinct deployment phases. Old code gets a free pass; new code has a zero-tolerance standard. The baseline run provides data for teams to improve test coverage over time without blocking velocity.

### CI Architecture (Fan-Out from Gate)

```
                    ┌─────────────────┐
                    │     gate job    │
                    │  build / lint   │
                    │  test / arch    │
                    │  coverage-check │
                    └────────┬────────┘
                             │ passes
                             ▼
              ┌──────────────────────────┐
              │      fan-out (parallel)  │
              ├────────────┬─────────────┤
              │            │             │
              ▼            ▼             ▼
     ┌────────────┐ ┌───────────┐ ┌───────────┐
     │ integration│ │ mutation  │ │  future   │
     │  (Docker)  │ │ (gomutants│ │  jobs     │
     │            │ │ PR-scoped)│ │           │
     └────────────┘ └───────────┘ └───────────┘
```

Both `integration` and `mutation` jobs specify `needs: gate`. The mutation job is independent of Docker and does not depend on integration. Results appear as separate GitHub status checks. The mutation job runs `make mutation-test-pr`, which blocks on any surviving mutant in changed code.

### Unified Gate Target

`make gate-all` includes `mutation-test-pr` alongside build, lint, test, coverage check, and architecture check. This means the full quality gate enforces mutation coverage on new code. There is no separate gating path — the same command developers run locally matches what CI runs.

```
gate-all: build lint test check-coverage arch-check mutation-test-pr
```

### gomutants CLI Integration

The tool is installed via `go install` with a pinned version in CI. It uses a `.gomutants.yml` config file at the repo root for shared settings between local and CI runs. The `--changed-since origin/main` flag scopes analysis to lines changed in the PR for fast feedback. No wrapper scripts are needed — gomutants' native exit codes (0 for pass, 10 for below-threshold) drive CI pass/fail directly.

## Data Flow

### PR-Scoped Run (Gate Path)

1. Developer pushes to a branch — GitHub triggers the workflow
2. `gate` job runs: checkout → build → lint → unit tests → arch check → coverage check
3. `gate` passes → `mutation` and `integration` jobs start in parallel
4. Mutation job: checkout with `fetch-depth: 0` (needed for `--changed-since`)
5. Install gomutants: `go install github.com/szhekpisov/gomutants@v0.3.0`
6. Run: `gomutants --config .gomutants.yml --changed-since origin/main --threshold-efficacy 100 ./internal/...`
7. gomutants produces `mutation-report.json` with per-mutant outcomes (KILLED, LIVED, NOT_COVERED, TIMED_OUT, NOT_VIABLE)
8. Exit code 0 → pass. Exit code 10 → fail (surviving mutants on changed lines). Exit code 1 → internal error, investigate
9. `mutation-report.json` uploaded as a CI artifact for click-through review
10. GitHub status check shows pass/fail for the mutation job

### Full-Tree Run (Manual, Baseline)

1. Run via `make mutation-test`
2. No `--changed-since` filter — all packages scanned
3. Uses `--threshold-efficacy 0` from `.gomutants.yml` — never fails
4. Cache (`.gomutants-cache.json`) enables incremental analysis on repeated runs
5. Useful for manual quality assessment and tracking baseline trends over time

## Interfaces

### gomutants CLI

Installed via `go install github.com/szhekpisov/gomutants@v0.3.0`. Key invocations:

```bash
# PR-scoped: mutants on changed lines only, 100% threshold (gate)
gomutants --config .gomutants.yml \
  --changed-since origin/main \
  --threshold-efficacy 100 ./internal/...

# Full tree: all packages, baseline measurement only (0% threshold from config)
gomutants --config .gomutants.yml ./internal/...

# Dry run: preview without executing tests
gomutants --config .gomutants.yml --dry-run ./internal/...
```

### .gomutants.yml Configuration

Config file at repo root. Shared between CI and local runs.

```yaml
# .gomutants.yml
workers: 4
timeout-coefficient: 10
threshold-efficacy: 0
cache: .gomutants-cache.json
checkpoint-interval: 10s
output: mutation-report.json

exclude-files:
  - vendor/.*
  - internal/.*/.*_test\.go

exclude-packages:
  - github.com/home/tt-bot/cmd/bot
```

Notes:
- `threshold-efficacy: 0` is the baseline for full-tree runs. The PR-scoped gate overrides this via CLI flag (`--threshold-efficacy 100`).
- `workers: 4` provides parallel mutant execution, scaled to typical CI runner cores.
- `exclude-packages` skips `cmd/bot` (composition root, minimal logic — low-value mutation targets).
- `exclude-files` prevents mutating test files themselves.

## Data/Storage Impact

- `mutation-report.json` — ephemeral CI artifact, not committed
- `.gomutants-cache.json` — local cache file, gitignored
- No persistent storage or schema changes
- No impact on application state (mutation testing is a CI-only concern)

## Error Handling

### gomutants Exit Codes

| Exit Code | Meaning | CI Interpretation |
|-----------|---------|-------------------|
| 0 | Thresholds met (or thresholds disabled at 0%) | Pass |
| 1 | Internal error (bad config, parse failure, no Go module) | Fail — unexpected, investigate |
| 10 | Efficacy below `--threshold-efficacy` | Fail — surviving mutants on changed lines |

No wrapper scripts interpret or remap exit codes. gomutants' native exit codes flow directly through Make to CI. Exit code 10 from `make mutation-test-pr` causes the CI job to fail immediately. Exit code 10 from `make mutation-test` (full-tree, 0% threshold) never occurs because the threshold is zero.

### Inline Directive Errors

`// gomutants:disable` directives are handled entirely by gomutants' built-in parser. gomutants reports warnings for malformed or semantically questionable directives to stderr. There is no separate directive guardrails script — gomutants' own validation is sufficient. The project does not enforce mandatory `reason` or `expires` fields beyond what gomutants provides natively.

## Security Considerations

- No secrets in `.gomutants.yml` — configuration is pure tool settings
- gomutants installation is version-pinned (`@v0.3.0`), not `@latest`, preventing supply-chain surprises
- `go install` fetches from the Go module proxy — standard Go security model applies
- Mutation testing never touches production data, Telegram credentials, or qBittorrent
- JSON report is an internal CI artifact with no sensitive information
- Inline directives are in committed source code — no dynamic allowlist loading

## Performance Considerations

### PR-Scoped Run (<10 minutes target)

- `--changed-since origin/main` limits mutant generation to lines changed in the PR
- Most PRs touch 10-50 lines, producing 20-200 mutants
- The gate job runs in serial before mutation: gate (2-3 min) + mutation (2-8 min) = 4-11 min total from push to mutation result
- Cache (`.gomutants-cache.json`) speeds up subsequent runs on the same branch

### Full-Tree Run (<30 minutes target)

- No `--changed-since` filter — all ~3000 lines of `internal/` are scanned
- Approximately 500-1500 mutants expected across 5 packages
- Expected duration: 15-25 minutes with `workers: 4`
- Cache hit rate improves on subsequent runs: ~80% of mutants skipped if code unchanged

### Caching Strategy

- `.gomutants-cache.json` stores per-mutant outcomes keyed by content hashes of the production file and covering test files
- On subsequent runs, mutants whose source and covering tests are byte-identical are skipped
- Gitignored to avoid committed cache drift
- Cache is invalidated automatically when source files change
- First run on a fresh checkout is always cache-miss (cold start)

## Tradeoffs

| Decision | Alternative | Rationale |
|----------|-------------|-----------|
| gomutants over ooze | ooze (JSON output, diff-scoping) | gomutants has native `--changed-since` for PR-scoped runs, JSON report output, active maintenance, and inline directive support. ooze would require custom diff-scoping logic and lacks directive support. |
| Inline `// gomutants:disable` directives over external YAML allowlist | External `mutation-allowlist.yml` | gomutants natively supports inline directives. Co-locating the directive with the disabled code makes review easier: the reviewer sees both the code and the justification in context. External allowlists drift from the code and require cross-file coordination. |
| Grandfathering model over two-phase rollout | Informational then blocking rollout | Without baseline data, a blocking threshold would surprise developers on the first run. Grandfathering achieves the same goal with less process: old code passes as-is, new code must be perfect. No need to schedule a Phase 1 to Phase 2 transition. |
| Unified gate-all over separate target | Separate target excluded from gate-all | Including `mutation-test-pr` in `gate-all` ensures the mutation gate is never skipped locally or in CI. No separate mental model or workflow. The tradeoff is slightly longer local gate-all runs, but PR-scoped runs are fast enough (2-8 min). |
| Single global PR threshold (100%) over per-package floors | Per-package floors via post-processing script | A single 100% threshold on changed lines is simpler to reason about and requires no script maintenance. Per-package floors would add a bash script that parses JSON and requires updates on every reorganization. The 100% bar is unambiguous: all new mutants on new code must be killed. |
| gomutants native directives over custom expiry enforcement | Mandatory reason + expiry + expiration script | gomutants' built-in `// gomutants:disable` handling is sufficient for equivalent mutant suppression. A custom guardrails script with expiry enforcement added maintenance overhead without proportional benefit, given that the project is small and directive count is expected to stay low (<10). |
| Separate CI job (not merged into gate) | Add mutation to the gate job | Mutation testing is 2-8 min minimum. Adding it to gate would push total gate time to 5-12 min, slowing the fast-feedback loop. A parallel job with `needs: gate` keeps gate fast while still running mutation before merge (it runs concurrently with integration). |
| Exclude `cmd/bot` from mutation testing | Include everything | `cmd/bot` is the composition root — it wires dependencies but has minimal logic. Mutation testing it yields low-value mutants (config parsing, main function wiring). Excluding it keeps the mutant count focused on `internal/` packages with business logic. |

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| gomutants v0.3.0 API changes before 1.0 | Medium | Medium | Pin to v0.3.0; test upgrades in a dedicated PR |
| `--changed-since` misses cross-package mutants | Low | Medium | Full-tree baseline run surfaces cross-package survivors the PR scope misses; CI does not gate on baseline, so these are review items, not blockers |
| gomutants produces false-positive NOT_VIABLE on complex generics | Low | Low | The project uses Go 1.26 with minimal generics; `NOT_VIABLE` mutants are excluded from efficacy calculation by gomutants |
| Inline directives create merge conflicts in large PRs | Low | Low | Directives are single-line comments attached to specific code; conflicts are resolved like any other comment conflict |
| 100% threshold on PR runs is too strict for some changes | Low | Medium | Team can use `// gomutants:disable` with a reason for genuinely equivalent mutants; the gate is per-PR, not per-line, so only mutants on changed lines count |

## Design Items

- **DES-1**: gomutants as the mutation testing tool selected over ooze
  - Satisfies: REQ-2, REQ-4
  - Covers: AC-2.1, AC-2.2, AC-4.1, AC-4.2

- **DES-2**: Grandfathering model — full-tree baseline at 0% threshold, PR-scoped gate at 100% threshold
  - Satisfies: REQ-1, REQ-2, REQ-3
  - Covers: AC-1.1, AC-1.3, AC-2.1, AC-2.2, AC-3.1, AC-3.2

- **DES-3**: Unified `gate-all` target includes `mutation-test-pr` alongside build/lint/test/check-coverage/arch-check
  - Satisfies: REQ-7
  - Covers: AC-7.1, AC-7.2

- **DES-4**: JSON report output (`--output mutation-report.json`) for artifact upload and review
  - Satisfies: REQ-4
  - Covers: AC-4.1, AC-4.2

- **DES-5**: Inline `// gomutants:disable` directives for equivalent mutant suppression, handled natively by gomutants
  - Satisfies: REQ-5
  - Covers: AC-5.1, AC-5.2, AC-5.3

- **DES-6**: CI job parallel to integration with `needs: gate` fan-out
  - Satisfies: REQ-6
  - Covers: AC-6.1, AC-6.2

- **DES-7**: Exclusion of `cmd/bot` via `.gomutants.yml` `exclude-packages`
  - Satisfies: REQ-8
  - Covers: AC-8.1, AC-8.2

- **DES-8**: Version-pinned gomutants installation (`go install ...@v0.3.0`)
  - Satisfies: REQ-8
  - Covers: AC-8.3

- **DES-9**: Separate Makefile targets (`mutation-test`, `mutation-test-pr`, `mutation-test-dry`)
  - Satisfies: REQ-7
  - Covers: AC-7.1, AC-7.2

- **DES-10**: `.gomutants.yml` shared configuration at repo root with workers, cache, exclusions
  - Satisfies: REQ-7, REQ-8
  - Covers: AC-7.2, AC-8.1

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
spec_reqs=$(grep -oP 'REQ-\d+' docs/features/mutation-testing/spec.md | sort -u)
design_reqs=$(grep -oP 'REQ-\d+' docs/features/mutation-testing/design.md | sort -u)
comm -23 <(echo "$spec_reqs") <(echo "$design_reqs")  # should be empty
```

## Requirement Mapping

| Design Item | Satisfies | Covers |
|-------------|-----------|--------|
| DES-1 | REQ-2, REQ-4 | AC-2.1, AC-2.2, AC-4.1, AC-4.2 |
| DES-2 | REQ-1, REQ-2, REQ-3 | AC-1.1, AC-1.3, AC-2.1, AC-2.2, AC-3.1, AC-3.2 |
| DES-3 | REQ-7 | AC-7.1, AC-7.2 |
| DES-4 | REQ-4 | AC-4.1, AC-4.2 |
| DES-5 | REQ-5 | AC-5.1, AC-5.2, AC-5.3 |
| DES-6 | REQ-6 | AC-6.1, AC-6.2 |
| DES-7 | REQ-8 | AC-8.1, AC-8.2 |
| DES-8 | REQ-8 | AC-8.3 |
| DES-9 | REQ-7 | AC-7.1, AC-7.2 |
| DES-10 | REQ-7, REQ-8 | AC-7.2, AC-8.1 |
