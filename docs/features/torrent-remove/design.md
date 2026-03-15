---
title: "Stop and Remove Torrent Actions — Design"
feature_id: torrent-remove
status: draft
depends_on_spec: "docs/features/torrent-remove/spec.md"
last_updated: 2026-03-15
---

# Stop and Remove Torrent Actions — Design

## Overview

Extends the stateless callback architecture introduced by `torrent-control` with four new callback prefixes for the remove confirmation flow. All navigation context (filter, page, hash) is encoded in callback data — no new server-side state. Adds `DeleteTorrents` to the `qbt.Client` interface, a confirmation view formatter, and four callback handlers. The detail keyboard gains a Remove button row; post-removal navigation reuses the existing back-to-list path.

## Architecture

### Callback Encoding Scheme

All new callbacks follow the existing `<prefix>:<filterChar>:<page>:<hash>` convention from `torrent-control`. Filter characters and their mappings are unchanged (`a` = all, `c` = active).

| Prefix | Format | Trigger | Max bytes (page=99, hash=40 chars) |
|--------|--------|---------|------------------------------------|
| `rm:` | `rm:<f>:<page>:<hash>` | Remove button pressed — show confirmation | 49 |
| `rd:` | `rd:<f>:<page>:<hash>` | Confirm remove torrent only (`deleteFiles=false`) | 49 |
| `rf:` | `rf:<f>:<page>:<hash>` | Confirm remove with files (`deleteFiles=true`) | 49 |
| `rc:` | `rc:<f>:<page>:<hash>` | Cancel confirmation — return to detail | 49 |

All four are under the 64-byte Telegram limit at worst case.

### Component Responsibilities

| Component | Change | Responsibility |
|-----------|--------|----------------|
| `qbt.Client` | Add method | `DeleteTorrents` API call |
| `qbt.HTTPClient` | Add method | POST to `/api/v2/torrents/delete` |
| `formatter` | Add functions | Confirmation view text, updated detail keyboard |
| `bot/callback.go` | Add handlers + dispatcher cases | Route and execute the four new prefixes |

### Updated Detail Keyboard Layout

```
Row 1: [⏸ Pause] or [▶ Start]   (existing, state-dependent)
Row 2: [🗑 Remove]
Row 3: [⬅ Back to list]
```

The Remove button is always present, independent of torrent state.

### Confirmation View Layout

```
<torrent name>

Are you sure you want to remove this torrent?

Row 1: [Remove torrent only]
Row 2: [Remove with files]
Row 3: [Cancel]
```

## Data Flow

### Remove Button Flow (show confirmation)

1. User taps Remove → Telegram sends callback `rm:<f>:<page>:<hash>`
2. `handleCallback` routes to `handleRemoveConfirmCallback`
3. Handler fetches torrent by hash via `qbt.ListTorrents` to retrieve its name for display
4. If not found → answer callback "Torrent not found", navigate to list
5. If found → format confirmation view with torrent name + confirmation keyboard
6. Edit message in place with confirmation view
7. Answer callback (no-op text)

### Confirm Remove Flow

1. User taps "Remove torrent only" → callback `rd:<f>:<page>:<hash>`
   OR taps "Remove with files" → callback `rf:<f>:<page>:<hash>`
2. `handleCallback` routes to `handleRemoveDeleteCallback`
3. Handler calls `qbt.DeleteTorrents(ctx, []string{hash}, deleteFiles)`
4. On error → answer callback with error text, no navigation change
5. On success → convert filter char to filter + prefix, fetch torrent list, format list view
6. Edit message to show torrent list at `<f>:<page>`
7. Answer callback "Removed"

### Cancel Flow

1. User taps Cancel → callback `rc:<f>:<page>:<hash>`
2. `handleCallback` routes to `handleRemoveCancelCallback`
3. Handler fetches torrent by hash via `qbt.ListTorrents`
4. If not found → answer callback "Torrent not found", navigate to list
5. If found → format detail view + detail keyboard (same as `handleSelectCallback`)
6. Edit message back to detail view
7. Answer callback (no-op text)

## Interfaces

### qbt.Client Addition

```go
// DeleteTorrents removes one or more torrents identified by info-hash.
// If deleteFiles is true, the associated downloaded data is also deleted from disk.
DeleteTorrents(ctx context.Context, hashes []string, deleteFiles bool) error
```

Updated full interface:

