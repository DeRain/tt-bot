---
title: "Search Torrents — Traceability Matrix"
feature_id: "search-torrents"
status: complete
last_updated: 2026-05-24
---

# Search Torrents — Traceability Matrix

## Forward Traceability (Requirement → Verification)

| Requirement | Acceptance Criteria | Design | Plan Tasks | Implementation Evidence | Verification | Status |
|-------------|---------------------|--------|------------|-------------------------|--------------|--------|
| REQ-1 | AC-1.1, AC-1.2, AC-1.3, AC-1.4 | DES-4 | TASK-4 | `internal/bot/handler.go` (`handleSearchCommand`, `handleSearchPromptReply`) | TEST-1, TEST-2 | Complete |
| REQ-2 | AC-2.1, AC-2.2, AC-2.3 | DES-2 | TASK-3, TASK-4 | `internal/formatter/format.go` (`FormatSearchResults`, `SearchResultKeyboard`, `SearchPaginationKeyboard`) | TEST-3, TEST-4 | Complete |
| REQ-3 | AC-3.1, AC-3.2 | DES-5 | TASK-4 | `internal/bot/handler.go` (`sortSearchResults`, `handleSearchSortCallback`) | TEST-5, TEST-6 | Complete |
| REQ-4 | AC-4.1, AC-4.2 | DES-6 | TASK-2 | `internal/qbt/http.go` (`StartSearch` with plugins param) | TEST-7 | Complete |
| REQ-5 | AC-5.1, AC-5.2, AC-5.3 | DES-3 | TASK-4 | `internal/bot/callback.go` (`handleSearchSelectCallback`, `handleSearchConfirmCallback`, `handleSearchBackCallback`) | TEST-8, TEST-9, TEST-10 | Complete |
| REQ-6 | AC-6.1 | DES-4 | TASK-4 | `internal/bot/handler.go` (`HandleUpdate` prompt abandonment logic) | TEST-11 | Complete |
| REQ-7 | AC-7.1, AC-7.2 | DES-1 | TASK-4 | `internal/bot/handler.go` (`pollSearchResults`, `evictExpired`) | TEST-12, TEST-13 | Complete |
| REQ-8 | AC-8.1, AC-8.2, AC-8.3, AC-8.4 | DES-1 | TASK-4 | `internal/bot/handler.go` (`handleSearchCommand`, `pollSearchResults`) | TEST-14, TEST-15 | Complete |

## Backward Traceability (Code → Requirement)

| Source File | Functions/Types | Traces To | Via |
|-------------|----------------|-----------|-----|
| `internal/qbt/types.go` | `SearchResult` struct | REQ-2 | TASK-1, DES-2 |
| `internal/qbt/client.go` | `Client` interface (search methods) | REQ-1, REQ-4, REQ-7 | TASK-1, DES-1 |
| `internal/qbt/http.go` | `StartSearch`, `SearchStatus`, `SearchResults`, `StopSearch`, `DeleteSearch` | REQ-1, REQ-4, REQ-7 | TASK-2, DES-1 |
| `internal/formatter/format.go` | `FormatSearchResults`, `SearchResultKeyboard`, `SearchPaginationKeyboard`, `FormatSearchConfirm`, `SearchConfirmKeyboard`, `SearchCancelKeyboard` | REQ-2, REQ-5 | TASK-3, DES-2, DES-3 |
| `internal/bot/handler.go` | `SearchState`, `SearchPrompt`, `handleSearchCommand`, `handleSearchPromptReply`, `pollSearchResults`, `sendSearchResultsPage`, `sortSearchResults`, `storeSearch`, `takeSearch`, `getSearch`, `storeSearchPrompt`, `takeSearchPrompt` | REQ-1, REQ-2, REQ-3, REQ-6, REQ-7, REQ-8 | TASK-4, DES-1, DES-4, DES-5 |
| `internal/bot/callback.go` | `handleSearchCallback`, `handleSearchSelectCallback`, `handleSearchPageCallback`, `handleSearchCancelCallback`, `handleSearchSortCallback`, `handleSearchConfirmCallback`, `handleSearchBackCallback` | REQ-2, REQ-3, REQ-5, REQ-7, REQ-8 | TASK-4, DES-3, DES-5 |
| `internal/bot/commands.go` | `/search` command registration | REQ-1 | TASK-4 |

## Coverage Summary

| Metric | Count | Covered | Gaps |
|--------|-------|---------|------|
| Requirements | 8 | 8 | 0 |
| Acceptance Criteria | 18 | 18 | 0 |
| Design Items | 6 | 6 | 0 |
| Plan Tasks | 7 | 7 | 0 |
| Verification Items | 15 | 15 | 0 |

## Rules

- No REQ-* may exist without at least one linked DES-*.
- No DES-* may exist without at least one linked TASK-*.
- No TASK-* may exist without at least one linked verification item.
- No AC-* may remain unverified.
- Status values: Complete | Partial | Blocked | Missing | N/A

## Harness Validation

```bash
# Count untraced requirements (should be 0)
grep "| TODO |" docs/features/search-torrents/traceability.md | wc -l

# Count missing verification (should be 0)
grep "| Missing |" docs/features/search-torrents/traceability.md | wc -l
```
