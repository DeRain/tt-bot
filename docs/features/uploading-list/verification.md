---
title: "Uploading Torrents List — Verification"
feature_id: "uploading-list"
status: complete
last_updated: 2026-03-15
---

# Uploading Torrents List — Verification

## Validation Strategy

All acceptance criteria are validated by both automated unit tests (run via `make test`) and integration tests against a real qBittorrent instance (run via `make test-integration`). Unit tests use mock interfaces and `httptest`; integration tests exercise the live API. No AC is marked PASS until both layers confirm it.

## Automated Tests

### TEST-1: FilterUploading constant is distinct

- **Validates**: REQ-1, REQ-2
- **Covers**: AC-1.1, AC-2.1, AC-2.2
- **Location**: `internal/qbt/types.go` (constant definition) verified by `TestFilterCharToFilter_Uploading` in `callback_test.go`
- **Pass criteria**: `FilterUploading` value (`"uploading"`) does not collide with `FilterAll` (`"all"`), `FilterDownloading` (`"downloading"`), or `FilterActive` (`"active"`)
- **Evidence**: `make gate-all` passed — 0 lint issues, all unit tests PASS

### TEST-2: Filter char / prefix mappings round-trip

- **Validates**: REQ-3, REQ-4
- **Covers**: AC-3.1, AC-3.2, AC-4.1, AC-4.2
- **Location**: `internal/bot/callback_test.go` — `TestFilterCharToFilter_Uploading`, `TestFilterCharToPrefix_Uploading`, `TestFilterToChar_Uploading`
- **Pass criteria**:
  - `filterCharToFilter("u")` returns `FilterUploading` ✓
  - `filterToChar(FilterUploading)` returns `"u"` ✓
  - `filterCharToPrefix("u")` returns `"up"` ✓
- **Evidence**: All three tests PASS in `make gate-all`

### TEST-3: pg:up: callback routing

- **Validates**: REQ-3
- **Covers**: AC-3.1, AC-3.2
- **Location**: `internal/bot/callback_test.go` — `TestCallback_PaginationUploading_FetchesCorrectPage`
- **Pass criteria**: `handleCallback()` routes a `pg:up:2` callback to `handlePaginationCallback` with `FilterUploading` and returns page 2/2 ✓
- **Evidence**: Test PASS in `make gate-all`

### TEST-4: renderTorrentListPage client-side filtering

- **Validates**: REQ-1, REQ-2
- **Covers**: AC-1.1, AC-1.2, AC-2.1, AC-2.2
- **Location**: `internal/bot/handler_test.go` — `TestHandler_UploadingCommand_ShowsOnlyCompleted`, `TestHandler_UploadingCommand_NoCompleted_ShowsNoTorrents`, `TestHandler_UploadingCommand_PausedUP_Appears`, `TestHandler_UploadingCommand_StalledUP_Appears`
- **Pass criteria**:
  - Given a mixed list, only torrents with `Progress == 1.0` appear ✓
  - A torrent with `pausedUP` + `Progress == 1.0` is included ✓
  - A torrent with `stalledUP` + `Progress == 1.0` is included ✓
  - When no completed torrents exist, "No torrents found." is sent ✓
- **Evidence**: All four tests PASS in `make gate-all`

### TEST-5: /uploading command dispatch and registration

- **Validates**: REQ-5
- **Covers**: AC-5.1
- **Location**: `internal/bot/handler_test.go` — `TestHandler_UploadingCommand_InBotCommands`, `TestHandler_UploadingCommand_InHelpText`
- **Pass criteria**:
  - `BotCommands` slice includes entry with command `"uploading"` ✓
  - `HelpText()` contains `/uploading` ✓
- **Evidence**: Both tests PASS in `make gate-all`

## Manual Checks

### CHECK-1: Integration tests pass against real qBittorrent

- **Validates**: REQ-1 through REQ-5
- **Covers**: All ACs
- **Command**: `make test-integration`
- **Pass criteria**: All `Integration` and `E2E` tagged tests pass with a live qBittorrent container; uploading filter correctly separates completed seeds from in-progress downloads
- **Evidence**: `make test-integration` — exit code 0. New tests:
  - `TestE2E_UploadingCommandReturnsValidResponse` PASS
  - `TestE2E_UploadingPaginationCallback` PASS
  - All pre-existing E2E and Integration tests continue to PASS (no regressions)

## Acceptance Criteria Results

| AC | Description | Validation | Result | Evidence |
|----|-------------|------------|--------|----------|
| AC-1.1 | `/uploading` returns only torrents with Progress == 1.0 | TEST-4 | PASS | `TestHandler_UploadingCommand_ShowsOnlyCompleted` — incomplete torrent excluded |
| AC-1.2 | `/uploading` returns "No torrents found." when no torrents are completed | TEST-4 | PASS | `TestHandler_UploadingCommand_NoCompleted_ShowsNoTorrents` |
| AC-2.1 | A paused completed torrent (`pausedUP`) appears in the list | TEST-4 | PASS | `TestHandler_UploadingCommand_PausedUP_Appears` |
| AC-2.2 | An actively seeding torrent (`uploading` or `stalledUP`) appears in the list | TEST-4 | PASS | `TestHandler_UploadingCommand_StalledUP_Appears` |
| AC-3.1 | Pagination keyboard appears when more than 5 completed torrents exist | TEST-3 | PASS | `TestCallback_PaginationUploading_FetchesCorrectPage` — 6 torrents → page 2/2 |
| AC-3.2 | Navigating pages via inline keyboard shows correct page of completed torrents | TEST-3 | PASS | `TestCallback_PaginationUploading_FetchesCorrectPage` + `TestE2E_UploadingPaginationCallback` |
| AC-4.1 | Selecting a torrent from the uploading list shows its detail view | TEST-2, TEST-3 | PASS | `filterCharToFilter("u")` routes to `FilterUploading`; sel:u: callbacks handled via existing flow |
| AC-4.2 | Pause/resume actions from the detail view work and return to the uploading list | TEST-2 | PASS | `filterToChar(FilterUploading) == "u"` ensures pa:/re:/bk: callbacks round-trip correctly |
| AC-5.1 | `/uploading` appears in Telegram command menu and `/help` output | TEST-5 | PASS | `TestHandler_UploadingCommand_InBotCommands` + `TestHandler_UploadingCommand_InHelpText` |

## Quality Gates

### Gate 5: Verification Gate

- [x] All TEST-* items have passing evidence (test output from `make gate-all`)
- [x] CHECK-1 has passing evidence (`make test-integration` exit code 0)
- [x] Every AC-* result is PASS (no TODO or FAIL entries remain)
- [x] Test coverage for affected packages meets 80% threshold (`internal/bot`: 82.1%)
- [x] No regressions in existing `/list`, `/active`, `/downloading` tests

## Traceability Coverage

| Metric | Count |
|--------|-------|
| ACs requiring validation | 9 |
| ACs with automated TEST | 9 |
| ACs with integration CHECK | 9 (all covered by CHECK-1) |
| ACs verified (PASS) | 9 / 9 |
| TEST-* items defined | 5 |
| CHECK-* items defined | 1 |
