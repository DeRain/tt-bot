---
title: Configuration Loading -- Traceability Matrix
feature_id: config
status: implemented
last_updated: 2026-03-15
---

# Configuration Loading -- Traceability Matrix

## Requirement to Design to Task to Test

| Requirement | Description | Design | Tasks | Tests | Status |
|-------------|-------------|--------|-------|-------|--------|
| REQ-1 | Required env vars must be present | DES-2, DES-5 | TASK-2 | TEST-1, TEST-2, TEST-3 | Implemented |
| REQ-2 | Missing vars produce clear errors | DES-2, DES-5 | TASK-2 | TEST-2, TEST-3 | Implemented |
| REQ-3 | User IDs parsed as comma-separated int64 | DES-3 | TASK-3 | TEST-4 | Implemented |
| REQ-4 | At least one user ID required | DES-3 | TASK-3 | TEST-6 | Implemented |
| REQ-5 | POLL_INTERVAL defaults to 30s | DES-4 | TASK-4 | TEST-7 | Implemented |
| REQ-6 | Invalid values produce descriptive errors | DES-3, DES-4, DES-5 | TASK-3, TASK-4 | TEST-5, TEST-8 | Implemented |
| REQ-7 | Config immutable after loading | DES-1 | TASK-1 | TEST-9 | Implemented |

## Design to Source

| Design | Source Location |
|--------|----------------|
| DES-1: Config struct | `internal/config/config.go` lines 14-34 |
| DES-2: Load() function | `internal/config/config.go` lines 39-83 |
| DES-3: User ID parsing | `internal/config/config.go` lines 98-119 |
| DES-4: Poll interval parsing | `internal/config/config.go` lines 123-136 |
| DES-5: Fail-fast validation | `internal/config/config.go` lines 39-83, `cmd/bot/main.go` lines 39-42 |

## Test to Source

| Test | Test Function | Source File |
|------|--------------|-------------|
| TEST-1 | `TestLoad_AllRequiredFields` | `internal/config/config_test.go` line 20 |
| TEST-2 | `TestLoad_MissingTelegramToken` | `internal/config/config_test.go` line 51 |
| TEST-3 | `TestLoad_MissingQBTURL` | `internal/config/config_test.go` line 81 |
| TEST-4 | `TestLoad_AllRequiredFields` (user ID assertions) | `internal/config/config_test.go` lines 31-35 |
| TEST-5 | `TestLoad_InvalidUserID` | `internal/config/config_test.go` line 71 |
| TEST-6 | `TestLoad_EmptyAllowedUsers` | `internal/config/config_test.go` line 117 |
| TEST-7 | `TestLoad_DefaultPollInterval` | `internal/config/config_test.go` line 104 |
| TEST-8 | `TestLoad_CustomPollInterval` | `internal/config/config_test.go` line 91 |
| TEST-9 | Structural (Config is value type, Load returns by value) | `internal/config/config.go` lines 14, 39 |

## Coverage

Unit test coverage: 91.3%
