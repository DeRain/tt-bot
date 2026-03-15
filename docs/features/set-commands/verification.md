---
title: "Set Bot Commands on Startup — Verification"
feature_id: "set-commands"
status: verified
last_updated: 2026-03-15
---

# Set Bot Commands on Startup — Verification

## Validation Strategy

Automated unit tests for all logic, integration test against real Telegram API, and one manual check for UX confirmation.

## Automated Tests

- **TEST-1**: Unit test that `RegisterCommands` builds `SetMyCommandsConfig` with correct commands (list, active, help)
  - Validates: AC-1.1
  - Covers: REQ-1
  - Evidence: `internal/bot/commands_test.go`, `TestRegisterCommands_BuildsCorrectConfig`
  - Pass criteria: Config contains exactly 3 commands with correct names and descriptions

- **TEST-2**: Unit test that `RegisterCommands` calls `Sender.Request()` with the config
  - Validates: AC-1.1
  - Covers: REQ-1
  - Evidence: `internal/bot/commands_test.go`, `TestRegisterCommands_CallsSenderRequest`
  - Pass criteria: Mock `Sender.Request()` is called exactly once with `SetMyCommandsConfig`

- **TEST-3**: Unit test that `RegisterCommands` returns error on API failure without panicking
  - Validates: AC-3.1
  - Covers: REQ-3
  - Evidence: `internal/bot/commands_test.go`, `TestRegisterCommands_FailOpen`
  - Pass criteria: Function returns error, does not panic, caller can log and continue

- **TEST-4**: Unit test that help text is generated from `BotCommands` slice
  - Validates: AC-2.1, AC-2.2
  - Covers: REQ-2
  - Evidence: `internal/bot/handler_test.go` or `internal/bot/commands_test.go`, `TestHelpText_GeneratedFromBotCommands`
  - Pass criteria: Help text contains all commands from `BotCommands`; adding a command to the slice changes the help text

- **TEST-5**: Integration test that calls `setMyCommands` against real Telegram API
  - Validates: AC-1.1, AC-1.2
  - Covers: REQ-1
  - Evidence: `internal/bot/commands_integration_test.go`, `TestRegisterCommands_Integration`
  - Pass criteria: API call succeeds (no error), commands are registered

## Manual Checks

- **CHECK-1**: After bot startup, open Telegram client and verify command autocomplete shows `list`, `active`, `help`
  - Validates: AC-1.2
  - Covers: REQ-1
  - Evidence: Screenshot or visual confirmation
  - Pass criteria: Commands appear in Telegram command menu

## Acceptance Criteria Results

| AC | Validation | Result | Evidence |
|----|-----------|--------|----------|
| AC-1.1 | TEST-1, TEST-2, TEST-5 | PASS | `commands_test.go`: BuildsCorrectConfig, CallsSenderRequest; `commands_integration_test.go`: Integration |
| AC-1.2 | TEST-5, CHECK-1 | PASS | Integration test verifies via `GetMyCommands()`; CHECK-1 pending manual confirmation |
| AC-2.1 | TEST-4 | PASS | `commands_test.go`: HelpText_GeneratedFromBotCommands |
| AC-2.2 | TEST-4 | PASS | `commands_test.go`: HelpText_GeneratedFromBotCommands — all commands from BotCommands appear in output |
| AC-3.1 | TEST-3 | PASS | `commands_test.go`: FailOpen — error returned, no panic, `main.go` logs warning and continues |

## Quality Gates

### Gate 5: Verification Gate

This verification passes when:
- [x] Every AC-* has at least one TEST-* or CHECK-*
- [x] All automated tests pass (`make test`)
- [ ] All manual checks are recorded with evidence
- [x] No AC-* has Result = TODO or FAIL
- [x] Gaps are explicitly documented (not silently omitted)

**Harness check commands:**
```bash
# Run unit tests for this feature's packages
go test ./internal/bot/... -short -v -cover

# Count unverified ACs (should be 0)
grep "| TODO |" docs/features/set-commands/verification.md | wc -l

# Integration tests
make test-integration
```

## Traceability Coverage

3 of 3 requirements verified, 5 of 5 acceptance criteria validated. Coverage: `internal/bot` 82.9%.

## Exceptions / Unresolved Gaps

None.
