---
title: Torrent Listing — Verification
feature_id: list-torrents
status: verified
last_updated: 2026-03-15
---

# Torrent Listing — Verification

## Validation Strategy

All acceptance criteria are validated through automated unit tests and E2E integration tests. The formatting logic is pure and deterministic; the command and callback handlers use mock interfaces for isolation.

## Automated Tests

- **TEST-1**: `/list` fetches all torrents and sends a formatted message
  - Validates: AC-1.1
  - Covers: REQ-1
  - Evidence: internal/bot/handler_test.go — `TestHandler_ListCommand_WithTorrents` (verifies torrent name appears in sent message), internal/bot/e2e_test.go — `TestE2E_ListReturnsRealTorrents` (verifies against real qBittorrent)

- **TEST-2**: `/active` fetches active torrents and sends a formatted message
  - Validates: AC-2.1
  - Covers: REQ-2
  - Evidence: internal/bot/e2e_test.go — `TestE2E_ActiveCommandShowsDownloading` (verifies `/active` returns valid response from real qBittorrent)

- **TEST-3**: Pagination splits torrents into correct pages
  - Validates: AC-3.1
  - Covers: REQ-3
  - Evidence: internal/bot/callback_test.go — `TestCallback_PaginationAll_FetchesCorrectPage` (7 torrents produce page 2/2), `TestCallback_PaginationActive_FetchesCorrectPage` (6 torrents produce page 2/2), internal/formatter/format_test.go — `TestTotalPages` (verifies page count arithmetic)

- **TEST-4**: Prev button hidden on page 1, Next button hidden on last page
  - Validates: AC-4.1, AC-4.2, AC-4.3
  - Covers: REQ-4
  - Evidence: internal/formatter/format_test.go — `TestPaginationKeyboard_FirstPage_NoPrev`, `TestPaginationKeyboard_LastPage_NoNext`, `TestPaginationKeyboard_MiddlePage_BothButtons`

- **TEST-5**: Page navigation edits the existing message in place
  - Validates: AC-4.4
  - Covers: REQ-4
  - Evidence: internal/bot/callback_test.go — `TestCallback_PaginationAll_FetchesCorrectPage` (verifies `hasEditText("page 2/2")`), internal/bot/e2e_test.go — `TestE2E_ListPagination` (verifies callback is answered and message is edited)

- **TEST-6**: Formatted message stays under 4096 characters
  - Validates: AC-6.1, AC-6.2
  - Covers: REQ-6
  - Evidence: internal/formatter/format_test.go — `TestFormatTorrentList_FiveTorrents_UnderLimit`, `TestFormatTorrentList_WorstCaseLongNames_UnderLimit` (5 torrents with 40-char names and max speeds stay under limit)

- **TEST-7**: Empty torrent list shows "No torrents found."
  - Validates: AC-8.1
  - Covers: REQ-8
  - Evidence: internal/bot/handler_test.go — `TestHandler_ListCommand_NoTorrents`, internal/formatter/format_test.go — `TestFormatTorrentList_Empty_ReturnsNotFound` (nil and empty slice both return exact string)

- **TEST-8**: Torrent names are truncated to 40 characters
  - Validates: AC-7.1
  - Covers: REQ-7
  - Evidence: internal/formatter/format_test.go — `TestFormatTorrentList_WorstCaseLongNames_UnderLimit` (uses 40-char names at truncation boundary)

- **TEST-9**: Speed formatting displays correct units (B/s, KB/s, MB/s)
  - Validates: AC-5.1 (speed portion)
  - Covers: REQ-5
  - Evidence: internal/formatter/format_test.go — `TestFormatSpeed_BytesPerSec`, `TestFormatSpeed_KilobytesPerSec`, `TestFormatSpeed_MegabytesPerSec`

- **TEST-10**: Progress bar renders correctly with percentage
  - Validates: AC-5.1 (progress portion)
  - Covers: REQ-5
  - Evidence: internal/formatter/format_test.go — `TestFormatProgress` (0%, 10%, 50%, 90%, 100%), `TestFormatProgress_EdgeValues` (negative and >1 values clamped)

## Manual Checks

None required.

## Acceptance Criteria Results

| AC | Validation | Result |
|----|-----------|--------|
| AC-1.1 | TEST-1 | Pass |
| AC-2.1 | TEST-2 | Pass |
| AC-3.1 | TEST-3 | Pass |
| AC-4.1 | TEST-4 | Pass |
| AC-4.2 | TEST-4 | Pass |
| AC-4.3 | TEST-4 | Pass |
| AC-4.4 | TEST-5 | Pass |
| AC-5.1 | TEST-8, TEST-9, TEST-10 | Pass |
| AC-6.1 | TEST-6 | Pass |
| AC-6.2 | TEST-6 | Pass |
| AC-7.1 | TEST-8 | Pass |
| AC-8.1 | TEST-7 | Pass |

## Traceability Coverage

All 8 requirements have automated verification. All 12 acceptance criteria are validated. No gaps.

## Exceptions / Unresolved Gaps

None.
