---
title: "Stop and Remove Torrent Actions — Decisions"
feature_id: "torrent-remove"
last_updated: 2026-03-15
---

# Stop and Remove Torrent Actions — Decisions

## Assumptions

1. **qBittorrent v5 delete endpoint is idempotent for unknown hashes.** If the torrent has already been removed by another client between the confirmation being displayed and the user clicking confirm, the `/api/v2/torrents/delete` call silently succeeds rather than returning an error. This means the bot can navigate to the list view as if deletion succeeded without special-casing "already gone" errors. Confirmed by manual testing against qBittorrent v5.x; no API documentation guarantee exists.

2. **Torrent hashes are stable 40-character hex strings.** The qBittorrent API uses SHA-1 info-hashes for all torrent identification. These are fixed at 40 hex characters (160 bits). No truncation or encoding is needed in callback data beyond the raw hex string.

3. **The existing `doWithAuth` re-login mechanism covers the delete endpoint.** The delete call goes through the same `doWithAuth` wrapper used by pause and resume. A 403 response triggers automatic re-login and a single retry. No special session handling is needed for TASK-2.

4. **No persistent state is needed for the confirmation flow.** The full context required to execute or cancel a deletion (filter char, page, hash) fits within the 64-byte Telegram callback data limit. The stateless design holds without any server-side session.

5. **`make test-integration` is the only valid mechanism for verifying AC-3.2.** File presence/absence on disk after deletion cannot be verified with `httptest`-based unit tests. The integration test environment must use a real qBittorrent instance with an accessible download directory.

## Design Choices

### Confirmation flow: fetch torrent name at show-time vs. encode in callback

**Decision:** Fetch the torrent name from qBittorrent when the confirmation view is shown (the `rm:` handler calls `ListTorrents` to look up the torrent by hash).

**Alternative considered:** Encode a truncated torrent name directly in the `rm:` callback data so no extra API call is needed to render the confirmation.

**Rationale:** Torrent names can be hundreds of bytes. Fitting a useful portion into the remaining ~44 bytes after `rm:<f>:<page>:<hash>` would require aggressive truncation that risks producing misleading display text (e.g., two different torrents with the same prefix). A fresh `ListTorrents` call is cheap (same cost as opening the detail view) and guarantees the confirmation prompt shows the accurate, full name. If the torrent has disappeared between the Remove tap and the confirmation render, the not-found path provides clearer feedback than a stale name would.

### Callback encoding: four distinct prefixes vs. a single prefix with sub-action field

**Decision:** Use four distinct prefixes: `rm:` (show confirmation), `rd:` (remove only), `rf:` (remove with files), `rc:` (cancel).

**Alternative considered:** Use one prefix (e.g., `rv:`) and encode the action as an additional colon-separated field: `rv:confirm_only:<f>:<page>:<hash>`.

**Rationale:** Four distinct prefixes are consistent with the existing codebase pattern (`pa:`, `re:`, `bk:`, `sel:`). The dispatcher switch routes by prefix string — no secondary parsing is needed, keeping each handler simple and independently testable. Each prefix consumes only 3 bytes, leaving 61 bytes for the `<f>:<page>:<hash>` payload. Worst-case at page=99, hash=40 chars: `rm:a:99:<40-char-hash>` = 49 bytes, well within the 64-byte limit.

### Post-deletion navigation: go directly to list vs. show a transient "Removed" message

**Decision:** After a successful deletion, edit the message directly to the torrent list view at the encoded filter and page.

**Alternative considered:** Edit the message to show a brief "Torrent removed" confirmation text, then either wait for a user tap or schedule a timed edit to navigate to the list.

**Rationale:** The list view is immediately useful — the user can verify the torrent is gone and take further actions without an extra tap. A transient message requires either a follow-up user action or a timer-based edit, both of which add complexity inconsistent with the project's stateless design. This matches the spirit of the Back button: navigating to a useful context rather than an intermediate screen.

### No separate "force stop" action

**Decision:** The feature does not add a force-stop button (stop without removing the torrent record).

**Rationale:** qBittorrent v5 does not expose a distinct "force stop" endpoint separate from `stop` (which maps to the existing Pause button's `PauseTorrents` call). The non-goal is explicitly called out in spec.md. The pause/resume flow already covers the stop use case. Adding a force-stop button would duplicate the Pause button's effect and confuse users.

### Remove button position in the detail keyboard

**Decision:** The Remove button occupies its own row between the pause/resume row (Row 1) and the Back row (Row 3), making it Row 2.

**Alternative considered:** Adding the Remove button to the same row as the pause/resume button, or placing it after the Back button.

**Rationale:** A dedicated row gives the Remove button visual separation from the state-control buttons (pause/resume), reducing the risk of an accidental tap. Placing it above Back keeps destructive actions in the middle of the keyboard rather than at the bottom where the thumb naturally rests on a phone. The `TorrentDetailKeyboard` function signature is unchanged, so all existing callers are unaffected; only the row count changes from 2 to 3.

## Deferred Work

| Item | Reason for Deferral | Potential Future Feature |
|------|---------------------|--------------------------|
| Batch removal of multiple torrents at once | Out of scope per spec.md non-goals; requires multi-select UI not present in current design | `torrent-batch-actions` |
| Remove by category or filter | Out of scope per spec.md non-goals; requires a different confirmation flow and broader API use | `torrent-batch-actions` |
| Undo / restore removed torrents | qBittorrent has no restore API; would require storing torrent metadata externally, contradicting the stateless design | Not planned |
| Confirmation timeout (auto-cancel after N seconds) | Adds timer-based state; not consistent with stateless design; Telegram inline keyboards do not expire automatically | Not planned |
| Audit log of removal actions | No logging infrastructure exists in the project; stateless design makes retroactive auditing impractical without external storage | `audit-log` |
