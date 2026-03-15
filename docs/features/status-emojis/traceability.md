---
title: "Human-Readable Statuses with Emojis — Traceability Matrix"
feature_id: "status-emojis"
status: draft
last_updated: 2026-03-15
---

# Human-Readable Statuses with Emojis — Traceability Matrix

## Forward Traceability: Requirements → Design → Plan → Tests

| REQ   | DES             | TASK            | Verification    | AC Coverage                                    |
|-------|-----------------|-----------------|-----------------|------------------------------------------------|
| REQ-1 | DES-1, DES-2    | TASK-1, TASK-3  | TEST-1          | AC-1.1, AC-1.2, AC-1.3                         |
| REQ-2 | DES-2           | TASK-1, TASK-3  | TEST-1          | AC-2.1, AC-2.2                                 |
| REQ-3 | DES-3           | TASK-2, TASK-4  | TEST-2          | AC-3.1, AC-3.2                                 |
| REQ-4 | DES-1           | TASK-1, TASK-3  | TEST-1          | AC-4.1, AC-4.2                                 |

## Backward Traceability: Tests → Plan → Design → Requirements

| Verification | TASK            | DES          | REQ             | AC Validated                                   |
|--------------|-----------------|--------------|-----------------|------------------------------------------------|
| TEST-1       | TASK-1, TASK-3  | DES-1, DES-2 | REQ-1, REQ-2, REQ-4 | AC-1.1, AC-1.2, AC-1.3, AC-2.1, AC-2.2, AC-4.1, AC-4.2 |
| TEST-2       | TASK-2, TASK-4  | DES-3        | REQ-3           | AC-3.1, AC-3.2                                 |
| CHECK-1      | TASK-5          | DES-1, DES-2, DES-3 | REQ-1, REQ-2, REQ-3, REQ-4 | (overall build health) |

## Acceptance Criteria Coverage

| AC      | Description                                                                                 | Design Item | Test     | Status |
|---------|---------------------------------------------------------------------------------------------|-------------|----------|--------|
| AC-1.1  | `stalledUP` → `Seeding (stalled)`, raw string absent                                        | DES-1, DES-2 | TEST-1  | PASS   |
| AC-1.2  | `pausedDL` → `Paused (Downloading)`, raw string absent                                      | DES-1, DES-2 | TEST-1  | PASS   |
| AC-1.3  | All 19 documented states produce a non-empty label distinct from the raw state string        | DES-1, DES-2 | TEST-1  | PASS   |
| AC-2.1  | Each mapped state's label begins with its designated emoji character                         | DES-2       | TEST-1   | PASS   |
| AC-2.2  | Fallback output for an unmapped state begins with ❓                                         | DES-1, DES-2 | TEST-1  | PASS   |
| AC-3.1  | `FormatTorrentList` displays the mapped label, not the raw state                             | DES-3       | TEST-2   | PASS   |
| AC-3.2  | `FormatTorrentDetail` displays the mapped label on the `State:` line, not the raw state      | DES-3       | TEST-2   | PASS   |
| AC-4.1  | Empty string input returns a non-empty fallback prefixed with ❓ and does not panic          | DES-1       | TEST-1   | PASS   |
| AC-4.2  | Novel unrecognized string (e.g. `"newState"`) returns `❓ newState` and does not panic       | DES-1       | TEST-1   | PASS   |

## Implementation Evidence

| File                                         | Change Description                                                              | TASK    | Status |
|----------------------------------------------|---------------------------------------------------------------------------------|---------|--------|
| `internal/formatter/format.go`               | Add `stateLabels` map and `FormatState` function; update `FormatTorrentList` and `FormatTorrentDetail` | TASK-3, TASK-4 | DONE |
| `internal/formatter/format_test.go`          | Add `TestFormatState` table-driven tests; extend existing list/detail tests     | TASK-1, TASK-2 | DONE |
| `internal/bot/callback_test.go`              | Update state assertion to use mapped label `⬇️ Downloading`                    | TASK-4  | DONE |
| `docs/features/status-emojis/traceability.md` | Updated post-implementation                                                   | TASK-5  | DONE |
| `docs/features/status-emojis/verification.md` | Updated post-implementation                                                   | TASK-5  | DONE |

## Design Item Coverage

| DES   | Description                                                                              | REQ Satisfied      | AC Covered                                             | Status |
|-------|------------------------------------------------------------------------------------------|--------------------|--------------------------------------------------------|--------|
| DES-1 | `FormatState(state string) string` function in `internal/formatter`                      | REQ-1, REQ-4       | AC-1.1, AC-1.2, AC-1.3, AC-4.1, AC-4.2               | DONE   |
| DES-2 | Package-level `stateLabels map[string]string` with all 19 documented state mappings      | REQ-1, REQ-2       | AC-1.3, AC-2.1, AC-2.2                                | DONE   |
| DES-3 | Update `FormatTorrentList` and `FormatTorrentDetail` to call `FormatState(t.State)`      | REQ-3              | AC-3.1, AC-3.2                                        | DONE   |
