---
title: Torrent Listing — Design
feature_id: list-torrents
status: implemented
depends_on_spec: docs/features/list-torrents/spec.md
last_updated: 2026-03-15
---

# Torrent Listing — Design

## Overview

The torrent listing feature spans three packages: `qbt` (data fetching), `formatter` (message rendering and keyboard construction), and `bot` (command dispatch, pagination callbacks, Telegram API interaction). All pagination is performed in Go after a single API call fetches all matching torrents.

## Architecture

- **DES-1**: Command dispatch in `handleCommand` — routes `/list` to `sendTorrentPage(ctx, chatID, FilterAll, 1)` and `/active` to `sendTorrentPage(ctx, chatID, FilterActive, 1)`.
  - Satisfies: REQ-1, REQ-2
  - Covers: AC-1.1, AC-2.1

- **DES-2**: Page rendering in `sendTorrentPage` — calls `qbt.Client.ListTorrents` with the appropriate filter, computes the page slice (`offset` to `offset + TorrentsPerPage`), and delegates formatting to `formatter.FormatTorrentList`. The result is sent as a new Telegram message with the pagination keyboard attached.
  - Satisfies: REQ-1, REQ-2, REQ-3, REQ-5, REQ-6, REQ-8
  - Covers: AC-1.1, AC-2.1, AC-3.1, AC-5.1, AC-6.1, AC-8.1

- **DES-3**: Pagination keyboard in `formatter.PaginationKeyboard` — builds a single-row inline keyboard with up to 3 buttons: "<< Prev" (omitted on page 1), "Page K/N" (callback data "noop"), "Next >>" (omitted on last page). Callback data format: `pg:<filterPrefix>:<page>` where filterPrefix is "all" or "act".
  - Satisfies: REQ-4
  - Covers: AC-4.1, AC-4.2, AC-4.3

- **DES-4**: Pagination callback in `handlePaginationCallback` — parses callback data (`pg:all:<page>` or `pg:act:<page>`), re-fetches torrents, computes the requested page slice, formats the message, and edits the original message in place via `editMessageText`. Answers the callback query to dismiss the spinner.
  - Satisfies: REQ-3, REQ-4
  - Covers: AC-3.1, AC-4.4

- **DES-5**: Message size guard in `FormatTorrentList` — iterates over the page's torrents, building the message string incrementally. Before appending each entry, checks whether doing so would exceed `MaxMessageLength - 1` (4095 characters). Entries that would overflow are silently dropped.
  - Satisfies: REQ-6
  - Covers: AC-6.1, AC-6.2

- **DES-6**: Formatting helpers — `truncateName` (rune-aware truncation to 40 chars with "..." suffix), `FormatProgress` (10-char bar using filled/empty block chars, clamped to [0,1]), `FormatSpeed` (B/s, KB/s, or MB/s based on magnitude).
  - Satisfies: REQ-5, REQ-7
  - Covers: AC-5.1, AC-7.1

## Data Flow

1. User sends `/list` or `/active`.
2. `handleCommand` calls `sendTorrentPage` with `FilterAll` or `FilterActive` and page 1.
3. `sendTorrentPage` calls `qbt.Client.ListTorrents(ctx, ListOptions{Filter: filter})` — fetches all matching torrents in one API call.
4. Page slice is computed: `offset = (page - 1) * 5`, `end = min(offset + 5, len(all))`.
5. `formatter.FormatTorrentList(torrents[offset:end], page, totalPages)` renders the message.
6. `formatter.PaginationKeyboard(page, totalPages, filterPrefix)` builds the inline keyboard.
7. Message is sent via `Sender.Send`.
8. On pagination callback, `handlePaginationCallback` repeats steps 3-6 and edits the message via `editMessageText`.

## Interfaces

- `qbt.Client.ListTorrents(ctx, ListOptions) ([]Torrent, error)` — returns all matching torrents; pagination is done client-side.
- `formatter.FormatTorrentList(torrents, page, totalPages) string` — pure function, no side effects.
- `formatter.PaginationKeyboard(currentPage, totalPages, filterPrefix) Keyboard` — pure function.
- `formatter.FormatSpeed(bytesPerSec int64) string` — pure function.
- `formatter.FormatProgress(progress float64) string` — pure function.

## Data/Storage Impact

None. No persistent state. The full torrent list is fetched on every command/callback.

## Error Handling

- `ListTorrents` error: sends error text to the user ("Error fetching torrents: ...").
- Pagination callback with invalid page number: answers callback with "Invalid page."
- Unknown callback prefix: answers callback with "Unknown action."
- `Sender.Send` / `editMessageText` error: logged, not surfaced to user.

## Security Considerations

- Auth check occurs before command dispatch; only whitelisted users reach listing logic.
- No user-controlled data is injected into messages without formatting (torrent names come from qBittorrent, not user input).

## Performance Considerations

- All torrents are fetched in a single API call. For typical personal use (tens to low hundreds of torrents) this is efficient.
- No caching: each command/callback triggers a fresh API call, ensuring up-to-date data.
- Message formatting is O(n) where n is the page size (constant 5).

## Tradeoffs

- Fetch-all-then-slice vs server-side pagination: chose client-side slicing for simplicity. The qBittorrent API supports offset/limit but the full list is small enough that one call is efficient.
- Edit-in-place vs new message: chose editing to avoid chat clutter.
- Fixed 5-per-page vs configurable: chose simplicity; 5 fits comfortably within 4096 chars.

## Risks

- If qBittorrent has thousands of torrents, the fetch-all approach could become slow (unlikely for personal use).

## Requirement Mapping

| Design Item | Satisfies | Covers |
|-------------|-----------|--------|
| DES-1 | REQ-1, REQ-2 | AC-1.1, AC-2.1 |
| DES-2 | REQ-1, REQ-2, REQ-3, REQ-5, REQ-6, REQ-8 | AC-1.1, AC-2.1, AC-3.1, AC-5.1, AC-6.1, AC-8.1 |
| DES-3 | REQ-4 | AC-4.1, AC-4.2, AC-4.3 |
| DES-4 | REQ-3, REQ-4 | AC-3.1, AC-4.4 |
| DES-5 | REQ-6 | AC-6.1, AC-6.2 |
| DES-6 | REQ-5, REQ-7 | AC-5.1, AC-7.1 |

## Quality Gates

### Gate 2: Design Gate

This design passes when:
- [x] Every REQ-* from spec.md is addressed by at least one DES-*
- [x] Every AC-* from spec.md is covered by at least one DES-*
- [x] Risks and tradeoffs are documented
- [x] No DES-* exists without a linked REQ-*

**Harness check command:**
```bash
spec_reqs=$(grep -oP 'REQ-\d+' docs/features/list-torrents/spec.md | sort -u)
design_reqs=$(grep -oP 'REQ-\d+' docs/features/list-torrents/design.md | sort -u)
comm -23 <(echo "$spec_reqs") <(echo "$design_reqs")  # should be empty
```
