---
title: "Search Torrent Download — Traceability Matrix"
feature_id: "search-torrent-download"
status: complete
last_updated: 2026-05-24
---

# Search Torrent Download — Traceability Matrix

## Forward Traceability (Requirement → Verification)

| Requirement | Acceptance Criteria | Design | Plan Tasks | Implementation Evidence | Verification | Status |
|-------------|---------------------|--------|------------|-------------------------|--------------|--------|
| REQ-1: Torrent File Download from FileURL | AC-1.1, AC-1.2 | DES-1, DES-2 | TASK-5 | `internal/bot/callback.go` (`handleSearchConfirmCallback` HTTP branch) | TEST-10, TEST-13 | Complete |
| REQ-2: SSRF Protection | AC-2.1 | DES-7 | TASK-3 | `internal/bot/download.go` (`isPublicHostname`) | TEST-1, TEST-2 | Complete |
| REQ-3: Download Size Limit | AC-3.1 | DES-3 | TASK-3 | `internal/bot/download.go` (`downloadSearchTorrent` with `io.LimitReader`) | TEST-5 | Complete |
| REQ-4: Download Timeout | AC-4.1 | DES-3 | TASK-3 | `internal/bot/download.go` (`newDownloadClient` with 15s timeout) | TEST-6, TEST-9 | Complete |
| REQ-5: Redirect Safety | AC-5.1, AC-5.2 | DES-3 | TASK-3 | `internal/bot/download.go` (`newDownloadClient` with `CheckRedirect`) | TEST-7, TEST-8 | Complete |
| REQ-6: Content-Type Validation | AC-6.1 | DES-3 | TASK-3 | `internal/bot/download.go` (`downloadSearchTorrent` Content-Type check) | TEST-4 | Complete |
| REQ-7: Pending Torrent Integration | AC-7.1 | DES-1, DES-2 | TASK-3, TASK-5 | `internal/bot/download.go` (`downloadSearchTorrent`), `internal/bot/callback.go` (`storePending` with FileData) | TEST-3, TEST-10 | Complete |
| REQ-8: Error Handling | AC-8.1 | DES-2 | TASK-5 | `internal/bot/callback.go` (error prefix "Failed to download torrent:") | TEST-11, TEST-12 | Complete |

## Backward Traceability (Code → Requirement)

| Source File | Functions/Types | Traces To | Via |
|-------------|----------------|-----------|-----|
| `internal/bot/download.go` | `isPublicHostname` | REQ-2 | TASK-3, DES-7 |
| `internal/bot/download.go` | `newDownloadClient` | REQ-4, REQ-5 | TASK-3, DES-3 |
| `internal/bot/download.go` | `downloadSearchTorrent` | REQ-3, REQ-4, REQ-5, REQ-6, REQ-7 | TASK-3, DES-1, DES-3 |
| `internal/bot/download.go` | `downloadSearchTorrentFn` | REQ-1, REQ-8 | TASK-3, TASK-5, DES-6 |
| `internal/bot/callback.go` | `handleSearchConfirmCallback` (HTTP branch) | REQ-1, REQ-7, REQ-8 | TASK-5, DES-1, DES-2 |
| `internal/bot/handler.go` | `PendingTorrent` (FileData, FileName fields) | REQ-7 | TASK-5, DES-1 |
| `internal/bot/download_test.go` | `TestIsPublicHostname_RejectsLoopback` | REQ-2 | TEST-1 |
| `internal/bot/download_test.go` | `TestIsPublicHostname_RejectsPrivate` | REQ-2 | TEST-2 |
| `internal/bot/download_test.go` | `TestIsPublicHostname_RejectsLinkLocal` | REQ-2 | TEST-2 |
| `internal/bot/download_test.go` | `TestIsPublicHostname_AcceptsPublic` | REQ-2 | TEST-2 |
| `internal/bot/download_test.go` | `TestIsPublicHostname_InvalidHostname` | REQ-2 | TEST-2 |
| `internal/bot/download_test.go` | `TestDownloadSearchTorrent_Success` | REQ-7 | TEST-3 |
| `internal/bot/download_test.go` | `TestDownloadSearchTorrent_HTMLRejected` | REQ-6 | TEST-4 |
| `internal/bot/download_test.go` | `TestDownloadSearchTorrent_TooLarge` | REQ-3 | TEST-5 |
| `internal/bot/download_test.go` | `TestDownloadSearchTorrent_Timeout` | REQ-4 | TEST-6 |
| `internal/bot/download_test.go` | `TestDownloadSearchTorrent_TooManyRedirects` | REQ-5 | TEST-7 |
| `internal/bot/download_test.go` | `TestDownloadSearchTorrent_HTTPSDowngrade` | REQ-5 | TEST-8 |
| `internal/bot/download_test.go` | `TestNewDownloadClient_Timeout` | REQ-4 | TEST-9 |
| `internal/bot/download_test.go` | `TestNewDownloadClient_RedirectPolicy` | REQ-5 | TEST-8 |
| `internal/bot/callback_test.go` | `TestCallback_SearchConfirmCallback_HTTPDownload_Success` | REQ-1, REQ-7 | TEST-10 |
| `internal/bot/callback_test.go` | `TestCallback_SearchConfirmCallback_HTTPDownload_Failure` | REQ-8 | TEST-11 |
| `internal/bot/callback_test.go` | `TestCallback_SearchConfirmCallback_HTTPDownload_SSRFReject` | REQ-8 | TEST-12 |
| `internal/bot/callback_test.go` | `TestCallback_SearchConfirmCallback_MagnetUnchanged` | REQ-1 | TEST-13 |

