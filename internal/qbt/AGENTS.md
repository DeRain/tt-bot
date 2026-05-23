# internal/qbt — qBittorrent API Client

**Scope:** Web API v2 client with SID cookie auth and auto-re-login on 403.

## STRUCTURE
```
client.go   → qbt.Client interface (all external deps behind this)
http.go     → HTTPClient implementation: auth, retry, JSON marshaling
types.go    → Request/response structs (Torrent, File, Category, etc.)
```

## KEY SYMBOLS
| Symbol | Type | File | Role |
|--------|------|------|------|
| `Client` | interface | client.go | Primary contract: Login, AddMagnet, ListTorrents, etc. |
| `HTTPClient` | struct | http.go | Concrete impl with cookie jar and mutex |
| `doWithRetry` | method | http.go | Auto-re-login loop on 403/401 |
| `Login` | method | http.go | Fetches SID cookie, stores in `sid` field |

## INTERFACE CONTRACT
When adding new qBittorrent API calls:
1. Add method to `Client` interface in `client.go`
2. Implement in `http.go`
3. Wire into bot handler via `bot/callback.go` or `bot/handler.go`

## QBITTORRENT v5+ ENDPOINTS
Always use v5 names; v4 endpoints return 404:

| Action | v4 (broken) | v5+ (use this) |
|--------|-------------|----------------|
| Pause | `/torrents/pause` | `/torrents/stop` |
| Resume | `/torrents/resume` | `/torrents/start` |

## CONVENTIONS
- `mu` protects `sid` — all authenticated requests hold lock
- `doWithRetry` handles 403 by calling `loginLocked`, then replaying original request
- Base URL includes `/api/v2` path prefix
- Categories endpoint used to populate inline keyboard before adding torrents

## ANTI-PATTERNS
- Do NOT call qBittorrent endpoints directly from `bot/` — always go through `Client`
- Do NOT skip `Login` before other calls; `doWithRetry` handles it, but explicit is safer in tests
- Do NOT use v4 endpoint names (pause/resume) — they 404 on qBittorrent v5+
