---
title: User Authorization — Plan
feature_id: auth
status: implemented
depends_on_design: docs/features/auth/design.md
last_updated: 2026-03-15
---

# User Authorization — Plan

## Overview

Implementation plan for the whitelist-based authorization system.

## Preconditions

- Go module initialized
- telegram-bot-api/v5 dependency available
- Config package exists with env var loading

## Task Sequence

- **TASK-1**: Implement config parsing for `TELEGRAM_ALLOWED_USERS`
  - Derived from: DES-3
  - Implements: REQ-3, REQ-4
  - Impacts: internal/config/config.go
  - Verification: TEST-1, TEST-2

- **TASK-2**: Implement `Authorizer` struct with `IsAllowed` method
  - Derived from: DES-1
  - Implements: REQ-1, REQ-2
  - Impacts: internal/bot/auth.go
  - Verification: TEST-3

- **TASK-3**: Integrate auth check into `HandleUpdate` dispatch
  - Derived from: DES-2
  - Implements: REQ-1, REQ-5
  - Impacts: internal/bot/handler.go
  - Verification: TEST-4, TEST-5

- **TASK-4**: Write unit tests for auth and config
  - Derived from: DES-1, DES-2, DES-3
  - Implements: REQ-1, REQ-2, REQ-3, REQ-4, REQ-5
  - Impacts: internal/bot/auth_test.go, internal/config/config_test.go, internal/bot/handler_test.go
  - Verification: TEST-1 through TEST-5

## Dependencies

- TASK-2 depends on TASK-1 (needs parsed user IDs)
- TASK-3 depends on TASK-2 (needs Authorizer)
- TASK-4 can partially run after each task

## Affected Files

- internal/config/config.go
- internal/config/config_test.go
- internal/bot/auth.go
- internal/bot/auth_test.go
- internal/bot/handler.go
- internal/bot/handler_test.go

## Rollout Notes

- No migration needed. Config change only requires env var.
- Existing deployments must set `TELEGRAM_ALLOWED_USERS` before upgrading.

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
tasks=$(grep -c "^\- \*\*TASK-" docs/features/auth/plan.md)
verifications=$(grep -c "Verification:" docs/features/auth/plan.md)
test "$tasks" -eq "$verifications"
```

### Iterative Harness Loop Protocol

When executing this plan via an agent harness loop:
1. Execute tasks in dependency order (TASK-1 → TASK-2 → TASK-3 → TASK-4)
2. After each TASK-*, run its verification target
3. If verification fails: fix, re-verify, max 3 retries
4. After all tasks: run `make gate-all` as the implementation gate
5. Update traceability.md with implementation evidence
6. Run verification.md checks

## Verification Steps

See verification.md for detailed test mapping.

## Blockers

None.
