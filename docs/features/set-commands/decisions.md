---
title: "Set Bot Commands on Startup — Decisions"
feature_id: "set-commands"
last_updated: 2026-03-15
---

# Set Bot Commands on Startup — Decisions

## Assumptions

- `tgbotapi` v5.5.1 `SetMyCommandsConfig` works as documented (verified via `go doc`)
- `Sender.Request()` is the correct method for API calls returning bool responses
- Existing commands (`list`, `active`, `help`) are the complete set to register
- `/start` is intentionally excluded from registration (Telegram handles it specially)

## Major Design Choices

### Choice 1: Package-level slice vs constants for command definitions

- **Decision**: Package-level `var BotCommands []BotCommand` slice
- **Alternatives**: String constants, map, or config file
- **Rationale**: Go doesn't support const slices. A slice allows iteration for both `setMyCommands` and help text generation. Hardcoded in source is appropriate since commands change with code.
- **Tradeoff**: Not configurable at runtime, but this is intentional — commands are a code-level concern.

### Choice 2: Fail-open on registration failure

- **Decision**: Log warning and continue startup if `setMyCommands` fails
- **Alternatives**: Exit on failure, retry with backoff
- **Rationale**: Command registration is a UX enhancement. The bot functions correctly without it. Retries add complexity for minimal benefit since the bot re-registers on next restart.
- **Tradeoff**: If registration consistently fails, users won't see autocomplete. Acceptable for a self-hosted bot.

### Choice 3: Use Sender interface vs direct BotAPI call

- **Decision**: Use `Sender.Request()` for testability
- **Alternatives**: Accept `*tgbotapi.BotAPI` directly
- **Rationale**: Consistent with existing codebase patterns. Enables unit testing with mock Sender.
- **Tradeoff**: None — Sender already has the required method.

## Unresolved Questions

None.

## Deferred Work

- Scoped commands (admin vs regular user) — not needed for current user base
- Localized command descriptions — single-language bot
- `deleteMyCommands` on graceful shutdown — unnecessary, commands persist until overwritten

## Out-of-Scope

- Adding new bot commands (this feature only registers existing ones)
- BotFather configuration automation beyond `setMyCommands`
