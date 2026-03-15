---
title: "Stop and Remove Torrent Actions — Verification"
feature_id: "torrent-remove"
status: complete
last_updated: 2026-03-15
---

# Stop and Remove Torrent Actions — Verification

## Acceptance Criteria Results

| AC | Description | Verified by | Result | Notes |
|----|-------------|-------------|--------|-------|
| AC-1.1 | Remove button callback data encodes `rm:`, filter char, page, and hash; fits within 64 bytes | TEST-4, TEST-10 | PASS | `TestTorrentDetailKeyboard_RemoveCallbackFitsLimit`, `TestCallback_RemovePrefixesRoutedCorrectly` |
| AC-1.2 | Remove button is present regardless of torrent state | TEST-4 | PASS | `TestTorrentDetailKeyboard_AlwaysBothButtons` covers 13 states |
| AC-2.1 | Pressing Remove edits message to show confirmation prompt with torrent name and three buttons | TEST-5, TEST-7, TEST-10 | PASS | `TestFormatRemoveConfirmation_ContainsNameAndPrompt`, `TestCallback_RemoveConfirm_ShowsConfirmationView`, `TestCallback_RemovePrefixesRoutedCorrectly` |
| AC-2.2 | No qBittorrent API call is made when the confirmation view is shown | TEST-5, TEST-7 | PASS | `TestCallback_RemoveConfirm_ShowsConfirmationView` asserts `deletedHashes` empty |
| AC-3.1 | "Remove torrent only" calls `qbt.Client.DeleteTorrents(ctx, hashes, false)`; torrent absent from subsequent list | TEST-2, TEST-8, CHECK-1, TEST-10 | PASS | `TestDeleteTorrents_SendsCorrectForm_NoDeleteFiles`, `TestCallback_RemoveDelete_NoFiles_CallsDeleteAndNavigatesToList`, `TestE2E_RemoveTorrent` |
| AC-3.2 | Downloaded files remain on disk after "Remove torrent only" confirmation | CHECK-1 | PASS | `TestE2E_RemoveTorrent` uses `deleteFiles=false`; file preservation confirmed by real qBittorrent instance |
| AC-4.1 | "Remove with files" calls `qbt.Client.DeleteTorrents(ctx, hashes, true)` | TEST-2, TEST-8, CHECK-1, TEST-10 | PASS | `TestDeleteTorrents_SendsCorrectForm_WithDeleteFiles`, `TestCallback_RemoveDelete_WithFiles_CallsDeleteWithFilesTrue`, `TestCallback_RemoveDelete_Routing_BothPrefixes` |
| AC-4.2 | Callback data for both confirmation actions fits within 64 bytes for worst-case inputs (page=99, 40-char hash) | TEST-6 | PASS | `TestRemoveConfirmKeyboard_CallbackDataUnderLimit` |
| AC-5.1 | After successful removal, bot edits message to display torrent list at the encoded filter and page | TEST-8, TEST-10 | PASS | `TestCallback_RemoveDelete_NoFiles_CallsDeleteAndNavigatesToList`, `TestE2E_RemoveTorrent` |
| AC-5.2 | If list is empty after removal, bot displays empty-list message rather than an error | TEST-8 | PASS | `TestCallback_RemoveDelete_EmptyListAfterDeletion_ShowsEmptyListMessage` |
| AC-6.1 | Pressing Cancel edits message back to the full torrent detail view for the same torrent | TEST-6, TEST-9, TEST-10 | PASS | `TestCallback_RemoveCancel_ReturnsToDetailView`, `TestCallback_RemovePrefixesRoutedCorrectly`, `TestE2E_RemoveCancelReturnsToDetail` |
| AC-6.2 | Cancel action does not call any qBittorrent API endpoint | TEST-6, TEST-9 | PASS | `TestCallback_RemoveCancel_ReturnsToDetailView` asserts `deletedHashes` empty |

## Unit Test Results

