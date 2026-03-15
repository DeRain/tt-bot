---
title: "Stop and Remove Torrent Actions — Traceability Matrix"
feature_id: "torrent-remove"
status: complete
last_updated: 2026-03-15
---

# Stop and Remove Torrent Actions — Traceability Matrix

## Forward Traceability: Requirements → Design → Plan → Tests

| REQ | AC | DES | TASK | TEST / CHECK | Implementation Evidence | Status |
|-----|----|-----|------|--------------|------------------------|--------|
| REQ-1 | AC-1.1, AC-1.2 | DES-3, DES-8 | TASK-4, TASK-10 | TEST-4, TEST-10 | `formatter.TorrentDetailKeyboard` adds `rm:` row; `TestTorrentDetailKeyboard_AlwaysBothButtons`, `TestTorrentDetailKeyboard_RemoveCallbackFitsLimit` | PASS |
| REQ-2 | AC-2.1, AC-2.2 | DES-4, DES-5, DES-8 | TASK-5, TASK-6, TASK-7, TASK-10 | TEST-5, TEST-6, TEST-7, TEST-10 | `formatter.FormatRemoveConfirmation`, `formatter.RemoveConfirmKeyboard`, `handleRemoveConfirmCallback`; `TestCallback_RemoveConfirm_ShowsConfirmationView` | PASS |
| REQ-3 | AC-3.1, AC-3.2 | DES-1, DES-2, DES-6, DES-8 | TASK-1, TASK-2, TASK-3, TASK-8, TASK-10 | TEST-1, TEST-2, TEST-8, CHECK-1, TEST-10 | `qbt.Client.DeleteTorrents`, `HTTPClient.DeleteTorrents`, `handleRemoveDeleteCallback` with `deleteFiles=false`; `TestE2E_RemoveTorrent` | PASS |
| REQ-4 | AC-4.1, AC-4.2 | DES-1, DES-2, DES-4, DES-6, DES-8 | TASK-1, TASK-2, TASK-3, TASK-6, TASK-8, TASK-10 | TEST-1, TEST-2, TEST-6, TEST-8, CHECK-1, TEST-10 | `HTTPClient.DeleteTorrents` with `deleteFiles=true`; `TestCallback_RemoveDelete_WithFiles_CallsDeleteWithFilesTrue`, `TestRemoveConfirmKeyboard_CallbackDataUnderLimit` | PASS |
| REQ-5 | AC-5.1, AC-5.2 | DES-6, DES-8 | TASK-8, TASK-10 | TEST-8, TEST-10 | `handleRemoveDeleteCallback` navigates to list; `TestCallback_RemoveDelete_NoFiles_CallsDeleteAndNavigatesToList`, `TestCallback_RemoveDelete_EmptyListAfterDeletion_ShowsEmptyListMessage` | PASS |
| REQ-6 | AC-6.1, AC-6.2 | DES-4, DES-7, DES-8 | TASK-6, TASK-9, TASK-10 | TEST-6, TEST-9, TEST-10 | `handleRemoveCancelCallback` returns to detail view; `TestCallback_RemoveCancel_ReturnsToDetailView`, `TestE2E_RemoveCancelReturnsToDetail` | PASS |

## Backward Traceability: Tests → Plan → Design → Requirements

| TEST / CHECK | TASK | DES | REQ | AC | Status |
|--------------|------|-----|-----|----|--------|
| TEST-1 | TASK-1 | DES-1 | REQ-3, REQ-4 | AC-3.1, AC-4.1 | PASS |
| TEST-2 | TASK-2 | DES-2 | REQ-3, REQ-4 | AC-3.1, AC-3.2, AC-4.1 | PASS |
| CHECK-1 | TASK-2 | DES-2 | REQ-3, REQ-4 | AC-3.1, AC-3.2, AC-4.1 | PASS |
| TEST-3 | TASK-3 | DES-1 | REQ-3, REQ-4 | AC-3.1, AC-4.1 | PASS |
| TEST-4 | TASK-4 | DES-3 | REQ-1 | AC-1.1, AC-1.2 | PASS |
| TEST-5 | TASK-5 | DES-4 | REQ-2 | AC-2.1, AC-2.2 | PASS |
| TEST-6 | TASK-6 | DES-4 | REQ-2, REQ-4, REQ-6 | AC-2.1, AC-4.2, AC-6.1, AC-6.2 | PASS |
| TEST-7 | TASK-7 | DES-5 | REQ-2 | AC-2.1, AC-2.2 | PASS |
| TEST-8 | TASK-8 | DES-6 | REQ-3, REQ-4, REQ-5 | AC-3.1, AC-4.1, AC-5.1, AC-5.2 | PASS |
| TEST-9 | TASK-9 | DES-7 | REQ-6 | AC-6.1, AC-6.2 | PASS |
| TEST-10 | TASK-10 | DES-8 | REQ-1, REQ-2, REQ-3, REQ-4, REQ-5, REQ-6 | AC-1.1, AC-2.1, AC-3.1, AC-4.1, AC-5.1, AC-6.1 | PASS |

