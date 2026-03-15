---
title: "Stop and Remove Torrent Actions"
feature_id: torrent-remove
status: draft
owner: TODO
source_files: []
last_updated: 2026-03-15
---

# Stop and Remove Torrent Actions — Specification

## Overview

Add a **Remove** button to the torrent detail view that lets users delete a torrent from qBittorrent, with a mandatory confirmation step that offers two options: remove the torrent record only, or remove the torrent and delete all downloaded files.

## Problem Statement

Users can pause and resume torrents via the bot but cannot remove them. Removing a finished or unwanted torrent currently requires leaving the Telegram conversation and opening the qBittorrent WebUI. This breaks the bot's goal of being the primary management interface.

## Goals

- Allow users to remove any torrent directly from the detail view
- Prevent accidental deletion via a confirmation step
- Give users explicit control over whether downloaded files are deleted
- Return the user to a clean list view after a successful removal

## Non-Goals

- Batch removal of multiple torrents at once
- Removing torrents by category or filter
- Force-stopping a torrent without removing it (pause already covers the stop use case; qBittorrent v5 does not expose a separate "force stop" endpoint distinct from `stop`)
- Undo / restore removed torrents

## Scope

This feature covers: a Remove button in the torrent detail keyboard, a confirmation view with two deletion options and a cancel option, the `DeleteTorrents` qBittorrent API call, and post-removal navigation back to the torrent list. It extends the `torrent-control` feature's detail view without modifying existing pause/resume behavior.

## Requirements

- **REQ-1**: The torrent detail view MUST display a **Remove** button alongside the existing Pause/Start and Back buttons.
- **REQ-2**: Pressing Remove MUST replace the detail view with a confirmation view that shows two action buttons ("Remove torrent only" and "Remove with files") and a Cancel button, before any deletion occurs.
- **REQ-3**: Confirming "Remove torrent only" MUST delete the torrent record from qBittorrent while preserving all downloaded files on disk (`deleteFiles=false`).
- **REQ-4**: Confirming "Remove with files" MUST delete the torrent record AND all associated downloaded files from disk (`deleteFiles=true`).
- **REQ-5**: After a successful removal (REQ-3 or REQ-4), the bot MUST navigate the user back to the torrent list at the same filter and page that was active when the detail view was opened.
- **REQ-6**: Pressing Cancel on the confirmation view MUST return the user to the torrent detail view for the same torrent, with the same filter and page context.

## Acceptance Criteria

- **AC-1.1**: The torrent detail keyboard contains a Remove button whose callback data encodes `rm:`, the filter character, the page number, and the torrent hash, and fits within 64 bytes.
- **AC-1.2**: The Remove button is present regardless of the torrent's current state (downloading, seeding, paused, etc.).
- **AC-2.1**: Pressing the Remove button edits the message to show a confirmation prompt that includes the torrent name and three buttons: "Remove torrent only", "Remove with files", and "Cancel".
- **AC-2.2**: No qBittorrent API call is made when the confirmation view is shown (showing it is non-destructive).
- **AC-3.1**: Confirming "Remove torrent only" calls `qbt.Client.DeleteTorrents(ctx, hashes, false)` and the torrent is absent from subsequent `ListTorrents` responses.
- **AC-3.2**: The downloaded files for the removed torrent remain on disk after a "Remove torrent only" confirmation (validated via integration test with a real qBittorrent instance).
- **AC-4.1**: Confirming "Remove with files" calls `qbt.Client.DeleteTorrents(ctx, hashes, true)`.
- **AC-4.2**: The callback data for both confirmation actions fits within 64 bytes for worst-case inputs (page 99, 40-char hash).
- **AC-5.1**: After a successful removal, the bot edits the message to display the torrent list at the filter and page encoded in the callback.
- **AC-5.2**: If the list is now empty (the removed torrent was the only one), the bot displays an appropriate empty-list message rather than an error.
- **AC-6.1**: Pressing Cancel on the confirmation view edits the message back to the full torrent detail view for the same torrent.
- **AC-6.2**: The Cancel action does not call any qBittorrent API endpoint.

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
grep -c "^- \*\*REQ-" docs/features/torrent-remove/spec.md  # count requirements
grep -c "^- \*\*AC-"  docs/features/torrent-remove/spec.md  # count acceptance criteria
grep -c "TODO:"        docs/features/torrent-remove/spec.md  # should be 0 for approved
```

## Risks

- **Accidental deletion**: Users may misread "Remove with files" as safe. Mitigation: confirmation message explicitly names the torrent and labels the destructive button clearly.
- **Torrent deleted between detail view and confirmation click**: The hash may no longer exist when the delete API call is made. Mitigation: handle the API error gracefully and navigate to the list view with an explanatory message.
- **Callback data overflow**: Four new callback prefixes each carry filter char, page, and full 40-char hash. Mitigation: worst-case byte calculations are verified in AC-1.1 and AC-4.2; the `rf:` prefix at page 99 with a 40-char hash is 49 bytes, well within the 64-byte limit.

## Open Questions

None — all resolved during planning.
