---
name: qbittorrent-v5-api-changes
description: "qBittorrent v5+ renamed pause/resume endpoints to stop/start — use stop/start for v5 instances"
user-invocable: false
origin: auto-extracted
---

# qBittorrent v5+ API Endpoint Renames

**Extracted:** 2026-03-15
**Context:** qBittorrent WebUI API v2 changed endpoint names starting in v5.0

## Problem
qBittorrent v5+ (WebAPI v2.10+) renamed several torrent action endpoints. Code
using the old endpoint names gets 404 responses. Unit tests with httptest can't
catch this because they mock the server — only integration tests against a real
instance reveal the breakage.

## Solution
Use the v5+ endpoint names:

| Action | v4 (old) | v5+ (current) |
|--------|----------|---------------|
| Pause/Stop | `/api/v2/torrents/pause` | `/api/v2/torrents/stop` |
| Resume/Start | `/api/v2/torrents/resume` | `/api/v2/torrents/start` |

The request format is unchanged: `POST` with form-encoded `hashes=h1|h2|...`.

To check which version is running:
```bash
curl http://<host>/api/v2/app/webapiVersion
# v2.11.4 → use /stop and /start
```

## When to Use
- Adding new qBittorrent API calls to this project
- Debugging 404 responses from qBittorrent endpoints
- Referencing qBittorrent API documentation (most online docs still show v4 names)
