---
title: User Authorization — Design
feature_id: auth
status: implemented
depends_on_spec: docs/features/auth/spec.md
last_updated: 2026-03-15
---

# User Authorization — Design

## Overview

A simple whitelist-based authorization layer that checks each incoming Telegram update against a set of allowed user IDs before dispatching to handlers.

## Architecture

- **DES-1**: Authorizer struct — holds allowed user IDs in a `map[int64]bool` for O(1) lookup.
  - Satisfies: REQ-1, REQ-2
  - Covers: AC-1.1, AC-2.1

- **DES-2**: Auth gate in HandleUpdate — the first check in the update dispatch path; rejects unauthorized users before any feature logic executes.
  - Satisfies: REQ-1, REQ-5
  - Covers: AC-1.1, AC-1.2, AC-5.1

- **DES-3**: Config parsing — `TELEGRAM_ALLOWED_USERS` parsed in config.Load(), split by comma, each value parsed as int64, minimum 1 required.
  - Satisfies: REQ-3, REQ-4
  - Covers: AC-3.1, AC-3.2, AC-4.1

## Data Flow

1. `config.Load()` reads `TELEGRAM_ALLOWED_USERS`, parses to `[]int64`, fails if empty or malformed.
2. `bot.New()` receives the parsed user IDs and constructs `Authorizer` with a map.
3. On each update, `HandleUpdate()` calls `Authorizer.IsAllowed(userID)`.
4. If not allowed: sends rejection message, returns early.
5. If allowed: proceeds to command/message dispatch.

## Interfaces

- `Authorizer.IsAllowed(userID int64) bool` — pure function, no side effects
- `config.Config.AllowedUsers []int64` — parsed from env var

## Data/Storage Impact

None. User IDs are stored only in memory (map). No persistence.

## Error Handling

- Missing env var → startup error: "TELEGRAM_ALLOWED_USERS is required"
- Empty value → startup error: "at least one allowed user is required"
- Non-numeric value → startup error: "invalid user ID: <value>"
- Unauthorized user → sends text message, no error returned

## Security Considerations

- User IDs are not secrets but should not be logged at info level.
- The whitelist is the sole access control mechanism; no secondary auth exists.
- Bot token must be kept secret to prevent impersonation.

## Performance Considerations

- Map lookup is O(1). No performance concern even with many users.
- Auth check adds negligible overhead per update.

## Tradeoffs

- Static whitelist (env var) vs dynamic management: chose simplicity over flexibility.
- Single rejection message vs detailed error: chose minimal information disclosure.

## Risks

- Restart required to change allowed users (acceptable for stateless design).

## Requirement Mapping

| Design Item | Satisfies | Covers |
|-------------|-----------|--------|
| DES-1 | REQ-1, REQ-2 | AC-1.1, AC-2.1 |
| DES-2 | REQ-1, REQ-5 | AC-1.1, AC-1.2, AC-5.1 |
| DES-3 | REQ-3, REQ-4 | AC-3.1, AC-3.2, AC-4.1 |

## Quality Gates

### Gate 2: Design Gate

This design passes when:
- [x] Every REQ-* from spec.md is addressed by at least one DES-*
- [x] Every AC-* from spec.md is covered by at least one DES-*
- [x] Risks and tradeoffs are documented
- [x] No DES-* exists without a linked REQ-*

**Harness check command:**
```bash
spec_reqs=$(grep -oP 'REQ-\d+' docs/features/auth/spec.md | sort -u)
design_reqs=$(grep -oP 'REQ-\d+' docs/features/auth/design.md | sort -u)
comm -23 <(echo "$spec_reqs") <(echo "$design_reqs")  # should be empty
```