## Test Mapping Detail

| Test ID | Test Name | File | Status |
|---------|-----------|------|--------|
| TEST-1 | `TestIsPublicHostname_RejectsLoopback` | `internal/bot/download_test.go` | Complete |
| TEST-2 | `TestIsPublicHostname_RejectsPrivate`, `RejectsLinkLocal`, `AcceptsPublic`, `InvalidHostname` | `internal/bot/download_test.go` | Complete |
| TEST-3 | `TestDownloadSearchTorrent_Success` | `internal/bot/download_test.go` | Complete |
| TEST-4 | `TestDownloadSearchTorrent_HTMLRejected` | `internal/bot/download_test.go` | Complete |
| TEST-5 | `TestDownloadSearchTorrent_TooLarge` | `internal/bot/download_test.go` | Complete |
| TEST-6 | `TestDownloadSearchTorrent_Timeout` | `internal/bot/download_test.go` | Complete |
| TEST-7 | `TestDownloadSearchTorrent_TooManyRedirects` | `internal/bot/download_test.go` | Complete |
| TEST-8 | `TestDownloadSearchTorrent_HTTPSDowngrade` | `internal/bot/download_test.go` | Complete |
| TEST-9 | `TestNewDownloadClient_Timeout`, `TestNewDownloadClient_RedirectPolicy` | `internal/bot/download_test.go` | Complete |
| TEST-10 | `TestCallback_SearchConfirmCallback_HTTPDownload_Success` | `internal/bot/callback_test.go` | Complete |
| TEST-11 | `TestCallback_SearchConfirmCallback_HTTPDownload_Failure` | `internal/bot/callback_test.go` | Complete |
| TEST-12 | `TestCallback_SearchConfirmCallback_HTTPDownload_SSRFReject` | `internal/bot/callback_test.go` | Complete |
| TEST-13 | `TestCallback_SearchConfirmCallback_MagnetUnchanged`, `TestCallback_SearchConfirmCallback_WithMagnet` | `internal/bot/callback_test.go` | Complete |

## Coverage Summary

| Metric | Count | Covered | Gaps |
|--------|-------|---------|------|
| Requirements | 8 | 8 | 0 |
| Acceptance Criteria | 11 | 11 | 0 |
| Design Items | 7 | 7 | 0 |
| Plan Tasks | 9 | 9 | 0 |
| Verification Items | 13 | 13 | 0 |

## Rules

- No REQ-* may exist without at least one linked DES-*.
- No DES-* may exist without at least one linked TASK-*.
- No TASK-* may exist without at least one linked verification item.
- No AC-* may remain unverified.
- Status values: Complete | Partial | Blocked | Missing | N/A

## Harness Validation

```bash
# Count untraced requirements (should be 0)
grep "| TODO |" docs/features/search-torrent-download/traceability.md | wc -l

# Count missing verification (should be 0)
grep "| Missing |" docs/features/search-torrent-download/traceability.md | wc -l
```
