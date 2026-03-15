---
title: "Uploading Torrents List — Decisions"
feature_id: "uploading-list"
last_updated: 2026-03-15
---

# Uploading Torrents List — Decisions

## Assumptions

- The client-side post-filtering pattern established by `downloading-list` (`FilterDownloading`, Progress < 1.0) is the correct model for `uploading-list` (`FilterUploading`, Progress == 1.0). No new architectural pattern is required.
- `Progress == 1.0` is a reliable and exhaustive indicator that a torrent has completed downloading, regardless of its current seeding state (active, stalled, paused, forced, checking).
- The qBittorrent instance runs v4.x or v5.x. The progress field is a float64 in the range [0.0, 1.0] in both versions.
- The personal-bot scale (personal use, <<1000 torrents) makes client-side filtering negligible in cost.

## Major Design Choices

### Choice 1: Client-side post-filtering (fetch all, keep Progress == 1.0)

**Decision**: Fetch all torrents via `FilterAll` and filter client-side where `Progress == 1.0`.

**Alternatives considered**:

| Option | Pros | Cons | Decision |
|--------|------|------|----------|
| `filter=completed` (qBittorrent API) | Single filtered API call | Misses `forcedUP`, `checkingUP` states; API semantics may drift between versions | Rejected |
| `filter=seeding` + separate `filter=pausedUP` call | Closer to server-side | Two API calls, more complexity, still may miss edge states | Rejected |
| `FilterAll` + client-side `Progress == 1.0` | Exhaustive, simple, immune to future state additions, matches existing `FilterDownloading` pattern | Slightly more data transferred | **Accepted** |

**Rationale**: Using `Progress == 1.0` as the sole criterion is the most resilient approach. qBittorrent's internal states change across versions (as seen with v5 renaming `/pause` → `/stop`), but the progress field semantics are stable. The `downloading-list` feature already established this pattern; reusing it keeps the codebase consistent.

### Choice 2: Reuse filter char `u` and prefix `up`

**Decision**: Assign `"u"` as the single-character filter code and `"up"` as the pagination callback prefix for `FilterUploading`.

**Rationale**: Existing codes are `a` (active/`act`), `d` (downloading/`dl`). The character `u` and prefix `up` are unambiguous, short, and mnemonic ("up" = uploading). Callback data must stay under Telegram's 64-byte limit; short prefixes preserve headroom for hash suffixes.

### Choice 3: No new interfaces or formatter changes

**Decision**: All existing interfaces (`qbt.Client`, `bot.Sender`, `poller.Notifier`) and formatter functions are reused as-is.

**Rationale**: The feature is purely additive — a new filter constant and routing wires. No formatting change is needed because the uploading list displays the same torrent fields as all other list views.

## Deferred Work

- **Server-side filtering**: If qBittorrent adds a stable API filter that covers all completed seeds (active + paused + forced + checking), the client-side post-filter in `renderTorrentListPage()` could be replaced. Deferred until such an API is available and verified stable.
- **Sorting by seed time or ratio**: Users may eventually want the uploading list sorted by upload ratio or time seeding. Not in scope for this feature; would require formatter and keyboard changes.

## Out-of-Scope

- Changes to `/list`, `/active`, or `/downloading` command behavior.
- New display fields or formatting styles for the uploading list.
- Filtering by upload ratio or seed time thresholds.
- Bulk pause/resume of all seeding torrents.
