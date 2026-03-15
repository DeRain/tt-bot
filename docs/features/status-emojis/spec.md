---
title: "Human-Readable Statuses with Emojis"
feature_id: status-emojis
status: draft
owner: TODO
source_files:
  - internal/formatter/formatter.go
last_updated: 2026-03-15
---

# Human-Readable Statuses with Emojis — Specification

## Overview

Map qBittorrent's internal state strings (e.g. `stalledUP`, `pausedDL`, `metaDL`) to human-readable labels with emoji prefixes in all bot message views.

## Problem Statement

qBittorrent's Web API returns raw machine-oriented state strings. The bot currently surfaces these strings directly to users, producing output like `State: stalledUP` or `... | stalledUP` in list rows. These identifiers are meaningless to end users who are not familiar with qBittorrent internals.

## Goals

- Replace all raw state strings in bot output with friendly labels.
- Prefix each label with a contextually appropriate emoji.
- Handle every documented qBittorrent state explicitly.
- Gracefully tolerate any future or undocumented state values.

## Non-Goals

- Changing what states qBittorrent reports (read-only mapping).
- Adding filtering or sorting by state.
- Localisation or i18n of labels.
- Storing or persisting state history.

## Scope

This feature is limited to the `internal/formatter` package. It adds a pure mapping function used by the two existing format functions (`FormatTorrentList` and `FormatTorrentDetail`). No other packages are modified.

## Requirements

- **REQ-1**: Every documented qBittorrent state string MUST be mapped to a human-readable label.
- **REQ-2**: Every mapped status MUST be prefixed with an emoji that reflects the state's semantics.
- **REQ-3**: The status mapping MUST be applied wherever the raw state is currently displayed — both the list view (`FormatTorrentList`) and the detail view (`FormatTorrentDetail`).
- **REQ-4**: Any state string not present in the mapping (unknown or future states) MUST fall back to displaying the raw string prefixed with ❓, without panicking or returning an error.

## Acceptance Criteria

- **AC-1.1**: Given a torrent with state `stalledUP`, the formatted output contains `Seeding (stalled)` and does not contain the raw string `stalledUP`.
- **AC-1.2**: Given a torrent with state `pausedDL`, the formatted output contains `Paused (Downloading)` and does not contain the raw string `pausedDL`.
- **AC-1.3**: All 19 documented states produce a non-empty label distinct from the raw state string.
- **AC-2.1**: Each mapped state's rendered label begins with its designated emoji character.
- **AC-2.2**: The fallback output for an unmapped state begins with ❓.
- **AC-3.1**: The list view row produced by `FormatTorrentList` displays the mapped label, not the raw state.
- **AC-3.2**: The detail view produced by `FormatTorrentDetail` displays the mapped label on the `State:` line, not the raw state.
- **AC-4.1**: Calling the mapping function with an empty string returns a non-empty fallback string prefixed with ❓ and does not panic.
- **AC-4.2**: Calling the mapping function with a novel unrecognised string (e.g. `"newState"`) returns `❓ newState` and does not panic.

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
grep -c "^- \*\*REQ-" docs/features/status-emojis/spec.md  # expect 4
grep -c "^- \*\*AC-"  docs/features/status-emojis/spec.md  # expect 9
grep -c "TODO:"        docs/features/status-emojis/spec.md  # expect 0
```

## Risks

- **New qBittorrent states**: Future API versions may introduce states not in the mapping. Mitigated by REQ-4's graceful fallback.
- **Emoji rendering**: Emoji display depends on the Telegram client. Risk is cosmetic only and out of scope to fix.

## Open Questions

- None. All states are documented in the qBittorrent Web API v2 reference and the mapping is fully specified above.
