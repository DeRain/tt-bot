---
title: "Torrent File Management — Verification"
feature_id: "torrent-files"
status: draft
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
**Location**: `internal/qbt/types_test.go` (or `internal/qbt/http_test.go`)
**Covers**: REQ-1, REQ-2, REQ-6 / AC-2.4, AC-6.1, AC-6.2, AC-6.3, AC-6.4

**What to verify**:
- `FilePrioritySkip == 0`, `FilePriorityNormal == 1`, `FilePriorityHigh == 6`, `FilePriorityMaximum == 7`.
- A sample JSON object `{"index":0,"name":"Season 1/ep01.mkv","size":1073741824,"progress":0.5,"priority":1}` unmarshals into a `TorrentFile` with all fields set correctly.

**Run command**:
```bash
go test ./internal/qbt/ -run TestFilePriority -short -v
go test ./internal/qbt/ -run TestTorrentFileUnmarshal -short -v
```

**Result**: TODO
**Evidence**: —

---

## CHECK-1 — Compilation passes after interface update

**TASK**: TASK-2
**Type**: Manual / CI check
**Location**: `internal/qbt/client.go`
**Covers**: REQ-1, REQ-4

**What to verify**:
- `make build` passes after `ListFiles` and `SetFilePriority` are added to the `qbt.Client` interface.
- Any existing mock or stub implementing `qbt.Client` in tests is updated to satisfy the new interface (or the build fails with a clear missing-method error, which is then fixed).

**Run command**:
```bash
make build
```

**Result**: TODO
**Evidence**: —

---

## TEST-2 — `ListFiles` and `SetFilePriority` HTTP unit tests

**TASK**: TASK-3
**Type**: Unit (httptest)
**Location**: `internal/qbt/http_test.go`
**Covers**: REQ-1, REQ-4 / AC-1.3, AC-4.2, AC-4.4

**What to verify**:
- `ListFiles` correctly parses a JSON array response from a fake `httptest.NewServer`.
- `ListFiles` returns a wrapped error when the server returns a non-200 status; error message is non-empty.
- `SetFilePriority` sends a POST request with the correct form fields (`hash`, `id`, `priority`).
- `SetFilePriority` returns nil on 200 OK and a non-nil error on non-200.

**Run command**:
```bash
go test ./internal/qbt/ -run TestListFiles -short -v
go test ./internal/qbt/ -run TestSetFilePriority -short -v
```

**Result**: TODO
**Evidence**: —

---

## TEST-3 — `FormatFileList`, `PriorityLabel` output format

**TASK**: TASK-4
**Type**: Unit
**Location**: `internal/formatter/format_test.go`
**Covers**: REQ-2, REQ-3, REQ-6 / AC-1.2, AC-2.1, AC-2.2, AC-2.3, AC-2.4, AC-3.3, AC-6.1, AC-6.2, AC-6.3, AC-6.4

**What to verify**:
- `PriorityLabel(0)` returns `"Skip"`.
- `PriorityLabel(1)` returns `"Normal"`.
- `PriorityLabel(6)` returns `"High"`.
- `PriorityLabel(7)` returns `"Max"`.
- `PriorityLabel` on an unknown value (e.g., `4`) returns `"Mixed"` (or similar fallback).
- `FormatFileList` output contains the torrent name header.
- File name with last path component longer than 40 UTF-8 chars is truncated with `…`.
- File name with a `/`-separated path shows only the last component.
- File size is formatted as a human-readable string (e.g., `1.2 GB`).
- Progress bar and numeric percentage appear in the output.
- Page indicator (e.g., `Page 1/3`) is present when `totalPages > 1`.
- Full message length is ≤ 4096 bytes for a 5-file page with maximum-length names and sizes.

**Run command**:
```bash
go test ./internal/formatter/ -run TestPriorityLabel -short -v
go test ./internal/formatter/ -run TestFormatFileList -short -v
```

**Result**: TODO
**Evidence**: —

---

## TEST-4 — `FileListKeyboard` and `PriorityKeyboard` callback data

**TASK**: TASK-5
**Type**: Unit
**Location**: `internal/formatter/format_test.go`
**Covers**: REQ-3, REQ-4, REQ-5 / AC-3.1, AC-4.1, AC-4.3, AC-5.3

**What to verify**:
- `FileListKeyboard` for a 6-file torrent (page 1 of 2) produces:
  - 5 file-tap buttons whose callback data starts with `fs:` and is ≤ 64 bytes.
  - A `pg:fl:` next-page button (no prev on page 1).
  - A `bk:fl:` back button.
- `FileListKeyboard` on a subsequent page produces both prev and next `pg:fl:` buttons.
- `FileListKeyboard` for a ≤5-file torrent produces no prev/next buttons.
- All `fs:` and `pg:fl:` callback data strings are ≤ 64 bytes.
- `PriorityKeyboard` produces exactly 4 priority buttons.
- The button for the current priority has a checkmark prefix.
- `fp:` callback data strings are ≤ 64 bytes.
- Back button in `PriorityKeyboard` encodes a `pg:fl:` callback that returns to the correct file list page.