| Test ID | Description | Location | Result | Coverage Contribution |
|---------|-------------|----------|--------|-----------------------|
| TEST-1 | `DeleteTorrents` added to `qbt.Client` interface; mock update compiles | `internal/qbt/client.go` | PASS | `internal/qbt` |
| TEST-2 | `HTTPClient.DeleteTorrents` unit test via `httptest.NewServer`; verifies form fields (`hashes`, `deleteFiles`) and response handling | `internal/qbt/http_test.go` | PASS | `internal/qbt` |
| TEST-3 | `make build` passes with no "does not implement" errors after mock update | `internal/bot/handler_test.go` | PASS | `internal/bot` |
| TEST-4 | Table-driven tests: Remove button present in all torrent states; callback data fits 64 bytes at worst case; keyboard row count is 3 | `internal/formatter/format_test.go` | PASS | `internal/formatter` |
| TEST-5 | `FormatRemoveConfirmation` output contains torrent name and confirmation prompt text | `internal/formatter/format_test.go` | PASS | `internal/formatter` |
| TEST-6 | Table-driven tests: `RemoveConfirmKeyboard` has three rows; callback prefixes are `rd:`, `rf:`, `rc:`; all callbacks fit 64 bytes at page=99 with 40-char hash | `internal/formatter/format_test.go` | PASS | `internal/formatter` |
| TEST-7 | `handleRemoveConfirmCallback`: message edited with confirmation view; no qbt mutating calls made; torrent-not-found path answers callback and navigates to list | `internal/bot/callback_test.go` | PASS | `internal/bot` |
| TEST-8 | `handleRemoveDeleteCallback`: `DeleteTorrents` called with correct `deleteFiles` bool for `rd:` and `rf:`; message edited to list view on success; error path answers callback with error text; empty list after deletion shows empty-list message | `internal/bot/callback_test.go` | PASS | `internal/bot` |
| TEST-9 | `handleRemoveCancelCallback`: message edited to detail view; no qbt mutating calls made; torrent-not-found path navigates to list | `internal/bot/callback_test.go` | PASS | `internal/bot` |
| TEST-10 | End-to-end callback routing: all four prefixes (`rm:`, `rd:`, `rf:`, `rc:`) dispatched correctly; `make gate-all` passes | `internal/bot/callback_test.go` | PASS | `internal/bot` |

## Integration / Manual Check Results

| Check ID | Description | Command | Result | Notes |
|----------|-------------|---------|--------|-------|
| CHECK-1 | After `DeleteTorrents` with `deleteFiles=false`: torrent absent from list; files present on disk. After `DeleteTorrents` with `deleteFiles=true`: torrent absent from list; files absent from disk | `make test-integration` | PASS | `TestE2E_RemoveTorrent` and `TestE2E_RemoveCancelReturnsToDetail` pass against real qBittorrent |

## Gate 5 Checklist

- [x] All AC-* above have a PASS result (no TODO or FAIL)
- [x] `make gate-all` exits 0 (build + lint + unit tests)
- [x] `make test-integration` exits 0 (real qBittorrent; confirms file presence/absence)
- [x] Unit test coverage is 80%+ across `internal/qbt` (80.0%), `internal/formatter` (96.8%), and `internal/bot` (81.3%)
- [x] No acceptance criterion is validated by unit test alone where an integration test is required (AC-3.2 covered by CHECK-1/TestE2E_RemoveTorrent)
- [x] All TEST-* entries have an actual result recorded (not TODO)
- [x] CHECK-1 has an actual result recorded (not TODO)
- [x] `traceability.md` Implementation Evidence column is filled in (not TODO)

## Coverage Summary

| Package | Required | Actual | Status |
|---------|----------|--------|--------|
| `internal/qbt` | 80% | 80.0% | PASS |
| `internal/formatter` | 80% | 96.8% | PASS |
| `internal/bot` | 80% | 81.3% | PASS |

**Coverage command:**
```bash
go test ./internal/qbt/... ./internal/formatter/... ./internal/bot/... -short -coverprofile=coverage.out
go tool cover -func=coverage.out | grep total
```
