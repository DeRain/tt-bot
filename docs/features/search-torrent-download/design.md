# Design: Search Torrent Download

## Architecture

```
SearchResult.FileURL
       │
       ├── magnet:? ──────► storePending{MagnetLink}
       │                          │
       │                          ▼
       │                   sendCategoryKeyboard
       │                          │
       │                          ▼
       │                   handleCategoryCallback
       │                          │
       │                          ▼
       │                   AddTorrent (magnet)
       │
       └── http:// or https:// ──► downloadSearchTorrent
                                        │
                                  ┌─────┴──────┐
                                  │  Safety     │
                                  │  Checks     │
                                  │             │
                                  │ • SSRF:     │
                                  │   isPublic  │
                                  │   Hostname  │
                                  │ • Timeout:  │
                                  │   15s       │
                                  │ • Size:     │
                                  │   10MB max  │
                                  │ • Redirects:│
                                  │   ≤5 hops   │
                                  │ • No HTTPS→ │
                                  │   HTTP      │
                                  │   downgrade │
                                  │ • Reject    │
                                  │   text/html │
                                  └─────┬──────┘
                                        │
                                        ▼
                                  []byte data
                                        │
                                        ▼
                             storePending{FileData, FileName}
                                        │
                                        ▼
                                  sendCategoryKeyboard
                                        │
                                        ▼
                                  handleCategoryCallback
                                        │
                                        ▼
                                  AddTorrentFile (HTTP)
```

## Key Decisions

### DES-1: Download Approach — Go-native HTTP download, not URL passthrough

The bot downloads the `.torrent` file in Go using `net/http`, then passes the raw bytes to `AddTorrentFile` (the existing endpoint for Telegram-uploaded `.torrent` files).

**Why not pass the URL to qBittorrent?** The qBittorrent Web API v2 does not expose an "add from URL" endpoint that accepts arbitrary HTTP sources in the search flow. Adding via `AddTorrentFile` reuses the same category picker pipeline and keeps the implementation simple.

**Why not use the existing `downloadFileURL` in handler.go?** That function is designed for Telegram CDN file downloads (bot token embedded in URL) and lacks SSRF protection, size limits, content-type checks, and redirect safety. A purpose-built function avoids coupling these two very different concerns.

### DES-2: Synchronous UX — download blocks the callback inline

The `downloadSearchTorrent` call runs synchronously inside the callback handler (same goroutine). The Telegram inline keyboard spinner keeps the user informed during the 1-3 second download. This is consistent with how magnet links work — there is no async progress bar for either path.

**Why not async?** The download takes 1-3 seconds for a typical `.torrent` file (usually 50-200 KB). Adding async state tracking (spinner message, goroutine lifecycle, error delivery) adds complexity for marginal UX gain. If downloads exceed 3 seconds, the synchronous call still completes within the Telegram callback timeout window.

### DES-3: No HEAD Probe — Content-Type from GET response is sufficient

The implementation does not send a preliminary `HEAD` request to check `Content-Type` before the full `GET`. The 15-second timeout and 10 MB size limit on the GET response provide adequate protection. If the GET returns `text/html`, the download is rejected after reading the headers (body bytes are discarded by `io.LimitReader`).

**Why no HEAD?** A HEAD probe doubles the latency for every download. Since the 15-second timeout already bounds worst-case behavior, and `io.LimitReader` prevents unbounded memory use, the extra round-trip is not justified.

### DES-4: No DescrLink Fallback — fix the primary case first

