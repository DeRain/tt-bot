---
title: "Set Bot Commands on Startup — Traceability Matrix"
feature_id: "set-commands"
status: complete
last_updated: 2026-03-15
---

# Set Bot Commands on Startup — Traceability Matrix

## Forward Traceability (Requirement → Verification)

| Requirement | Acceptance Criteria | Design | Plan Tasks | Implementation Evidence | Verification | Status |
|-------------|---------------------|--------|------------|-------------------------|--------------|--------|
| REQ-1 | AC-1.1, AC-1.2 | DES-2 | TASK-1, TASK-4 | `internal/bot/commands.go` (`RegisterCommands`) | TEST-1, TEST-2, TEST-5 | Complete |
| REQ-2 | AC-2.1, AC-2.2 | DES-1, DES-4 | TASK-1, TASK-3 | `internal/bot/commands.go` (`BotCommands`, `HelpText`), `internal/bot/handler.go` (`helpText`) | TEST-4 | Complete |
| REQ-3 | AC-3.1 | DES-3 | TASK-2 | `cmd/bot/main.go` (startup step 2a) | TEST-3, CHECK-1 | Complete |

## Backward Traceability (Code → Requirement)

| Source File | Functions/Types | Traces To | Via |
|-------------|----------------|-----------|-----|
| `internal/bot/commands.go` | `CommandDef`, `BotCommands`, `RegisterCommands`, `HelpText` | REQ-1, REQ-2 | TASK-1, DES-1, DES-2 |
| `cmd/bot/main.go` | startup step 2a | REQ-3 | TASK-2, DES-3 |
| `internal/bot/handler.go` | `helpText` (var, generated from `HelpText()`) | REQ-2 | TASK-3, DES-4 |

## Coverage Summary

| Metric | Count | Covered | Gaps |
|--------|-------|---------|------|
| Requirements | 3 | 3 | 0 |
| Acceptance Criteria | 5 | 5 | 0 |
| Design Items | 4 | 4 | 0 |
| Plan Tasks | 4 | 4 | 0 |
| Verification Items | 6 | 6 | 0 |

## Rules

- No REQ-* may exist without at least one linked DES-*.
- No DES-* may exist without at least one linked TASK-*.
- No TASK-* may exist without at least one linked verification item.
- No AC-* may remain unverified.
- Status values: Complete | Partial | Blocked | Missing | N/A

## Harness Validation

```bash
# Count untraced requirements (should be 0)
grep "| TODO |" docs/features/set-commands/traceability.md | wc -l

# Count missing verification (should be 0)
grep "| Missing |" docs/features/set-commands/traceability.md | wc -l
```
