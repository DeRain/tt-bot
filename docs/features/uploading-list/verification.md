---
title: "Uploading Torrents List — Verification"
feature_id: "uploading-list"
status: draft
last_updated: 2026-03-15
---

# Uploading Torrents List — Verification

## Validation Strategy

All acceptance criteria are validated by both automated unit tests (run via `make test`) and integration tests against a real qBittorrent instance (run via `make test-integration`). Unit tests use mock interfaces and `httptest`; integration tests exercise the live API. No AC is marked PASS until both layers confirm it.

## Automated Tests

### TEST-1: FilterUploading constant is distinct

- **Validates**: REQ-1, REQ-2
- **Covers**: AC-1.1, AC-2.1, AC-2.2
- **Location**: `internal/qbt/types_test.go` (to be created in TASK-5)
- **Pass criteria**: `FilterUploading` value does not collide with any existing `TorrentFilter` constant (`FilterAll`, `FilterDownloading`, `FilterActive`)
- **Evidence**: TBD

### TEST-2: Filter char / prefix mappings round-trip

- **Validates**: REQ-3, REQ-4
- **Covers**: AC-3.1, AC-3.2, AC-4.1, AC-4.2
- **Location**: `internal/bot/callback_test.go` (to be created in TASK-5)
- **Pass criteria**:
  - `filterCharToFilter("u")` returns `FilterUploading`
  - `filterToChar(FilterUploading)` returns `"u"`
  - `filterCharToPrefix("u")` returns `"up"`
- **Evidence**: TBD

### TEST-3: pg:up: callback routing

- **Validates**: REQ-3
- **Covers**: AC-3.1, AC-3.2
- **Location**: `internal/bot/callback_test.go` (to be created in TASK-5)
- **Pass criteria**: `handleCallback()` routes a `pg:up:<page>` callback to `sendTorrentPage` with `FilterUploading` and the correct page number
- **Evidence**: TBD

### TEST-4: renderTorrentListPage client-side filtering

- **Validates**: REQ-1, REQ-2
- **Covers**: AC-1.1, AC-1.2, AC-2.1, AC-2.2
- **Location**: `internal/bot/handler_test.go` (to be created in TASK-5)
- **Pass criteria**:
  - Given a mixed list of torrents, only those with `Progress == 1.0` are returned
  - A torrent with state `pausedUP` and `Progress == 1.0` is included
  - A torrent with state `uploading` and `Progress == 1.0` is included
  - When no torrents have `Progress == 1.0`, the "No torrents found." message is sent
- **Evidence**: TBD

### TEST-5: /uploading command dispatch and registration

- **Validates**: REQ-5
- **Covers**: AC-5.1
- **Location**: `internal/bot/handler_test.go` (to be created in TASK-5)
- **Pass criteria**:
  - `/uploading` command triggers `sendTorrentPage` with `FilterUploading` and page 1
  - `BotCommands` slice includes an entry with command `"uploading"`
- **Evidence**: TBD

## Manual Checks

### CHECK-1: Integration tests pass against real qBittorrent

- **Validates**: REQ-1 through REQ-5
- **Covers**: All ACs
- **Command**: `make test-integration`
- **Pass criteria**: All `Integration` and `E2E` tagged tests pass with a live qBittorrent container; uploading filter correctly separates completed seeds from in-progress downloads
- **Evidence**: TBD

## Acceptance Criteria Results

| AC | Description | Validation | Result | Evidence |
|----|-------------|------------|--------|----------|
| AC-1.1 | `/uploading` returns only torrents with Progress == 1.0 | TEST-4 | TODO | TBD |
| AC-1.2 | `/uploading` returns "No torrents found." when no torrents are completed | TEST-4 | TODO | TBD |
| AC-2.1 | A paused completed torrent (`pausedUP`) appears in the list | TEST-4 | TODO | TBD |
| AC-2.2 | An actively seeding torrent (`uploading` or `stalledUP`) appears in the list | TEST-4 | TODO | TBD |
| AC-3.1 | Pagination keyboard appears when more than 5 completed torrents exist | TEST-3, TEST-4 | TODO | TBD |
| AC-3.2 | Navigating pages via inline keyboard shows correct page of completed torrents | TEST-3 | TODO | TBD |
| AC-4.1 | Selecting a torrent from the uploading list shows its detail view | TEST-2, TEST-3 | TODO | TBD |
| AC-4.2 | Pause/resume actions from the detail view work and return to the uploading list | TEST-2 | TODO | TBD |
| AC-5.1 | `/uploading` appears in Telegram command menu and `/help` output | TEST-5 | TODO | TBD |

## Quality Gates

### Gate 5: Verification Gate

- [ ] All TEST-* items have passing evidence (test output or CI run link)
- [ ] CHECK-1 has passing evidence (`make test-integration` output)
- [ ] Every AC-* result is PASS (no TODO or FAIL entries remain)
- [ ] Test coverage for affected packages meets 80% threshold (`make test` coverage output)
- [ ] No regressions in existing `/list`, `/active`, `/downloading` tests

## Traceability Coverage

| Metric | Count |
|--------|-------|
| ACs requiring validation | 9 |
| ACs with automated TEST | 9 |
| ACs with integration CHECK | 9 (all covered by CHECK-1) |
| ACs verified (PASS) | 0 / 9 — not yet implemented |
| TEST-* items defined | 5 |
| CHECK-* items defined | 1 |
