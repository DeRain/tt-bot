---
title: Configuration Loading -- Implementation Plan
feature_id: config
status: complete
last_updated: 2026-03-15
---

# Configuration Loading -- Implementation Plan

## Summary

Implementation of the config feature in `internal/config/config.go` (136 lines) with unit tests in `internal/config/config_test.go` (126 lines). All tasks are complete.

## Tasks

### TASK-1: Config struct definition

Define the `Config` struct as a value type with exported fields for all configuration values.

| Field | Type | Source |
|-------|------|--------|
| `TelegramToken` | `string` | `TELEGRAM_BOT_TOKEN` |
| `AllowedUsers` | `[]int64` | `TELEGRAM_ALLOWED_USERS` |
| `QBTBaseURL` | `string` | `QBITTORRENT_URL` |
| `QBTUsername` | `string` | `QBITTORRENT_USERNAME` |
| `QBTPassword` | `string` | `QBITTORRENT_PASSWORD` |
| `PollInterval` | `time.Duration` | `POLL_INTERVAL` |

**Status:** Complete
**Implements:** DES-1
**File:** `internal/config/config.go` (lines 14-34)

### TASK-2: Load() function and requireEnv helper

Implement the public `Load()` function that reads all required environment variables via `requireEnv()`, validates them, and returns a `Config` value or descriptive error.

- `Load()` calls `requireEnv()` for each of the five required variables.
- Returns `Config{}` and an error on first failure (fail-fast).
- Assembles and returns the final `Config` value on success.

**Status:** Complete
**Implements:** DES-2, DES-5
**File:** `internal/config/config.go` (lines 39-93)

### TASK-3: User ID parsing

Implement `parseAllowedUsers(raw string)` to:
- Split on commas.
- Trim whitespace from each token.
- Skip empty tokens.
- Parse each token as int64.
- Enforce at least one valid ID.
- Return descriptive error quoting the invalid token on parse failure.

**Status:** Complete
**Implements:** DES-3
**File:** `internal/config/config.go` (lines 98-119)

### TASK-4: Poll interval parsing

Implement `parsePollInterval(raw string)` to:
- Return `30 * time.Second` when input is empty.
- Delegate to `time.ParseDuration` otherwise.
- Wrap parse errors with variable name and raw value.

**Status:** Complete
**Implements:** DES-4
**File:** `internal/config/config.go` (lines 123-136)

### TASK-5: Unit tests

Write comprehensive unit tests covering:
- All-required-fields happy path.
- Missing required variable (token, URL, allowed users).
- Invalid user ID (non-integer).
- Empty/whitespace-only allowed users.
- Default poll interval.
- Custom poll interval.

Test helper `setRequiredEnv(t)` sets all five required variables to valid defaults, allowing individual tests to override specific variables.

**Status:** Complete (91.3% coverage)
**Validates:** REQ-1 through REQ-7
**File:** `internal/config/config_test.go` (lines 1-126)

## Quality Gates

### Gate 3: Plan Gate

- [x] Every TASK has a status (Complete / In Progress / Not Started)
- [x] Every TASK maps to at least one DES or REQ
- [x] All tasks have file references with line numbers
- [x] Dependency order is documented
- [x] No unresolved TODOs in task descriptions

**Harness:**
```bash
# Task count must equal verification count
TASK_COUNT=$(grep -c '^### TASK-' docs/features/config/plan.md)
MAP_COUNT=$(grep -c 'Implements:' docs/features/config/plan.md)
echo "Tasks: $TASK_COUNT, Mapped: $MAP_COUNT"
# TASK-5 uses Validates instead of Implements, so count both
TOTAL_MAPPED=$((MAP_COUNT + $(grep -c 'Validates:' docs/features/config/plan.md)))
test "$TASK_COUNT" -eq "$TOTAL_MAPPED" && echo "PASS" || echo "FAIL: task count mismatch"
```

**Iterative Harness Loop Protocol:**
1. Run all harness commands from Gates 1-3.
2. If any check fails, fix the failing document and re-run.
3. Repeat until all gates pass with zero failures.
4. Only then proceed to implementation.

## Dependency Order

```
TASK-1 (struct) --> TASK-2 (Load + requireEnv) --> TASK-3 (user parsing)
                                                --> TASK-4 (interval parsing)
                                                --> TASK-5 (tests)
```

TASK-1 is the foundation. TASK-2 depends on TASK-1. TASK-3 and TASK-4 are independent of each other but both feed into TASK-2. TASK-5 depends on all other tasks.
