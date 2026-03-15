---
title: Add Torrent - Implementation Plan
feature_id: add-torrent
depends_on_design: docs/features/add-torrent/design.md
last_updated: 2026-03-15
---

# Add Torrent - Implementation Plan

## Tasks

### TASK-1: PendingTorrent Struct and Map

**Derived from:** DES-1
**Implements:** REQ-5, REQ-6
**Impacts:** `internal/bot/handler.go`
**Verification:** Unit test -- verify `storePending` overwrites existing entry; verify `takePending` retrieves and removes entry, returns nil when empty.

Define the `PendingTorrent` struct with fields `MagnetLink string`, `FileData []byte`, `FileName string`, and `CreatedAt time.Time`. Add `pending map[int64]*PendingTorrent` and `mu sync.Mutex` to the `Handler` struct. Implement `storePending(chatID, pt)` and `takePending(chatID)` methods with mutex protection.

### TASK-2: Magnet Link Handling

**Derived from:** DES-2
**Implements:** REQ-1
**Impacts:** `internal/bot/handler.go`
**Verification:** Unit tests `TestHandler_MagnetLink_StoresPendingAndShowsCategories`, `TestHandler_MagnetLink_MidText` -- verify magnet extraction from standalone and embedded text, pending storage, and category keyboard display.

Implement `handleMagnet(ctx, msg)`: scan `msg.Text` for `magnet:?`, extract URI to next whitespace or end of string, call `storePending`, call `sendCategoryKeyboard`. Wire into `HandleUpdate` dispatch after command check with `strings.Contains(msg.Text, "magnet:?")`.

### TASK-3: .torrent File Handling

**Derived from:** DES-3
**Implements:** REQ-2
**Impacts:** `internal/bot/handler.go`
**Verification:** Unit tests `TestHandler_TorrentFile_StoresPendingAndShowsCategories`, `TestDownloadFile_Success`, `TestDownloadFile_HTTPError` -- verify file download, pending storage, and error handling.

Implement `handleTorrentFile(ctx, msg)`: check `msg.Document` for `.torrent` suffix, call `sender.GetFile()` for file path, download bytes via `downloadFile(ctx, filePath)`, store in `PendingTorrent{FileData, FileName}`, call `sendCategoryKeyboard`. Implement `downloadFile` with token-based URL construction and error sanitization.

### TASK-4: CategoryKeyboard Formatter

**Derived from:** DES-4, DES-8
**Implements:** REQ-3, REQ-4, REQ-7
**Impacts:** `internal/formatter/format.go`
**Verification:** Unit tests `TestCategoryKeyboard_Normal`, `TestCategoryKeyboard_Empty`, `TestCategoryKeyboard_LongNameTruncated`, `TestCategoryKeyboard_CallbackDataUnderLimit` -- verify keyboard construction, empty-category fallback, and 64-byte truncation with UTF-8 alignment.

Implement `CategoryKeyboard(categories []qbt.Category) Keyboard`: iterate categories, build `cat:<name>` callback data, truncate to `MaxCallbackData` bytes with UTF-8 back-off, return "No category" button when list is empty.

### TASK-5: Category Callback Handler

**Derived from:** DES-5
**Implements:** REQ-1, REQ-2, REQ-3, REQ-8, REQ-9
**Impacts:** `internal/bot/callback.go`
**Verification:** Unit tests `TestCallback_CategoryWithPendingMagnet_CallsAddMagnet`, `TestCallback_CategoryWithNoPending_ReturnsError`, `TestCallback_CategoryWithNoCategory_ShowsGenericConfirm`, `TestCallback_AddMagnetError` -- verify magnet/file dispatch, expired-entry handling, confirmation editing, and error propagation.

Implement `handleCategoryCallback(ctx, cq, category)`: call `takePending(chatID)`, dispatch to `AddMagnet` or `AddTorrentFile` based on `PendingTorrent` contents, edit message with confirmation or error, answer callback to dismiss spinner. Wire into `handleCallback` with `strings.HasPrefix(data, "cat:")`.

### TASK-6: TTL Eviction Goroutine

**Derived from:** DES-6
**Implements:** REQ-5
**Impacts:** `internal/bot/handler.go`
**Verification:** Unit test `TestEvictExpired_RemovesOldEntries` -- verify stale entries are removed while fresh entries survive.

Implement `runCleanup(ctx)`: create 1-minute ticker, on each tick call `evictExpired()`. Implement `evictExpired()`: lock mutex, compute cutoff as `time.Now().Add(-pendingTTL)`, delete entries with `CreatedAt.Before(cutoff)`. Launch goroutine from `New()` with the provided context.

### TASK-7: qBittorrent AddMagnet and AddTorrentFile

**Derived from:** DES-5 (depends on qbt client interface)
**Implements:** REQ-1, REQ-2
**Impacts:** `internal/qbt/http.go`
**Verification:** Integration tests via `make test-integration` -- verify actual API calls against a Docker qBittorrent instance; E2E test `TestE2E_AddMagnetWithCategorySelection`.

Implement `AddMagnet(ctx, magnet, category)`: build multipart form with `urls` and `category` fields, POST to `/api/v2/torrents/add` via `doWithAuth`. Implement `AddTorrentFile(ctx, filename, data, category)`: buffer file data for retry safety, build multipart form with `torrents` file part and `category` field, POST via `doWithRetry` with a request-builder closure.

### TASK-8: Test Coverage

**Derived from:** All design items
**Implements:** All requirements (verification)
**Impacts:** `internal/bot/handler_test.go`, `internal/bot/callback_test.go`, `internal/bot/handler_extra_test.go`, `internal/bot/e2e_test.go`, `internal/formatter/format_test.go`
**Verification:** `make gate-all` passes; `make test-integration` passes; coverage meets 80% threshold.

Write unit tests for all handler, callback, and formatter functions. Write integration/E2E tests for the full magnet-to-confirmation flow against a real qBittorrent instance. Ensure mock implementations of `qbt.Client` and `bot.Sender` cover all error paths.

## Quality Gates

### Gate 3: Plan Gate

- [x] Every design item (DES-*) maps to at least one task (TASK-*)
- [x] Every task has a verification method defined
- [x] Task dependency order is documented
- [x] All tasks have impacts (files) identified
- [x] No TODO placeholders remain in task descriptions

#### Harness Check

```bash
# Verify task count = verification count (every task has a verification)
TASK_COUNT=$(grep -c '^### TASK-' docs/features/add-torrent/plan.md)
VERIF_COUNT=$(grep -c '^\*\*Verification:\*\*' docs/features/add-torrent/plan.md)
echo "Tasks: $TASK_COUNT, Verifications: $VERIF_COUNT"
# Expected: counts are equal
```

#### Iterative Harness Loop Protocol

1. Execute tasks in dependency order (TASK-1 through TASK-8).
2. After each task, run its verification method before proceeding.
3. If verification fails, retry the task with fixes (max 3 retries per task).
4. After all tasks pass individually, run `make gate-all` for full quality gate.
5. Update the Requirement-to-Design mapping and Task traceability if any changes occurred.
6. Run all verification checks end-to-end to confirm no regressions.

## Task Dependency Order

```
TASK-1 (pending struct/map)
  |
  +---> TASK-2 (magnet handling)
  |       |
  +---> TASK-3 (.torrent handling)
  |       |
  +---> TASK-6 (TTL eviction)
  |
  +---> TASK-4 (CategoryKeyboard)
  |       |
  +---> TASK-5 (category callback) <--- TASK-7 (qbt AddMagnet/AddTorrentFile)
  |
  +---> TASK-8 (tests -- depends on all above)
```
