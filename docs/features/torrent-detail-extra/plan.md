---
title: "Extended Torrent Detail Info — Plan"
feature_id: torrent-detail-extra
status: draft
last_updated: 2026-03-15
---

# Plan: Extended Torrent Detail Info

## Task Summary

| TASK | Description | DES | REQ | Verification |
|------|-------------|-----|-----|--------------|
| TASK-1 | Add `Uploaded` and `Ratio` fields to `Torrent` struct | DES-1, DES-3 | REQ-3 | TEST-1 |
| TASK-2 | Update `FormatTorrentDetail` to render new fields | DES-2 | REQ-1, REQ-2 | TEST-2, TEST-3 |
| TASK-3 | Add/update unit tests for formatter | DES-2 | REQ-1, REQ-2 | TEST-2, TEST-3 |
| TASK-4 | Add integration test assertions for new fields | DES-3 | REQ-3 | TEST-4 |

## Tasks

### TASK-1: Add struct fields

**File:** `internal/qbt/client.go`
**Depends on:** none
**Verification:** TEST-1

Add to `Torrent` struct (after existing fields, before closing brace):

```go
Uploaded int64   `json:"uploaded"`
Ratio    float64 `json:"ratio"`
```

**Done when:** `make build` passes with no new warnings.

---

### TASK-2: Update `FormatTorrentDetail`

**File:** `internal/formatter/formatter.go`
**Depends on:** TASK-1
**Verification:** TEST-2, TEST-3

After the `Upload` speed line, add:

```
Uploaded: <formatBytes(t.Uploaded)>
Ratio:    <fmt.Sprintf("%.2f", t.Ratio)>
```

**Done when:** `make build` passes and manual output matches the target format in `spec.md`.

---

### TASK-3: Unit tests for formatter

**File:** `internal/formatter/formatter_test.go`
**Depends on:** TASK-1 (struct fields must exist to compile)
**Write tests BEFORE TASK-2 (TDD — RED first)**
**Verification:** TEST-2, TEST-3

Add table-driven test cases to the existing `TestFormatTorrentDetail` test (or create it if absent):

- **TEST-2:** Torrent with `Uploaded=3_435_973_837` (≈3.2 GB) and `Ratio=2.13` → output contains `Uploaded: 3.2 GB` and `Ratio: 2.13`.
- **TEST-3:** Torrent with `Uploaded=0` and `Ratio=0.0` → output contains `Uploaded: 0 B` and `Ratio: 0.00`.

**Done when:** `make test` passes (all unit tests green).

---

### TASK-4: Integration test assertions

**File:** `internal/qbt/http_integration_test.go` (or nearest integration test file)
**Depends on:** TASK-1
**Verification:** TEST-4

- **TEST-4:** In the integration test that fetches torrent list, assert that at least one returned `Torrent` has `Uploaded >= 0` and `Ratio >= 0.0` (fields deserialise without error; for a seeding torrent assert `Uploaded > 0`).

**Done when:** `make test-integration` passes.

---

## Verification Targets

| TEST | Type | AC Coverage | Command |
|------|------|-------------|---------|
| TEST-1 | Build check | AC-3.1, AC-3.2 | `make build` |
| TEST-2 | Unit | AC-1.1, AC-1.3, AC-2.1, AC-2.3 | `make test` |
| TEST-3 | Unit | AC-1.2, AC-2.2 | `make test` |
| TEST-4 | Integration | AC-3.3 | `make test-integration` |

## Execution Order

```
TASK-3 (write tests, RED) → TASK-1 → TASK-2 (GREEN) → TASK-4 → make gate-all → make test-integration
```

## Gate 3 Check

- [x] Every TASK maps to at least one DES-* and REQ-*.
- [x] Every TASK has a verification target (TEST-*).
- [x] All AC-* from spec.md are covered by at least one TEST-*.
- [x] TDD order is explicit (tests written before implementation).
- [x] Integration test is included (mandatory per project rules).
- [x] No TASK is orphaned.
