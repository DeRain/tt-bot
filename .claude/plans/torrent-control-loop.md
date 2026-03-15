# Torrent Control — Sequential Loop Runbook

## Pattern: Sequential (Safe Mode)
## Feature: torrent-control
## Branch: feat/torrent-control
## Created: 2026-03-15

## Execution Order

| Step | Task | Phase | Depends On | Verification |
|------|------|-------|------------|--------------|
| 1 | TASK-1: Add Pause/Resume to qbt.Client interface | qbt | — | go build ./internal/qbt/... |
| 2 | TASK-3: Update mockQBTClient | qbt | TASK-1 | go build ./internal/bot/... |
| 3 | TASK-2: Implement HTTPClient Pause/Resume | qbt | TASK-1 | go test ./internal/qbt/... -run TestPause\|TestResume -v |
| 4 | TASK-4: Add FormatSize helper | formatter | — | go test ./internal/formatter/... -run TestFormatSize -v |
| 5 | TASK-5: Add IsPaused helper | formatter | — | go test ./internal/formatter/... -run TestIsPaused -v |
| 6 | TASK-8: Add TorrentSelectionKeyboard | formatter | — | go test ./internal/formatter/... -run TestTorrentSelection -v |
| 7 | TASK-9: Add filter char mapping helpers | bot | — | go test ./internal/bot/... -run TestFilterChar -v |
| 8 | TASK-6: Add FormatTorrentDetail | formatter | TASK-4 | go test ./internal/formatter/... -run TestFormatTorrentDetail -v |
| 9 | TASK-7: Add TorrentDetailKeyboard | formatter | TASK-5 | go test ./internal/formatter/... -run TestTorrentDetailKeyboard -v |
| 10 | TASK-10: Add handleSelectCallback | bot | TASK-6,7,9 | go test ./internal/bot/... -run TestHandleSelect -v |
| 11 | TASK-11: Add handlePause/ResumeCallback | bot | TASK-2,3,10 | go test ./internal/bot/... -run TestHandlePause\|TestHandleResume -v |
| 12 | TASK-12: Add handleBackCallback | bot | TASK-8,9 | go test ./internal/bot/... -run TestHandleBack -v |
| 13 | TASK-13: Integrate selection keyboard + dispatcher | bot | TASK-8,9,10,11,12 | go test ./internal/bot/... -v |
| 14 | TASK-14: Full gate verification | all | all | go build ./... && go test ./... -short -cover |

## Safety Rules

- TDD: write test FIRST, then implementation
- After each step: run verification command
- If verification fails: fix → re-verify → max 3 retries → stop
- Checkpoint: commit after each phase completion
- Stop condition: TASK-14 verification passes

## Gate Commands

```bash
# Per-step verification: run the command from the table above
# Phase checkpoint: go build ./... && go test ./... -short
# Final gate: go build ./... && go test ./... -short -cover
```
