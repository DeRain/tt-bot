---
title: Download Completion Notifications -- Design
feature_id: completion-notifications
last_updated: 2026-03-15
---

# Download Completion Notifications -- Design

## Design Decisions

### DES-1: Poller Goroutine

The `Poller.Run` method executes in a dedicated goroutine launched from `main`. It uses `time.NewTicker` for periodic wake-ups and `select` on both the ticker channel and `ctx.Done()` for graceful shutdown. The ticker is deferred-stopped to prevent resource leaks.

**Source:** `internal/poller/poller.go` -- `Run(ctx context.Context)`

### DES-2: Hash Seeding

On the first call to `Run`, before the ticker starts, `seedKnownHashes` fetches all torrents via `ListTorrents(FilterAll)` and populates `knownHashes` with every hash where `Progress >= 1.0`. This prevents spurious notifications for torrents that were already complete before the bot started. Seed errors are logged but non-fatal; the polling loop proceeds regardless.

**Source:** `internal/poller/poller.go` -- `seedKnownHashes(ctx)`

### DES-3: Completion Detection

Each poll cycle fetches the full torrent list. For each torrent with `Progress >= 1.0`, the poller checks `knownHashes` under a mutex lock. If the hash is absent, it is added to `knownHashes` and notifications are dispatched. If already present, the torrent is silently skipped. This ensures exactly-once notification per completion event.

**Source:** `internal/poller/poller.go` -- `poll(ctx)`

### DES-4: Notifier Interface

The `Notifier` interface decouples the poller from the Telegram API:

```go
type Notifier interface {
    NotifyCompletion(ctx context.Context, chatID int64, torrent qbt.Torrent) error
}
```

This allows unit tests to use a mock notifier and makes it possible to swap notification backends without modifying the poller.

**Source:** `internal/poller/poller.go` -- `Notifier` interface

### DES-5: telegramNotifier

The `telegramNotifier` struct in `cmd/bot/main.go` implements `Notifier` by formatting the message as `"Download complete!\n\n<name>"` and sending it via the `tgbotapi.BotAPI.Send` method. It is a thin adapter between the poller's interface and the Telegram library.

**Source:** `cmd/bot/main.go` -- `telegramNotifier.NotifyCompletion`

### DES-6: Hash Pruning

After processing completions, `poll` calls `pruneDeleted(currentHashes)`. This method builds a set of all hashes currently present in qBittorrent and removes from `knownHashes` any entry not in that set. Pruning ensures that if a torrent is deleted and later re-added and completes again, the user receives a fresh notification.

**Source:** `internal/poller/poller.go` -- `pruneDeleted(currentHashes)`

### DES-7: Configurable Interval

The poll interval is loaded from the `POLL_INTERVAL` environment variable by `config.Load()`. The `parsePollInterval` function defaults to 30 seconds when the variable is empty and returns a descriptive error for malformed values. The parsed `time.Duration` is passed to `poller.New` at construction time.

**Source:** `internal/config/config.go` -- `parsePollInterval(raw)`

## Concurrency Model

- The `knownHashes` map is protected by a `sync.Mutex` to allow safe concurrent access.
- Notifications to multiple users are dispatched in parallel goroutines coordinated by `sync.WaitGroup`.
- The `chatIDs` slice is defensively copied at construction time (`poller.New`) to prevent external mutation.

## Requirement-to-Design Mapping

| Requirement | Design Decision(s) |
|-------------|-------------------|
| REQ-1: Periodic Polling | DES-1 (Poller Goroutine) |
| REQ-2: Configurable Interval | DES-7 (Configurable Interval) |
| REQ-3: Seed on Startup | DES-2 (Hash Seeding) |
| REQ-4: Notify All Users | DES-3 (Completion Detection), DES-4 (Notifier Interface), DES-5 (telegramNotifier) |
| REQ-5: Message Includes Name | DES-5 (telegramNotifier) |
| REQ-6: Prune Deleted Hashes | DES-6 (Hash Pruning) |
| REQ-7: Graceful Shutdown | DES-1 (Poller Goroutine) |

## Quality Gates

### Gate 2: Design Gate

- [x] Every DES-N has a source file reference
- [x] Every REQ from spec is mapped to at least one DES
- [x] Requirement-to-Design mapping table is complete
- [x] Concurrency model documented
- [x] No unresolved TODOs

**Harness:**
```bash
# Verify every spec REQ appears in the design mapping table
for req in $(grep -oP 'REQ-\d+' docs/features/completion-notifications/spec.md | sort -u); do
  grep -q "$req" docs/features/completion-notifications/design.md || echo "MISSING: $req"
done
```
