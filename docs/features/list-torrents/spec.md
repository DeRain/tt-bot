---
title: Torrent Listing
feature_id: list-torrents
status: implemented
owner: TODO
source_files:
  - internal/bot/handler.go
  - internal/bot/callback.go
  - internal/formatter/format.go
  - internal/qbt/http.go
last_updated: 2026-03-15
---

# Torrent Listing — Specification

## Overview

Paginated torrent listing displayed via Telegram inline messages. Users can view all torrents or only active ones, navigate between pages using inline keyboard buttons, and see per-torrent details including name, progress, speeds, and state.

## Problem Statement

Users need visibility into their qBittorrent download queue directly from Telegram. The listing must fit within Telegram's message constraints (4096 UTF-8 characters, 64-byte callback data) and support navigation when the torrent count exceeds a single page.

## Goals

- Show all or active torrents on demand via bot commands
- Paginate results to keep messages readable and within Telegram limits
- Display meaningful per-torrent details at a glance
- Allow seamless page navigation via inline keyboard buttons

## Non-Goals

- Torrent management actions (pause, resume, delete) from the list view
- Real-time auto-refresh of the listing
- Sorting or filtering beyond all/active
- Search within the torrent list

## Scope

Covers the `/list` and `/active` commands, the pagination callback handler, and all formatting logic for torrent list messages and navigation keyboards.

## Requirements

- **REQ-1**: `/list` MUST display all torrents from qBittorrent.
- **REQ-2**: `/active` MUST display only active (downloading/uploading) torrents.
- **REQ-3**: Results MUST be paginated with 5 torrents per page.
- **REQ-4**: An inline keyboard MUST provide Prev / Page N/M / Next navigation.
- **REQ-5**: Each torrent entry MUST show name, progress bar, download/upload speeds, and state.
- **REQ-6**: The formatted message MUST NOT exceed 4096 UTF-8 characters.
- **REQ-7**: Torrent names MUST be truncated to 40 characters maximum, with "..." appended when truncated.
- **REQ-8**: When no torrents match the filter, the message MUST read "No torrents found."

## Acceptance Criteria

- **AC-1.1**: Sending `/list` calls `ListTorrents` with `FilterAll` and returns a formatted message with all torrents.
- **AC-2.1**: Sending `/active` calls `ListTorrents` with `FilterActive` and returns a formatted message with active torrents only.
- **AC-3.1**: A list of 7 torrents produces 2 pages (5 + 2), and the page header shows the correct page count.
- **AC-4.1**: On page 1 of a multi-page list, the keyboard shows "Page 1/N" and "Next >>" but no "Prev" button.
- **AC-4.2**: On the last page, the keyboard shows "<< Prev" and "Page N/N" but no "Next" button.
- **AC-4.3**: On a middle page, the keyboard shows "<< Prev", "Page K/N", and "Next >>".
- **AC-4.4**: Pressing a pagination button edits the existing message in place (no new message sent).
- **AC-5.1**: Each torrent entry in the message contains the torrent name, a 10-character progress bar with percentage, download speed, upload speed, and state string.
- **AC-6.1**: A page with 5 torrents using worst-case 40-character names and maximum speed values stays under 4096 characters.
- **AC-6.2**: If adding the next torrent entry would exceed the limit, it is silently omitted.
- **AC-7.1**: A torrent name longer than 40 characters is displayed as the first 37 characters followed by "...".
- **AC-8.1**: When `ListTorrents` returns an empty slice, the message text is exactly "No torrents found."

## Quality Gates

### Gate 1: Spec Gate

This spec passes when:
- [x] All requirements are clear and unambiguous
- [x] All acceptance criteria are testable
- [x] Scope and non-goals are defined
- [x] No unresolved open questions block implementation
- [x] At least one AC exists per requirement

**Harness check command:**
```bash
# Verify spec completeness
test $(grep -c "^\- \*\*REQ-" docs/features/list-torrents/spec.md) -gt 0
test $(grep -c "^\- \*\*AC-" docs/features/list-torrents/spec.md) -gt 0
test $(grep -c "TODO:" docs/features/list-torrents/spec.md) -eq 0
```

## Risks

- If qBittorrent returns an error, the user sees an error message instead of the listing (acceptable; error is displayed).
- Very large torrent counts could produce high page numbers, but callback data for pagination (`pg:all:9999`) stays well under the 64-byte Telegram limit.

## Open Questions

None.
