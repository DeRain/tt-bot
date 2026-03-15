---
title: Torrent Listing — Plan
feature_id: list-torrents
status: implemented
depends_on_design: docs/features/list-torrents/design.md
last_updated: 2026-03-15
---

# Torrent Listing — Plan

## Overview

Implementation plan for the paginated torrent listing feature, covering the qBittorrent API call, message formatting, pagination keyboard, command handlers, and callback handler.

## Preconditions

- Go module initialized with telegram-bot-api/v5 and qbt package
- `qbt.Client` interface defined with `ListTorrents` method
- `bot.Handler` and `bot.Sender` interface established
- Auth feature implemented (auth gate protects all commands)

## Task Sequence

- **TASK-1**: Implement `ListTorrents` in `qbt.HTTPClient` with filter support
  - Derived from: DES-2
  - Implements: REQ-1, REQ-2
  - Impacts: internal/qbt/http.go, internal/qbt/client.go
  - Verification: TEST-1, TEST-2

- **TASK-2**: Implement `FormatTorrentList` with message size guard
  - Derived from: DES-2, DES-5
  - Implements: REQ-5, REQ-6, REQ-8
  - Impacts: internal/formatter/format.go
  - Verification: TEST-6, TEST-7, TEST-8

- **TASK-3**: Implement `PaginationKeyboard` with Prev/Page/Next logic
  - Derived from: DES-3
  - Implements: REQ-4
  - Impacts: internal/formatter/format.go
  - Verification: TEST-4, TEST-5

- **TASK-4**: Implement `/list` and `/active` command handlers (`handleCommand`, `sendTorrentPage`)
  - Derived from: DES-1, DES-2
  - Implements: REQ-1, REQ-2, REQ-3
  - Impacts: internal/bot/handler.go
  - Verification: TEST-1, TEST-2, TEST-3

- **TASK-5**: Implement pagination callback handler (`handlePaginationCallback`)
  - Derived from: DES-4
  - Implements: REQ-3, REQ-4
  - Impacts: internal/bot/callback.go
  - Verification: TEST-5, TEST-6

- **TASK-6**: Implement formatting helpers (`FormatSpeed`, `FormatProgress`, `truncateName`)
  - Derived from: DES-6
  - Implements: REQ-5, REQ-7
  - Impacts: internal/formatter/format.go
  - Verification: TEST-8, TEST-9, TEST-10

- **TASK-7**: Write unit tests and E2E tests
  - Derived from: DES-1 through DES-6
  - Implements: REQ-1 through REQ-8
  - Impacts: internal/bot/handler_test.go, internal/bot/callback_test.go, internal/formatter/format_test.go, internal/bot/e2e_test.go
  - Verification: TEST-1 through TEST-10

## Dependencies

- TASK-4 depends on TASK-1 (needs `ListTorrents`), TASK-2 (needs `FormatTorrentList`), TASK-3 (needs `PaginationKeyboard`), TASK-6 (needs formatting helpers)
- TASK-5 depends on TASK-2, TASK-3, TASK-6
- TASK-2 depends on TASK-6 (uses `truncateName`, `FormatProgress`, `FormatSpeed`)
- TASK-7 can partially run after each preceding task

## Affected Files

- internal/qbt/client.go (interface definition)
- internal/qbt/http.go (HTTP implementation)
- internal/formatter/format.go (formatting and keyboard logic)
- internal/bot/handler.go (command dispatch, `sendTorrentPage`)
- internal/bot/callback.go (pagination callback handler)
- internal/bot/sender.go (`toTGKeyboard` helper)
- internal/bot/handler_test.go (unit tests)
- internal/bot/callback_test.go (callback unit tests)
- internal/formatter/format_test.go (formatter unit tests)
- internal/bot/e2e_test.go (E2E integration tests)

## Rollout Notes

- No migration needed. Feature is available immediately after deployment.
- Existing `/list` and `/active` commands are the only entry points.

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
tasks=$(grep -c "^\- \*\*TASK-" docs/features/list-torrents/plan.md)
verifications=$(grep -c "Verification:" docs/features/list-torrents/plan.md)
test "$tasks" -eq "$verifications"
```

### Iterative Harness Loop Protocol

When executing this plan via an agent harness loop:
1. Execute tasks in dependency order (TASK-6 → TASK-1 → TASK-2 → TASK-3 → TASK-4 → TASK-5 → TASK-7)
2. After each TASK-*, run its verification target
3. If verification fails: fix, re-verify, max 3 retries
4. After all tasks: run `make gate-all` as the implementation gate
5. Update traceability.md with implementation evidence
6. Run verification.md checks

## Verification Steps

See verification.md for detailed test mapping.

## Blockers

None.
