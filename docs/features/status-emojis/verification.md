---
title: "Human-Readable Statuses with Emojis — Verification"
feature_id: "status-emojis"
status: draft
last_updated: 2026-03-15
---

# Human-Readable Statuses with Emojis — Verification

## Test Inventory

| ID      | Type        | Location                                          | Mapped AC                             | Status |
|---------|-------------|---------------------------------------------------|---------------------------------------|--------|
| TEST-1  | Unit        | `internal/formatter/formatter_test.go` — `TestFormatState` | AC-1.1, AC-1.2, AC-1.3, AC-2.1, AC-2.2, AC-4.1, AC-4.2 | TODO |
| TEST-2  | Unit        | `internal/formatter/formatter_test.go` — `TestFormatTorrentList`, `TestFormatTorrentDetail` | AC-3.1, AC-3.2 | TODO |
| CHECK-1 | Manual/CI   | `make gate-all`                                   | Overall build health                  | TODO |

## TEST-1: FormatState Unit Tests

**Description**: Table-driven test covering all 19 documented states, empty string input, and an unrecognised state input.

**Assertions**:
- Each of the 19 documented states returns a non-empty string not equal to the raw input.
- Each mapped label begins with the designated emoji character.
- `stalledUP` returns a string containing `Seeding (stalled)` and not containing `stalledUP`.
- `pausedDL` returns a string containing `Paused (Downloading)` and not containing `pausedDL`.
- `""` returns a non-empty string beginning with `❓` and does not panic.
- `"newState"` returns `❓ newState` and does not panic.

**Run command**:
```bash
go test ./internal/formatter/... -short -v -run TestFormatState
```

**Result**: TODO

**Evidence**: TODO

---

## TEST-2: FormatTorrentList and FormatTorrentDetail Integration Tests

**Description**: Extended unit tests asserting that the mapped label appears in the formatted output and the raw state string does not.

**Assertions**:
- `FormatTorrentList` output contains the mapped label for a torrent's state and does not contain the raw state string.
- `FormatTorrentDetail` output contains the mapped label on the `State:` line and does not contain the raw state string.

**Run command**:
```bash
go test ./internal/formatter/... -short -v -run "TestFormatTorrentList|TestFormatTorrentDetail"
```

**Result**: TODO

**Evidence**: TODO

---

## CHECK-1: make gate-all

**Description**: Full quality gate — build, lint, and all unit tests pass with no warnings.

**Run command**:
```bash
make gate-all
```

**Expected**: Exit code 0, no lint warnings, all tests pass.

**Result**: TODO

**Evidence**: TODO

---

## Acceptance Criteria Results

| AC      | Description                                                                                  | Verified by | Result |
|---------|----------------------------------------------------------------------------------------------|-------------|--------|
| AC-1.1  | `stalledUP` → `Seeding (stalled)`, raw string absent in output                              | TEST-1      | TODO   |
| AC-1.2  | `pausedDL` → `Paused (Downloading)`, raw string absent in output                            | TEST-1      | TODO   |
| AC-1.3  | All 19 documented states produce a non-empty label distinct from the raw state string        | TEST-1      | TODO   |
| AC-2.1  | Each mapped state's label begins with its designated emoji character                          | TEST-1      | TODO   |
| AC-2.2  | Fallback output for an unmapped state begins with ❓                                         | TEST-1      | TODO   |
| AC-3.1  | `FormatTorrentList` displays the mapped label, not the raw state                              | TEST-2      | TODO   |
| AC-3.2  | `FormatTorrentDetail` displays the mapped label on the `State:` line, not the raw state       | TEST-2      | TODO   |
| AC-4.1  | Empty string input returns a non-empty fallback prefixed with ❓ and does not panic           | TEST-1      | TODO   |
| AC-4.2  | `"newState"` input returns `❓ newState` and does not panic                                  | TEST-1      | TODO   |

## Gate 5 Checklist

- [ ] Every AC-* has a corresponding TEST-* or CHECK-* entry above.
- [ ] Every TEST-* result is PASS (not TODO or FAIL).
- [ ] Every CHECK-* result is PASS (not TODO or FAIL).
- [ ] No acceptance criterion has a TODO result.
- [ ] `make gate-all` exits 0 with no lint warnings.
- [ ] `make test-integration` has been run and passes (if integration tests exist for this feature).
- [ ] `traceability.md` is updated to reflect final implementation.
- [ ] All implementation evidence entries in `traceability.md` are marked complete (not TODO).
