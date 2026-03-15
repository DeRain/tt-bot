---
title: "Downloading Torrents List"
feature_id: "downloading-list"
status: implemented
owner: user
source_files:
  - internal/bot/handler.go
  - internal/bot/callback.go
  - internal/bot/commands.go
  - internal/qbt/types.go
last_updated: 2026-03-15
---

# Downloading Torrents List — Specification

## Overview

Add a `/downloading` command that lists all non-completed torrents (both paused and actively downloading), with the same pagination and torrent control UX as `/list` and `/active`.

## Problem Statement

Users want to quickly see only torrents that are still downloading, regardless of whether they are paused or active. The existing `/active` command shows only actively transferring torrents (downloading + seeding), which excludes paused downloads and includes completed seeds.

## Goals

- Provide a focused view of incomplete downloads
- Include both paused and active non-completed torrents
- Reuse existing pagination, selection, and control UX

## Non-Goals

- Changing the behavior of `/list` or `/active`
- Adding new formatting or display styles
- Server-side filtering (qBittorrent API has no single filter for "all incomplete")

## Scope

This feature covers the `/downloading` command, its pagination callbacks, and integration with torrent selection/control. It reuses all existing formatter, keyboard, and control infrastructure.

## Requirements

- **REQ-1**: `/downloading` command displays all non-completed torrents (Progress < 1.0)
- **REQ-2**: Both paused and active incomplete torrents are included
- **REQ-3**: Pagination works identically to `/list` and `/active` (5 per page, inline keyboard)
- **REQ-4**: Torrent selection and control (pause/resume/back) works from the downloading list
- **REQ-5**: Command is registered with Telegram and appears in help text

## Acceptance Criteria

- **AC-1.1**: `/downloading` returns only torrents with Progress < 1.0
- **AC-1.2**: `/downloading` returns "No torrents found." when all torrents are completed
- **AC-2.1**: A paused incomplete torrent appears in the `/downloading` list
- **AC-2.2**: An actively downloading torrent appears in the `/downloading` list
- **AC-3.1**: Pagination keyboard appears when more than 5 incomplete torrents exist
- **AC-3.2**: Navigating pages via inline keyboard shows correct page of incomplete torrents
- **AC-4.1**: Selecting a torrent from the downloading list shows its detail view
- **AC-4.2**: Pause/resume actions from the detail view work and return to the downloading list
- **AC-5.1**: `/downloading` appears in Telegram command menu and `/help` output

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
