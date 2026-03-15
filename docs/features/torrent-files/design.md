---
title: "Torrent File Management — Design"
feature_id: "torrent-files"
status: draft
depends_on_spec: "docs/features/torrent-files/spec.md"
last_updated: 2026-03-15
---

# Torrent File Management — Design

## Overview

Extend the existing layered architecture (qbt client → formatter → bot handler) with file-level operations. No new packages are introduced. The design follows the same patterns as torrent-control and downloading-list: thin qbt client methods, stateless formatter functions, and new callback prefixes routed in the existing callback dispatcher.

## Architecture

```
internal/qbt/
  types.go          ← TorrentFile struct, FilePriority constants
  client.go         ← ListFiles + SetFilePriority added to Client interface
  http.go           ← HTTP implementations of both methods

internal/formatter/
  format.go         ← FormatFileList, FileListKeyboard, PriorityKeyboard, PriorityLabel

internal/bot/
  callback.go       ← route pg:fl:, fp: callbacks
  handler.go        ← add "Files" button to DetailKeyboard
```

### Component Responsibilities

| Component | Responsibility |
|-----------|---------------|
| `qbt.TorrentFile` | Data type returned by the API; carries index, name, size, progress, priority |
| `qbt.Client.ListFiles` | Calls `GET /api/v2/torrents/files?hash=<hash>`, returns `[]TorrentFile` |
| `qbt.Client.SetFilePriority` | Calls `POST /api/v2/torrents/filePrio`, sets priority for one or more file indices |
| `formatter.FormatFileList` | Renders a page of files as a Telegram message (≤4096 chars) |
| `formatter.FileListKeyboard` | Builds inline keyboard: one button per file (tap to set priority) + prev/next paging + Back |
| `formatter.PriorityKeyboard` | Builds inline keyboard: 4 priority options (current marked) + Back to file list |
| `formatter.PriorityLabel` | Maps int priority → human-readable string |
| `bot.handleCallback` | Routes `pg:fl:` and `fp:` callbacks to handler functions |
| `bot.DetailKeyboard` | Extended to include a "Files" button |

## Data Flow

### View file list
1. User taps "Files" on torrent detail keyboard → callback `fl:<filterChar>:<page>:<hash>` is sent.
2. `handleCallback` routes to `handleFilesPage`.
3. `handleFilesPage` calls `qbt.Client.ListFiles(ctx, hash)`.
4. Result is passed to `formatter.FormatFileList(files, page)` and `formatter.FileListKeyboard(files, hash, filterChar, listPage, filePage)`.
5. Bot edits the current message with the formatted file list and keyboard.

### Navigate file list pages
1. User taps prev/next → callback `pg:fl:<hash>:<filePage>:<filterChar>:<listPage>`.
2. `handleCallback` routes to `handleFilesPage` with updated `filePage`.
3. Steps 3–5 from above repeat.

### Change file priority
1. User taps a file button → callback `fp:<hash>:<fileIndex>:<currentPriority>:<filePage>:<filterChar>:<listPage>`.
2. `handleCallback` routes to `handleFilePrioritySelect`.
3. Bot edits message to show `formatter.PriorityKeyboard(hash, fileIndex, currentPriority, filePage, filterChar, listPage)`.
4. User taps a priority option → callback `fp:<hash>:<fileIndex>:<newPriority>:<filePage>:<filterChar>:<listPage>`.
5. `handleCallback` detects this as a priority-set action (different from the show-keyboard step via a distinguishing prefix or sub-action — see Tradeoffs).
6. `handleFilePrioritySet` calls `qbt.Client.SetFilePriority(ctx, hash, []int{fileIndex}, newPriority)`.
7. Re-fetches file list and re-renders updated file list page (same as step 3–5 of "view file list").

### Back navigation
- From file list → `bk:fl:<filterChar>:<listPage>:<hash>` → re-renders torrent detail view.
- From priority keyboard → `pg:fl:<hash>:<filePage>:<filterChar>:<listPage>` (reuses file list page render).

## Interfaces

### qbt/types.go additions

```go
// FilePriority represents a qBittorrent file download priority.
type FilePriority int

const (
    FilePrioritySkip    FilePriority = 0
    FilePriorityNormal  FilePriority = 1
    FilePriorityHigh    FilePriority = 6
    FilePriorityMaximum FilePriority = 7
)

// TorrentFile represents a single file within a torrent.
type TorrentFile struct {
    Index    int          `json:"index"`
    Name     string       `json:"name"`
    Size     int64        `json:"size"`
    Progress float64      `json:"progress"`
    Priority FilePriority `json:"priority"`
}
```

