---
title: "Torrent File Management"
feature_id: "torrent-files"
status: draft
owner: user
source_files: []
last_updated: 2026-03-15
---

# Torrent File Management — Specification

## Overview

Allow users to view the files within a torrent and manage their individual download priorities directly from the Telegram bot. From the torrent detail view, a "Files" button opens a paginated file list showing each file's name, size, progress, and priority. Users can tap a file to change its priority (skip, normal, high, maximum).

## Problem Statement

The existing torrent detail view shows aggregate progress for the whole torrent but gives no visibility into individual files. Users who want to selectively download parts of a multi-file torrent (e.g., skip extras, prioritise the main episode) currently must open the qBittorrent WebUI. Exposing file-level priority control in the bot removes this friction for the common case.

## Goals

- Show users the file list of any torrent with per-file name, size, progress, and priority.
- Let users change the download priority of any individual file without leaving Telegram.
- Keep navigation consistent with the existing detail / list / back pattern.
- Fit all callback data within Telegram's 64-byte limit.
- Handle large file counts gracefully with pagination.

## Non-Goals

- Bulk priority changes across multiple files in a single action.
- Renaming, moving, or deleting files within qBittorrent.
- Sorting or filtering the file list.
- Showing the full file path when the torrent is a single-file torrent.

## Scope

This feature covers:
- A new `TorrentFile` type and two new `qbt.Client` methods (`ListFiles`, `SetFilePriority`).
- A new formatter function `FormatFileList` and keyboard builder `FileListKeyboard` in `internal/formatter`.
- New callback prefixes `pg:fl:` (file list pagination) and `fp:` (file priority change) routed in `internal/bot/callback.go`.
- A "Files" button added to the existing torrent detail inline keyboard.
- Back navigation from file list to torrent detail.

It stops at the boundary of priority confirmation — priority changes are applied immediately without an undo step.

## Requirements

- **REQ-1**: The user can view the list of files within any torrent from its detail view.
- **REQ-2**: Each file entry displays the file name (truncated if necessary), size, download progress, and current download priority.
- **REQ-3**: The file list is paginated at 5 files per page with inline prev/next navigation.
- **REQ-4**: The user can change the download priority of an individual file to one of: Skip (0), Normal (1), High (6), or Maximum (7).
- **REQ-5**: A "Files" button is present on the torrent detail keyboard; a "Back" button on the file list returns to the torrent detail view.
- **REQ-6**: Priority values are displayed as human-readable labels (Skip, Normal, High, Max) rather than raw integers.

## Acceptance Criteria

- **AC-1.1**: Tapping the "Files" button on a torrent detail view sends a message listing that torrent's files.
- **AC-1.2**: The file list message includes the torrent name as a header.
- **AC-1.3**: If qBittorrent returns an error for `ListFiles`, the bot responds with a user-friendly error message and logs the error.

- **AC-2.1**: Each file entry shows the file name (last path component, truncated to 40 chars with `…` if longer).
- **AC-2.2**: Each file entry shows the file size formatted as human-readable bytes (e.g., `1.2 GB`).
- **AC-2.3**: Each file entry shows a textual progress bar and numeric percentage.
- **AC-2.4**: Each file entry shows the current priority as a human-readable label.

- **AC-3.1**: When a torrent has more than 5 files, pagination navigation buttons appear.
- **AC-3.2**: Navigating pages via inline keyboard displays the correct subset of files.
- **AC-3.3**: Page indicator (e.g., "Page 1/3") is shown in the message or keyboard when multiple pages exist.

- **AC-4.1**: Tapping a file in the file list presents a priority selection keyboard showing all four priority options.
- **AC-4.2**: Tapping a priority option calls `SetFilePriority` and sends an updated file list message confirming the change.
- **AC-4.3**: The current priority is visually distinguished (e.g., checkmark prefix) in the priority selection keyboard.
- **AC-4.4**: If `SetFilePriority` returns an error, the bot responds with a user-friendly error message and logs the error.

- **AC-5.1**: The torrent detail keyboard includes a "Files" button alongside the existing Pause / Start / Back buttons.
- **AC-5.2**: The "Back" button on the file list keyboard returns the user to the torrent detail view for the same torrent.
- **AC-5.3**: The "Back" button on the priority selection keyboard returns the user to the file list page they came from.

- **AC-6.1**: Priority integer 0 is displayed as "Skip" everywhere (file list and priority keyboard).
- **AC-6.2**: Priority integer 1 is displayed as "Normal".
- **AC-6.3**: Priority integer 6 is displayed as "High".
- **AC-6.4**: Priority integer 7 is displayed as "Max".

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
grep -c "^- \*\*REQ-" docs/features/torrent-files/spec.md  # should be 6
grep -c "^- \*\*AC-"  docs/features/torrent-files/spec.md  # should be 14
grep -c "TODO:"        docs/features/torrent-files/spec.md  # should be 0
```

## Risks

- **MEDIUM**: Callback data compactness — `fp:<hash>:<fileIndex>:<priority>` must stay ≤ 64 bytes. SHA-1 hashes are 40 hex chars; with prefix and separators, this is `fp:` (3) + 40 + `:` (1) + up to 5 digits for index + `:` (1) + 1 digit for priority = 51 bytes maximum. Within limit.
- **LOW**: Torrents with hundreds of files generate many callback round-trips. Pagination at 5-per-page keeps each page manageable. Performance impact is negligible at personal-bot scale.
- **LOW**: File name display — qBittorrent returns the full relative path within the torrent (e.g., `Season 1/Episode 01.mkv`). Truncation logic must handle both flat and nested paths consistently.

## Open Questions

None. All design decisions are resolved above.