```go
type Client interface {
    Login(ctx context.Context) error
    AddMagnet(ctx context.Context, magnet string, category string) error
    AddTorrentFile(ctx context.Context, filename string, data io.Reader, category string) error
    ListTorrents(ctx context.Context, opts ListOptions) ([]Torrent, error)
    Categories(ctx context.Context) ([]Category, error)
    PauseTorrents(ctx context.Context, hashes []string) error
    ResumeTorrents(ctx context.Context, hashes []string) error
    DeleteTorrents(ctx context.Context, hashes []string, deleteFiles bool) error
}
```

### New Formatter Functions

```go
// FormatRemoveConfirmation renders the confirmation prompt text for a torrent removal.
// Includes the torrent name so the user can verify before confirming.
func FormatRemoveConfirmation(torrentName string) string

// RemoveConfirmKeyboard builds the three-button confirmation keyboard.
// Encodes hash, filterChar, and page into all four callback variants.
func RemoveConfirmKeyboard(hash, filterChar string, page int) Keyboard

// TorrentDetailKeyboard (existing function, modified) now adds a Remove row.
// Signature unchanged; implementation adds rm: row between pause/resume and back rows.
func TorrentDetailKeyboard(hash, filterChar string, page int, state string) Keyboard
```

## Data/Storage Impact

None. All context is encoded in callback data. No new persistent or in-memory state. The feature is fully stateless consistent with the project's design philosophy.

## Error Handling

| Scenario | User sees | Logged |
|----------|-----------|--------|
| Torrent not found when showing confirmation | Callback answer "Torrent not found"; navigates to list | Warning: hash not in list |
| `DeleteTorrents` API call fails | Callback answer with error text; confirmation view unchanged | Error with context |
| Torrent not found when cancelling (deleted between confirmation and cancel) | Callback answer "Torrent not found"; navigates to list | Warning: hash not in list |
| List empty after deletion | Empty-list message (existing behavior in list formatter) | — |
| Invalid/malformed callback format | Callback answered silently, no action | Warning: malformed callback |

## Security Considerations

- All four new callback handlers are gated by the existing `Authorizer.IsAllowed(userID)` check in `HandleUpdate` — no changes to the auth layer.
- Torrent hashes come from previously issued Telegram callbacks, not from user-typed input. They are passed directly to the qBittorrent API, which treats them as opaque identifiers.
- No new secrets, credentials, or environment variables are introduced.
- Destructive action (`deleteFiles=true`) requires two separate user taps: the Remove button and then the "Remove with files" confirmation button. One tap cannot trigger file deletion.

## Performance Considerations

- Showing the confirmation view requires one `ListTorrents` call (to fetch the torrent name). This is the same cost as the existing `handleSelectCallback`.
- The delete API call itself is a lightweight POST with no response body.
- Cancel requires one `ListTorrents` call to re-render the detail view — same cost as any detail refresh.
- No caching introduced; consistent with stateless design.

## Tradeoffs

### Fetch torrent name for confirmation vs. encode name in callback

- **Decision**: Fetch the torrent name at confirmation-show time via `ListTorrents`.
- **Alternative**: Encode the torrent name (truncated) in the `rm:` callback data.
- **Rationale**: Torrent names can be long (hundreds of bytes). Encoding them in callback data would eat into the 64-byte budget and risk truncation artefacts. A fresh fetch ensures accuracy and keeps callbacks small.

### Four separate callback prefixes vs. a single prefix with sub-action parameter

- **Decision**: Use four distinct prefixes (`rm:`, `rd:`, `rf:`, `rc:`).
- **Alternative**: Use one prefix and encode the action as an extra field, e.g., `rv:confirm_only:<f>:<page>:<hash>`.
- **Rationale**: Consistent with the existing codebase pattern (`pa:`, `re:`, `bk:`, `sel:`). Simpler dispatcher routing — no secondary parsing needed. Prefix length is 3 bytes, which leaves 61 bytes for the rest of the payload.

### Navigate to list after deletion vs. show a "Removed" message

- **Decision**: Navigate directly to the torrent list view after deletion.
- **Alternative**: Show a transient "Torrent removed" message, then navigate.
- **Rationale**: The list is immediately useful. A transient message requires an extra interaction (or a timed edit) and adds complexity. Consistent with the spirit of the Back button flow.

## Risks

- **Race between confirmation display and user action**: Torrent could be removed by another client between the confirmation being shown and the user clicking confirm. Mitigation: handle `DeleteTorrents` errors gracefully; if the hash no longer exists, qBittorrent silently succeeds (its delete endpoint is idempotent for unknown hashes), so the user is navigated to the list as if deletion succeeded.
- **Existing `TorrentDetailKeyboard` callers**: Adding a new row to the detail keyboard changes the keyboard layout for all existing callers. Mitigation: the function signature is unchanged; callers do not need updating.
- **Existing unit tests for `TorrentDetailKeyboard`**: Row count assertions will break. Mitigation: update expected row counts in affected tests as part of TASK-4.

