---
title: Configuration Loading -- Design
feature_id: config
status: implemented
last_updated: 2026-03-15
---

# Configuration Loading -- Design

## Overview

The design follows a single-function-load pattern: one public `Load()` function reads all environment variables, validates them, and returns an immutable `Config` value. Internal helpers keep the function body concise and each validation concern isolated.

## Design Decisions

### DES-1: Config struct (value type)

`Config` is a plain Go struct with exported fields. Being a value type, it is copied on assignment and function calls, which prevents accidental mutation of shared configuration state.

```
Config {
    TelegramToken  string
    AllowedUsers   []int64
    QBTBaseURL     string
    QBTUsername    string
    QBTPassword    string
    PollInterval   time.Duration
}
```

The `AllowedUsers` field is a slice (reference type internally), but because `Config` is always returned fresh from `Load()` and no setter methods exist, the slice is effectively owned by the caller.

**Satisfies:** REQ-7

### DES-2: Load() function

`Load()` is the single public entry point. It:

1. Reads each required variable via `requireEnv()`.
2. Parses `TELEGRAM_ALLOWED_USERS` via `parseAllowedUsers()`.
3. Parses `POLL_INTERVAL` via `parsePollInterval()`.
4. Returns the assembled `Config` value or the first error encountered.

The function returns `(Config, error)` -- Config by value, not pointer. On any validation failure, it returns the zero-value `Config{}` and a descriptive error.

**Satisfies:** REQ-1, REQ-2

### DES-3: User ID parsing

`parseAllowedUsers(raw string)` splits on commas, trims whitespace from each token, skips empty tokens, and parses each as `int64` via `strconv.ParseInt`. After processing, it enforces that at least one valid ID was found.

Error messages quote the offending token for diagnosability (e.g., `invalid user ID "abc" in TELEGRAM_ALLOWED_USERS: must be an integer`).

**Satisfies:** REQ-3, REQ-4, REQ-6

### DES-4: Poll interval parsing

`parsePollInterval(raw string)` returns the default `30 * time.Second` when the input is empty. Otherwise it delegates to `time.ParseDuration`, wrapping any error with the variable name and raw value for context.

**Satisfies:** REQ-5, REQ-6

### DES-5: Fail-fast validation

`Load()` checks variables sequentially and returns on the first error. This fail-fast approach keeps error output simple (one issue at a time) and avoids partial configuration states. The caller (`cmd/bot/main.go`) calls `log.Fatalf` on error, terminating the process immediately.

**Satisfies:** REQ-1, REQ-2, REQ-6

## Internal helpers

| Helper | Visibility | Purpose |
|--------|-----------|---------|
| `requireEnv(key)` | unexported | Returns env value or error naming the missing key |
| `parseAllowedUsers(raw)` | unexported | Comma-split, trim, parse int64, enforce min-1 |
| `parsePollInterval(raw)` | unexported | Parse duration with 30s default |

## Design-to-Requirement Mapping

| Design | Requirements |
|--------|-------------|
| DES-1: Config struct | REQ-7 |
| DES-2: Load() function | REQ-1, REQ-2 |
| DES-3: User ID parsing | REQ-3, REQ-4, REQ-6 |
| DES-4: Poll interval parsing | REQ-5, REQ-6 |
| DES-5: Fail-fast validation | REQ-1, REQ-2, REQ-6 |

## Quality Gates

### Gate 2: Design Gate

- [x] Every DES-N has a "Satisfies" reference to at least one REQ
- [x] Every REQ from spec is mapped to at least one DES
- [x] Design-to-Requirement mapping table is complete
- [x] Internal helpers documented
- [x] No unresolved TODOs

**Harness:**
```bash
# Verify every spec REQ appears in the design mapping table
for req in $(grep -oP 'REQ-\d+' docs/features/config/spec.md | sort -u); do
  grep -q "$req" docs/features/config/design.md || echo "MISSING: $req"
done
```