When `FileURL` is present and valid (HTTP/HTTPS), the bot downloads it. There is no fallback to `DescrLink` (the search result's description page link) for results that lack a `FileURL`. This avoids speculative heuristics like "scrape the tracker's HTML page for a download link."

**Future consideration:** If user feedback shows that many trackers only expose download links via `DescrLink`, a follow-up feature can add a heuristic scraper. For now, the feature addresses the primary bug: trackers that expose `FileURL` but whose magnets are not available.

### DES-5: Code Location — package-level functions in `download.go`

The download logic lives in `internal/bot/download.go` as package-level functions in the `bot` package:

| Function | Purpose |
|----------|---------|
| `isPublicHostname(host string) error` | SSRF guard — rejects private, loopback, link-local, and unspecified IPs |
| `newDownloadClient() *http.Client` | Factory for download HTTP client (15s timeout, redirect safety) |
| `downloadSearchTorrent(ctx, client, url) ([]byte, error)` | Full download pipeline: scheme check, SSRF check, GET, Content-Type check, size limit |

**Why no new interface?** These functions are pure computation with no external dependencies beyond the standard library. An interface would add ceremony for no benefit. The test-injection hook (DES-6) handles the only case where indirection is needed.

### DES-6: Test Injection — `var downloadSearchTorrentFn`

```go
var downloadSearchTorrentFn = downloadSearchTorrent
```

A package-level variable holds the download function. Tests can replace it:

```go
downloadSearchTorrentFn = func(ctx context.Context, client *http.Client, url string) ([]byte, error) {
    return []byte("fake torrent data"), nil
}
```

This allows callback integration tests to exercise the HTTP download path without:
- Needing a real HTTP server (which would fail the SSRF check on 127.0.0.1)
- Threading a dependency through the `Handler` struct

**Why not a method on Handler?** The function has no dependency on `Handler` state (no `Sender`, no `qbt.Client`, no `token`). A package-level var is the lightest-weight injection mechanism. It is reset per test via `t.Cleanup`.

### DES-7: SSRF Protection — `isPublicHostname` before HTTP request

Before making any HTTP request, the bot parses the URL's host and checks it against RFC 1918 private ranges, loopback (127.0.0.0/8, ::1), link-local unicast (169.254.0.0/16), and unspecified (0.0.0.0, ::) addresses. For bare IP literals the check is exact (no DNS). For hostnames, the function currently allows them without resolution (a known gap — see below).

**Why not full DNS resolution?** The search results come from qBittorrent search plugins (Jackett), which in turn query public trackers. A hostname like `example.com` arriving in a search result is nearly always legitimate, and resolving each hostname at callback time would add latency and a DNS exfiltration vector. Full DNS resolution is deferred as a follow-up enhancement.

**Protected ranges:**

| Range | Type | Rejected |
|-------|------|----------|
| 10.0.0.0/8 | Private (RFC 1918) | Yes |
| 172.16.0.0/12 | Private (RFC 1918) | Yes |
| 192.168.0.0/16 | Private (RFC 1918) | Yes |
| 127.0.0.0/8 | Loopback | Yes |
| ::1 | Loopback (IPv6) | Yes |
| 169.254.0.0/16 | Link-local unicast | Yes |
| 0.0.0.0, :: | Unspecified | Yes |

## Error Handling

| Scenario | Bot Response |
|----------|--------------|
| URL is neither magnet nor HTTP | "This result doesn't have a valid download link. Try another result." |
| SSRF rejection (private/loopback IP) | "Failed to download torrent: ..." (sanitized, does not leak raw error) |
| HTTP non-200 status | "Failed to download torrent: ..." |
| Content-Type is text/html | "Failed to download torrent: ..." |
| Body exceeds 10 MB | "Failed to download torrent: ..." |
| Timeout (15s exceeded) | "Failed to download torrent: ..." |
| Redirect chain >5 hops | "Failed to download torrent: ..." |
| HTTPS to HTTP scheme downgrade | "Failed to download torrent: ..." |
| Invalid URL / unsupported scheme | "Failed to download torrent: ..." |

All errors from `downloadSearchTorrent` are prefixed with "Failed to download torrent: " and surfaced inline via `editMessageText` on the confirmation message. No raw HTTP details, IP addresses, or internal error types leak to the user.

## Constants

```go
downloadTimeout = 15 * time.Second  // Per-request timeout for .torrent downloads
downloadMaxSize = 10 * 1024 * 1024  // 10 MB maximum response body
downloadMaxRedirects = 5             // Maximum redirect hops
```

## File Layout

```
internal/bot/
  download.go        → isPublicHostname, newDownloadClient, downloadSearchTorrent, downloadSearchTorrentFn
  download_test.go   → TDD-red safety function tests (isPublicHostname, downloadSearchTorrent, newDownloadClient)
  callback.go        → handleSearchConfirmCallback modified to branch on FileURL prefix
  callback_test.go   → Callback integration tests with downloadSearchTorrentFn injection
```
