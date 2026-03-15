---
title: "Torrent Control — Design"
feature_id: "torrent-control"
status: draft
depends_on_spec: "docs/features/torrent-control/spec.md"
last_updated: 2026-03-15
---

# Torrent Control — Design

## Overview

Extends the existing stateless callback architecture with four new callback prefixes for torrent selection, pause, resume, and back-to-list navigation. All navigation context is encoded in callback data (no server-side state). Adds `PauseTorrents`/`ResumeTorrents` to the `qbt.Client` interface and new formatter functions for torrent detail rendering.

## Architecture

### Callback Encoding Scheme

All context is encoded in callbacks to stay truly stateless:

| Prefix | Format | Example | Max bytes |
|--------|--------|---------|-----------|
| `sel:` | `sel:<f>:<page>:<hash>` | `sel:a:1:abc123...def` | 49 |
| `pa:` | `pa:<f>:<page>:<hash>` | `pa:a:12:abc123...def` | 49 |
| `re:` | `re:<f>:<page>:<hash>` | `re:c:99:abc123...def` | 49 |
| `bk:` | `bk:<f>:<page>` | `bk:a:1` | 6 |

Where `<f>` is `a` (all) or `c` (active) — shortened from `all`/`act` to save bytes.

### Filter Character Mapping

| Char | Filter Constant | Pagination Prefix |
|------|----------------|-------------------|
| `a` | `qbt.FilterAll` | `all` |
| `c` | `qbt.FilterActive` | `act` |

### Component Responsibilities

| Component | Responsibility |
|-----------|---------------|
| `qbt.Client` | Pause/Resume API calls to qBittorrent |
| `formatter` | Detail view text, action keyboard, selection keyboard, size formatting, state helpers |
| `bot/callback` | Route new prefixes, handle select/pause/resume/back actions |
| `bot/handler` | Integrate selection keyboard into existing list views |

## Data Flow

### Torrent Selection Flow

1. User taps torrent button → Telegram sends callback `sel:a:1:<hash>`
2. `handleCallback` routes to `handleSelectCallback`
3. Handler fetches all torrents via `qbt.ListTorrents`
4. Handler finds torrent by hash (linear scan)
5. If not found → answer callback "Torrent not found"
6. If found → format detail view + build action keyboard
7. Edit message in place with detail view

### Pause/Resume Flow

1. User taps Pause → callback `pa:a:1:<hash>`
2. `handleCallback` routes to `handlePauseCallback`
3. Handler calls `qbt.PauseTorrents(ctx, []string{hash})`
4. On error → answer callback with error
5. On success → re-fetch torrent, format detail view, edit message
6. Answer callback "Paused"

### Back-to-List Flow

1. User taps Back → callback `bk:a:1`
2. `handleCallback` routes to `handleBackCallback`
3. Handler converts filter char to filter constant and prefix
4. Re-fetches torrents, computes page, formats list + keyboards
5. Edits message back to list view

## Interfaces

### qbt.Client Additions

```go
// PauseTorrents pauses one or more torrents identified by info-hash.
PauseTorrents(ctx context.Context, hashes []string) error

// ResumeTorrents resumes one or more torrents identified by info-hash.
ResumeTorrents(ctx context.Context, hashes []string) error
```

### New Formatter Functions

```go
// FormatSize renders bytes as human-readable size (B, KB, MB, GB, TB).
func FormatSize(bytes int64) string

// IsPaused returns true if the torrent state represents a paused condition.
func IsPaused(state string) bool

// FormatTorrentDetail renders a single torrent's full metadata.
func FormatTorrentDetail(t qbt.Torrent) string

// TorrentDetailKeyboard builds Pause/Resume + Back buttons for the detail view.
func TorrentDetailKeyboard(hash, filterChar string, page int, state string) Keyboard

// TorrentSelectionKeyboard builds numbered torrent buttons for the list view.
func TorrentSelectionKeyboard(torrents []qbt.Torrent, filterChar string, page int) Keyboard
```

### New Bot Helpers

