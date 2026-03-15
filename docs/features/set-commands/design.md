---
title: "Set Bot Commands on Startup — Design"
feature_id: "set-commands"
status: implemented
depends_on_spec: "docs/features/set-commands/spec.md"
last_updated: 2026-03-15
---

# Set Bot Commands on Startup — Design

## Overview

Define a package-level `BotCommands` slice in `internal/bot/` as the single source of truth for command metadata. Add a `RegisterCommands` function that calls `Sender.Request()` with `tgbotapi.NewSetMyCommands()`. Call it from `main.go` on startup. Refactor help text to generate from the same slice.

## Architecture

The feature adds one new file (`commands.go`) and modifies two existing files (`handler.go`, `main.go`):

```
cmd/bot/main.go                 → calls RegisterCommands() during startup
internal/bot/commands.go (new)  → BotCommands slice + RegisterCommands()
internal/bot/handler.go         → help text generated from BotCommands
```

## Data Flow

1. Bot starts in `main.go`
2. After creating `tgbotapi.BotAPI`, calls `bot.RegisterCommands(ctx, botAPI)`
3. `RegisterCommands` builds `tgbotapi.SetMyCommandsConfig` from `BotCommands` slice
4. Calls `Sender.Request()` to send `setMyCommands` to Telegram API
5. On success: logs confirmation, continues startup
6. On failure: logs warning, continues startup (fail-open)
7. When user sends `/help` or `/start`, help text is generated from the same `BotCommands` slice

## Interfaces

```go
// BotCommand defines a single bot command for registration and help text.
type BotCommand struct {
    Command     string
    Description string
}

// BotCommands is the single source of truth for all bot commands.
var BotCommands = []BotCommand{...}

// RegisterCommands registers bot commands with the Telegram API.
// Returns an error if the API call fails, but callers should treat this as non-fatal.
func RegisterCommands(sender Sender) error
```

`Sender.Request()` already exists and returns `(*tgbotapi.APIResponse, error)` — suitable for `setMyCommands` which returns a bool response.

## Data/Storage Impact

None. This feature is stateless.

## Error Handling

- `RegisterCommands` returns an error if the API call fails
- `main.go` logs the error at WARN level and continues startup
- No retry logic — the bot will re-register on next restart

## Security Considerations

- No new secrets or auth needed — uses existing bot token
- No user input involved in command registration
- Command definitions are hardcoded in source, not configurable at runtime

## Performance Considerations

- Single API call on startup — negligible overhead
- No runtime performance impact after startup

## Tradeoffs

### Using package-level slice vs constants
- **Chosen**: Package-level `var` slice — allows iteration for both registration and help text
- **Alternative**: String constants — would require duplication between registration and help
- **Rationale**: Slice enables single source of truth; var instead of const because Go doesn't support const slices

### Fail-open vs fail-closed
- **Chosen**: Fail-open (log warning, continue)
- **Alternative**: Fail-closed (exit on registration failure)
- **Rationale**: Command registration is a UX enhancement, not a functional requirement. Bot commands work regardless of registration.

## Risks

- **LOW**: `tgbotapi.BotAPI.Request()` signature change in future library versions — pinned at v5.5.1.

## Design Items

- **DES-1**: Define `BotCommands` slice in `internal/bot/commands.go` as the single source of truth for command names and descriptions.
  - Satisfies: REQ-2
  - Covers: AC-2.1, AC-2.2

- **DES-2**: Implement `RegisterCommands(sender Sender) error` that builds `SetMyCommandsConfig` from `BotCommands` and calls `sender.Request()`.
  - Satisfies: REQ-1
  - Covers: AC-1.1, AC-1.2

- **DES-3**: Call `RegisterCommands` from `main.go` startup sequence with fail-open error handling (log warning on failure, do not exit).
  - Satisfies: REQ-3
  - Covers: AC-3.1

- **DES-4**: Refactor help text in `handler.go` to generate from `BotCommands` slice instead of hardcoded string.
  - Satisfies: REQ-2
  - Covers: AC-2.1, AC-2.2

## Quality Gates

### Gate 2: Design Gate

This design passes when:
- [x] Every REQ-* from spec.md is addressed by at least one DES-*
- [x] Every AC-* from spec.md is covered by at least one DES-*
- [x] Risks and tradeoffs are documented
- [x] No DES-* exists without a linked REQ-*

**Harness check command:**
```bash
spec_reqs=$(grep -oP 'REQ-\d+' docs/features/set-commands/spec.md | sort -u)
design_reqs=$(grep -oP 'REQ-\d+' docs/features/set-commands/design.md | sort -u)
comm -23 <(echo "$spec_reqs") <(echo "$design_reqs")  # should be empty
```

## Requirement Mapping

| Design Item | Satisfies | Covers |
|-------------|-----------|--------|
| DES-1 | REQ-2 | AC-2.1, AC-2.2 |
| DES-2 | REQ-1 | AC-1.1, AC-1.2 |
| DES-3 | REQ-3 | AC-3.1 |
| DES-4 | REQ-2 | AC-2.1, AC-2.2 |
