---
title: "Uploading Torrents List"
feature_id: "uploading-list"
status: draft
owner: user
source_files:
  - internal/bot/handler.go
  - internal/bot/callback.go
  - internal/bot/commands.go
  - internal/qbt/types.go
last_updated: 2026-03-15
---

# Uploading Torrents List — Specification

## Overview

Add a `/uploading` command that lists all completed torrents that are currently seeding or paused after completion (Progress == 1.0), with the same pagination and torrent control UX as `/list`, `/active`, and `/downloading`.

## Problem Statement

Users want to quickly see only torrents that have finished downloading and are now uploading (seeding), including those that have been paused after completion. The existing `/active` command shows actively transferring torrents (downloading + seeding) but excludes paused seeds and includes active downloads. The `/downloading` command is the inverse — showing Progress < 1.0. Neither provides a focused seeding view.

## Goals

- Provide a focused view of completed torrents that are seeding or paused after completion
- Include both paused-complete and actively seeding torrents
- Reuse existing pagination, selection, and control UX

## Non-Goals

- Changing the behavior of `/list`, `/active`, or `/downloading`
- Adding new formatting or display styles
- Server-side filtering (qBittorrent API has no single filter for "all completed seeds including paused")

## Scope

This feature covers the `/uploading` command, its pagination callbacks, and integration with torrent selection/control. It reuses all existing formatter, keyboard, and control infrastructure.

## Requirements

- **REQ-1**: `/uploading` command displays all completed torrents (Progress == 1.0)
- **REQ-2**: Both paused-complete and actively seeding torrents are included
- **REQ-3**: Pagination works identically to `/list`, `/active`, and `/downloading` (5 per page, inline keyboard)
- **REQ-4**: Torrent selection and control (pause/resume/back) works from the uploading list
- **REQ-5**: Command is registered with Telegram and appears in help text

## Acceptance Criteria

- **AC-1.1**: `/uploading` returns only torrents with Progress == 1.0
- **AC-1.2**: `/uploading` returns "No torrents found." when no torrents are completed
- **AC-2.1**: A paused completed torrent (state `pausedUP`) appears in the `/uploading` list
- **AC-2.2**: An actively seeding torrent (state `uploading` or `stalledUP`) appears in the `/uploading` list
- **AC-3.1**: Pagination keyboard appears when more than 5 completed torrents exist
- **AC-3.2**: Navigating pages via inline keyboard shows correct page of completed torrents
- **AC-4.1**: Selecting a torrent from the uploading list shows its detail view
- **AC-4.2**: Pause/resume actions from the detail view work and return to the uploading list
- **AC-5.1**: `/uploading` appears in Telegram command menu and `/help` output

## Quality Gates

### Gate 1: Spec Gate

This spec passes when:
- [x] All requirements are clear and unambiguous
- [x] All acceptance criteria are testable
- [x] Scope and non-goals are defined
- [x] No unresolved open questions block implementation
- [x] At least one AC exists per requirement

## Risks

- **LOW**: Client-side filtering fetches all torrents then filters — acceptable for personal bot scale

## Open Questions

None.
