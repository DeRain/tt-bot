---
title: Add Torrent - Architectural Decisions
feature_id: add-torrent
last_updated: 2026-03-15
---

# Add Torrent - Architectural Decisions

## ADR-1: In-Memory Pending Storage (Not Persistent)

**Status:** Accepted
**Context:** Pending torrents need to be stored between the user sending a magnet/file and selecting a category. Options: in-memory map, SQLite, Redis, filesystem.
**Decision:** Use an in-memory `map[int64]*PendingTorrent` protected by `sync.Mutex`.
**Rationale:**
- The bot is designed to be stateless and single-instance. Pending state is short-lived (5-minute TTL) and low-volume (one entry per active user).
- No external dependency is needed. A database or cache would add operational complexity disproportionate to the problem.
- Loss of pending state on restart is acceptable -- users simply resend the link/file. This aligns with the project's intentional stateless design.
**Consequences:** Pending torrents are lost on restart. If the bot is scaled horizontally, users would need sticky sessions (not planned).

## ADR-2: One Pending Torrent Per User

**Status:** Accepted
**Context:** Users may send multiple magnet links or files before selecting a category for any of them.
**Decision:** The pending map allows only one entry per chat ID. A new submission overwrites the previous one.
**Rationale:**
- Simplifies the UX: the category keyboard always refers to the most recently submitted torrent, avoiding ambiguity.
- Simplifies the implementation: no need for a selection step to disambiguate which pending torrent a category applies to.
- Real-world usage pattern is sequential: send one torrent, pick a category, repeat.
**Consequences:** If a user sends two magnets quickly before selecting a category, the first one is silently replaced. This is documented behavior.

## ADR-3: 5-Minute TTL for Pending Entries

**Status:** Accepted
**Context:** Pending entries consume memory and become stale if the user abandons the flow.
**Decision:** Entries expire after 5 minutes (`pendingTTL = 5 * time.Minute`).
**Rationale:**
- 5 minutes is long enough for a user to read category options and tap a button, even on a slow connection.
- Short enough to prevent memory accumulation from abandoned flows.
- The value is a constant, easily tunable if user feedback indicates it should be longer.
**Consequences:** Users who take longer than 5 minutes to select a category will see an "expired" error and must resend.

## ADR-4: Background Ticker Cleanup (Not On-Demand)

**Status:** Accepted
**Context:** Expired pending entries need to be cleaned up. Options: (a) check TTL only when `takePending` is called, (b) background goroutine on a timer.
**Decision:** A background goroutine runs every 1 minute (`cleanupInterval = 1 * time.Minute`) and evicts all expired entries.
**Rationale:**
- On-demand cleanup (option a) only removes entries when the same user triggers a callback. Entries from users who abandon the flow entirely would never be cleaned up, causing a slow memory leak.
- A 1-minute interval keeps memory bounded regardless of user behavior.
- The goroutine is lightweight (one mutex acquisition per tick) and exits cleanly via context cancellation.
**Consequences:** There is a brief window (up to 1 minute) where an entry may persist slightly beyond its 5-minute TTL. This is acceptable -- the actual TTL enforcement on callback is done by `takePending` returning nil for entries that no longer exist.

## ADR-5: Edit Original Message for Confirmation (Not New Message)

**Status:** Accepted
**Context:** After a torrent is added, the user needs feedback. Options: (a) send a new message, (b) edit the category selection message in place.
**Decision:** Edit the category selection message to replace the inline keyboard with a confirmation string.
**Rationale:**
- Editing keeps the chat clean. A new message per torrent would clutter the conversation, especially for power users adding many torrents.
- The inline keyboard becomes useless after selection. Editing removes it and replaces it with the outcome, creating a clear audit trail.
- Telegram's `editMessageText` API is purpose-built for this pattern.
- Error messages also edit the original, so the user sees the failure context next to where they interacted.
**Consequences:** The original "Select category" prompt text is lost. This is acceptable since the category keyboard is no longer relevant after selection. Uses `sender.Request()` instead of `sender.Send()` because Telegram returns a boolean for edit operations.

## Deferred Decisions

### Multiple Pending Torrents Per User

**Status:** Deferred
**Context:** Some users may want to queue several torrents and assign categories in bulk.
**Rationale for deferral:** Adds significant UX complexity (need to show which torrent each category applies to) and implementation complexity (list management, disambiguation). Current sequential flow is simple and sufficient. Will revisit if user feedback indicates demand.

### Persistent Pending State

**Status:** Deferred
**Context:** Pending state could be persisted to survive bot restarts.
**Rationale for deferral:** The project's architecture is intentionally stateless. The 5-minute TTL means the maximum data loss on restart is negligible. Adding persistence would require a storage dependency that contradicts the design philosophy. Will revisit only if the bot moves to a multi-instance deployment model.