## Design Items

- **DES-1**: Add `DeleteTorrents(ctx context.Context, hashes []string, deleteFiles bool) error` to the `qbt.Client` interface in `internal/qbt/client.go`.
  - Satisfies: REQ-3, REQ-4
  - Covers: AC-3.1, AC-4.1

- **DES-2**: Implement `DeleteTorrents` on `HTTPClient` in `internal/qbt/http.go`. POSTs `hashes=<pipe-separated>&deleteFiles=<true|false>` to `/api/v2/torrents/delete` using `doWithAuth` for automatic re-login on 403.
  - Satisfies: REQ-3, REQ-4
  - Covers: AC-3.1, AC-3.2, AC-4.1

- **DES-3**: Extend `TorrentDetailKeyboard` in `internal/formatter/format.go` to include a Remove row (`rm:<f>:<page>:<hash>`) between the pause/resume row and the back row.
  - Satisfies: REQ-1
  - Covers: AC-1.1, AC-1.2

- **DES-4**: Add `FormatRemoveConfirmation(torrentName string) string` and `RemoveConfirmKeyboard(hash, filterChar string, page int) Keyboard` to `internal/formatter/format.go`. The confirmation keyboard has three rows: "Remove torrent only" (`rd:`), "Remove with files" (`rf:`), "Cancel" (`rc:`).
  - Satisfies: REQ-2, REQ-6
  - Covers: AC-2.1, AC-2.2, AC-4.2, AC-6.1, AC-6.2

- **DES-5**: Add `handleRemoveConfirmCallback` in `internal/bot/callback.go`. Parses `rm:<f>:<page>:<hash>`, fetches torrent by hash, renders confirmation view, edits message. No API mutation.
  - Satisfies: REQ-2
  - Covers: AC-2.1, AC-2.2

- **DES-6**: Add `handleRemoveDeleteCallback` in `internal/bot/callback.go`. Handles both `rd:` and `rf:` prefixes; the prefix determines the `deleteFiles` bool passed to `DeleteTorrents`. On success, fetches the torrent list and navigates to the list view at the encoded filter and page.
  - Satisfies: REQ-3, REQ-4, REQ-5
  - Covers: AC-3.1, AC-4.1, AC-5.1, AC-5.2

- **DES-7**: Add `handleRemoveCancelCallback` in `internal/bot/callback.go`. Parses `rc:<f>:<page>:<hash>`, re-fetches torrent, renders detail view, edits message. No API mutation.
  - Satisfies: REQ-6
  - Covers: AC-6.1, AC-6.2

- **DES-8**: Register all four new callback prefixes (`rm:`, `rd:`, `rf:`, `rc:`) in the callback dispatcher switch in `internal/bot/callback.go`.
  - Satisfies: REQ-1, REQ-2, REQ-3, REQ-4, REQ-5, REQ-6
  - Covers: AC-1.1, AC-2.1, AC-3.1, AC-4.1, AC-5.1, AC-6.1

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
spec_reqs=$(grep -oP 'REQ-\d+' docs/features/torrent-remove/spec.md | sort -u)
design_reqs=$(grep -oP 'REQ-\d+' docs/features/torrent-remove/design.md | sort -u)
comm -23 <(echo "$spec_reqs") <(echo "$design_reqs")  # should be empty
```

## Requirement Mapping

| Design Item | Satisfies | Covers |
|-------------|-----------|--------|
| DES-1 | REQ-3, REQ-4 | AC-3.1, AC-4.1 |
| DES-2 | REQ-3, REQ-4 | AC-3.1, AC-3.2, AC-4.1 |
| DES-3 | REQ-1 | AC-1.1, AC-1.2 |
| DES-4 | REQ-2, REQ-6 | AC-2.1, AC-2.2, AC-4.2, AC-6.1, AC-6.2 |
| DES-5 | REQ-2 | AC-2.1, AC-2.2 |
| DES-6 | REQ-3, REQ-4, REQ-5 | AC-3.1, AC-4.1, AC-5.1, AC-5.2 |
| DES-7 | REQ-6 | AC-6.1, AC-6.2 |
| DES-8 | REQ-1, REQ-2, REQ-3, REQ-4, REQ-5, REQ-6 | AC-1.1, AC-2.1, AC-3.1, AC-4.1, AC-5.1, AC-6.1 |
