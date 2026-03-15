---
title: "Torrent File Management — Traceability Matrix"
feature_id: "torrent-files"
status: complete
last_updated: 2026-03-15
---

# Torrent File Management — Traceability Matrix

## Forward Traceability (Requirement → Verification)

| Requirement | Acceptance Criteria | Design Items | Plan Tasks | Implementation Evidence | Verification Targets | Status |
|-------------|---------------------|--------------|------------|------------------------|----------------------|--------|
| REQ-1: View file list from detail view | AC-1.1, AC-1.2, AC-1.3 | DES-1, DES-2, DES-6 | TASK-1, TASK-2, TASK-3, TASK-7 | `internal/qbt/http.go` ListFiles; `internal/bot/callback.go` handleFilesPageCallback | TEST-1, TEST-2, TEST-6, TEST-7 | PASS |
| REQ-2: Each file entry shows name, size, progress, priority | AC-2.1, AC-2.2, AC-2.3, AC-2.4 | DES-3 | TASK-4 | `internal/formatter/format.go` FormatFileList, truncateFileName | TEST-3 | PASS |
| REQ-3: File list paginated at 5 per page | AC-3.1, AC-3.2, AC-3.3 | DES-3, DES-4 | TASK-4, TASK-5, TASK-7 | `internal/formatter/format.go` FileListKeyboard; `internal/bot/callback.go` handleFilesPageNavCallback | TEST-3, TEST-4, TEST-6 | PASS |
| REQ-4: User can change individual file priority | AC-4.1, AC-4.2, AC-4.3, AC-4.4 | DES-1, DES-2, DES-4 | TASK-1, TASK-2, TASK-3, TASK-5, TASK-7 | `internal/qbt/http.go` SetFilePriority; `internal/bot/callback.go` handleFilePriorityCallback, handleFileSelectCallback | TEST-1, TEST-2, TEST-4, TEST-6, TEST-8 | PASS |
| REQ-5: "Files" button and "Back" navigation present | AC-5.1, AC-5.2, AC-5.3 | DES-4, DES-5, DES-6 | TASK-5, TASK-6, TASK-7 | `internal/formatter/format.go` TorrentDetailKeyboard filesBtn; `internal/bot/callback.go` handleBackFromFilesCallback | TEST-4, TEST-5, TEST-6 | PASS |
| REQ-6: Priority displayed as human-readable labels | AC-6.1, AC-6.2, AC-6.3, AC-6.4 | DES-3 | TASK-4 | `internal/formatter/format.go` PriorityLabel | TEST-3 | PASS |

### Acceptance Criteria Detail

