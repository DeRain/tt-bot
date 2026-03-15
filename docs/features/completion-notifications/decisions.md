---
title: Download Completion Notifications -- Design Decisions
feature_id: completion-notifications
last_updated: 2026-03-15
---

# Download Completion Notifications -- Design Decisions

## ADR-1: Polling vs Webhooks

**Status:** Accepted
**Context:** qBittorrent can notify of completion events through either periodic API polling or RSS/webhook mechanisms.

**Decision:** Use periodic polling via `ListTorrents(FilterAll)`.

**Rationale:**
- qBittorrent's Web API v2 does not provide a native webhook or push-notification mechanism for torrent completion events.
- Polling is simple to implement, requires no additional infrastructure, and works with any qBittorrent installation out of the box.
- The default 30-second interval keeps API load negligible while providing timely notifications.

**Trade-offs:**
- Notifications are delayed by up to one poll interval after actual completion.
- Each poll fetches the entire torrent list, which is acceptable for typical home-server workloads but could be optimised with filtering if the list grows very large.

---

## ADR-2: In-Memory Hash Tracking

**Status:** Accepted
**Context:** The poller needs to track which torrents have already triggered a notification to avoid duplicates.

**Decision:** Use an in-memory `map[string]bool` (`knownHashes`) protected by a `sync.Mutex`.

**Rationale:**
- The bot is designed to be stateless across restarts (CLAUDE.md: "All state is lost on restart -- this is intentional").
- An in-memory map is the simplest possible tracking mechanism with zero external dependencies.
- The mutex provides safe concurrent access from the poll goroutine without introducing a channel-based design that would complicate the code.

**Trade-offs:**
- On restart, the bot loses knowledge of previously notified torrents. The seed strategy (ADR-3) mitigates false re-notifications.
- Memory usage grows linearly with the number of tracked hashes, but pruning (ADR-5) prevents unbounded growth.

---

## ADR-3: Seed Strategy -- Mark All Completed on Startup

**Status:** Accepted
**Context:** After a restart, the poller has an empty `knownHashes` map and would otherwise send notifications for every currently-completed torrent.

**Decision:** Before the first poll tick, call `seedKnownHashes` to fetch all torrents and mark every torrent with `Progress >= 1.0` as known.

**Rationale:**
- Users do not want a flood of notifications for torrents that completed hours or days ago.
- The seed pass uses the same `ListTorrents` call as regular polling, adding no new API surface.
- If the seed call fails (e.g. qBittorrent is temporarily unreachable), the error is logged and polling proceeds. This means some false notifications may occur in that edge case, but the alternative (refusing to start) is worse for availability.

**Trade-offs:**
- A torrent that completes during the narrow window between the seed fetch and the first poll tick could be missed or double-notified. In practice, this window is sub-second and the risk is negligible.

---

## ADR-4: Notify All Whitelisted Users

**Status:** Accepted
**Context:** When a torrent completes, the system must decide which users to notify.

**Decision:** Send a notification to every chat ID in the `AllowedUsers` whitelist.

**Rationale:**
- tt-bot is a shared household tool. All whitelisted users are interested in all downloads.
- The `AllowedUsers` list already exists for authentication purposes, so reusing it for notification targeting avoids introducing a separate "subscribers" concept.
- Notifications are sent concurrently (one goroutine per user) with a `WaitGroup`, so a slow or failing notification to one user does not block others.

**Trade-offs:**
- No per-user subscription or mute capability. If a user does not want notifications, they must be removed from the whitelist entirely (which also removes bot access).
- Could be extended in the future with a `/mute` command and a separate subscriber list if needed.

---

## ADR-5: Hash Pruning for Deleted Torrents

**Status:** Accepted
**Context:** When a torrent is deleted from qBittorrent, its hash remains in `knownHashes` indefinitely, wasting memory and preventing re-notification if the same torrent is added again.

**Decision:** After each poll cycle, call `pruneDeleted` to remove from `knownHashes` any hash that is no longer present in the current torrent list.

**Rationale:**
- Prevents unbounded memory growth over the bot's lifetime.
- Enables the correct behaviour when a user deletes a torrent and later re-adds it: the second completion triggers a fresh notification.
- Pruning is O(n) in the number of known hashes and runs once per poll cycle, adding negligible overhead.

**Trade-offs:**
- If a torrent is temporarily absent from the API response (e.g. due to a transient qBittorrent bug), its hash would be pruned and a re-notification could fire. This is considered extremely unlikely and acceptable.
