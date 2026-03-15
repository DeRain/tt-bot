---
title: Add Torrent - Design
feature_id: add-torrent
depends_on_spec: docs/features/add-torrent/spec.md
last_updated: 2026-03-15
---

# Add Torrent - Design

## Design Items

### DES-1: Pending Torrent Storage

**Satisfies:** REQ-5, REQ-6
**Covers:** AC-5.1, AC-6.1

The `Handler` struct holds a `pending map[int64]*PendingTorrent` field protected by a `sync.Mutex`. The map is keyed by chat ID (int64), enforcing one pending torrent per user (REQ-6). Each `PendingTorrent` stores either a `MagnetLink` string or `FileData`/`FileName` byte pair, plus a `CreatedAt` timestamp for TTL eviction (REQ-5).

`storePending(chatID, pt)` unconditionally overwrites any existing entry, implementing the "last write wins" replacement semantics of REQ-6. `takePending(chatID)` atomically retrieves and removes the entry, returning `nil` when no entry exists.

### DES-2: Magnet Link Detection

**Satisfies:** REQ-1
**Covers:** AC-1.1

`handleMagnet` scans the message text for the substring `magnet:?` using `strings.Index`. It extracts from that position to the next whitespace character (space, tab, newline, carriage return) or end of string. This allows magnets embedded in surrounding text to be correctly extracted. The extracted URI is stored via `storePending`.

### DES-3: .torrent File Handling

**Satisfies:** REQ-2
**Covers:** AC-2.1

`handleTorrentFile` checks `msg.Document` for a `.torrent` suffix (case-insensitive). It calls `sender.GetFile()` to obtain the Telegram file path, then `downloadFile()` to fetch the raw bytes from the Telegram CDN (`https://api.telegram.org/file/bot<token>/<path>`). The file bytes and original filename are stored in `PendingTorrent.FileData` and `PendingTorrent.FileName`.

The download uses a dedicated `http.Client` with a 30-second timeout. Errors are sanitized to avoid leaking the bot token that appears in the CDN URL.

### DES-4: Category Keyboard

**Satisfies:** REQ-3, REQ-4, REQ-7
**Covers:** AC-3.1, AC-4.1, AC-7.1

`sendCategoryKeyboard` calls `qbt.Client.Categories()` to fetch the current category list, then delegates to `formatter.CategoryKeyboard()` which builds a `Keyboard` (slice of `ButtonRow`) with one button per category. Each button's callback data is `cat:<name>`.

When no categories exist, a single "No category" button with callback data `cat:` is returned, ensuring the user always has at least one option and the flow can complete.

### DES-5: Category Callback Processing

**Satisfies:** REQ-1, REQ-2, REQ-3, REQ-8, REQ-9
**Covers:** AC-3.1, AC-8.1, AC-9.1

`handleCategoryCallback` is triggered when callback data starts with `cat:`. It calls `takePending(chatID)` to atomically retrieve and remove the pending entry:

- **Entry exists with `MagnetLink`:** Calls `qbt.Client.AddMagnet(ctx, magnet, category)`.
- **Entry exists with `FileData`:** Calls `qbt.Client.AddTorrentFile(ctx, filename, reader, category)`.
- **Entry is nil:** Answers the callback with the expired-torrent error message (AC-9.1).

On success, the original category selection message is edited to show "Torrent added to <category>!" (or "Torrent added!" for empty category). On failure, the message is edited to show the error. The callback spinner is always dismissed via `answerCallback`.

### DES-6: TTL Eviction

**Satisfies:** REQ-5
**Covers:** AC-5.1

A background goroutine (`runCleanup`) runs on a 1-minute ticker. On each tick, `evictExpired` acquires the mutex and deletes all entries whose `CreatedAt` is older than `pendingTTL` (5 minutes). The goroutine exits when the context passed to `New()` is cancelled.

The cleanup interval (1 minute) is deliberately shorter than the TTL (5 minutes) to ensure entries are evicted reasonably close to their expiry time without excessive lock contention.

### DES-7: Confirmation Message

**Satisfies:** REQ-8
**Covers:** AC-8.1

Confirmation is delivered by editing the existing category selection message via `editMessageText`, which uses `sender.Request()` (not `Send()`) because Telegram's `editMessageText` API returns a boolean, not a `Message` object. This replaces the inline keyboard with a clean confirmation string, giving the user clear feedback without adding a new message to the chat.

### DES-8: UTF-8 Callback Truncation

**Satisfies:** REQ-7
**Covers:** AC-7.1

`formatter.CategoryKeyboard` enforces the 64-byte limit on callback data (`MaxCallbackData = 64`). When `cat:<name>` exceeds 64 bytes, it is truncated to exactly 64 bytes, then backed off one byte at a time until the result is valid UTF-8. This prevents splitting a multi-byte character sequence at the boundary.

## Requirement-to-Design Mapping

| Requirement | Design Items |
|-------------|-------------|
| REQ-1 | DES-2, DES-5 |
| REQ-2 | DES-3, DES-5 |
| REQ-3 | DES-4, DES-5 |
| REQ-4 | DES-4 |
| REQ-5 | DES-1, DES-6 |
| REQ-6 | DES-1 |
| REQ-7 | DES-4, DES-8 |
| REQ-8 | DES-5, DES-7 |
| REQ-9 | DES-5 |

## Quality Gates

### Gate 2: Design Gate

- [x] Every spec requirement (REQ-*) maps to at least one design item (DES-*)
- [x] Every design item traces back to at least one requirement
- [x] All acceptance criteria are covered by design items
- [x] No TODO placeholders remain in design items
- [x] Requirement-to-Design mapping table is complete

#### Harness Check

```bash
# Verify all spec REQs appear in the design mapping
comm -23 \
  <(grep -oP 'REQ-\d+' docs/features/add-torrent/spec.md | sort -u) \
  <(grep -oP 'REQ-\d+' docs/features/add-torrent/design.md | sort -u)
# Expected: empty (all spec REQs present in design)
```
