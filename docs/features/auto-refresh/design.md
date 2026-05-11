---
title: "Auto-Refresh Views — Design"
feature_id: "auto-refresh"
status: draft
depends_on_spec: "docs/features/auto-refresh/spec.md"
last_updated: 2026-05-12
---

# Auto-Refresh Views — Design

## Overview

Adds a background goroutine to the bot handler that polls qBittorrent on a configurable ticker, re-renders active views, and edits Telegram messages only when the rendered content has changed. A `LiveView` struct tracks the active view per chat, and a per-chat map enforces the single-active-view constraint. The bot remains stateless across restarts.

## Architecture

### Component Responsibilities

| Component | Responsibility |
|-----------|---------------|
| `config` | Parse `VIEW_REFRESH_INTERVAL` env var, provide `ViewRefreshInterval()` accessor |
| `LiveView` struct | Track active view state: chatID, messageID, view type, filter, page, torrent hash, last content hash |
| `liveViews map[int64]LiveView` | Per-chat view registry in `Handler` struct, protected by `sync.Mutex` |
| `runAutoRefresh` goroutine | Tick-based loop that calls `refreshViews` on each tick |
| `refreshViews` | Iterate `liveViews`, call `refreshLiveView` for each |
| `refreshLiveView` | Fetch torrents, re-render view, hash new content, compare, edit if changed |
| `bot/callback` | Register/deregister views at lifecycle points (send, select, paginate, back, actions, help, categories) |

### LiveView Struct

```go
type ViewType int

const (
    ViewTypeList ViewType = iota
    ViewTypeDetail
)

type LiveView struct {
    ChatID      int64
    MessageID   int
    ViewType    ViewType
    Filter      qbt.TorrentFilter  // only for list views
    Page        int                // only for list views
    Hash        string             // torrent info-hash, only for detail views
    ContentHash string             // SHA-256 hex of last rendered content
}
```

### Per-Chat View Tracking

```go
type Handler struct {
    // ... existing fields ...
    liveViews   map[int64]LiveView
    liveViewsMu sync.Mutex
}
```

Only one `LiveView` entry per `chatID`. Registering a new view for the same chat overwrites the previous entry, effectively deregistering the old view.

## Data Flow

### List View Auto-Refresh Flow

1. Ticker fires every N seconds (from `VIEW_REFRESH_INTERVAL`)
2. `runAutoRefresh` calls `refreshViews`
3. For each `LiveView` in `liveViews`:
   - Fetch all torrents via `qbt.ListTorrents(ctx, filter)` (or unfiltered for list views)
   - Re-render the list page using existing `renderTorrentListPage` / formatter functions
   - Compute SHA-256 of the rendered text
   - If hash matches `LiveView.ContentHash` → skip edit
   - If hash differs → call `bot.EditMessageText(chatID, messageID, newText, replyMarkup)`
   - On success → update `LiveView.ContentHash`
   - On failure (message deleted, permission lost) → delete from `liveViews`, log warning

### Detail View Auto-Refresh Flow

