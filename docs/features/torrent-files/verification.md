---
title: "Torrent File Management — Verification"
feature_id: "torrent-files"
status: complete
last_updated: 2026-03-15
---

# Torrent File Management — Verification

## Overview

This document records the verification state for each acceptance criterion in `spec.md`. It is updated during and after Gate 4/5 execution. All results start as TODO and must reach PASS before the feature is considered complete.

Gate 5 is met when:
- Every AC-* has a result of PASS (no TODO, no FAIL).
- Every TEST-* has been executed with evidence.
- `make gate-all` passes.
- `make test-integration` passes.

---

## TEST-1 — `TorrentFile` JSON round-trip and `FilePriority` constant values

**TASK**: TASK-1
**Type**: Unit
**Location**: `internal/qbt/http_test.go` (TestListFiles_ParsesResponse)
**Covers**: REQ-1, REQ-2, REQ-6 / AC-2.4, AC-6.1, AC-6.2, AC-6.3, AC-6.4

**What was verified**:
- `FilePrioritySkip == 0`, `FilePriorityNormal == 1`, `FilePriorityHigh == 6`, `FilePriorityMaximum == 7` (verified by TestPriorityLabel in formatter tests).
- `TestListFiles_ParsesResponse` confirms JSON `{"index":0,"name":"Season 1/ep01.mkv","size":...,"progress":0.5,"priority":1}` unmarshals into `TorrentFile` with Priority == FilePriorityNormal.

**Result**: PASS
**Evidence**: `go test ./internal/qbt/ -run TestListFiles -short -v` — PASS; `go test ./internal/formatter/ -run TestPriorityLabel -short -v` — PASS

---

## CHECK-1 — Compilation passes after interface update

**TASK**: TASK-2
**Type**: Manual / CI check
**Location**: `internal/qbt/client.go`
**Covers**: REQ-1, REQ-4

**What was verified**:
- `make build` passes with `ListFiles` and `SetFilePriority` on `qbt.Client` interface.
- All mock implementations updated (mockQBTClient, errorQBTClient).

**Result**: PASS
**Evidence**: `make build` — 0 errors; `make gate-all` — PASS

---

## TEST-2 — `ListFiles` and `SetFilePriority` HTTP unit tests

**TASK**: TASK-3
**Type**: Unit (httptest)
**Location**: `internal/qbt/http_test.go`
**Covers**: REQ-1, REQ-4 / AC-1.3, AC-4.2, AC-4.4

**What was verified**:
- `TestListFiles_ParsesResponse`: parses JSON array; sends correct `hash` query param.
- `TestListFiles_ErrorOnNon200`: returns non-nil error for 500 response.
- `TestSetFilePriority_SendsCorrectForm`: POST to `/api/v2/torrents/filePrio` with correct `hash`, `id` (pipe-separated), and `priority` fields.
- `TestSetFilePriority_ErrorOnNon200`: returns non-nil error for 400 response.

**Result**: PASS
**Evidence**: `go test ./internal/qbt/ -run 'TestListFiles|TestSetFilePriority' -short -v` — all PASS

---

## TEST-3 — `FormatFileList`, `PriorityLabel` output format

**TASK**: TASK-4
**Type**: Unit
**Location**: `internal/formatter/format_test.go`
**Covers**: REQ-2, REQ-3, REQ-6 / AC-1.2, AC-2.1, AC-2.2, AC-2.3, AC-2.4, AC-3.3, AC-6.1, AC-6.2, AC-6.3, AC-6.4

**What was verified**:
- `TestPriorityLabel`: Skip/Normal/High/Max/Mixed labels.
- `TestFormatFileList_ContainsHeader`: torrent name in header.
- `TestFormatFileList_ShowsLastPathComponent`: only last component shown.
- `TestFormatFileList_TruncatesLongFileName`: truncated to 40 chars with `…`.
- `TestFormatFileList_ShowsPriorityLabel`: priority label in output.
- `TestFormatFileList_PageIndicatorMultiPage`: "Page N/M" present for multi-page.
- `TestFormatFileList_MessageUnderLimit`: ≤4096 chars at worst case.

**Result**: PASS
**Evidence**: `go test ./internal/formatter/ -run 'TestPriorityLabel|TestFormatFileList' -short -v` — all PASS

---

## TEST-4 — `FileListKeyboard` and `PriorityKeyboard` callback data

**TASK**: TASK-5
**Type**: Unit
**Location**: `internal/formatter/format_test.go`
**Covers**: REQ-3, REQ-4, REQ-5 / AC-3.1, AC-4.1, AC-4.3, AC-5.3

**What was verified**:
- `TestFileListKeyboard_FileButtons`: 5 file buttons with `fs:` prefix ≤64 bytes.
- `TestFileListKeyboard_PaginationButtons_FirstPage`: Next but no Prev on page 1.
- `TestFileListKeyboard_PaginationButtons_MiddlePage`: both Prev and Next.
- `TestFileListKeyboard_NoPageButtons_SinglePage`: no `pg:fl:` on single page.
- `TestFileListKeyboard_BackButton`: `bk:fl:` back button present ≤64 bytes.
- `TestFileListKeyboard_AllCallbacksUnderLimit`: all ≤64 bytes at max page numbers.
- `TestPriorityKeyboard_FourPriorityOptions`: exactly 4 `fp:` buttons + back.
- `TestPriorityKeyboard_CurrentMarkedWithCheckmark`: exactly 1 checkmark on current priority.
- `TestPriorityKeyboard_BackButtonIsPgFL`: back button uses `pg:fl:` callback.
- `TestPriorityKeyboard_AllCallbacksUnderLimit`: all ≤64 bytes at worst case.

