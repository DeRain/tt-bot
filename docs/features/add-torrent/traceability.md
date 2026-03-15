---
title: Add Torrent - Traceability Matrix
feature_id: add-torrent
last_updated: 2026-03-15
---

# Add Torrent - Traceability Matrix

| Requirement | AC | Design | Tasks | Implementation Evidence | Verification | Status |
|-------------|-----|--------|-------|------------------------|-------------|--------|
| REQ-1: Magnet link support | AC-1.1 | DES-2, DES-5 | TASK-2, TASK-5, TASK-7 | `handler.go:handleMagnet` extracts magnet URI and stores pending; `callback.go:handleCategoryCallback` calls `qbt.Client.AddMagnet`; `http.go:AddMagnet` POSTs multipart form to qBittorrent API | `TestHandler_MagnetLink_StoresPendingAndShowsCategories`, `TestHandler_MagnetLink_MidText`, `TestCallback_CategoryWithPendingMagnet_CallsAddMagnet`, `TestE2E_AddMagnetWithCategorySelection` | PASS |
| REQ-2: .torrent file support | AC-2.1 | DES-3, DES-5 | TASK-3, TASK-5, TASK-7 | `handler.go:handleTorrentFile` downloads file from Telegram CDN and stores pending; `callback.go:handleCategoryCallback` calls `qbt.Client.AddTorrentFile`; `http.go:AddTorrentFile` POSTs multipart file upload | `TestHandler_TorrentFile_StoresPendingAndShowsCategories`, `TestDownloadFile_Success`, `TestDownloadFile_HTTPError` | PASS |
| REQ-3: Category selection required | AC-3.1 | DES-4, DES-5 | TASK-4, TASK-5 | `handleMagnet` and `handleTorrentFile` call `sendCategoryKeyboard` instead of adding directly; `handleCategoryCallback` performs the actual add only after user selects | `TestCallback_CategoryWithPendingMagnet_CallsAddMagnet`, `TestCallback_CategoryWithNoCategory_ShowsGenericConfirm` | PASS |
| REQ-4: Dynamic category fetch | AC-4.1 | DES-4 | TASK-4 | `sendCategoryKeyboard` calls `qbt.Client.Categories()` on each request; `formatter.CategoryKeyboard` builds keyboard from returned list | `TestCategoryKeyboard_Normal`, `TestCategoryKeyboard_Empty` | PASS |
| REQ-5: 5-minute pending TTL | AC-5.1 | DES-1, DES-6 | TASK-1, TASK-6 | `pendingTTL = 5 * time.Minute` constant; `evictExpired` deletes entries older than cutoff; `runCleanup` ticks every 1 minute | `TestEvictExpired_RemovesOldEntries` | PASS |
| REQ-6: One pending per user | AC-6.1 | DES-1 | TASK-1 | `storePending` unconditionally overwrites `h.pending[chatID]`; map keyed by chat ID guarantees single entry | `TestHandler_MagnetLink_StoresPendingAndShowsCategories` (implicit -- stores and verifies single entry) | PASS |
| REQ-7: 64-byte callback limit | AC-7.1 | DES-4, DES-8 | TASK-4 | `formatter.CategoryKeyboard` truncates `cat:<name>` to `MaxCallbackData` (64) bytes with UTF-8 back-off | `TestCategoryKeyboard_LongNameTruncated`, `TestCategoryKeyboard_CallbackDataUnderLimit`, `TestAllCallbackDataUnderLimit` | PASS |
| REQ-8: Confirmation message | AC-8.1 | DES-5, DES-7 | TASK-5 | `handleCategoryCallback` calls `editMessageText` with "Torrent added to <category>!" or "Torrent added!" on success | `TestCallback_CategoryWithPendingMagnet_CallsAddMagnet` (checks edit text contains category), `TestCallback_CategoryWithNoCategory_ShowsGenericConfirm` | PASS |
| REQ-9: Expired pending error | AC-9.1 | DES-5 | TASK-5 | `handleCategoryCallback` returns "No pending torrent. Please resend the magnet link or file." when `takePending` returns nil | `TestCallback_CategoryWithNoPending_ReturnsError` | PASS |