```go
// Filter character mapping
func filterCharToFilter(char string) (qbt.TorrentFilter, bool)
func filterCharToPrefix(char string) string
func filterToChar(filter qbt.TorrentFilter) string
```

## Data/Storage Impact

None. All navigation context is encoded in callback data. No new persistent or in-memory state.

## Error Handling

| Error | User sees | Log |
|-------|-----------|-----|
| Torrent hash not found | "Torrent not found" callback answer | Warning: hash not in list |
| qbt.PauseTorrents fails | Callback answer with error text | Error with context |
| qbt.ResumeTorrents fails | Callback answer with error text | Error with context |
| Invalid callback format | Callback answered, no action | Warning: malformed callback |
| Empty torrent list on back | Empty list message (existing behavior) | — |

## Security Considerations

- All callback handlers are gated by the existing `Authorizer.IsAllowed(userID)` check in `HandleUpdate`.
- Torrent hashes are opaque identifiers — no user-supplied input is used in URL construction beyond what qBittorrent expects.
- No new secrets or credentials introduced.

## Performance Considerations

- Select/pause/resume each require one `ListTorrents` call to find the torrent by hash. This fetches all torrents but is bounded by qBittorrent's own limits. Acceptable for typical home use (hundreds of torrents).
- Pause/resume API calls are lightweight POST requests.
- No caching introduced — consistent with stateless design.

## Tradeoffs

### Stateless callbacks vs. in-memory state

- **Decision**: Encode all context in callback data.
- **Alternative**: Store return context in an in-memory map (like `pendingTorrents`).
- **Rationale**: Avoids additional state management and expiry logic. Callback data fits well within 64-byte limit. Consistent with project's stateless philosophy.

### Re-fetching torrents on every action vs. caching

- **Decision**: Re-fetch from qBittorrent on every select/pause/resume/back.
- **Alternative**: Cache torrent list for short duration.
- **Rationale**: Ensures fresh data. Caching adds complexity for minimal benefit in a single-user bot context.

### Single-character filter codes vs. full names

- **Decision**: Use `a`/`c` in callbacks instead of `all`/`act`.
- **Alternative**: Keep existing `all`/`act` prefixes.
- **Rationale**: Saves 2–4 bytes per callback, providing more headroom within the 64-byte limit.

## Risks

- **Race condition**: Torrent state may not immediately reflect pause/resume in qBittorrent's API response. Mitigation: user can re-select to see updated state; documented as known behavior.
- **Existing test breakage**: Modifying `sendTorrentPage` and `handlePaginationCallback` could break pagination tests. Mitigation: extract shared `renderTorrentListPage` helper; keep existing keyboard structure unchanged.

## Design Items

- **DES-1**: Add `PauseTorrents(ctx, hashes []string) error` and `ResumeTorrents(ctx, hashes []string) error` to `qbt.Client` interface. HTTP implementation POSTs form-encoded `hashes=hash1|hash2` to `/api/v2/torrents/stop` and `/api/v2/torrents/start` (qBittorrent v5+ endpoints) using `doWithAuth` for automatic re-login on 403.
  - Satisfies: REQ-5, REQ-6, REQ-10
  - Covers: AC-5.1, AC-6.1, AC-10.1, AC-10.2

- **DES-2**: Add `FormatTorrentDetail(torrent qbt.Torrent) string` to `formatter` package. Renders multi-line detail view: full name, size, progress bar, speeds, state, category. Stays under 4096 chars.
  - Satisfies: REQ-2, REQ-9
  - Covers: AC-2.1, AC-2.2

- **DES-3**: Add `FormatSize(bytes int64) string` helper to `formatter` package. Formats bytes as B, KB, MB, GB, TB with one decimal place.
  - Satisfies: REQ-2
  - Covers: AC-2.1

- **DES-4**: Add `TorrentDetailKeyboard(hash, filterChar string, page int, state string) Keyboard` to `formatter`. Row 1 = Pause or Resume button (based on `IsPaused`), row 2 = Back to list button. Uses compact callback encoding.
  - Satisfies: REQ-3, REQ-4, REQ-7, REQ-8
  - Covers: AC-3.1, AC-3.2, AC-4.1, AC-4.2, AC-7.1, AC-8.1

