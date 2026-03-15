---
title: User Authorization — Decisions
feature_id: auth
last_updated: 2026-03-15
---

# User Authorization — Decisions

## Assumptions

- Telegram user IDs are stable and unique (Telegram guarantees this).
- The bot will have a small number of allowed users (family/personal use).
- Restart is acceptable when changing the allowed user list.

## Major Design Choices

1. **Static whitelist via env var** — Chose simplicity over dynamic user management. A database or config file would add complexity without clear benefit for the target use case (personal/small-group bot).

2. **Map-based lookup** — O(1) per check. Slice-based linear scan would also work for small lists but map is idiomatic Go for membership checks.

3. **Single rejection message** — No detailed error or retry suggestion. Minimizes information disclosure to unauthorized users.

4. **Auth check before dispatch** — Placed at the top of `HandleUpdate` to ensure no feature code runs for unauthorized users.

## Unresolved Questions

None.

## Deferred Work

- Dynamic user management (add/remove users without restart) — not planned.
- Role-based access (admin vs user permissions) — not planned.

## Out-of-Scope

- Per-command authorization (all-or-nothing access).
- Audit logging of unauthorized access attempts.
- IP-based restrictions (Telegram handles transport).
