---
title: "Downloading Torrents List — Traceability"
feature_id: "downloading-list"
status: complete
last_updated: 2026-03-15
---

# Downloading Torrents List — Traceability Matrix

## Forward Traceability

| REQ | AC | DES | TASK | Source File | TEST | Status |
|-----|----|-----|------|-------------|------|--------|
| REQ-1 | AC-1.1, AC-1.2 | DES-1, DES-2 | TASK-1, TASK-4 | types.go, handler.go | TEST-1, TEST-4 | Complete |
| REQ-2 | AC-2.1, AC-2.2 | DES-1, DES-2 | TASK-1, TASK-4 | types.go, handler.go | TEST-1, TEST-4 | Complete |
| REQ-3 | AC-3.1, AC-3.2 | DES-4 | TASK-2, TASK-3 | callback.go | TEST-2, TEST-3 | Complete |
| REQ-4 | AC-4.1, AC-4.2 | DES-4 | TASK-2, TASK-3 | callback.go | TEST-2, TEST-3 | Complete |
| REQ-5 | AC-5.1 | DES-3 | TASK-4 | commands.go, handler.go | TEST-5 | Complete |

## Backward Traceability

| Source File | Functions/Types | Traces To | Via |
|-------------|----------------|-----------|-----|
| internal/qbt/types.go | FilterDownloading | REQ-1, REQ-2 | DES-1, TASK-1 |
| internal/bot/callback.go | filterCharToFilter, filterToChar, filterCharToPrefix, handleCallback | REQ-3, REQ-4 | DES-4, TASK-2, TASK-3 |
| internal/bot/handler.go | handleCommand, renderTorrentListPage | REQ-1, REQ-2, REQ-5 | DES-2, DES-3, TASK-4 |
| internal/bot/commands.go | BotCommands | REQ-5 | DES-3, TASK-4 |

## Coverage Summary

- Requirements: 5 defined, 5 complete, 0 pending
- Acceptance Criteria: 9 defined, 9 validated
- Design Items: 4 defined, all mapped
- Tasks: 5 defined, all mapped
- Gaps: None identified
