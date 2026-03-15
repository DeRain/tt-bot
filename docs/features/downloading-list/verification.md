---
title: "Downloading Torrents List — Verification"
feature_id: "downloading-list"
status: complete
last_updated: 2026-03-15
---

# Downloading Torrents List — Verification

## Validation Strategy

Unit tests validate filter mappings, client-side filtering logic, and command dispatch. Integration tests validate against a real qBittorrent instance in Docker.

## Automated Tests

- **TEST-1**: Unit test verifying `FilterDownloading` constant exists and is distinct
  - Validates: AC-1.1
  - Pass criteria: Constant compiles and is not equal to `FilterAll` or `FilterActive`

- **TEST-2**: Unit test verifying filter char `d` ↔ `FilterDownloading` roundtrip
  - Validates: AC-3.1, AC-4.1
  - Pass criteria: `filterCharToFilter("d")` returns `FilterDownloading`; `filterToChar(FilterDownloading)` returns `"d"`; `filterCharToPrefix("d")` returns `"dw"`

- **TEST-3**: Unit test verifying `pg:dw:` callback routing dispatches pagination
  - Validates: AC-3.2
  - Pass criteria: Callback with `pg:dw:2` triggers pagination handler with `FilterDownloading`

- **TEST-4**: Unit test verifying client-side filtering keeps only Progress < 1.0
  - Validates: AC-1.1, AC-1.2, AC-2.1, AC-2.2
  - Pass criteria: Given mixed torrents (complete + incomplete + paused incomplete), only incomplete ones appear

- **TEST-5**: Unit test verifying `/downloading` command registered and dispatches correctly
  - Validates: AC-5.1
  - Pass criteria: `BotCommands` includes `downloading`; `handleCommand` dispatches to `sendTorrentPage` with `FilterDownloading`

## Manual Checks

- **CHECK-1**: Visual verification that `/downloading` appears in Telegram command menu after bot restart

## Acceptance Criteria Results

| AC | Validation | Result | Evidence |
|----|-----------|--------|----------|
| AC-1.1 | TEST-1, TEST-4 | PASS | Unit tests pass via make gate-all |
| AC-1.2 | TEST-4 | PASS | Unit tests pass via make gate-all |
| AC-2.1 | TEST-4 | PASS | Unit tests pass via make gate-all |
| AC-2.2 | TEST-4 | PASS | Unit tests pass via make gate-all |
| AC-3.1 | TEST-2 | PASS | Unit tests pass via make gate-all |
| AC-3.2 | TEST-3 | PASS | Unit tests pass via make gate-all |
| AC-4.1 | TEST-2 | PASS | Unit tests pass via make gate-all |
| AC-4.2 | TEST-2 | PASS | Unit tests pass via make gate-all |
| AC-5.1 | TEST-5 | PASS | Unit tests pass via make gate-all |

## Quality Gates

### Gate 5: Verification Gate

- [x] All ACs have TEST-* or CHECK-* assigned
- [x] All automated tests pass
- [x] No TODO results remain
- [x] `make gate-all` passes
- [x] `make test-integration` passes
