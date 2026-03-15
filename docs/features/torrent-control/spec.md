---
title: "Torrent Control"
feature_id: "torrent-control"
status: draft
owner: TODO
source_files: []
last_updated: 2026-03-15
---

# Torrent Control — Specification

## Overview

Add the ability for users to select individual torrents from any paginated list view, see a detail view with torrent metadata, pause or resume them via inline keyboard buttons, and navigate back to the originating list page.

## Problem Statement

Users can list torrents but cannot interact with them individually. There is no way to pause, resume, or inspect a single torrent from within the bot. Users must open the qBittorrent WebUI for any torrent management beyond adding new downloads.

## Goals

- Allow users to select any torrent from `/list` or `/active` views
- Display a detailed single-torrent view with full metadata
- Provide pause/resume actions directly in the Telegram conversation
- Preserve navigation context so users can return to the same list page
- Work with all current and future torrent list views

## Non-Goals

- Delete torrent functionality (future feature)
- Torrent priority/queue management
- File-level management within a torrent
- Sorting or advanced filtering options
- Batch operations (pause/resume multiple at once)

## Scope

This feature covers: individual torrent selection from any list, detail view rendering, pause/resume API integration, and back-to-list navigation. It extends the existing list-torrents and add-torrent features but does not modify their core behavior.

## Requirements

- **REQ-1**: Each torrent in a list view MUST be selectable via an inline keyboard button.
- **REQ-2**: Selecting a torrent MUST display a detail view showing: full name, size, progress bar with percentage, download speed, upload speed, state, and category.
- **REQ-3**: The detail view MUST show a **Pause** button when the torrent is in a pausable state (downloading, seeding, stalledDL, stalledUP, queuedDL, queuedUP, forcedDL, forcedUP, uploading, allocating, metaDL, forcedMetaDL, checkingDL, checkingUP, checkingResumeData, moving).
- **REQ-4**: The detail view MUST show a **Resume** button when the torrent is in a paused state (pausedDL, pausedUP).
- **REQ-5**: Pressing Pause MUST pause the torrent in qBittorrent and refresh the detail view to show the Resume button.
- **REQ-6**: Pressing Resume MUST resume the torrent in qBittorrent and refresh the detail view to show the Pause button.
- **REQ-7**: The detail view MUST include a **Back to list** button that returns the user to the same page and filter they came from.
- **REQ-8**: All callback data MUST fit within the 64-byte Telegram limit.
- **REQ-9**: The detail view message MUST NOT exceed 4096 UTF-8 characters.
- **REQ-10**: The `qbt.Client` interface MUST expose `PauseTorrents` and `ResumeTorrents` methods.

## Acceptance Criteria

- **AC-1.1**: The `/list` and `/active` commands display a torrent selection keyboard with one button per torrent, numbered 1–5 per page.
- **AC-1.2**: Each selection button's callback data encodes the filter, page, and torrent hash, and fits within 64 bytes.
- **AC-2.1**: Selecting a torrent edits the message to show a detail view with name, size, progress, speeds, state, and category.
- **AC-2.2**: The detail view message stays under 4096 characters even with a worst-case long torrent name.
- **AC-3.1**: A torrent in "downloading" state shows a Pause button (not Resume).
- **AC-3.2**: A torrent in "seeding" state shows a Pause button (not Resume).
- **AC-4.1**: A torrent in "pausedDL" state shows a Resume button (not Pause).
- **AC-4.2**: A torrent in "pausedUP" state shows a Resume button (not Pause).
- **AC-5.1**: Pressing Pause calls `qbt.Client.PauseTorrents` with the torrent hash, answers the callback, and refreshes the detail view.
- **AC-5.2**: After pausing, the detail view shows the updated state and a Resume button.
- **AC-6.1**: Pressing Resume calls `qbt.Client.ResumeTorrents` with the torrent hash, answers the callback, and refreshes the detail view.
- **AC-6.2**: After resuming, the detail view shows the updated state and a Pause button.
- **AC-7.1**: Pressing Back edits the message back to the paginated torrent list at the correct page and filter.
- **AC-8.1**: All new callback data formats (`sel:`, `pa:`, `re:`, `bk:`) are verified to fit within 64 bytes for worst-case inputs (page 99, 40-char hash).
- **AC-9.1**: If the torrent hash is not found when displaying detail (deleted between list and click), the user sees an error message.
- **AC-10.1**: `PauseTorrents(ctx, hashes)` sends a POST to `/api/v2/torrents/stop` with the pipe-separated hash list.
- **AC-10.2**: `ResumeTorrents(ctx, hashes)` sends a POST to `/api/v2/torrents/start` with the pipe-separated hash list.

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
# Verify spec completeness (used by iterative harness loops)
grep -c "^- \*\*REQ-" docs/features/torrent-control/spec.md  # count requirements
grep -c "^- \*\*AC-"  docs/features/torrent-control/spec.md  # count acceptance criteria
grep -c "TODO:"        docs/features/torrent-control/spec.md  # should be 0 for approved
```

## Risks

- **Stale state after pause/resume**: qBittorrent may not immediately reflect the new state after an API call. Mitigation: accept slight staleness; user can re-select the torrent to see updated state.
- **Torrent deleted between list and click**: User clicks a torrent that no longer exists. Mitigation: handle gracefully with "Torrent not found" error message (AC-9.1).
- **Callback data overflow**: Complex encoding could exceed 64 bytes. Mitigation: use single-character filter codes and verified worst-case calculations (AC-8.1).

## Open Questions

None — all resolved during planning.
