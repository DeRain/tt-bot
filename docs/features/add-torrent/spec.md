---
title: Add Torrent
feature_id: add-torrent
status: implemented
owner: TODO
source_files:
  - internal/bot/handler.go
  - internal/bot/callback.go
  - internal/qbt/http.go
  - internal/formatter/format.go
last_updated: 2026-03-15
---

# Add Torrent

## Overview

The add-torrent feature allows whitelisted Telegram users to submit torrents to a qBittorrent instance by sending magnet links or uploading `.torrent` files through the bot. The bot stores the submission in a pending map (keyed by chat ID, 5-minute TTL), fetches categories from qBittorrent, and shows an inline keyboard. The user picks a category via callback, the bot adds the torrent, and the confirmation message edits the original. A background goroutine evicts expired entries every 1 minute. Callback format is `cat:<name>` (64-byte Telegram limit). For `.torrent` files, the bot downloads from Telegram CDN.

## Problem Statement

Users need a conversational interface to add torrents to a headless qBittorrent instance from Telegram. The two-step flow (submit torrent, pick category) must handle asynchronous user interaction, enforce Telegram API constraints (64-byte callback data, 4096-char messages), and clean up abandoned requests.

## Goals

- Provide a frictionless two-step flow: submit torrent, select category.
- Support both magnet URIs and `.torrent` file uploads.
- Dynamically fetch categories from qBittorrent so configuration changes are reflected immediately.
- Confirm success or failure inline by editing the original message.
- Automatically clean up stale pending state to prevent resource leaks.

## Non-Goals

- Bulk torrent submission (multiple torrents in one message).
- Torrent search or discovery within the bot.
- Custom save-path selection (categories determine save paths in qBittorrent).
- Persistent pending state across bot restarts.
- Multiple simultaneous pending torrents per user.
- Category creation or management from Telegram.
- Torrent priority or configuration options during add.

## Scope

Covers the full lifecycle from receiving a magnet link or `.torrent` file through category selection to qBittorrent submission and user confirmation. Includes:

- Magnet link detection and extraction from message text.
- `.torrent` file download from Telegram CDN.
- Pending torrent storage with TTL-based expiry.
- Category fetching and inline keyboard presentation.
- Category callback handling and torrent addition via qBittorrent API.
- Confirmation/error message editing.
- Background cleanup of expired pending entries.

## Requirements

| ID | Priority | Description |
|----|----------|-------------|
| REQ-1 | MUST | Users MUST be able to add torrents by sending magnet links. |
| REQ-2 | MUST | Users MUST be able to add torrents by uploading `.torrent` files. |
| REQ-3 | MUST | Users MUST select a category before a torrent is added. |
| REQ-4 | MUST | Categories MUST be fetched dynamically from qBittorrent. |
| REQ-5 | MUST | Pending torrents MUST expire after 5 minutes. |
| REQ-6 | MUST | Only one pending torrent per user at a time. |
| REQ-7 | MUST | Category callback data MUST respect the 64-byte Telegram limit. |
| REQ-8 | MUST | Users MUST receive confirmation when a torrent is added. |
| REQ-9 | MUST | Users MUST receive an error if their pending torrent expired. |

## Acceptance Criteria

| ID | Requirement | Criterion |
|----|-------------|-----------|
| AC-1.1 | REQ-1 | A message containing `magnet:?` is detected, the magnet URI is extracted (up to next whitespace), stored as a pending torrent keyed by chat ID, and the category selection keyboard is displayed. |
| AC-2.1 | REQ-2 | An uploaded document with a `.torrent` extension triggers file download from Telegram CDN; the file bytes and original filename are stored as a pending torrent keyed by chat ID, and the category selection keyboard is displayed. |
| AC-3.1 | REQ-3 | The torrent is not submitted to qBittorrent until the user selects a category via the inline keyboard. When a "No category" option exists (empty category list), the torrent is added with an empty category string. |
| AC-4.1 | REQ-4 | Category names displayed in the keyboard match the current categories returned by `qbt.Client.Categories()` at the time of the request. If fetching categories fails, the user receives an error message. |
| AC-5.1 | REQ-5 | A pending torrent entry created more than 5 minutes ago is removed by the background cleanup goroutine, which runs every 1 minute. |
| AC-6.1 | REQ-6 | Sending a second magnet link or `.torrent` file replaces the existing pending entry for the same chat ID. |
| AC-7.1 | REQ-7 | Category callback data (`cat:<name>`) is truncated to at most 64 bytes with valid UTF-8 boundary alignment when the category name is long. |
| AC-8.1 | REQ-8 | After successful torrent addition, the category selection message is edited to show "Torrent added to <category>!" (or "Torrent added!" for empty category), and the callback spinner is dismissed with "Torrent added!". |
| AC-9.1 | REQ-9 | If no pending torrent exists when the user selects a category, the callback answer contains "No pending torrent. Please resend the magnet link or file." |

## Quality Gates

### Gate 1: Spec Gate

- [x] All requirements have unique IDs
- [x] All requirements have priority levels
- [x] Every requirement has at least one acceptance criterion
- [x] All acceptance criteria are testable
- [x] No TODO placeholders remain in requirements or acceptance criteria
- [x] Risks and open questions are documented

#### Harness Check

```bash
# Verify REQ count matches expectations
grep -c '^| REQ-' docs/features/add-torrent/spec.md

# Verify AC count matches expectations
grep -c '^| AC-' docs/features/add-torrent/spec.md

# Verify no TODOs remain in requirements/AC sections
grep -c 'TODO' docs/features/add-torrent/spec.md
# Expected: 0 in requirements/AC (owner TODO in frontmatter is acceptable)
```

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Telegram CDN download failure for `.torrent` files | User cannot add file-based torrents | Error message prompts retry; HTTP client has 30s timeout. |
| qBittorrent API unavailable during category fetch | User sees error instead of keyboard | Error message shown; user can resend the link/file to retry. |
| Race condition between TTL eviction and category selection | User selects category but entry was just evicted | AC-9.1 handles this gracefully with an informative error. |
| Category name exceeds 64 bytes when prefixed with `cat:` | Telegram rejects the callback | Truncation with UTF-8 boundary alignment (REQ-7). |
| Bot token leaked in error messages from file download | Security exposure | `downloadFile` sanitizes errors to exclude the URL containing the token. |
| Large `.torrent` files cause memory pressure | Pending map holds raw bytes in memory | Telegram bot API limits files to 20 MB; 5-minute TTL bounds retention. |
| qBittorrent unreachable when user selects category | Pending entry consumed by `takePending` but torrent not added | Error shown to user; entry is not re-stored (user must resend). |

## Open Questions

| # | Question | Status |
|---|----------|--------|
| 1 | Should the bot re-store the pending entry on qBittorrent add failure so the user can retry without resending? | Open |
| 2 | Should there be a configurable TTL instead of the hardcoded 5-minute value? | Open |
| 3 | Should the bot support multiple pending torrents per user? | Deferred |
| 4 | Should pending state survive bot restarts? | Deferred -- stateless design is intentional. |