**Result**: PASS
**Evidence**: `go test ./internal/formatter/ -run 'TestFileListKeyboard|TestPriorityKeyboard' -short -v` — all PASS

---

## TEST-5 — Detail keyboard contains "Files" button

**TASK**: TASK-6
**Type**: Unit
**Location**: `internal/formatter/format_test.go` (TestTorrentDetailKeyboard_FilesButton), `internal/bot/callback_test.go` (TestDetailKeyboardFilesButton)
**Covers**: REQ-5 / AC-5.1

**What was verified**:
- `TestTorrentDetailKeyboard_FilesButton`: row 2 of detail keyboard has `fl:` callback ≤64 bytes.
- `TestTorrentDetailKeyboard_AlwaysBothButtons`: updated to expect 4 rows (added Files row).
- `TestDetailKeyboardFilesButton`: full handler integration — selecting a torrent produces an edited message with `fl:` button in keyboard.

**Result**: PASS
**Evidence**: `go test ./internal/formatter/ -run TestTorrentDetailKeyboard -short -v` — PASS; `go test ./internal/bot/ -run TestDetailKeyboardFilesButton -short -v` — PASS

---

## TEST-6 — Callback routing unit tests

**TASK**: TASK-7
**Type**: Unit (mock qbt.Client)
**Location**: `internal/bot/callback_test.go`
**Covers**: REQ-1, REQ-3, REQ-4, REQ-5 / AC-1.1, AC-1.2, AC-1.3, AC-3.2, AC-4.1, AC-4.2, AC-4.4, AC-5.2, AC-5.3

**What was verified**:
- `TestCallbackFL_ShowsFileList`: `fl:` calls ListFiles with correct hash; torrent name in edited message (AC-1.1, AC-1.2).
- `TestCallbackFL_ListFilesError`: `fl:` on error answers "Failed to load files" (AC-1.3).
- `TestCallbackPgFL_NavigatesToCorrectPage`: `pg:fl:` renders page 2/2 header (AC-3.2).
- `TestCallbackFS_ShowsPriorityKeyboard`: `fs:` edits message without calling SetFilePriority (AC-4.1).
- `TestCallbackFP_SetsFilePriorityAndRefreshes`: `fp:` calls SetFilePriority + re-renders file list (AC-4.2).
- `TestCallbackFP_SetPriorityError_AnswersWithError`: `fp:` error answers "Failed to set priority" (AC-4.4).
- `TestCallbackBkFL_ReturnsToDetailView`: `bk:fl:` shows detail view with Size: (AC-5.2).

**Result**: PASS
**Evidence**: `go test ./internal/bot/ -run 'TestCallbackFL|TestCallbackPgFL|TestCallbackFS|TestCallbackFP|TestCallbackBkFL|TestDetailKeyboard' -short -v` — all PASS

---

## TEST-7 — `ListFiles` integration test against real qBittorrent

**TASK**: TASK-8
**Type**: Integration
**Build tag**: `//go:build integration`
**Location**: `internal/qbt/http_integration_test.go` (TestIntegration_ListFiles)
**Covers**: REQ-1 / AC-1.1, AC-1.3

**What was verified**:
- `TestIntegration_ListFiles`: against live qBittorrent (Docker) returns non-nil slice for a known torrent hash; each file has non-empty Name and non-negative Priority.

**Result**: PASS
**Evidence**: `make test-integration` — TestIntegration_ListFiles PASS

---

## TEST-8 — `SetFilePriority` integration test with observable state change

**TASK**: TASK-8
**Type**: Integration
**Build tag**: `//go:build integration`
**Location**: `internal/qbt/http_integration_test.go` (TestIntegration_SetFilePriority)
**Covers**: REQ-4 / AC-4.2, AC-4.4

**What was verified**:
- `TestIntegration_SetFilePriority`: sets priority to Skip and restores original priority; no error returned.

**Result**: PASS
**Evidence**: `make test-integration` — TestIntegration_SetFilePriority PASS

---

## Gate 5 Checklist

Before marking this feature complete, confirm all items below:

- [x] TEST-1: PASS
- [x] CHECK-1: PASS
- [x] TEST-2: PASS
- [x] TEST-3: PASS
- [x] TEST-4: PASS
- [x] TEST-5: PASS
- [x] TEST-6: PASS
- [x] TEST-7: PASS
- [x] TEST-8: PASS
- [x] `make gate-all` passes (build + lint + unit tests)
- [x] `make test-integration` passes
- [x] All 22 AC-* have result PASS in traceability.md
- [x] Backward traceability table in traceability.md is fully populated
- [x] No TODO results remain in this document
