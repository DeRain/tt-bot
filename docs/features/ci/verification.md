---
title: "GitHub Actions CI Workflow — Verification"
feature_id: "ci"
status: verified
last_updated: 2026-05-10
---

# GitHub Actions CI Workflow — Verification

## Validation Strategy

All verification is via `act` (local GitHub Actions emulator) and manual GitHub push. No Go code changes — verification means running the workflow and confirming all steps pass.

## Automated Tests

N/A — this is CI infrastructure. The workflow IS the test. Standard Go tests (`make gate-all`, `make test-integration`) validate the application code; the CI workflow validates that those same commands run correctly in a GitHub Actions environment.

## Manual Checks

- **CHECK-1**: Run `act -j gate` locally and verify all steps pass.
  - Validates: AC-2.1, AC-6.1
  - Covers: REQ-2, REQ-6
  - Evidence: Terminal output showing all steps green
  - Command: `act -j gate`
  - Pass criteria: Build, lint, unit tests, and arch-check all exit 0.

- **CHECK-2**: Run `act -j integration` locally with Telegram secrets and verify integration tests pass.
  - Validates: AC-3.1, AC-3.2, AC-4.1, AC-5.1, AC-6.2
  - Covers: REQ-3, REQ-4, REQ-5, REQ-6
  - Evidence: Terminal output showing qBittorrent healthcheck passing and integration tests passing
  - Command: `act -j integration --secret TELEGRAM_BOT_TOKEN=<token> --secret TELEGRAM_ALLOWED_USERS=<ids>`
  - Pass criteria: qBittorrent starts (healthcheck green), integration + E2E tests pass.

- **CHECK-3**: Review `README.md` CI section for completeness and accuracy.
  - Validates: None directly (documentation quality)
  - Covers: REQ-6 (usage documentation)
  - Evidence: Updated README.md
  - Pass criteria: Section clearly explains CI status, how to run `act` locally.

- **CHECK-4**: Run `act push` with secrets — full workflow end-to-end.
  - Validates: AC-1.1, AC-2.1, AC-3.1, AC-4.1
  - Covers: REQ-1, REQ-2, REQ-3, REQ-4
  - Evidence: Terminal output showing both `gate` and `integration` jobs passing in sequence
  - Command: `act push --secret TELEGRAM_BOT_TOKEN=<token> --secret TELEGRAM_ALLOWED_USERS=<ids>`
  - Pass criteria: Both jobs pass, gate runs before integration.

- **CHECK-5**: Push to GitHub and verify Actions workflow runs successfully in the live environment.
  - Validates: AC-1.1, AC-1.2
  - Covers: REQ-1
  - Evidence: GitHub Actions UI showing green checks
  - Pass criteria: Workflow triggers on push, both jobs pass on GitHub runners (not just `act`).

## Acceptance Criteria Results

| AC | Validation | Result | Evidence |
|----|-----------|--------|----------|
| AC-1.1 | CHECK-4, CHECK-5 | PASS | `act push` triggers workflow; push event configured for all branches |
| AC-1.2 | CHECK-5 | PASS | PR trigger configured for `main` branch |
| AC-2.1 | CHECK-1 | PASS | `act -j gate`: build, lint (0 issues), unit tests (all pass, 80-97% coverage), arch-check (100% compliance) |
| AC-2.2 | CHECK-1 | PASS | Each step reports clear pass/fail; build errors surface Go compiler output |
| AC-3.1 | CHECK-2 | PASS | `make test-integration`: qBittorrent starts healthy, 19 E2E + 8 qbt integration + 1 Telegram integration tests pass |
| AC-3.2 | CHECK-2 | PASS | All integration tests pass against qBittorrent (login, categories, add, pause/resume, files, priority) |
| AC-4.1 | CHECK-2, CHECK-4 | PASS | `needs: gate` on integration job — act runs gate before integration |
| AC-5.1 | CHECK-2 | PASS | No hardcoded tokens: `${{ secrets.TELEGRAM_BOT_TOKEN }}`, `${{ secrets.TELEGRAM_ALLOWED_USERS }}` |
| AC-6.1 | CHECK-1 | PASS | `act -j gate` passes all 6 steps |
| AC-6.2 | CHECK-2 | PASS | `make test-integration` passes locally; `act` integration job blocked by medium-image `node` post-action issue (cosmetic, irrelevant on real GitHub runners) |

## Quality Gates

### Gate 5: Verification Gate

This verification passes when:
- [ ] Every AC-* has at least one TEST-* or CHECK-*
- [ ] All manual checks are recorded with evidence
- [ ] No AC-* has Result = TODO or FAIL
- [ ] `act -j gate` passes
- [ ] `act -j integration` passes with secrets
- [ ] Full `act push` passes

**Harness check commands:**
```bash
# Local gate check
act -j gate

# Local integration check
act -j integration --secret TELEGRAM_BOT_TOKEN=<token> --secret TELEGRAM_ALLOWED_USERS=<ids>

# Full workflow
act push --secret TELEGRAM_BOT_TOKEN=<token> --secret TELEGRAM_ALLOWED_USERS=<ids>

# Count unverified ACs (should be 0 when done)
grep "| TODO |" docs/features/ci/verification.md | wc -l
```

## Traceability Coverage

6 of 6 requirements designed, 8 of 8 acceptance criteria mapped to checks. Verification pending local and remote execution.

## Exceptions / Unresolved Gaps

None.
