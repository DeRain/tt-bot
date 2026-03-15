---
title: Download Completion Notifications -- Traceability Matrix
feature_id: completion-notifications
last_updated: 2026-03-15
---

# Download Completion Notifications -- Traceability Matrix

## Full Requirement Traceability

| Requirement | Design | Implementation | Task | Tests | Status |
|-------------|--------|---------------|------|-------|--------|
| REQ-1: Periodic Polling | DES-1 (Poller Goroutine) | `poller.Run` -- ticker loop with `select` on `ticker.C` and `ctx.Done()` | TASK-2, TASK-3 | TEST-2 (no false notifications across ticks), TEST-3 (new completion detected) | Implemented |
| REQ-2: Configurable Interval | DES-7 (Configurable Interval) | `config.parsePollInterval` -- defaults to 30s, parses `POLL_INTERVAL` env var | TASK-5 | TEST-7 (interval configurable) | Implemented |
| REQ-3: Seed on Startup | DES-2 (Hash Seeding) | `poller.seedKnownHashes` -- fetches all torrents, marks `Progress >= 1.0` as known | TASK-3 | TEST-1 (seeds hashes, no startup notifications) | Implemented |
| REQ-4: Notify All Users | DES-3 (Completion Detection), DES-4 (Notifier Interface), DES-5 (telegramNotifier) | `poller.poll` -- concurrent goroutines per chatID with `WaitGroup`; `telegramNotifier.NotifyCompletion` | TASK-1, TASK-3, TASK-4 | TEST-3 (new completion notified), TEST-8 (multi-user notification) | Implemented |
| REQ-5: Message Includes Name | DES-5 (telegramNotifier) | `telegramNotifier.NotifyCompletion` -- formats `"Download complete!\n\n%s"` with `t.Name` | TASK-4 | TEST-4 (message has torrent name) | Implemented |
| REQ-6: Prune Deleted Hashes | DES-6 (Hash Pruning) | `poller.pruneDeleted` -- removes hashes not in current torrent set | TASK-3 | TEST-5 (pruning works, re-added torrent triggers new notification) | Implemented |
| REQ-7: Graceful Shutdown | DES-1 (Poller Goroutine) | `poller.Run` -- `select` on `ctx.Done()` returns; `defer ticker.Stop()` | TASK-3 | TEST-6 (context cancellation stops poller) | Implemented |

## Source File Coverage

| Source File | Lines | Requirements Addressed |
|-------------|-------|----------------------|
| `internal/poller/poller.go` | 141 | REQ-1, REQ-3, REQ-4, REQ-5, REQ-6, REQ-7 |
| `internal/poller/poller_test.go` | 335 | REQ-1 through REQ-7 (verification) |
| `cmd/bot/main.go` | 97 | REQ-4, REQ-5 (telegramNotifier adapter + wiring) |
| `internal/config/config.go` | 137 | REQ-2 (POLL_INTERVAL parsing) |

## Interface Dependency Map

```
main.go
  |
  +-- telegramNotifier (implements poller.Notifier)
  |     |
  |     +-- tgbotapi.BotAPI.Send
  |
  +-- poller.New(qbt.Client, Notifier, interval, chatIDs)
  |     |
  |     +-- poller.Run(ctx)
  |           |
  |           +-- seedKnownHashes -> qbt.Client.ListTorrents
  |           +-- poll -> qbt.Client.ListTorrents
  |           |         -> Notifier.NotifyCompletion (per chatID)
  |           +-- pruneDeleted
  |
  +-- config.Load() -> PollInterval
```