### qbt/client.go interface additions

```go
ListFiles(ctx context.Context, hash string) ([]TorrentFile, error)
SetFilePriority(ctx context.Context, hash string, fileIndices []int, priority FilePriority) error
```

### formatter/format.go additions

```go
// FormatFileList formats a page of torrent files as a Telegram message.
// Returns the message text (≤4096 chars).
func FormatFileList(torrentName string, files []TorrentFile, page, totalPages int) string

// FileListKeyboard builds the inline keyboard for the file list view.
// filterChar and listPage allow the Back button to return to the correct torrent detail page.
func FileListKeyboard(files []TorrentFile, hash string, fileIndexOffset int, filePage, totalFilePages int, filterChar string, listPage int) tgbotapi.InlineKeyboardMarkup

// PriorityKeyboard builds the inline keyboard for priority selection of a single file.
func PriorityKeyboard(hash string, fileIndex int, currentPriority FilePriority, filePage int, filterChar string, listPage int) tgbotapi.InlineKeyboardMarkup

// PriorityLabel returns the human-readable label for a file priority.
func PriorityLabel(p FilePriority) string
```

## Callback Encoding

All callback data must fit within 64 bytes.

| Action | Prefix | Format | Max length |
|--------|--------|--------|-----------|
| Open file list (from detail) | `fl:` | `fl:<filterChar>:<listPage>:<hash>` | 3+1+1+1+1+40 = 47 bytes |
| File list page navigation | `pg:fl:` | `pg:fl:<hash>:<filePage>:<filterChar>:<listPage>` | 7+40+1+3+1+1+1+3 = 57 bytes |
| Show priority selector | `fs:` | `fs:<hash>:<fileIndex>:<filePage>:<filterChar>:<listPage>` | 3+40+1+5+1+3+1+1+1+3 = 59 bytes |
| Set priority | `fp:` | `fp:<hash>:<fileIndex>:<priority>:<filePage>:<filterChar>:<listPage>` | 3+40+1+5+1+1+1+3+1+1+1+3 = 61 bytes |
| Back from file list | `bk:fl:` | `bk:fl:<filterChar>:<listPage>:<hash>` | 7+1+1+1+3+1+40 = 54 bytes |

All values confirmed ≤ 64 bytes. File index max 99999 (5 digits), list page max 999 (3 digits), file page max 999 (3 digits).

## Data/Storage Impact

None. Stateless — no new in-memory state beyond the existing patterns.

## Error Handling

- `ListFiles` API error: bot answers the callback with "Failed to load files" and logs the full error.
- `SetFilePriority` API error: bot answers the callback with "Failed to set priority" and logs the full error.
- Unknown priority integer from API (e.g., mixed-priority sentinel): displayed as "Mixed" label, non-tappable in the priority keyboard (the change action is still available for the four standard values).
- Malformed callback data: silently ignored (existing pattern).

## Security Considerations

- All callbacks are gated behind the existing auth whitelist (`isAuthorized`).
- Hash and file index values from callback data are passed directly to qBittorrent. The qBittorrent client already handles API-level validation. No additional sanitisation is needed beyond ensuring the index is a non-negative integer (parsed with `strconv.Atoi`; error → ignore callback).

## Performance Considerations

- `ListFiles` is called on every file list page render. For a personal bot with torrents up to ~10,000 files, the response is a JSON array that is typically small (<100 KB). No caching is needed.
- `SetFilePriority` triggers a re-fetch of the file list to show the updated priority. This is two sequential API calls but adds negligible latency at personal-bot scale.

## Tradeoffs

### Callback data compactness

**Problem**: Encoding enough context (hash, file index, page state, filter, list page) for back-navigation within 64 bytes is tight.

**Decision**: Use two-character or three-character prefixes and colon-delimited fields. Separate prefixes for "show priority selector" (`fs:`) and "set priority" (`fp:`) avoid encoding an action sub-type. This keeps parsing simple and deterministic.

**Alternative considered**: Base64-encode a struct. Rejected — harder to debug and no byte savings at this field count.

### File name display

**Problem**: qBittorrent returns the full relative path within the torrent (e.g., `Season 1/Episode 01.mkv`). Showing the full path in a button label wastes space and looks noisy for deeply nested files.

**Decision**: Display only the last path component (after the final `/`). Truncate to 40 UTF-8 characters with a trailing `…` if longer. This is handled in `FormatFileList` and `FileListKeyboard`.

**Alternative considered**: Show the full path. Rejected — exceeds Telegram's inline button label limits and reduces readability.

### Files button placement

