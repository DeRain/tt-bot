---
title: Download Completion Notifications
feature_id: completion-notifications
status: implemented
owner: TODO
source_files:
  - internal/poller/poller.go
  - internal/poller/poller_test.go
  - cmd/bot/main.go
  - internal/config/config.go
last_updated: 2026-03-15
---

# Download Completion Notifications -- Specification

## Overview

A background poller goroutine periodically queries qBittorrent for torrent status changes and sends Telegram notifications to all whitelisted users when a download completes. The poller is stateless across restarts: it seeds known hashes on startup to avoid false notifications, tracks completions in memory, and prunes hashes for deleted torrents.

## Requirements

### REQ-1: Periodic Polling

The system shall poll qBittorrent for torrent status at a regular interval.

**Acceptance Criteria:**
- AC-1.1: The poller calls `ListTorrents` on every tick of a recurring timer.
- AC-1.2: The polling loop runs continuously until the context is cancelled.
- AC-1.3: Errors from `ListTorrents` are logged and do not crash the poller.

### REQ-2: Configurable Poll Interval

The poll interval shall be configurable via the `POLL_INTERVAL` environment variable, with a default of 30 seconds.

**Acceptance Criteria:**
- AC-2.1: When `POLL_INTERVAL` is unset or empty, the interval defaults to 30 seconds.
- AC-2.2: When `POLL_INTERVAL` is set to a valid Go duration string (e.g. `"10s"`, `"1m"`), that duration is used.
- AC-2.3: When `POLL_INTERVAL` is set to an invalid value, config loading returns a descriptive error.

### REQ-3: Seed Known Hashes on Startup

On startup, the poller shall record all currently-completed torrent hashes so that pre-existing completions do not trigger notifications.

**Acceptance Criteria:**
- AC-3.1: Before the first poll tick, the poller fetches all torrents and marks those with `Progress >= 1.0` as known.
- AC-3.2: Torrents that were already complete at startup never produce a notification.
- AC-3.3: If the seed fetch fails, the error is logged and polling still proceeds.

### REQ-4: Notify All Whitelisted Users on Completion

When a torrent newly completes, the system shall send a notification to every whitelisted user.

**Acceptance Criteria:**
- AC-4.1: Each chat ID in the configured whitelist receives exactly one notification per newly completed torrent.
- AC-4.2: Notifications for multiple users are sent concurrently (goroutines with `sync.WaitGroup`).
- AC-4.3: A notification failure for one user is logged but does not prevent notifications to other users.

### REQ-5: Notification Message Includes Torrent Name

Each completion notification message shall include the torrent's name.

**Acceptance Criteria:**
- AC-5.1: The notification text follows the format: `"Download complete!\n\n<torrent name>"`.
- AC-5.2: The `Torrent` struct (including `Name`) is passed to the `Notifier` interface.

### REQ-6: Prune Hashes for Deleted Torrents

After each poll, the system shall remove from the known-hashes set any hash that no longer appears in the qBittorrent torrent list.

**Acceptance Criteria:**
- AC-6.1: When a torrent is deleted from qBittorrent, its hash is removed from `knownHashes` on the next poll cycle.
- AC-6.2: If the same torrent hash reappears as completed after pruning, a new notification is sent.

### REQ-7: Graceful Shutdown

The poller shall stop cleanly when its context is cancelled, without leaking goroutines or blocking indefinitely.

**Acceptance Criteria:**
- AC-7.1: When the context is cancelled, the `Run` method returns promptly.
- AC-7.2: The ticker is stopped on exit (no resource leak).
- AC-7.3: The poller does not block on in-flight notifications after cancellation.

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
grep -c '^### REQ-' docs/features/completion-notifications/spec.md
grep -c '^- AC-' docs/features/completion-notifications/spec.md
# Verify no TODOs in spec body (frontmatter owner: TODO is acceptable)
grep -c 'TODO' <(tail -n +12 docs/features/completion-notifications/spec.md)  # expect 0
```
