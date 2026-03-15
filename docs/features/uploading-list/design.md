---
title: "Uploading Torrents List — Design"
feature_id: "uploading-list"
status: draft
depends_on_spec: "docs/features/uploading-list/spec.md"
last_updated: 2026-03-15
---

# Uploading Torrents List — Design

## Overview

Extend the existing torrent listing architecture with a new `FilterUploading` filter that uses client-side post-filtering (Progress == 1.0) since no single qBittorrent API filter matches all completed seeds including paused ones.

## Architecture

The existing filter-agnostic architecture (filter constant → handler → formatter) is reused without modification. The only new concept is client-side post-filtering in `renderTorrentListPage()` when the filter is `FilterUploading`, mirroring the pattern established by `FilterDownloading`.

## Data Flow

1. User sends `/uploading`
2. `handleCommand()` calls `sendTorrentPage(ctx, chatID, FilterUploading, 1)`
3. `renderTorrentListPage()` detects `FilterUploading`, fetches with `FilterAll`, then filters for `Progress == 1.0`
4. Pagination, formatting, and keyboard generation proceed unchanged
5. Callback prefix `pg:up:` handles page navigation; filter char `u` handles selection/control callbacks

## Interfaces

No new interfaces. Extends existing:
- `qbt.TorrentFilter` with new constant `FilterUploading`
- `filterCharToFilter()` / `filterToChar()` / `filterCharToPrefix()` with `"u"` ↔ `FilterUploading` ↔ `"up"`

## Data/Storage Impact

None. Stateless.

## Error Handling

Same as existing list commands — API errors shown via callback answer or message text.

## Security Considerations

Same auth whitelist as all other commands.

## Performance Considerations

Fetches all torrents and filters client-side. For a personal bot with <1000 torrents, this is negligible. The `FilterUploading` path follows the identical allocation pattern as `FilterDownloading`.

## Tradeoffs

- **Client-side filtering vs. multiple API calls**: Client-side is simpler and sufficient at this scale. Using the qBittorrent API `filter=completed` would miss `forcedUP` and `checkingUP` states; using `filter=seeding` would miss `pausedUP`. Fetching all and filtering on `Progress == 1.0` is exhaustive and immune to future qBittorrent state additions.
- **Reusing FilterAll**: Fetching all torrents ensures no completed state is missed regardless of qBittorrent's internal state classification.

## Design Items

- **DES-1**: Add `FilterUploading` constant to `qbt.TorrentFilter` in `internal/qbt/types.go`
  - Satisfies: REQ-1, REQ-2
  - Covers: AC-1.1, AC-2.1, AC-2.2

- **DES-2**: Client-side post-filtering in `renderTorrentListPage()` for `FilterUploading` (fetch all, keep Progress == 1.0)
  - Satisfies: REQ-1, REQ-2
  - Covers: AC-1.1, AC-1.2, AC-2.1, AC-2.2

- **DES-3**: Add `/uploading` command dispatch in `handleCommand()` and register in `BotCommands`
  - Satisfies: REQ-5
  - Covers: AC-5.1

- **DES-4**: Add `pg:up:` pagination prefix and `u` filter char mappings in callback routing
  - Satisfies: REQ-3, REQ-4
  - Covers: AC-3.1, AC-3.2, AC-4.1, AC-4.2

## Quality Gates

### Gate 2: Design Gate

This design passes when:
- [x] Every REQ-* from spec.md is addressed by at least one DES-*
- [x] Every AC-* from spec.md is covered by at least one DES-*
- [x] Risks and tradeoffs are documented
- [x] No DES-* exists without a linked REQ-*

## Requirement Mapping

| Design Item | Satisfies | Covers |
|-------------|-----------|--------|
| DES-1 | REQ-1, REQ-2 | AC-1.1, AC-2.1, AC-2.2 |
| DES-2 | REQ-1, REQ-2 | AC-1.1, AC-1.2, AC-2.1, AC-2.2 |
| DES-3 | REQ-5 | AC-5.1 |
| DES-4 | REQ-3, REQ-4 | AC-3.1, AC-3.2, AC-4.1, AC-4.2 |
