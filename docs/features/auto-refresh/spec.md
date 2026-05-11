---
title: "Auto-Refresh Views"
feature_id: "auto-refresh"
status: draft
owner: TODO
source_files: []
last_updated: 2026-05-12
---

# Auto-Refresh Views — Specification

## Overview

List views and torrent detail views automatically refresh at a configurable interval by polling qBittorrent and editing Telegram messages in-place when data changes. No new messages are ever sent during refresh.

## Problem Statement

All torrent views in the bot are static. A user who opens `/all` or selects a torrent detail sees a snapshot at that moment. To see updated progress, speeds, or state changes, the user must manually press a pagination button or re-select the torrent. This is tedious for active downloads where progress changes every few seconds, and it forces users to generate unnecessary interaction noise just to get fresh data.

## Goals

- Automatically refresh list views (`/all`, `/active`, `/downloading`, `/uploading`) on a timer
- Automatically refresh torrent detail views after selection
- Avoid unnecessary Telegram API calls by detecting actual data changes before editing
- Support only one active auto-refreshing view per chat to prevent message clutter
- Make the refresh interval configurable via environment variable
- Fail silently when edits fail, without disrupting the bot or the user

## Non-Goals

- Persisting view state across bot restarts (stateless is acceptable)
- Auto-refresh for category picker or confirmation dialogs
- WebSocket or push-based refresh (polling only)
- Refresh-on-focus or activity-based triggers
- Multi-view-per-chat support

## Scope

This feature covers: background polling for list and detail views, hash-based change detection, Telegram message editing, per-chat view registration and deregistration, and a configurable refresh interval. It extends the existing list-torrents, downloading-list, uploading-list, and torrent-control features but does not modify their core behavior.

## Requirements

- **REQ-1**: List views (`/all`, `/active`, `/downloading`, `/uploading`) MUST auto-refresh at a configurable interval after being rendered.
- **REQ-2**: Torrent detail views MUST auto-refresh at a configurable interval after selection.
- **REQ-3**: Auto-refresh MUST compare rendered content hashes (SHA-256) and only edit the message when data has actually changed.
- **REQ-4**: Only the most recently rendered view per chat MUST auto-refresh. Rendering a new view deregisters any previously active view for that chat.
- **REQ-5**: The refresh interval MUST be configurable via a `VIEW_REFRESH_INTERVAL` environment variable, defaulting to `5s`.
- **REQ-6**: Failed message edits during refresh MUST silently deregister the view for that chat without sending error messages to the user.
- **REQ-7**: Auto-refresh MUST only edit existing messages via `EditMessageText`. It MUST NEVER send new messages.

## Acceptance Criteria

- **AC-1.1**: After sending `/all`, the list message updates in-place within one refresh interval when torrent data changes.
- **AC-1.2**: `/active`, `/downloading`, and `/uploading` list views each auto-refresh independently when active.
- **AC-1.3**: Paginating within a list view keeps the view registered for auto-refresh at the new page.
- **AC-2.1**: After selecting a torrent from a list, the detail view auto-refreshes to reflect state and progress changes.
- **AC-2.2**: Pressing Pause or Resume on a detail view immediately refreshes the view and keeps it registered for continued auto-refresh.
- **AC-3.1**: If qBittorrent returns identical data on consecutive polls, the Telegram message is not edited (no flashing, no "edited" label).
- **AC-3.2**: If a single field changes (e.g., progress increments by 0.1%), the message is edited to reflect the update.
- **AC-4.1**: Sending `/all` then `/active` in the same chat deregisters the `/all` view; only `/active` auto-refreshes.
- **AC-4.2**: Selecting a torrent detail deregisters the originating list view; only the detail view auto-refreshes.
- **AC-5.1**: Setting `VIEW_REFRESH_INTERVAL=10s` causes views to refresh on a 10-second ticker.
- **AC-5.2**: Omitting `VIEW_REFRESH_INTERVAL` defaults the ticker to 5 seconds.
- **AC-5.3**: An invalid value (e.g., `VIEW_REFRESH_INTERVAL=abc`) falls back to the 5-second default with a logged warning.
- **AC-6.1**: If a message is deleted by the user between refreshes, the next edit attempt fails and the view is silently deregistered (no error sent to chat).
- **AC-6.2**: If the bot loses permission to edit the message, the next edit attempt fails and the view is silently deregistered.
- **AC-7.1**: During a full refresh cycle, no `SendMessage` API calls are made; only `EditMessageText` is used for existing views.
- **AC-7.2**: If no view is registered for a chat, no API calls are made during the refresh tick.

## Quality Gates

### Gate 1: Spec Gate

This spec passes when:
- [ ] All requirements are clear and unambiguous
- [ ] All acceptance criteria are testable
- [ ] Scope and non-goals are defined
- [ ] No unresolved open questions block implementation
- [ ] At least one AC exists per requirement

**Harness check command:**
```bash
# Verify spec completeness (used by iterative harness loops)
grep -c "^- \*\*REQ-" docs/features/auto-refresh/spec.md  # count requirements
grep -c "^- \*\*AC-"  docs/features/auto-refresh/spec.md  # count acceptance criteria
grep -c "TODO:"        docs/features/auto-refresh/spec.md  # should be 0 for approved
```

## Risks

- **Polling load on qBittorrent**: Each active chat triggers a `ListTorrents` API call per tick. For a single-user bot this is negligible; for multi-user deployments, the interval should be tuned accordingly. Mitigation: configurable interval and hash-based change detection minimize unnecessary API calls to Telegram.
- **Telegram rate limits**: Rapid edits to the same message (e.g., every 5s) stay well within Telegram's limit of ~30 messages per second per chat. No practical risk.
- **Stale content hash**: If the hash comparison fails to detect a real change (extremely unlikely with SHA-256), the view would not refresh. Mitigation: SHA-256 collision probability is negligible.
- **Memory growth**: The `liveViews` map is bounded by active chats and cleaned up on deregistration or failed edits. No unbounded growth.

## Open Questions

- None at this stage.