1. Ticker fires, `refreshViews` iterates
2. For detail-type `LiveView`:
   - Fetch all torrents, find by hash (linear scan)
   - If hash not found → skip (torrent may have been deleted, view stays registered but hash won't match until user navigates away)
   - Re-render detail via `formatter.FormatTorrentDetail`
   - Compute hash, compare, edit if changed
   - On failure → deregister

### Registration Lifecycle

| Trigger | Action |
|---------|--------|
| List command sent (`/all`, `/active`, etc.) | Register list view with filter, page 1 |
| User paginates | Update `LiveView.Page` and `LiveView.ContentHash` |
| User selects torrent | Register detail view (overwrites list view) |
| User presses Back from detail | Register list view (overwrites detail view) |
| User presses Pause/Resume | Keep detail view registered; refresh immediately + let ticker continue |
| User sends `/categories`, `/help`, or adds torrent | Deregister view for that chat |
| Edit fails (message deleted/permission) | Deregister view silently |

### Immediate Refresh After User Action

When a user interacts with a view (paginate, select, pause, resume), the handler refreshes the message immediately in the callback handler. After that, the view is registered (or re-registered) in `liveViews` so the ticker continues background refreshes. This means the user always sees fresh data after their action, and the view stays alive for automatic updates.

## Interfaces

### Config Addition

```go
// ViewRefreshInterval returns the configured auto-refresh interval.
// Defaults to 5s if VIEW_REFRESH_INTERVAL is unset or invalid.
func ViewRefreshInterval() time.Duration
```

### Handler Method Additions

```go
// registerView adds or overwrites the live view for a chat.
func (h *Handler) registerView(v LiveView)

// deregisterView removes the live view for a chat.
func (h *Handler) deregisterView(chatID int64)

// runAutoRefresh starts the background refresh loop. Runs until ctx is cancelled.
func (h *Handler) runAutoRefresh(ctx context.Context, interval time.Duration)

// refreshViews iterates all registered views and refreshes changed ones.
func (h *Handler) refreshViews()

// refreshLiveView fetches data, re-renders, and edits if changed.
func (h *Handler) refreshLiveView(v LiveView)
```

### Hash Helper

```go
// contentHash returns the SHA-256 hex digest of a string.
func contentHash(s string) string
```

## Data/Storage Impact

None. The `liveViews` map is purely in-memory and lost on restart. Views are re-registered on the next user interaction. This is consistent with the bot's stateless design — users simply re-open a command to resume auto-refresh after a restart.

## Error Handling

| Error | Behavior | Log |
|-------|----------|-----|
| qbt.ListTorrents fails | Skip this view for this tick, keep registered | Warning with error |
| Torrent hash not found (detail view) | Skip this view for this tick, keep registered | Debug (may have been deleted) |
| EditMessageText fails (message deleted) | Deregister view, no user message | Info |
| EditMessageText fails (permission) | Deregister view, no user message | Warning |
| EditMessageText fails (other) | Deregister view, no user message | Warning with error |
| config.ViewRefreshInterval returns invalid duration | Fall back to 5s default | Warning |
| panic in refreshViews | Recover in runAutoRefresh loop, log, continue | Error with stack trace |

All edit failures are silent from the user's perspective per REQ-6. The user can always re-send a command to start a fresh view.

## Security Considerations

- No new secrets or credentials are introduced.
- All refresh operations use the existing authorized `qbt.Client` instance — no elevation of privilege.
- Hash comparison does not expose any data; it operates on rendered message text only.
- The `liveViews` map lives entirely in-memory and is not exposed through any API.

## Performance Considerations

- **qBittorrent load**: One `ListTorrents` call per active chat per tick. A single-user bot with one view does one call every 5 seconds (default). Acceptable.
- **Hash computation**: SHA-256 over a typical Telegram message (<4KB) is sub-millisecond. Negligible.
- **Telegram API**: `EditMessageText` is called only when content actually changes. In practice, for a seeding torrent with no changes, zero Telegram API calls occur after the initial render.
- **Goroutine overhead**: One goroutine for `runAutoRefresh`, O(active chats) per tick for `refreshViews`. Bounded.

## Tradeoffs

### Polling vs. push notifications

- **Decision**: Polling. qBittorrent has no WebSocket or webhook API for state changes.
- **Alternative**: None viable without middleware between qBittorrent and the bot.

### Editing vs. sending new messages

- **Decision**: Edit existing messages in-place (REQ-7).
- **Rationale**: Sending new messages on each refresh would flood the chat. Editing keeps the conversation clean and the view anchored in the same position.

### Hash-based vs. always-edit

- **Decision**: Compare content hashes before editing (REQ-3).
- **Rationale**: Avoids Telegram "edited" label flickering and reduces API calls when data is static.

### Single view per chat vs. multiple

- **Decision**: Single active view per chat (REQ-4).
- **Rationale**: Multiple concurrent views would create confusing message edits scattered across the chat. Users rarely need two auto-refreshing views simultaneously.

### Ephemeral state vs. persistent

- **Decision**: Ephemeral in-memory state, lost on restart (DES-6).
- **Rationale**: Consistent with the bot's stateless architecture. Users re-send a command to re-register. No database dependency introduced.

## Design Items

- **DES-1**: LiveView struct — captures chatID, messageID, view type, filter, page, torrent hash, and last content hash to represent an active auto-refreshing view.
  - Satisfies: REQ-1, REQ-2, REQ-4
  - Covers: AC-1.1, AC-2.1, AC-4.1

- **DES-2**: Per-chat view tracking — a `map[int64]LiveView` in the `Handler` struct, protected by a mutex, enforcing the single-view-per-chat constraint.
  - Satisfies: REQ-4
  - Covers: AC-4.1, AC-4.2

- **DES-3**: SHA-256 change detection — compare rendered content hashes before editing to skip unnecessary Telegram API calls.
  - Satisfies: REQ-3
  - Covers: AC-3.1, AC-3.2

- **DES-4**: Background goroutine — `runAutoRefresh` with a configurable ticker reuses existing `renderTorrentListPage` and `FormatTorrentDetail` for consistent rendering.
  - Satisfies: REQ-1, REQ-2, REQ-5
  - Covers: AC-1.1, AC-1.2, AC-2.1, AC-5.1, AC-5.2, AC-5.3

- **DES-5**: Registration lifecycle — views are registered on send/select/back, updated on paginate/action, and deregistered on categories/help/failure. Keeps the liveViews map accurate.
  - Satisfies: REQ-4, REQ-6
  - Covers: AC-1.3, AC-2.2, AC-4.1, AC-4.2, AC-6.1, AC-6.2

- **DES-6**: Stateless across restarts — `liveViews` is purely in-memory, consistent with the bot's existing design philosophy.
  - Satisfies: (architectural constraint)
  - Covers: (none — design principle)

## Quality Gates

### Gate 2: Design Gate

This design passes when:
- [ ] Every REQ-* from spec.md is addressed by at least one DES-*
- [ ] Every AC-* from spec.md is covered by at least one DES-*
- [ ] Risks and tradeoffs are documented
- [ ] No DES-* exists without a linked REQ-*

**Harness check command:**
```bash
# Verify design-to-spec coverage
spec_reqs=$(grep -oP 'REQ-\d+' docs/features/auto-refresh/spec.md | sort -u)
design_reqs=$(grep -oP 'REQ-\d+' docs/features/auto-refresh/design.md | sort -u)
comm -23 <(echo "$spec_reqs") <(echo "$design_reqs")  # should be empty
```

## Requirement Mapping

| Design Item | Satisfies | Covers |
|-------------|-----------|--------|
| DES-1 | REQ-1, REQ-2, REQ-4 | AC-1.1, AC-2.1, AC-4.1 |
| DES-2 | REQ-4 | AC-4.1, AC-4.2 |
| DES-3 | REQ-3 | AC-3.1, AC-3.2 |
| DES-4 | REQ-1, REQ-2, REQ-5 | AC-1.1, AC-1.2, AC-2.1, AC-5.1, AC-5.2, AC-5.3 |
| DES-5 | REQ-4, REQ-6 | AC-1.3, AC-2.2, AC-4.1, AC-4.2, AC-6.1, AC-6.2 |
| DES-6 | (constraint) | — |
