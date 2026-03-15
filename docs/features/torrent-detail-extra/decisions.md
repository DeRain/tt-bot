---
title: "Extended Torrent Detail Info — Decisions"
feature_id: "torrent-detail-extra"
last_updated: 2026-03-15
---

# Extended Torrent Detail Info — Decisions

## Assumptions

- The qBittorrent v2 `/api/v2/torrents/info` response always includes `uploaded` and `ratio` keys, even for newly added torrents (where both are `0` and `0.0` respectively).
- `formatBytes` correctly handles `0` and produces `0 B`, matching AC-1.2.
- The existing `FormatTorrentDetail` function has a well-defined line order that allows additive insertion between the `Upload` speed line and the `State` line without ambiguity.

## Major Design Choices

### Choice 1: Reuse `formatBytes` for `Uploaded`

- **Decision**: Use the existing internal `formatBytes` helper rather than a new formatter.
- **Alternatives**: Introduce a dedicated `FormatUploadedBytes` function; inline the formatting directly.
- **Rationale**: `formatBytes` is already used for `Size`, `DLSpeed`, and `UPSpeed`. Reusing it keeps formatting consistent and avoids code duplication.
- **Tradeoff**: Couples `Uploaded` display to the same precision/unit thresholds as all other byte values. Acceptable for this use case.

### Choice 2: Fields added directly to `Torrent` struct (no separate DTO)

- **Decision**: Add `Uploaded` and `Ratio` as fields on the existing `Torrent` struct in `internal/qbt/client.go`.
- **Alternatives**: Create a separate extended struct or a wrapper type.
- **Rationale**: Consistent with all other fields on `Torrent`; avoids an unnecessary type conversion layer and keeps the data model flat and simple.
- **Tradeoff**: The struct grows slightly; acceptable given the small number of fields added.

### Choice 3: `Ratio` formatted to two decimal places

- **Decision**: Render ratio as `fmt.Sprintf("%.2f", t.Ratio)` — always two decimal places.
- **Alternatives**: Strip trailing zeros; use more or fewer decimal places.
- **Rationale**: Two decimal places is the convention in most torrent clients and gives sufficient precision without unnecessary noise (e.g., `2.13` rather than `2.1300000`).
- **Tradeoff**: A ratio of exactly `1` displays as `1.00`, which is slightly verbose. Accepted as consistent and unambiguous.

## Unresolved Questions

None. The feature is small and well-bounded; all design questions were resolved in `design.md`.

## Deferred Work

- `num_seeds` and `num_leechs` fields: considered during spec, deferred as not requested in the current iteration.

## Out-of-Scope

- Changes to list-view formatters (`FormatTorrentList`, `FormatActiveTorrents`, `FormatDownloadingTorrents`).
- Persistent storage of upload statistics.
- Any other new fields beyond `Uploaded` and `Ratio`.
