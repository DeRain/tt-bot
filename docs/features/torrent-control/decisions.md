---
title: "Torrent Control — Decisions"
feature_id: "torrent-control"
last_updated: 2026-03-15
---

# Torrent Control — Decisions

## Assumptions

- Torrent info-hashes are always exactly 40 hexadecimal characters (SHA-1). qBittorrent v2 hashes (SHA-256, 64 chars) are not yet in use by the target deployment.
- qBittorrent v5+ renamed the pause/resume endpoints: `/api/v2/torrents/stop` (was `/pause`) and `/api/v2/torrents/start` (was `/resume`). Both accept `hashes` as a pipe-separated string in a form-encoded POST body. This was discovered during integration testing against qBittorrent v5.1.4 (WebAPI v2.11.4).
- Users interact with the bot one at a time per chat — there is no concurrent multi-user concern for the same chat ID.
- The bot's Telegram user whitelist (`TELEGRAM_ALLOWED_USERS`) gates all callback interactions, so no additional auth is needed for pause/resume.

## Major Design Choices

### Choice 1: Stateless callback encoding vs. in-memory state

- **Decision**: Encode all navigation context (filter, page, hash) directly in callback data.
- **Alternatives**: Store return context in an in-memory map keyed by chat ID (similar to `pendingTorrents`).
- **Rationale**: Avoids additional state management, expiry logic, and mutex coordination. Callback data fits well within the 64-byte Telegram limit (worst case: 49 bytes). Consistent with the project's intentionally stateless philosophy.
- **Tradeoff**: Slightly more complex callback parsing, but no server-side state to manage or lose on restart.

### Choice 2: Single-character filter codes

- **Decision**: Use `a` (all) and `c` (active) in torrent-control callbacks instead of `all`/`act`.
- **Alternatives**: Reuse existing `all`/`act` prefixes.
- **Rationale**: Saves 2-4 bytes per callback, providing more headroom within the 64-byte limit. The mapping is trivial and isolated in helper functions.
- **Tradeoff**: Introduces a second encoding convention alongside the existing `all`/`act` pagination prefixes. Documented in DES-11.

### Choice 3: Re-fetch on every action vs. caching

- **Decision**: Re-fetch the full torrent list from qBittorrent on every select, pause, resume, and back action.
- **Alternatives**: Cache the torrent list in memory for a short TTL.
- **Rationale**: Ensures the user always sees the latest state. Caching adds complexity (invalidation, memory management) for minimal benefit in a single-user bot context. qBittorrent API responses are fast on a local network.
- **Tradeoff**: More API calls to qBittorrent, but acceptable given the low request volume.

### Choice 4: Find torrent by hash via full list scan

- **Decision**: Fetch all torrents matching the filter and linearly scan for the hash, rather than adding a "get torrent by hash" API method.
- **Alternatives**: Add `GetTorrent(ctx, hash) (Torrent, error)` to `qbt.Client` using `/api/v2/torrents/info?hashes=<hash>`.
- **Rationale**: The existing `ListTorrents` endpoint already supports filtering and returns all needed data. Adding a dedicated endpoint creates more API surface for minimal benefit. The linear scan is negligible for typical torrent counts (< 1000).
- **Tradeoff**: Slightly less efficient for very large torrent lists, but avoids interface bloat.

### Choice 5: IsPaused as negative match

- **Decision**: `IsPaused` returns true only for `pausedDL` and `pausedUP`. All other states are considered "pausable" (showing the Pause button).
- **Alternatives**: Enumerate all pausable states explicitly.
- **Rationale**: qBittorrent has many states (downloading, seeding, stalledDL, stalledUP, etc.) and may add more. Matching only the two paused states is simpler and forward-compatible. States like "error" or "missingFiles" will show a Pause button, which is harmless (the API call is a no-op).
- **Tradeoff**: The Pause button may appear for states where pausing has no effect, but this causes no harm.

## Unresolved Questions

None.

## Deferred Work

- **Delete torrent**: Separate feature with its own confirmation flow and safety checks.
- **Batch operations**: Pause/resume all torrents at once. Could be added as buttons on the list view.
- **Torrent priority management**: Queue position, bandwidth limits — separate feature.
- **Refresh button on detail view**: Auto-refresh or manual refresh to see updated progress. Could be added as a simple callback that re-renders the detail view.

## Out-of-Scope

- File-level management within a torrent (selecting which files to download).
- Sorting or advanced filtering options beyond all/active.
- Notification when a paused torrent completes after resuming (covered by existing completion-notifications feature).
- qBittorrent v2 hash support (SHA-256, 64 chars) — would exceed callback data limits and requires a different approach.
