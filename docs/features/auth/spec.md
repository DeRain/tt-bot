---
title: User Authorization
feature_id: auth
status: implemented
owner: TODO
source_files:
  - internal/bot/auth.go
  - internal/bot/handler.go
  - internal/config/config.go
last_updated: 2026-03-15
---

# User Authorization — Specification

## Overview

Access control mechanism that restricts the Telegram bot to a pre-approved set of users identified by their numeric Telegram user ID.

## Problem Statement

The bot manages qBittorrent downloads and must not be accessible to arbitrary Telegram users. Only explicitly whitelisted users should be able to interact with the bot.

## Goals

- Restrict all bot functionality to authorized users only
- Provide clear feedback to unauthorized users
- Support configuration of allowed users via environment variable

## Non-Goals

- Role-based access control (admin vs user)
- Dynamic user management (add/remove at runtime)
- Rate limiting per user

## Scope

Covers the whitelist check applied to every incoming Telegram update before dispatch to feature handlers.

## Requirements

- **REQ-1**: The bot MUST reject updates from users not in the allowed list.
- **REQ-2**: The bot MUST accept updates from users in the allowed list.
- **REQ-3**: The allowed user list MUST be configurable via the `TELEGRAM_ALLOWED_USERS` environment variable as comma-separated int64 values.
- **REQ-4**: At least one allowed user MUST be configured; the bot MUST fail to start with zero allowed users.
- **REQ-5**: Unauthorized users MUST receive a text message indicating they are not authorized.

## Acceptance Criteria

- **AC-1.1**: A message from an unknown user ID returns an "unauthorized" response and is not processed further.
- **AC-1.2**: No bot commands or callbacks execute for unauthorized users.
- **AC-2.1**: A message from a whitelisted user ID is processed normally.
- **AC-3.1**: Setting `TELEGRAM_ALLOWED_USERS=123,456` results in user IDs 123 and 456 being authorized.
- **AC-3.2**: Non-numeric or empty values in `TELEGRAM_ALLOWED_USERS` produce a startup error.
- **AC-4.1**: An empty `TELEGRAM_ALLOWED_USERS` causes a startup error with a descriptive message.
- **AC-5.1**: Unauthorized users see the message "You are not authorized to use this bot."

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
# Verify spec completeness
test $(grep -c "^\- \*\*REQ-" docs/features/auth/spec.md) -gt 0
test $(grep -c "^\- \*\*AC-" docs/features/auth/spec.md) -gt 0
test $(grep -c "TODO:" docs/features/auth/spec.md) -eq 0
```

## Risks

- If the env var is misconfigured, no users can access the bot (mitigated by clear error messages).
- User IDs are numeric and could theoretically collide across Telegram accounts (extremely unlikely, not mitigated).

## Open Questions

None.
