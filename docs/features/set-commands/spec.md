---
title: "Set Bot Commands on Startup"
feature_id: "set-commands"
status: implemented
owner: DeRain
source_files: ["internal/bot/commands.go", "internal/bot/commands_test.go", "internal/bot/commands_integration_test.go", "cmd/bot/main.go", "internal/bot/handler.go"]
last_updated: 2026-03-15
---

# Set Bot Commands on Startup — Specification

## Overview

Register the bot's available commands with Telegram on startup by calling the `setMyCommands` API, replacing the current manual BotFather setup.

## Problem Statement

Commands are registered manually via @BotFather. This means command definitions can drift from the actual code, and any new deployment requires manual BotFather interaction to update the command list.

## Goals

- Automatically register bot commands with Telegram on every startup
- Keep command definitions as a single source of truth in code
- Eliminate manual BotFather command configuration

## Non-Goals

- Scoped commands (different commands for different users/chats)
- Localized command descriptions (multi-language)
- Dynamic command registration based on runtime state

## Scope

This feature covers: defining commands in code, calling `setMyCommands` API on startup, and updating help text to use the same command definitions. It does not cover adding new bot commands — only registering the existing ones (`list`, `active`, `help`).

## Requirements

- **REQ-1**: The bot MUST call the Telegram `setMyCommands` API on startup to register its available commands.
- **REQ-2**: Command definitions MUST be a single source of truth used for both registration and help text generation.
- **REQ-3**: A failure to register commands MUST NOT prevent the bot from starting (fail-open).

## Acceptance Criteria

- **AC-1.1**: On startup, the bot sends a `setMyCommands` request to the Telegram API with commands: `list`, `active`, `help`.
- **AC-1.2**: After startup, Telegram clients show the registered commands in the command autocomplete menu.
- **AC-2.1**: The help text displayed by `/start` and `/help` is generated from the same command definitions used for registration.
- **AC-2.2**: Adding a new command to the definitions slice automatically includes it in both registration and help text without additional changes.
- **AC-3.1**: If the `setMyCommands` API call fails, the bot logs a warning and continues to start and process updates normally.

## Quality Gates

### Gate 1: Spec Gate

This spec passes when:
- [x] All requirements are clear and unambiguous
- [x] All acceptance criteria are testable
- [x] Scope and non-goals are defined
- [x] No unresolved open questions block implementation
- [x] At least one AC exists per requirement

**Harness check command:**
```bash
grep -c "^- \*\*REQ-" docs/features/set-commands/spec.md  # count requirements
grep -c "^- \*\*AC-"  docs/features/set-commands/spec.md  # count acceptance criteria
grep -c "TODO:"        docs/features/set-commands/spec.md  # should be 0 for approved
```

## Risks

- **LOW**: Telegram API rate limiting on startup — mitigated by being a single call per start.
- **LOW**: Library `tgbotapi` v5.5.1 `SetMyCommandsConfig` behaves differently than expected — mitigated by integration test.

## Open Questions

None.
