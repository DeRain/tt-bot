---
title: "Set Bot Commands on Startup — Plan"
feature_id: "set-commands"
status: implemented
depends_on_design: "docs/features/set-commands/design.md"
last_updated: 2026-03-15
---

# Set Bot Commands on Startup — Plan

## Overview

Four sequential tasks: define command data, implement registration, wire startup call, refactor help text. TDD approach — write tests first for each task, then implement.

## Preconditions

- `tgbotapi` v5.5.1 supports `SetMyCommandsConfig` and `NewSetMyCommands` (verified)
- `Sender` interface already has `Request()` method
- Existing unit test infrastructure with mock `Sender`

## Task Sequence

- **TASK-1**: Define `BotCommands` slice and `RegisterCommands` function in `internal/bot/commands.go`
  - Derived from: DES-1, DES-2
  - Implements: REQ-1, REQ-2
  - Impacts: `internal/bot/commands.go` (new file)
  - Verification: TEST-1, TEST-2
  - Gate: 4

- **TASK-2**: Wire `RegisterCommands` call in `cmd/bot/main.go` startup with fail-open error handling
  - Derived from: DES-3
  - Implements: REQ-3
  - Impacts: `cmd/bot/main.go`
  - Verification: TEST-3, CHECK-1
  - Gate: 4

- **TASK-3**: Refactor help text in `handler.go` to generate from `BotCommands` slice
  - Derived from: DES-4
  - Implements: REQ-2
  - Impacts: `internal/bot/handler.go`
  - Verification: TEST-4
  - Gate: 4

- **TASK-4**: Write integration test that verifies `setMyCommands` against real Telegram API
  - Derived from: DES-2
  - Implements: REQ-1
  - Impacts: `internal/bot/commands_integration_test.go` (new file)
  - Verification: TEST-5
  - Gate: 5

## Dependencies

```
TASK-1 → TASK-2 (RegisterCommands must exist before wiring)
TASK-1 → TASK-3 (BotCommands must exist before help refactor)
TASK-1 → TASK-4 (RegisterCommands must exist before integration test)
TASK-2 and TASK-3 are independent of each other
```

## Affected Files

| File | Action | Tasks |
|------|--------|-------|
| `internal/bot/commands.go` | Create | TASK-1 |
| `internal/bot/commands_test.go` | Create | TASK-1 |
| `internal/bot/commands_integration_test.go` | Create | TASK-4 |
| `cmd/bot/main.go` | Modify | TASK-2 |
| `internal/bot/handler.go` | Modify | TASK-3 |
| `internal/bot/handler_test.go` | Modify | TASK-3 |

## Rollout Notes

- No migration needed — stateless feature
- No new env vars required
- Command registration happens on every startup automatically
- Backward compatible — if API call fails, bot works as before

## Quality Gates

### Gate 3: Plan Gate

This plan passes when:
- [x] Every TASK-* maps to at least one DES-* and REQ-*
- [x] Task sequencing is coherent (dependencies respected)
- [x] Every TASK-* has a verification target (TEST-* or CHECK-*)
- [x] No TASK-* exists without implementation evidence path
- [x] Affected files are listed

**Harness check command:**
```bash
design_items=$(grep -oP 'DES-\d+' docs/features/set-commands/design.md | sort -u)
plan_items=$(grep -oP 'DES-\d+' docs/features/set-commands/plan.md | sort -u)
comm -23 <(echo "$design_items") <(echo "$plan_items")  # should be empty

grep "^- \*\*TASK-" docs/features/set-commands/plan.md | wc -l  # task count
grep "Verification:" docs/features/set-commands/plan.md | wc -l  # should match
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

See `verification.md` for full test definitions:
- TEST-1: Unit test `RegisterCommands` builds correct config
- TEST-2: Unit test `RegisterCommands` calls `Sender.Request()`
- TEST-3: Unit test fail-open behavior (error logged, no panic)
- TEST-4: Unit test help text generated from `BotCommands`
- TEST-5: Integration test `setMyCommands` against real Telegram API
- CHECK-1: Manual check — bot starts and commands appear in Telegram client

## Blockers

None.
