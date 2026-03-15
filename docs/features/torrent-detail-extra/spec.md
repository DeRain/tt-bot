---
title: "Extended Torrent Detail Info"
feature_id: torrent-detail-extra
status: draft
last_updated: 2026-03-15
---

# Spec: Extended Torrent Detail Info

## Overview

The torrent detail view currently omits upload totals and share ratio. This feature adds `Uploaded` (total bytes uploaded) and `Ratio` (share ratio) to the detail view so users can assess seeding performance at a glance.

## Requirements

### REQ-1: Display uploaded amount

The torrent detail view MUST display the total bytes uploaded in a human-readable format (e.g., `3.2 GB`), using the same byte-formatting logic applied to `Size`.

#### Acceptance Criteria

- **AC-1.1:** `FormatTorrentDetail` output includes a line `Uploaded: <human-readable size>` when `Uploaded > 0`.
- **AC-1.2:** `FormatTorrentDetail` output includes `Uploaded: 0 B` when `Uploaded == 0`.
- **AC-1.3:** The `Uploaded` line appears between the `Upload` speed line and the `State` line.

### REQ-2: Display share ratio

The torrent detail view MUST display the share ratio formatted to two decimal places (e.g., `Ratio: 2.13`).

#### Acceptance Criteria

- **AC-2.1:** `FormatTorrentDetail` output includes a line `Ratio: <value>` formatted to two decimal places.
- **AC-2.2:** When ratio is `0.00` the line still appears (e.g., freshly added torrent with no upload yet).
- **AC-2.3:** The `Ratio` line appears immediately after the `Uploaded` line.

### REQ-3: Fields populated from qBittorrent API

The `Torrent` struct MUST expose `Uploaded` and `Ratio` fields that are correctly populated when deserialising the `/api/v2/torrents/info` JSON response.

#### Acceptance Criteria

- **AC-3.1:** `Torrent.Uploaded` is populated from the `uploaded` JSON key (int64).
- **AC-3.2:** `Torrent.Ratio` is populated from the `ratio` JSON key (float64).
- **AC-3.3:** An integration test confirms both fields are non-zero for a seeding torrent.

## Out of Scope

- `num_seeds` and `num_leechs` fields (deferred; not requested).
- Persistent storage of upload statistics.
- Changes to list-view formatting (`FormatTorrentList`, `FormatActiveTorrents`, `FormatDownloadingTorrents`).

## Gate 1 Check

- [x] All requirements are clear and unambiguous.
- [x] Each requirement has at least one testable acceptance criterion.
- [x] Scope boundaries are explicit.
- [x] No hidden dependencies on unreleased features.
