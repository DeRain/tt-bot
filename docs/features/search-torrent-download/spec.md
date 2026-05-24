# Feature: Search Torrent Download

## Overview

Search results from qBittorrent search plugins may include results that lack magnet links but provide a `FileURL` pointing to a `.torrent` file over HTTP/HTTPS. The bot must detect such URLs, download the `.torrent` file securely, and feed it into the existing `AddTorrentFile` flow — the same category picker UX used for magnet links and manually uploaded `.torrent` files. This requires careful handling of network safety (SSRF protection, timeouts, redirect limits, content-type checks) and graceful error reporting.

## Requirements

### REQ-1: Torrent File Download from FileURL
When `SearchResult.FileURL` starts with `http://` or `https://`, download the `.torrent` file and add via `AddTorrentFile`.

### REQ-2: SSRF Protection
Reject URLs whose hostname resolves to private (RFC 1918), loopback (127.0.0.0/8, ::1), or link-local (169.254.0.0/16) IPs.

### REQ-3: Download Size Limit
Download body capped at 10 MB via `io.LimitReader`.

### REQ-4: Download Timeout
External HTTP downloads use a 15-second timeout (separate from the shared 30s `httpClient`).

### REQ-5: Redirect Safety
Max 5 redirect hops; reject HTTPS to HTTP protocol downgrades.

### REQ-6: Content-Type Validation
Reject responses with `Content-Type: text/html`.

### REQ-7: Pending Torrent Integration
Successful download stores `FileData` + `FileName` in `PendingTorrent`, then shows category keyboard (same UX as magnet links).

### REQ-8: Error Handling
Download failures show user-friendly error message; errors never leak raw HTTP details.

## Acceptance Criteria

- AC-1.1: Tapping search result with HTTP FileURL (valid .torrent) downloads and shows category keyboard
- AC-1.2: Tapping search result with magnet FileURL continues to work unchanged (regression)
- AC-2.1: URL resolving to 127.0.0.1 is rejected before HTTP request
- AC-3.1: Response body exceeding 10MB is rejected
- AC-4.1: Download exceeding 15 seconds is rejected
- AC-5.1: Redirect chain exceeding 5 hops is rejected
- AC-5.2: HTTPS to HTTP protocol downgrade redirect is rejected
- AC-6.1: Response with Content-Type: text/html is rejected
- AC-7.1: Successful download results in pending torrent with FileData populated
- AC-8.1: Download failure shows user-friendly error message (not raw HTTP status)
