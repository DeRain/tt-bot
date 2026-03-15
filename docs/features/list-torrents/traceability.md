---
title: Torrent Listing — Traceability Matrix
feature_id: list-torrents
status: implemented
last_updated: 2026-03-15
---

# Torrent Listing — Traceability Matrix

| Requirement | Acceptance Criteria | Design | Plan Tasks | Implementation Evidence | Verification | Status |
|-------------|---------------------|--------|------------|-------------------------|--------------|--------|
| REQ-1 | AC-1.1 | DES-1, DES-2 | TASK-1, TASK-4 | internal/bot/handler.go (`handleCommand` routes `/list` to `sendTorrentPage` with `FilterAll`), internal/qbt/http.go (`ListTorrents` with filter param) | TEST-1 | Complete |
| REQ-2 | AC-2.1 | DES-1, DES-2 | TASK-1, TASK-4 | internal/bot/handler.go (`handleCommand` routes `/active` to `sendTorrentPage` with `FilterActive`), internal/qbt/http.go (`ListTorrents` with filter param) | TEST-2 | Complete |
| REQ-3 | AC-3.1 | DES-2, DES-4 | TASK-4, TASK-5 | internal/bot/handler.go (`sendTorrentPage` slices by `TorrentsPerPage`), internal/bot/callback.go (`handlePaginationCallback` slices the same way), internal/formatter/format.go (`TorrentsPerPage = 5`, `TotalPages`) | TEST-3 | Complete |
| REQ-4 | AC-4.1, AC-4.2, AC-4.3, AC-4.4 | DES-3, DES-4 | TASK-3, TASK-5 | internal/formatter/format.go (`PaginationKeyboard` omits Prev on page 1, Next on last page), internal/bot/callback.go (`handlePaginationCallback` edits message in place via `editMessageText`) | TEST-4, TEST-5 | Complete |
| REQ-5 | AC-5.1 | DES-2, DES-6 | TASK-2, TASK-6 | internal/formatter/format.go (`FormatTorrentList` builds entry with name, progress bar, speeds, state) | TEST-8, TEST-9, TEST-10 | Complete |
| REQ-6 | AC-6.1, AC-6.2 | DES-5 | TASK-2 | internal/formatter/format.go (`FormatTorrentList` checks `sb.Len()+len(entry) > MaxMessageLength-1` before appending) | TEST-6 | Complete |
| REQ-7 | AC-7.1 | DES-6 | TASK-6 | internal/formatter/format.go (`truncateName` truncates at 40 runes, appends "...") | TEST-8 | Complete |
| REQ-8 | AC-8.1 | DES-2 | TASK-2 | internal/formatter/format.go (`FormatTorrentList` returns "No torrents found." for empty input) | TEST-7 | Complete |
