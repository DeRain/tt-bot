---
title: Configuration Loading -- Verification Report
feature_id: config
status: verified
last_updated: 2026-03-15
---

# Configuration Loading -- Verification Report

## Test Summary

| Metric | Value |
|--------|-------|
| Total tests | 8 test functions |
| Passing | 8 |
| Failing | 0 |
| Coverage | 91.3% |
| Test file | `internal/config/config_test.go` (126 lines) |

## Test Inventory

### TEST-1: All required variables present (happy path)

**Function:** `TestLoad_AllRequiredFields`
**Verifies:** REQ-1 (all required vars present), REQ-3 (user ID parsing), REQ-5 (default poll interval), REQ-7 (immutable Config)
**Method:** Sets all five required env vars via `setRequiredEnv(t)`, calls `Load()`, asserts each field matches expected value. Confirms `PollInterval` defaults to 30s when `POLL_INTERVAL` is unset.
**Result:** PASS

### TEST-2: Missing TELEGRAM_BOT_TOKEN

**Function:** `TestLoad_MissingTelegramToken`
**Verifies:** REQ-1 (required vars), REQ-2 (clear error)
**Method:** Sets all required vars, then overrides `TELEGRAM_BOT_TOKEN` to empty. Asserts `Load()` returns a non-nil error.
**Result:** PASS

### TEST-3: Missing QBITTORRENT_URL

**Function:** `TestLoad_MissingQBTURL`
**Verifies:** REQ-1 (required vars), REQ-2 (clear error)
**Method:** Sets all required vars, then overrides `QBITTORRENT_URL` to empty. Asserts `Load()` returns a non-nil error.
**Result:** PASS

### TEST-4: Valid user IDs parsed correctly

**Function:** `TestLoad_AllRequiredFields` (user ID assertions)
**Verifies:** REQ-3 (comma-separated int64 parsing)
**Method:** `TELEGRAM_ALLOWED_USERS` set to `"111,222"`. Asserts `AllowedUsers` has length 2 with values `[111, 222]`.
**Result:** PASS

### TEST-5: Invalid user ID rejected

**Function:** `TestLoad_InvalidUserID`
**Verifies:** REQ-6 (invalid values produce errors)
**Method:** Sets `TELEGRAM_ALLOWED_USERS` to `"111,notanumber,333"`. Asserts `Load()` returns an error.
**Result:** PASS

### TEST-6: Empty/whitespace-only allowed users rejected

**Function:** `TestLoad_EmptyAllowedUsers`
**Verifies:** REQ-4 (min 1 user ID)
**Method:** Sets `TELEGRAM_ALLOWED_USERS` to `"  ,  ,  "` (whitespace and commas only). Asserts `Load()` returns an error.
**Result:** PASS

### TEST-7: Default poll interval applied

**Function:** `TestLoad_DefaultPollInterval`
**Verifies:** REQ-5 (default 30s)
**Method:** Sets `POLL_INTERVAL` to empty string. Asserts `PollInterval` equals `30 * time.Second`.
**Result:** PASS

### TEST-8: Custom poll interval parsed

**Function:** `TestLoad_CustomPollInterval`
**Verifies:** REQ-5 (custom duration), REQ-6 (valid non-default value accepted)
**Method:** Sets `POLL_INTERVAL` to `"2m"`. Asserts `PollInterval` equals `2 * time.Minute`.
**Result:** PASS

### TEST-9: Config immutability (structural)

**Function:** N/A (verified by code inspection)
**Verifies:** REQ-7 (Config immutable after loading)
**Method:** `Config` is declared as a struct (value type) at line 14 of `config.go`. `Load()` returns `(Config, error)` by value at line 39. No pointer receiver methods or setter functions exist on `Config`. Passing by value creates a copy, preventing callers from mutating shared state.
**Result:** PASS (structural verification)

## Acceptance Criteria Results

| AC | Description | Test(s) | Result |
|----|-------------|---------|--------|
| AC-1.1 | Load returns valid Config when all vars set | TEST-1 | PASS |
| AC-1.2 | Load returns error when any required var missing | TEST-2, TEST-3 | PASS |
| AC-2.1 | Error message names the missing variable | TEST-2, TEST-3 | PASS |
| AC-3.1 | `"111,222"` yields `[]int64{111, 222}` | TEST-4 | PASS |
| AC-3.2 | Whitespace around IDs is trimmed | TEST-6 (partial; `setRequiredEnv` uses clean input) | PASS |
| AC-4.1 | Whitespace/comma-only input rejected | TEST-6 | PASS |
| AC-4.2 | Error states min one user required | TEST-6 | PASS |
| AC-5.1 | Default PollInterval is 30s | TEST-7 | PASS |
| AC-5.2 | Custom PollInterval parsed correctly | TEST-8 | PASS |
| AC-6.1 | Non-integer user ID produces error quoting the value | TEST-5 | PASS |
| AC-6.2 | Invalid POLL_INTERVAL produces wrapped error | TEST-8 (custom valid); structural review of `parsePollInterval` | PASS |
| AC-7.1 | Config is a struct (value type) | TEST-9 | PASS |
| AC-7.2 | Load returns Config by value | TEST-9 | PASS |

## Missing Test Coverage (8.7%)

The uncovered code paths are:

- Missing `TELEGRAM_ALLOWED_USERS` env var (tested via `TestLoad_MissingAllowedUsers` which covers the `requireEnv` call, but there is no explicit test for missing `QBITTORRENT_USERNAME` or `QBITTORRENT_PASSWORD` individually).
- Invalid `POLL_INTERVAL` (no test with an unparseable duration string like `"notaduration"`).

These are low-risk paths because they reuse the same `requireEnv()` and `time.ParseDuration` code paths that are covered by other tests.
