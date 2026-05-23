# internal/bot — Telegram Bot Handlers

**Scope:** Telegram update dispatch, command handlers, inline callbacks, auth whitelist.

## STRUCTURE
```
handler.go    → Handler struct, command routing, live view refresh
callback.go   → All inline keyboard callbacks (pagination, torrent actions, file management)
commands.go   → Bot command registration (/list, /active, /downloading, etc.)
auth.go       → Whitelist check by Telegram user ID
sender.go     → Sender interface abstraction for Telegram API
```

## KEY SYMBOLS
| Symbol | Type | File | Role |
|--------|------|------|------|
| `Handler` | struct | handler.go | Central dispatcher; 40+ method refs across handler+callback |
| `HandleUpdate` | method | handler.go | Entry point for every Telegram update |
| `handleCallback` | method | callback.go | Routes all inline keyboard callbacks |
| `PendingTorrent` | struct | handler.go | In-memory pending state (5-min TTL) |
| `LiveView` | struct | handler.go | Auto-refreshing message tracking |
| `Sender` | interface | sender.go | Abstracted Telegram send/edit API |

## TELEGRAM CONSTRAINTS
- **Messages:** max 4096 UTF-8 chars (formatter handles truncation)
- **Callback data:** max 64 bytes — use short prefixes only:
  - `cat:<name>`, `pg:all:<N>`, `pg:act:<N>`, `pg:dw:<N>`, `pg:up:<N>`, `pg:fl:<hash>:<page>`
  - `sel:<hash>`, `rm:<hash>`, `rd:`, `rf:`, `rc:` (remove variants)
  - `pa:`, `re:` (pause/resume), `bk:` (back), `fl:` (files), `fs:`, `fp:`, `noop`
- **Auth:** numeric Telegram user IDs only; usernames not supported

## CALLBACK FLOW
```
HandleUpdate → handleCallback → parse prefix → dispatch:
  cat:  → handleCategoryCallback
  pg:   → handlePaginationCallback
  sel:  → handleSelectCallback (detail view)
  rm:   → handleRemoveConfirmCallback / handleRemoveDeleteCallback
  file actions → handleFilesPageCallback / handleFilePriorityCallback
```

## CONVENTIONS
- Handler methods are pointer receivers on `*Handler`
- All outgoing Telegram operations go through `Sender` interface (mock in tests)
- `liveViews` map is protected by `liveViewsMu`; `pending` by `mu`
- Auto-refresh runs on `viewRefreshInterval` ticker (default 5s)

## ANTI-PATTERNS
- Do NOT hardcode Telegram size limits — use `formatter` package
- Do NOT bypass `Sender` interface for Telegram calls
- Do NOT add state persistence — bot is intentionally stateless
- Never use raw byte truncation on user strings; truncate at valid UTF-8 boundaries