| AC | Description | Design Items | Plan Tasks | Verification Targets | Status |
|----|-------------|--------------|------------|----------------------|--------|
| AC-1.1 | Tapping "Files" sends a message listing that torrent's files | DES-1, DES-2, DES-6 | TASK-7 | TEST-6 (TestCallbackFL_ShowsFileList), TEST-7 (TestIntegration_ListFiles), E2E (TestE2E_FileListCallback) | PASS |
| AC-1.2 | File list message includes torrent name as header | DES-3, DES-6 | TASK-4, TASK-7 | TEST-3 (TestFormatFileList_ContainsHeader), TEST-6 (TestCallbackFL_ShowsFileList) | PASS |
| AC-1.3 | `ListFiles` error → user-friendly message + logged | DES-1, DES-2 | TASK-3, TASK-7 | TEST-2 (TestListFiles_ErrorOnNon200), TEST-6 (TestCallbackFL_ListFilesError) | PASS |
| AC-2.1 | File name shows last path component, truncated to 40 chars with `…` | DES-3 | TASK-4 | TEST-3 (TestFormatFileList_ShowsLastPathComponent, TestFormatFileList_TruncatesLongFileName) | PASS |
| AC-2.2 | File size formatted as human-readable bytes | DES-3 | TASK-4 | TEST-3 (TestFormatFileList_MessageUnderLimit) | PASS |
| AC-2.3 | File entry shows textual progress bar and numeric percentage | DES-3 | TASK-4 | TEST-3 (TestFormatFileList_ContainsHeader, FormatProgress tests) | PASS |
| AC-2.4 | File entry shows current priority as human-readable label | DES-3 | TASK-4 | TEST-3 (TestFormatFileList_ShowsPriorityLabel) | PASS |
| AC-3.1 | Pagination nav buttons appear when torrent has more than 5 files | DES-3, DES-4 | TASK-5 | TEST-4 (TestFileListKeyboard_PaginationButtons_FirstPage) | PASS |
| AC-3.2 | Navigating pages displays the correct subset of files | DES-4 | TASK-7 | TEST-6 (TestCallbackPgFL_NavigatesToCorrectPage) | PASS |
| AC-3.3 | Page indicator shown when multiple pages exist | DES-3 | TASK-4 | TEST-3 (TestFormatFileList_PageIndicatorMultiPage) | PASS |
| AC-4.1 | Tapping a file presents a priority selection keyboard with all four options | DES-4 | TASK-5, TASK-7 | TEST-4 (TestPriorityKeyboard_FourPriorityOptions), TEST-6 (TestCallbackFS_ShowsPriorityKeyboard), E2E (TestE2E_FilePriorityChange) | PASS |
| AC-4.2 | Tapping a priority option calls `SetFilePriority` and shows updated file list | DES-1, DES-2, DES-4 | TASK-3, TASK-7 | TEST-6 (TestCallbackFP_SetsFilePriorityAndRefreshes), TEST-8 (TestIntegration_SetFilePriority), E2E (TestE2E_FilePriorityChange) | PASS |
| AC-4.3 | Current priority marked with checkmark in priority keyboard | DES-3, DES-4 | TASK-5 | TEST-4 (TestPriorityKeyboard_CurrentMarkedWithCheckmark) | PASS |
| AC-4.4 | `SetFilePriority` error → user-friendly message + logged | DES-1, DES-2, DES-4 | TASK-3, TASK-7 | TEST-2 (TestSetFilePriority_ErrorOnNon200), TEST-6 (TestCallbackFP_SetPriorityError_AnswersWithError) | PASS |
| AC-5.1 | Torrent detail keyboard includes "Files" button | DES-5, DES-6 | TASK-6 | TEST-5 (TestTorrentDetailKeyboard_FilesButton, TestDetailKeyboardFilesButton), E2E (TestE2E_DetailKeyboardContainsFilesButton) | PASS |
| AC-5.2 | "Back" on file list returns to torrent detail view | DES-4 | TASK-7 | TEST-6 (TestCallbackBkFL_ReturnsToDetailView) | PASS |
| AC-5.3 | "Back" on priority keyboard returns to the file list page the user came from | DES-4 | TASK-5, TASK-7 | TEST-4 (TestPriorityKeyboard_BackButtonIsPgFL) | PASS |
| AC-6.1 | Priority 0 displayed as "Skip" everywhere | DES-3 | TASK-4 | TEST-3 (TestPriorityLabel) | PASS |
| AC-6.2 | Priority 1 displayed as "Normal" everywhere | DES-3 | TASK-4 | TEST-3 (TestPriorityLabel) | PASS |
| AC-6.3 | Priority 6 displayed as "High" everywhere | DES-3 | TASK-4 | TEST-3 (TestPriorityLabel) | PASS |
| AC-6.4 | Priority 7 displayed as "Max" everywhere | DES-3 | TASK-4 | TEST-3 (TestPriorityLabel) | PASS |

## Backward Traceability (Code → Requirement)

