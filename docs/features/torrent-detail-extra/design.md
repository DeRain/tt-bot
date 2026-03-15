---
title: "Extended Torrent Detail Info — Design"
feature_id: torrent-detail-extra
status: draft
last_updated: 2026-03-15
---

# Design: Extended Torrent Detail Info

## Requirement Coverage

| REQ | Design Item(s) |
|-----|---------------|
| REQ-1 | DES-1, DES-2 |
| REQ-2 | DES-1, DES-2 |
| REQ-3 | DES-1, DES-3 |

## Design Items

### DES-1: Extend `Torrent` struct

**File:** `internal/qbt/client.go`

Add two fields to the existing `Torrent` struct:

```go
Uploaded int64   `json:"uploaded"`
Ratio    float64 `json:"ratio"`
```

No constructor changes are needed; the struct is populated via `json.Unmarshal` from the API response. Existing fields are unaffected (immutability at the data-layer is maintained — no mutation of existing fields).

**Rationale:** Keeping both fields on `Torrent` avoids a separate DTO and matches the existing pattern used by all other qBittorrent fields.

### DES-2: Update `FormatTorrentDetail`

**File:** `internal/formatter/formatter.go`

Insert two lines after the `Upload` speed line and before `State`:

```
Uploaded: <formatBytes(t.Uploaded)>
Ratio:    <fmt.Sprintf("%.2f", t.Ratio)>
```

`formatBytes` is the existing internal helper already used for `Size`, `DLSpeed`, and `UPSpeed`. No new formatting utility is required.

The function signature and return type remain unchanged (`func FormatTorrentDetail(t qbt.Torrent) string`). The change is purely additive.

### DES-3: JSON field mapping

**File:** `internal/qbt/client.go` (struct tags on DES-1 fields)

The qBittorrent v2 API `/api/v2/torrents/info` response object uses snake_case keys:

| JSON key   | Go field        | Type    |
|------------|-----------------|---------|
| `uploaded` | `Torrent.Uploaded` | int64   |
| `ratio`    | `Torrent.Ratio`    | float64 |

These tags are applied directly in DES-1. No separate mapping layer is needed.

## Affected Files Summary

| File | Change Type | Design Item |
|------|-------------|-------------|
| `internal/qbt/client.go` | Additive (struct fields + JSON tags) | DES-1, DES-3 |
| `internal/formatter/formatter.go` | Additive (two output lines) | DES-2 |
| `internal/formatter/formatter_test.go` | New test cases | DES-2 |
| `internal/qbt/http_integration_test.go` | New assertions | DES-3 |

## Gate 2 Check

- [x] Every REQ-* is mapped to at least one DES-*.
- [x] No design item is orphaned (each maps back to a REQ-*).
- [x] Changes are additive only; no existing behaviour is modified.
- [x] No new dependencies introduced.
- [x] Affected files are identified.