## AC Coverage Matrix

| AC | Requirement | Covered by DES | Verified by TEST / CHECK | Status |
|----|-------------|----------------|--------------------------|--------|
| AC-1.1 | REQ-1 | DES-3, DES-8 | TEST-4, TEST-10 | PASS |
| AC-1.2 | REQ-1 | DES-3 | TEST-4 | PASS |
| AC-2.1 | REQ-2 | DES-4, DES-5, DES-8 | TEST-5, TEST-7, TEST-10 | PASS |
| AC-2.2 | REQ-2 | DES-4, DES-5 | TEST-5, TEST-7 | PASS |
| AC-3.1 | REQ-3 | DES-1, DES-2, DES-6, DES-8 | TEST-2, TEST-8, CHECK-1, TEST-10 | PASS |
| AC-3.2 | REQ-3 | DES-2 | CHECK-1 | PASS |
| AC-4.1 | REQ-4 | DES-1, DES-2, DES-6, DES-8 | TEST-2, TEST-8, CHECK-1, TEST-10 | PASS |
| AC-4.2 | REQ-4 | DES-4 | TEST-6 | PASS |
| AC-5.1 | REQ-5 | DES-6, DES-8 | TEST-8, TEST-10 | PASS |
| AC-5.2 | REQ-5 | DES-6 | TEST-8 | PASS |
| AC-6.1 | REQ-6 | DES-4, DES-7, DES-8 | TEST-6, TEST-9, TEST-10 | PASS |
| AC-6.2 | REQ-6 | DES-4, DES-7 | TEST-6, TEST-9 | PASS |

## Design Item Coverage

| DES | Satisfies REQ | Covers AC | Implemented by TASK | Status |
|-----|---------------|-----------|---------------------|--------|
| DES-1 | REQ-3, REQ-4 | AC-3.1, AC-4.1 | TASK-1, TASK-3 | PASS |
| DES-2 | REQ-3, REQ-4 | AC-3.1, AC-3.2, AC-4.1 | TASK-2 | PASS |
| DES-3 | REQ-1 | AC-1.1, AC-1.2 | TASK-4 | PASS |
| DES-4 | REQ-2, REQ-6 | AC-2.1, AC-2.2, AC-4.2, AC-6.1, AC-6.2 | TASK-5, TASK-6 | PASS |
| DES-5 | REQ-2 | AC-2.1, AC-2.2 | TASK-7 | PASS |
| DES-6 | REQ-3, REQ-4, REQ-5 | AC-3.1, AC-4.1, AC-5.1, AC-5.2 | TASK-8 | PASS |
| DES-7 | REQ-6 | AC-6.1, AC-6.2 | TASK-9 | PASS |
| DES-8 | REQ-1, REQ-2, REQ-3, REQ-4, REQ-5, REQ-6 | AC-1.1, AC-2.1, AC-3.1, AC-4.1, AC-5.1, AC-6.1 | TASK-10 | PASS |

## Task Completion Status

| TASK | Derived from DES | Implements REQ | Verification Target | Affected Files | Status |
|------|-----------------|----------------|---------------------|----------------|--------|
| TASK-1 | DES-1 | REQ-3, REQ-4 | TEST-1 | `internal/qbt/client.go` | PASS |
| TASK-2 | DES-2 | REQ-3, REQ-4 | TEST-2, CHECK-1 | `internal/qbt/http.go`, `internal/qbt/http_test.go` | PASS |
| TASK-3 | DES-1 | REQ-3, REQ-4 | TEST-3 | `internal/bot/handler_test.go`, `internal/bot/callback_test.go`, `internal/poller/poller_test.go` | PASS |
| TASK-4 | DES-3 | REQ-1 | TEST-4 | `internal/formatter/format.go`, `internal/formatter/format_test.go` | PASS |
| TASK-5 | DES-4 | REQ-2 | TEST-5 | `internal/formatter/format.go`, `internal/formatter/format_test.go` | PASS |
| TASK-6 | DES-4 | REQ-2, REQ-6 | TEST-6 | `internal/formatter/format.go`, `internal/formatter/format_test.go` | PASS |
| TASK-7 | DES-5 | REQ-2 | TEST-7 | `internal/bot/callback.go`, `internal/bot/callback_test.go` | PASS |
| TASK-8 | DES-6 | REQ-3, REQ-4, REQ-5 | TEST-8 | `internal/bot/callback.go`, `internal/bot/callback_test.go` | PASS |
| TASK-9 | DES-7 | REQ-6 | TEST-9 | `internal/bot/callback.go`, `internal/bot/callback_test.go` | PASS |
| TASK-10 | DES-8 | REQ-1, REQ-2, REQ-3, REQ-4, REQ-5, REQ-6 | TEST-10 | `internal/bot/callback.go`, `internal/bot/callback_test.go`, `internal/bot/e2e_test.go` | PASS |
| TASK-11 | DES-1 through DES-8 | REQ-1 through REQ-6 | `make gate-all`, `make test-integration` | `docs/features/torrent-remove/traceability.md`, `docs/features/torrent-remove/verification.md` | PASS |
