---
title: Configuration Loading
feature_id: config
status: implemented
owner: TODO
source_files:
  - internal/config/config.go
  - internal/config/config_test.go
  - cmd/bot/main.go
last_updated: 2026-03-15
---

# Configuration Loading -- Feature Specification

## Overview

The config feature loads and validates all runtime configuration for tt-bot from environment variables at startup. It produces an immutable `Config` value type that the rest of the application consumes by value. The feature enforces fail-fast semantics: the process exits immediately with a descriptive error if any required variable is missing or malformed.

## Requirements

### REQ-1: Required environment variables must be present

All five required environment variables must be set and non-empty for the application to start:

| Variable | Purpose |
|----------|---------|
| `TELEGRAM_BOT_TOKEN` | Telegram Bot API token |
| `TELEGRAM_ALLOWED_USERS` | Comma-separated whitelist of Telegram user IDs |
| `QBITTORRENT_URL` | qBittorrent WebUI base URL |
| `QBITTORRENT_USERNAME` | qBittorrent WebUI username |
| `QBITTORRENT_PASSWORD` | qBittorrent WebUI password |

**Acceptance criteria:**
- AC-1.1: `Load()` returns a valid `Config` when all required variables are set with valid values.
- AC-1.2: `Load()` returns an error when any required variable is unset or empty.

### REQ-2: Missing variables produce clear errors

When a required variable is missing, the error message must identify which variable is absent.

**Acceptance criteria:**
- AC-2.1: Error message includes the name of the missing variable (e.g., `"required environment variable TELEGRAM_BOT_TOKEN is not set"`).

### REQ-3: User IDs parsed as comma-separated int64

`TELEGRAM_ALLOWED_USERS` is a comma-separated list of int64 values. Whitespace around individual values is trimmed.

**Acceptance criteria:**
- AC-3.1: `"111,222"` produces `[]int64{111, 222}`.
- AC-3.2: `" 111 , 222 "` produces `[]int64{111, 222}` (whitespace trimmed).

### REQ-4: At least one user ID required

The allowed-users list must contain at least one valid integer after parsing.

**Acceptance criteria:**
- AC-4.1: `Load()` returns an error when `TELEGRAM_ALLOWED_USERS` resolves to zero valid IDs (e.g., `"  ,  ,  "`).
- AC-4.2: Error message states that at least one valid user ID is required.

### REQ-5: POLL_INTERVAL defaults to 30 seconds

`POLL_INTERVAL` is optional. When unset or empty, it defaults to `30s`.

**Acceptance criteria:**
- AC-5.1: `Config.PollInterval` equals `30 * time.Second` when `POLL_INTERVAL` is empty.
- AC-5.2: `Config.PollInterval` equals the parsed duration when `POLL_INTERVAL` is set to a valid Go duration string (e.g., `"2m"` yields `2 * time.Minute`).

### REQ-6: Invalid values produce descriptive errors

Malformed values for any variable result in an error that identifies the variable and the nature of the problem.

**Acceptance criteria:**
- AC-6.1: A non-integer token in `TELEGRAM_ALLOWED_USERS` (e.g., `"111,notanumber,333"`) produces an error naming the invalid value.
- AC-6.2: An unparseable `POLL_INTERVAL` (e.g., `"notaduration"`) produces an error naming the variable and wrapping the underlying parse error.

### REQ-7: Config is immutable after loading

The `Config` type is a value type (struct, not pointer). Passing it by value prevents callers from mutating shared state.

**Acceptance criteria:**
- AC-7.1: `Config` is declared as a struct (value type), not a pointer.
- AC-7.2: `Load()` returns `Config` by value (not `*Config`).

## Quality Gates

### Gate 1: Spec Gate

- [x] All requirements have unique IDs (REQ-N)
- [x] Every requirement has at least one acceptance criterion (AC-N.M)
- [x] No TODOs remain in spec body
- [x] Overview section present and concise
- [x] Requirements are testable and unambiguous

**Harness:**
```bash
# Count requirements and acceptance criteria
grep -c '^### REQ-' docs/features/config/spec.md
grep -c '^- AC-' docs/features/config/spec.md
# Verify no TODOs in spec body (frontmatter owner: TODO is acceptable)
grep -c 'TODO' <(tail -n +12 docs/features/config/spec.md)  # expect 0
```
