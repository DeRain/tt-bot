---
title: "Extended Torrent Detail Info — Verification"
feature_id: "torrent-detail-extra"
status: draft
last_updated: 2026-03-15
---

# Extended Torrent Detail Info — Verification

## Validation Strategy

All acceptance criteria are validated through automated tests. REQ-1 and REQ-2 (formatter output) are covered by unit tests. REQ-3 (API field deserialization) is covered by a build check and an integration test against a real qBittorrent instance. No manual checks are required — the feature is fully exercisable via the test suite.

## Automated Tests

- **TEST-1**: Build check confirming `Torrent.Uploaded` and `Torrent.Ratio` fields compile with correct types and JSON tags.
  - Validates: AC-3.1, AC-3.2
  - Covers: REQ-3
  - Evidence: `internal/qbt/client.go` (`Torrent` struct); command: `make build`
  - Pass criteria: `make build` exits 0 with no new warnings or type errors.

- **TEST-2**: Unit test — torrent with `Uploaded=3_435_973_837` (~3.2 GB) and `Ratio=2.13` produces correct `FormatTorrentDetail` output.
  - Validates: AC-1.1, AC-1.3, AC-2.1, AC-2.3
  - Covers: REQ-1, REQ-2
  - Evidence: `internal/formatter/formatter_test.go` (`TestFormatTorrentDetail`); command: `make test`
  - Pass criteria: output contains `Uploaded: 3.2 GB`, contains `Ratio: 2.13`, `Uploaded` line appears between the `Upload` speed line and `State` line, `Ratio` line appears immediately after `Uploaded` line.

- **TEST-3**: Unit test — torrent with `Uploaded=0` and `Ratio=0.0` still renders both lines.
  - Validates: AC-1.2, AC-2.2
  - Covers: REQ-1, REQ-2
  - Evidence: `internal/formatter/formatter_test.go` (`TestFormatTorrentDetail`); command: `make test`
  - Pass criteria: output contains `Uploaded: 0 B` and `Ratio: 0.00`.

- **TEST-4**: Integration test — fetch torrent list from a real qBittorrent instance and assert deserialized fields.
  - Validates: AC-3.3
  - Covers: REQ-3
  - Evidence: `internal/qbt/http_integration_test.go`; command: `make test-integration`
  - Pass criteria: at least one returned `Torrent` has `Uploaded >= 0` and `Ratio >= 0.0`; for a seeding torrent, `Uploaded > 0`.

## Manual Checks

None required. All acceptance criteria are covered by automated tests.

## Acceptance Criteria Results

| AC | Validation | Result | Evidence |
|----|-----------|--------|----------|
| AC-1.1 | TEST-2 | TODO | — |
| AC-1.2 | TEST-3 | TODO | — |
| AC-1.3 | TEST-2 | TODO | — |
| AC-2.1 | TEST-2 | TODO | — |
| AC-2.2 | TEST-3 | TODO | — |
| AC-2.3 | TEST-2 | TODO | — |
| AC-3.1 | TEST-1 | TODO | — |
| AC-3.2 | TEST-1 | TODO | — |
| AC-3.3 | TEST-4 | TODO | — |

## Quality Gates

### Gate 5: Verification Gate

This verification passes when:
- [ ] Every AC-* has at least one TEST-* or CHECK-*
- [ ] All automated tests pass (`make test`)
- [ ] All manual checks are recorded with evidence
- [ ] No AC-* has Result = TODO or FAIL
- [ ] Gaps are explicitly documented (not silently omitted)

**Harness check commands:**
```bash
# Run unit tests for formatter and qbt packages
go test ./internal/formatter/... ./internal/qbt/... -short -v -cover

# Count unverified ACs (should be 0 at Gate 5)
grep "| TODO |" docs/features/torrent-detail-extra/verification.md | wc -l

# Integration tests (mandatory)
make test-integration
```

## Traceability Coverage

0 of 3 requirements verified, 0 of 9 acceptance criteria validated. All items pending implementation.

## Exceptions / Unresolved Gaps

None at this stage. All AC-* have an assigned TEST-*; no coverage gaps exist in the plan.
