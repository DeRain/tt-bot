## EVAL REPORT: torrent-control
================================

**Date:** 2026-03-15
**Feature:** torrent-control (select, pause/resume, back-to-list)
**Branch:** feat/torrent-control

### Capability Evals (Unit)

| # | Eval | Grader | Result | Attempts |
|---|------|--------|--------|----------|
| 1 | qbt.Client interface has PauseTorrents/ResumeTorrents | Code (`go build`) | PASS | pass@1 |
| 2 | HTTP client uses /stop and /start (v5+ endpoints) | Code (`grep`) | PASS | pass@2 |
| 3 | FormatSize formats bytes correctly | Code (`TestFormatSize`) | PASS | pass@1 |
| 4 | IsPaused identifies paused states | Code (`TestIsPaused`) | PASS | pass@1 |
| 5 | FormatTorrentDetail renders all fields, under 4096 chars | Code (`TestFormatTorrentDetail*`) | PASS | pass@1 |
| 6 | TorrentDetailKeyboard shows Pause/Resume based on state | Code (`TestTorrentDetailKeyboard*`) | PASS | pass@1 |
| 7 | TorrentSelectionKeyboard generates numbered buttons | Code (`TestTorrentSelectionKeyboard*`) | PASS | pass@1 |
| 8 | All callback data under 64 bytes | Code (`*CallbackDataUnderLimit`) | PASS | pass@1 |
| 9 | handleSelectCallback shows detail view | Code (`TestCallback_Select*`) | PASS | pass@1 |
| 10 | handlePauseCallback calls API and refreshes | Code (`TestCallback_Pause*`) | PASS | pass@1 |
| 11 | handleResumeCallback calls API and refreshes | Code (`TestCallback_Resume*`) | PASS | pass@1 |
| 12 | handleBackCallback returns to list | Code (`TestCallback_Back*`) | PASS | pass@1 |
| 13 | Filter char mapping helpers correct | Code (`TestFilterChar*`) | PASS | pass@1 |

**Note on #2**: Initially FAIL (pass@1) — used v4 endpoints `/pause` and `/resume` which return 404 on qBittorrent v5+. Fixed to `/stop` and `/start` after integration test caught the issue. Passed on second attempt.

### Capability Evals (Integration — real qBittorrent v5.1.4)

| # | Eval | Grader | Result | Attempts |
|---|------|--------|--------|----------|
| 14 | PauseTorrents/ResumeTorrents against real instance | Code (`TestIntegration_PauseAndResumeTorrent`) | PASS | pass@2 |
| 15 | Select torrent from list shows real detail data | Code (`TestE2E_SelectTorrentShowsDetail`) | PASS | pass@1 |
| 16 | Pause → Resume → Back flow with real data | Code (`TestE2E_PauseResumeTorrent`) | PASS | pass@2 |

**Note on #14, #16**: FAIL on first run (v4 endpoints). PASS after endpoint fix.

### Regression Evals

| # | Eval | Result |
|---|------|--------|
| 1 | /list command with no torrents | PASS |
| 2 | /list command with torrents | PASS |
| 3 | /active command | PASS |
| 4 | Pagination (all filter) | PASS |
| 5 | Pagination (active filter) | PASS |
| 6 | Category selection with pending magnet | PASS |
| 7 | Category selection with no pending | PASS |
| 8 | No-category button | PASS |
| 9 | Noop callback | PASS |
| 10 | Invalid page callback | PASS |
| 11 | Unauthorized user rejected | PASS |
| 12 | Help/start commands | PASS |
| 13 | Magnet link parsing (mid-text) | PASS |
| 14 | AddMagnet error handling | PASS |
| 15 | Unknown callback data | PASS |
| 16 | QBT connection error on /list | PASS |

### Metrics

```
Capability (unit):      pass@1: 12/13 (92%)    pass@2: 13/13 (100%)
Capability (integration): pass@1: 1/3  (33%)    pass@2: 3/3   (100%)
Regression:             pass^1: 16/16 (100%)

Coverage:
  bot:        81.1%  (target: 80%) ✓
  formatter:  96.6%  (target: 80%) ✓
  qbt:        78.7%  (target: 80%) ✗ (pre-existing gap, new code at 84.6%)
```

### Key Finding

Integration tests caught a critical API contract issue that unit tests could not:
- qBittorrent v5+ renamed `/api/v2/torrents/pause` → `/stop` and `/resume` → `/start`
- httptest-based unit tests passed because they mock the server endpoints
- `make test-integration` (real qBittorrent in Docker) returned 404, exposing the bug
- **Lesson codified in**: `gates.md` (mandatory integration tests), `CLAUDE.md` (hard rule), `feedback_no_optional_tests.md` (memory)

### Status: VERIFIED

All capability evals pass (pass@2: 100%). All regression evals pass (pass^1: 100%). Integration tests confirmed against real qBittorrent v5.1.4.