- **DES-5**: Add `IsPaused(state string) bool` helper to `formatter` package. Returns true for `"pausedDL"` and `"pausedUP"`.
  - Satisfies: REQ-3, REQ-4
  - Covers: AC-3.1, AC-3.2, AC-4.1, AC-4.2

- **DES-6**: Add `TorrentSelectionKeyboard(torrents []qbt.Torrent, filterChar string, page int) Keyboard` to `formatter`. One button per torrent with numbered label and callback `sel:<f>:<page>:<hash>`.
  - Satisfies: REQ-1, REQ-8
  - Covers: AC-1.1, AC-1.2

- **DES-7**: Modify `sendTorrentPage` and `handlePaginationCallback` to include torrent selection keyboard below pagination keyboard. Extract shared `renderTorrentListPage` helper.
  - Satisfies: REQ-1
  - Covers: AC-1.1

- **DES-8**: Add `handleSelectCallback` in `bot/callback.go`. Parses `sel:<f>:<page>:<hash>`, fetches torrents, finds by hash, renders detail view, edits message.
  - Satisfies: REQ-2, REQ-3, REQ-4
  - Covers: AC-2.1, AC-3.1, AC-4.1, AC-9.1

- **DES-9**: Add `handlePauseCallback` and `handleResumeCallback` in `bot/callback.go`. Parse callbacks, call qbt API, re-fetch torrent, refresh detail view.
  - Satisfies: REQ-5, REQ-6
  - Covers: AC-5.1, AC-5.2, AC-6.1, AC-6.2

- **DES-10**: Add `handleBackCallback` in `bot/callback.go`. Parses `bk:<f>:<page>`, converts filter, re-fetches torrents, edits message back to list view.
  - Satisfies: REQ-7
  - Covers: AC-7.1

- **DES-11**: Add filter character mapping helpers: `filterCharToFilter`, `filterCharToPrefix`, `filterToChar`.
  - Satisfies: REQ-8
  - Covers: AC-8.1

## Quality Gates

### Gate 2: Design Gate

This design passes when:
- [x] Every REQ-* from spec.md is addressed by at least one DES-*
- [x] Every AC-* from spec.md is covered by at least one DES-*
- [x] Risks and tradeoffs are documented
- [x] No DES-* exists without a linked REQ-*

**Harness check command:**
```bash
# Verify design-to-spec coverage
spec_reqs=$(grep -oP 'REQ-\d+' docs/features/torrent-control/spec.md | sort -u)
design_reqs=$(grep -oP 'REQ-\d+' docs/features/torrent-control/design.md | sort -u)
comm -23 <(echo "$spec_reqs") <(echo "$design_reqs")  # should be empty
```

## Requirement Mapping

| Design Item | Satisfies | Covers |
|-------------|-----------|--------|
| DES-1 | REQ-5, REQ-6, REQ-10 | AC-5.1, AC-6.1, AC-10.1, AC-10.2 |
| DES-2 | REQ-2, REQ-9 | AC-2.1, AC-2.2 |
| DES-3 | REQ-2 | AC-2.1 |
| DES-4 | REQ-3, REQ-4, REQ-7, REQ-8 | AC-3.1, AC-3.2, AC-4.1, AC-4.2, AC-7.1, AC-8.1 |
| DES-5 | REQ-3, REQ-4 | AC-3.1, AC-3.2, AC-4.1, AC-4.2 |
| DES-6 | REQ-1, REQ-8 | AC-1.1, AC-1.2 |
| DES-7 | REQ-1 | AC-1.1 |
| DES-8 | REQ-2, REQ-3, REQ-4 | AC-2.1, AC-3.1, AC-4.1, AC-9.1 |
| DES-9 | REQ-5, REQ-6 | AC-5.1, AC-5.2, AC-6.1, AC-6.2 |
| DES-10 | REQ-7 | AC-7.1 |
| DES-11 | REQ-8 | AC-8.1 |