**Run command**:
```bash
go test ./internal/formatter/ -run TestFileListKeyboard -short -v
go test ./internal/formatter/ -run TestPriorityKeyboard -short -v
```

**Result**: TODO
**Evidence**: —

---

## TEST-5 — Detail keyboard contains "Files" button

**TASK**: TASK-6
**Type**: Unit
**Location**: `internal/bot/handler_test.go`
**Covers**: REQ-5 / AC-5.1

**What to verify**:
- The inline keyboard returned by `DetailKeyboard` (or the equivalent function in `handler.go`) for a torrent with a non-empty hash contains at least one button whose callback data starts with `fl:`.
- The callback data of that button is ≤ 64 bytes.

**Run command**:
```bash
go test ./internal/bot/ -run TestDetailKeyboardFilesButton -short -v
```

**Result**: TODO
**Evidence**: —

---

## TEST-6 — Callback routing unit tests

**TASK**: TASK-7
**Type**: Unit (mock qbt.Client)
**Location**: `internal/bot/callback_test.go`
**Covers**: REQ-1, REQ-3, REQ-4, REQ-5 / AC-1.1, AC-1.2, AC-1.3, AC-3.2, AC-4.1, AC-4.2, AC-4.4, AC-5.2, AC-5.3

**What to verify**:
- `fl:` callback: `ListFiles` is called with the correct hash; bot edits the message with file list content; torrent name appears in the edited message.
- `fl:` callback when `ListFiles` returns an error: bot answers with a user-friendly error string; error is logged.
- `pg:fl:` callback: `ListFiles` is called; the page number in the rendered content matches the requested page.
- `fs:` callback: bot edits message to show a priority keyboard (does not call `SetFilePriority`).
- `fp:` callback: `SetFilePriority` is called with the correct hash, file index, and priority; `ListFiles` is subsequently called to refresh the file list.
- `fp:` callback when `SetFilePriority` returns an error: bot answers with a user-friendly error string; error is logged.
- `bk:fl:` callback: bot edits message to show torrent detail view (same hash).

**Run command**:
```bash
go test ./internal/bot/ -run TestCallbackFL -short -v
go test ./internal/bot/ -run TestCallbackPgFL -short -v
go test ./internal/bot/ -run TestCallbackFS -short -v
go test ./internal/bot/ -run TestCallbackFP -short -v
go test ./internal/bot/ -run TestCallbackBkFL -short -v
```

**Result**: TODO
**Evidence**: —

---

## TEST-7 — `ListFiles` integration test against real qBittorrent

**TASK**: TASK-8
**Type**: Integration
**Build tag**: `//go:build integration`
**Location**: `internal/qbt/http_integration_test.go`
**Covers**: REQ-1 / AC-1.1, AC-1.3

**What to verify**:
- `ListFiles` against a live qBittorrent instance (started by `make test-integration`) returns a non-error result for a known torrent hash.
- The returned slice contains at least one `TorrentFile` with a non-empty `Name` and a valid `Priority` value.
- The JSON contract matches the struct (no unmarshal panics or zero-value anomalies).

**Run command**:
```bash
make test-integration
# or directly:
go test ./internal/qbt/ -run TestIntegrationListFiles -v -count=1
```

**Result**: TODO
**Evidence**: —

---

## TEST-8 — `SetFilePriority` integration test with observable state change

**TASK**: TASK-8
**Type**: Integration
**Build tag**: `//go:build integration`
**Location**: `internal/qbt/http_integration_test.go`
**Covers**: REQ-4 / AC-4.2, AC-4.4

**What to verify**:
- `SetFilePriority` sets a file's priority to `FilePrioritySkip` (0) without error.
- A follow-up `ListFiles` call returns the same file with `Priority == FilePrioritySkip`.
- `SetFilePriority` restores the original priority (cleanup).
- Passing an invalid priority value results in either an API-level error or a graceful no-op (document observed behaviour).

**Run command**:
```bash
make test-integration
# or directly:
go test ./internal/qbt/ -run TestIntegrationSetFilePriority -v -count=1
```

**Result**: TODO
**Evidence**: —

---

## Gate 5 Checklist

Before marking this feature complete, confirm all items below:

- [ ] TEST-1: PASS
- [ ] CHECK-1: PASS
- [ ] TEST-2: PASS
- [ ] TEST-3: PASS
- [ ] TEST-4: PASS
- [ ] TEST-5: PASS
- [ ] TEST-6: PASS
- [ ] TEST-7: PASS
- [ ] TEST-8: PASS
- [ ] `make gate-all` passes (build + lint + unit tests)
- [ ] `make test-integration` passes
- [ ] All 22 AC-* have result PASS in traceability.md
- [ ] Backward traceability table in traceability.md is fully populated
- [ ] No TODO results remain in this document
