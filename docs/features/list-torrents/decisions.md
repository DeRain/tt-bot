---
title: Torrent Listing — Decisions
feature_id: list-torrents
last_updated: 2026-03-15
---

# Torrent Listing — Decisions

## Assumptions

- Typical deployments have tens to low hundreds of torrents (personal/small-group use).
- Telegram's 4096-character message limit is a hard constraint that cannot be negotiated.
- Telegram's 64-byte callback data limit is a hard constraint.
- Torrent names from qBittorrent can be arbitrarily long.

## Major Design Choices

1. **5 torrents per page** — Chosen to stay well within the 4096-char message limit even with worst-case entry sizes (40-char name, maximum speed values, full progress bar). At ~120 characters per entry plus header, 5 entries use roughly 650 characters, leaving ample headroom. A higher count (e.g., 10) risks hitting the limit with long names and high speeds.

2. **40-character name truncation** — Balances readability with space efficiency. Torrent names are often very long (release group tags, codec info, resolution). 40 characters preserves enough of the name for identification while keeping each entry compact. Truncation uses rune-aware slicing to handle multi-byte characters correctly, appending "..." (3 ASCII dots) when truncation occurs.

3. **Edit message in place (not send new message)** — Pagination callbacks edit the original message via `editMessageText` rather than sending a new message. This prevents chat clutter: navigating 10 pages would otherwise produce 10 separate messages. The inline keyboard remains attached to the same message throughout navigation.

4. **10-character progress bar** — Uses filled block (U+2588) and empty block (U+2591) characters for a visual progress indicator. 10 characters provides 10% granularity, which is sufficient for at-a-glance status. The bar is followed by the integer percentage for precision.

5. **Client-side pagination (fetch-all-then-slice)** — The qBittorrent API supports offset/limit parameters, but the implementation fetches all matching torrents in a single call and slices in Go. This simplifies the code (no need to track total count separately) and is efficient for the expected torrent counts. The qBittorrent API does not provide a total count in list responses, so server-side pagination would require an extra call anyway.

## Unresolved Questions

None.

## Deferred Work

- Per-torrent actions (pause, resume, delete) from the list view — not planned.
- Configurable page size — not planned; 5 is hardcoded as `TorrentsPerPage`.
- Sort order selection (by name, progress, speed) — not planned.
- Auto-refresh via Telegram message editing on a timer — not planned.

## Out-of-Scope

- Torrent search/filtering beyond all vs active.
- Inline query mode (searching torrents from the Telegram chat input).
- Rich media (thumbnails, file trees) in torrent listings.