**Problem**: The detail keyboard already has Pause, Start, and Back. Adding Files makes four buttons. Inline keyboards support multiple rows.

**Decision**: Add "Files" as a second row of one button below the Pause/Start row. Back remains on its own row. This maintains visual grouping (control actions vs. navigation).

### Priority selection UX

**Problem**: Two taps (file → priority) is slightly more friction than a single tap. However, fitting four priority options as direct buttons on the file list row would require 4 extra buttons per file, making the keyboard unreadable.

**Decision**: Two-tap flow. First tap opens a dedicated priority keyboard; second tap sets the priority. Current priority is marked with a checkmark in the priority keyboard so users can confirm before changing.

## Risks

- **LOW**: Torrent hash in callback — SHA-1 hashes are always 40 hex chars so length is fixed and predictable.
- **LOW**: qBittorrent may return priority value `4` (mixed) for some files. This is handled gracefully by the `PriorityLabel` fallback.

## Design Items

- **DES-1**: `TorrentFile` struct and `FilePriority` constants in `internal/qbt/types.go`; `ListFiles` and `SetFilePriority` added to `qbt.Client` interface in `internal/qbt/client.go`
  - Satisfies: REQ-1, REQ-4
  - Covers: AC-1.1, AC-1.3, AC-4.2, AC-4.4

- **DES-2**: HTTP implementations of `ListFiles` (`GET /api/v2/torrents/files`) and `SetFilePriority` (`POST /api/v2/torrents/filePrio`) in `internal/qbt/http.go`
  - Satisfies: REQ-1, REQ-4
  - Covers: AC-1.1, AC-1.3, AC-4.2, AC-4.4

- **DES-3**: `FormatFileList`, `FileListKeyboard`, `PriorityKeyboard`, and `PriorityLabel` in `internal/formatter/format.go`
  - Satisfies: REQ-2, REQ-3, REQ-6
  - Covers: AC-2.1, AC-2.2, AC-2.3, AC-2.4, AC-3.1, AC-3.2, AC-3.3, AC-6.1, AC-6.2, AC-6.3, AC-6.4

- **DES-4**: `fp:` and `fs:` callback handlers in `internal/bot/callback.go` for priority selection and priority-set actions; `pg:fl:` for file list page navigation; `bk:fl:` for back-from-file-list
  - Satisfies: REQ-3, REQ-4, REQ-5
  - Covers: AC-3.1, AC-3.2, AC-4.1, AC-4.2, AC-4.3, AC-4.4, AC-5.2, AC-5.3

- **DES-5**: "Files" button added to the torrent detail inline keyboard (new row) via `fl:` callback prefix; rendered in `internal/bot/handler.go` (or whichever function builds `DetailKeyboard`)
  - Satisfies: REQ-5
  - Covers: AC-5.1

- **DES-6**: `fl:` callback routing in `internal/bot/callback.go` to open the first page of a torrent's file list
  - Satisfies: REQ-1, REQ-5
  - Covers: AC-1.1, AC-1.2, AC-5.1

## Quality Gates

### Gate 2: Design Gate

This design passes when:
- [x] Every REQ-* from spec.md is addressed by at least one DES-*
- [x] Every AC-* from spec.md is covered by at least one DES-*
- [x] Risks and tradeoffs are documented
- [x] No DES-* exists without a linked REQ-*

**Harness check command:**
```bash
spec_reqs=$(grep -oP 'REQ-\d+' docs/features/torrent-files/spec.md | sort -u)
design_reqs=$(grep -oP 'REQ-\d+' docs/features/torrent-files/design.md | sort -u)
comm -23 <(echo "$spec_reqs") <(echo "$design_reqs")  # should be empty
```

## Requirement Mapping

| Design Item | Satisfies | Covers |
|-------------|-----------|--------|
| DES-1 | REQ-1, REQ-4 | AC-1.1, AC-1.3, AC-4.2, AC-4.4 |
| DES-2 | REQ-1, REQ-4 | AC-1.1, AC-1.3, AC-4.2, AC-4.4 |
| DES-3 | REQ-2, REQ-3, REQ-6 | AC-2.1, AC-2.2, AC-2.3, AC-2.4, AC-3.1, AC-3.2, AC-3.3, AC-6.1, AC-6.2, AC-6.3, AC-6.4 |
| DES-4 | REQ-3, REQ-4, REQ-5 | AC-3.1, AC-3.2, AC-4.1, AC-4.2, AC-4.3, AC-4.4, AC-5.2, AC-5.3 |
| DES-5 | REQ-5 | AC-5.1 |
| DES-6 | REQ-1, REQ-5 | AC-1.1, AC-1.2, AC-5.1 |
