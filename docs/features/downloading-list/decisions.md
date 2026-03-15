---
title: "Downloading Torrents List — Decisions"
feature_id: "downloading-list"
last_updated: 2026-03-15
---

# Downloading Torrents List — Decisions

## Assumptions

- The bot serves a single user or small group; fetching all torrents for client-side filtering is acceptable
- qBittorrent's `active` filter includes seeding torrents, making it unsuitable for "downloads only"

## Major Design Choices

### Client-side filtering vs. qBittorrent API filter

- **Decision**: Fetch with `FilterAll` and filter client-side for `Progress < 1.0`
- **Alternatives**: (a) Use qBittorrent `downloading` API filter — misses paused downloads; (b) Two API calls (`downloading` + `paused`) — misses edge states like `stalledDL`, `metaDL`, `allocating`
- **Rationale**: Client-side filtering is simplest and catches all non-completed states regardless of qBittorrent's state classification
- **Tradeoff**: Slightly more data transferred per request; negligible at personal bot scale

### Filter char `d` and prefix `dw`

- **Decision**: Use `d` as filter char and `dw` as pagination prefix
- **Alternatives**: `dl`, `dn` — all fit within 64-byte callback limit
- **Rationale**: `d` is shortest unique char; `dw` follows the 2-char prefix pattern of `act`

## Unresolved Questions

None.

## Deferred Work

None.

## Out-of-Scope

- Upload-only / seeding list view
- Filter by download speed threshold