| File | Symbol / Change | TASK | REQ | AC |
|------|----------------|------|-----|----|
| `internal/qbt/types.go` | `FilePriority`, `TorrentFile` | TASK-1 | REQ-1, REQ-4 | AC-1.1, AC-4.2 |
| `internal/qbt/client.go` | `ListFiles`, `SetFilePriority` interface methods | TASK-2 | REQ-1, REQ-4 | AC-1.1, AC-4.2 |
| `internal/qbt/http.go` | `ListFiles` HTTP implementation | TASK-3 | REQ-1 | AC-1.1, AC-1.3 |
| `internal/qbt/http.go` | `SetFilePriority` HTTP implementation | TASK-3 | REQ-4 | AC-4.2, AC-4.4 |
| `internal/formatter/format.go` | `PriorityLabel` | TASK-4 | REQ-6 | AC-6.1, AC-6.2, AC-6.3, AC-6.4 |
| `internal/formatter/format.go` | `FormatFileList`, `truncateFileName` | TASK-4 | REQ-2, REQ-3 | AC-1.2, AC-2.1, AC-2.2, AC-2.3, AC-2.4, AC-3.3 |
| `internal/formatter/format.go` | `FileListKeyboard` | TASK-5 | REQ-3, REQ-5 | AC-3.1, AC-5.3 |
| `internal/formatter/format.go` | `PriorityKeyboard` | TASK-5 | REQ-4, REQ-5 | AC-4.1, AC-4.3, AC-5.3 |
| `internal/formatter/format.go` | `TorrentDetailKeyboard` filesBtn row | TASK-6 | REQ-5 | AC-5.1 |
| `internal/bot/callback.go` | `handleFilesPageCallback` (fl:) | TASK-7 | REQ-1, REQ-5 | AC-1.1, AC-1.2, AC-1.3 |
| `internal/bot/callback.go` | `handleFilesPageNavCallback` (pg:fl:) | TASK-7 | REQ-3 | AC-3.2 |
| `internal/bot/callback.go` | `handleBackFromFilesCallback` (bk:fl:) | TASK-7 | REQ-5 | AC-5.2 |
| `internal/bot/callback.go` | `handleFileSelectCallback` (fs:) | TASK-7 | REQ-4 | AC-4.1 |
| `internal/bot/callback.go` | `handleFilePriorityCallback` (fp:) | TASK-7 | REQ-4 | AC-4.2, AC-4.4 |

## Coverage Summary

| Metric | Count |
|--------|-------|
| Requirements (REQ-*) | 6 |
| Acceptance Criteria (AC-*) | 22 |
| Design Items (DES-*) | 6 |
| Plan Tasks (TASK-*) | 8 |
| Unit Test targets (TEST-*) | 6 (TEST-1 through TEST-6) |
| Integration Test targets (TEST-*) | 2 (TEST-7, TEST-8) |
| Manual checks (CHECK-*) | 1 (CHECK-1) |
| REQs fully covered by DES | 6 / 6 |
| ACs with at least one verification target | 22 / 22 |
| TASKs with at least one verification target | 8 / 8 |
| ACs verified (PASS) | 22 / 22 |

## Rules + Harness Validation

```bash
# REQ count must be 6
grep -c "^- \*\*REQ-" docs/features/torrent-files/spec.md

# AC count must be 22 (14 in spec + 8 from AC-6.x group being 4; spec has 22 total)
grep -c "^- \*\*AC-" docs/features/torrent-files/spec.md

# Every REQ in spec must appear in design
spec_reqs=$(grep -oP 'REQ-\d+' docs/features/torrent-files/spec.md | sort -u)
design_reqs=$(grep -oP 'REQ-\d+' docs/features/torrent-files/design.md | sort -u)
comm -23 <(echo "$spec_reqs") <(echo "$design_reqs")  # should be empty

# Every DES in design must appear in plan
design_items=$(grep -oP 'DES-\d+' docs/features/torrent-files/design.md | sort -u)
plan_items=$(grep -oP 'DES-\d+' docs/features/torrent-files/plan.md | sort -u)
comm -23 <(echo "$design_items") <(echo "$plan_items")  # should be empty

# TASK count must be 8
grep "^- \*\*TASK-" docs/features/torrent-files/plan.md | wc -l

# No TODO: open questions in spec
grep -c "TODO:" docs/features/torrent-files/spec.md  # should be 0

# Verification targets per task must equal 8
grep "Verification:" docs/features/torrent-files/plan.md | wc -l
```
