---
title: Download Completion Notifications -- Implementation Plan
feature_id: completion-notifications
last_updated: 2026-03-15
---

# Download Completion Notifications -- Implementation Plan

## Task Breakdown

### TASK-1: Define Notifier Interface

**File:** `internal/poller/poller.go`
**Status:** Complete

Define the `Notifier` interface with a single method:

```go
type Notifier interface {
    NotifyCompletion(ctx context.Context, chatID int64, torrent qbt.Torrent) error
}
```

This decouples the poller from the Telegram SDK and enables mock-based testing.

**Maps to:** REQ-4 (Notify All Users)

---

### TASK-2: Implement Poller Struct and Constructor

**File:** `internal/poller/poller.go`
**Status:** Complete

Create the `Poller` struct holding:
- `qbt.Client` for fetching torrent state
- `Notifier` for sending completion messages
- `interval` (`time.Duration`) for poll frequency
- `chatIDs` (`[]int64`) for notification recipients (defensively copied)
- `knownHashes` (`map[string]bool`) for tracking seen completions
- `sync.Mutex` for thread-safe hash access

Provide a `New` constructor that initialises all fields.

**Maps to:** REQ-1, REQ-2, REQ-4

---

### TASK-3: Implement Completion Detection and Notification Loop

**File:** `internal/poller/poller.go`
**Status:** Complete

Implement the `Run` method:
1. Call `seedKnownHashes` to populate initial state.
2. Start a `time.NewTicker` at the configured interval.
3. Loop on `select` between `ticker.C` (call `poll`) and `ctx.Done()` (return).
4. Defer `ticker.Stop()` for cleanup.

Implement the `poll` method:
1. Fetch all torrents via `ListTorrents(FilterAll)`.
2. Build `currentHashes` set from the response.
3. For each torrent with `Progress >= 1.0` not in `knownHashes`:
   - Add to `knownHashes`.
   - Send concurrent notifications to all `chatIDs` via `WaitGroup`.
4. Call `pruneDeleted(currentHashes)`.

**Maps to:** REQ-1, REQ-3, REQ-4, REQ-5, REQ-6, REQ-7

---

### TASK-4: Implement telegramNotifier

**File:** `cmd/bot/main.go`
**Status:** Complete

Create a `telegramNotifier` struct wrapping `*tgbotapi.BotAPI`. Implement `NotifyCompletion` to format the message as `"Download complete!\n\n<name>"` and send via `botAPI.Send`.

Wire the notifier into `main`:
1. Construct `telegramNotifier{bot: botAPI}`.
2. Pass it to `poller.New` alongside the qBittorrent client, poll interval, and allowed-user chat IDs.
3. Launch `p.Run(ctx)` in a goroutine.

**Maps to:** REQ-4, REQ-5

---

### TASK-5: Add Poll Interval Configuration

**File:** `internal/config/config.go`
**Status:** Complete

Add `PollInterval time.Duration` to the `Config` struct. Implement `parsePollInterval`:
- Empty string returns `30 * time.Second` (default).
- Non-empty string parsed via `time.ParseDuration`; return error on invalid input.

Integrate into `config.Load()` by reading `os.Getenv("POLL_INTERVAL")`.

**Maps to:** REQ-2

---

### TASK-6: Write Unit Tests

**File:** `internal/poller/poller_test.go`
**Status:** Complete

Implement mocks:
- `mockQBT`: thread-safe fake `qbt.Client` with swappable torrent list.
- `mockNotifier`: records all `NotifyCompletion` calls for assertion.

Test cases:
1. **Seeds known hashes** -- pre-existing completed torrents produce zero notifications.
2. **New completion notified** -- incomplete-to-complete transition triggers notification.
3. **No duplicate notifications** -- completed torrent across multiple polls sends only one notification.
4. **Pruning works** -- deleted torrent hash is removed; re-added completion triggers new notification.
5. **Multi-user notification** -- all chat IDs receive notifications for a single completion.

Helper: `waitFor(t, timeout, cond)` polls a condition to handle asynchronous poller behaviour in tests.

**Maps to:** REQ-1 through REQ-7

## Quality Gates

### Gate 3: Plan Gate

- [x] Every TASK has a status (Complete / In Progress / Not Started)
- [x] Every TASK maps to at least one REQ or DES
- [x] Task-to-Requirement mapping table is complete
- [x] All tasks have file references
- [x] No unresolved TODOs in task descriptions

**Harness:**
```bash
# Task count must equal verification count
TASK_COUNT=$(grep -c '^### TASK-' docs/features/completion-notifications/plan.md)
MAP_COUNT=$(grep -c '^| TASK-' docs/features/completion-notifications/plan.md)
echo "Tasks: $TASK_COUNT, Mapped: $MAP_COUNT"
test "$TASK_COUNT" -eq "$MAP_COUNT" && echo "PASS" || echo "FAIL: task count mismatch"
```

**Iterative Harness Loop Protocol:**
1. Run all harness commands from Gates 1-3.
2. If any check fails, fix the failing document and re-run.
3. Repeat until all gates pass with zero failures.
4. Only then proceed to implementation.

## Task-to-Requirement Mapping

| Task | Requirements Covered |
|------|---------------------|
| TASK-1 | REQ-4 |
| TASK-2 | REQ-1, REQ-2, REQ-4 |
| TASK-3 | REQ-1, REQ-3, REQ-4, REQ-5, REQ-6, REQ-7 |
| TASK-4 | REQ-4, REQ-5 |
| TASK-5 | REQ-2 |
| TASK-6 | REQ-1, REQ-2, REQ-3, REQ-4, REQ-5, REQ-6, REQ-7 |
