# Refresh Status Views Design

## Summary

Add an explicit Refresh button to every status-bearing Telegram bot view so the user can re-fetch current qBittorrent state without leaving the current message. The button applies to torrent list views, torrent detail, file list, and file-priority selection. It does not apply to category selection or remove confirmation.

## Goals

- Let users refresh the current view in place.
- Keep refresh behavior consistent across all status-bearing views.
- Preserve existing navigation context such as filter, list page, file page, and selected file.
- Stay within Telegram's 64-byte callback-data limit.

## Non-Goals

- No automatic polling or timed refresh.
- No refresh button on non-status screens such as category selection or remove confirmation.
- No new navigation behavior when a refreshed item disappears.

## Affected Views

The feature applies to these views:

- Torrent list views for all supported filters.
- Torrent detail view.
- Torrent file list view.
- File-priority selection view.

The feature does not apply to these views:

- Category selection.
- Remove confirmation.

## Callback Design

Use explicit refresh callback families per view type:

- `rf:list:<filterChar>:<page>`
- `rf:td:<filterChar>:<page>:<hash>`
- `rf:files:<hash>:<filePage>:<filterChar>:<listPage>`
- `rf:prio:<hash>:<fileIndex>:<filePage>:<filterChar>:<listPage>`

This keeps each refresh action self-describing and aligned with the current callback architecture, where view types already map to distinct prefixes and handlers.

## Rendering Behavior

### Torrent Lists

When the user presses Refresh on a torrent list, the bot re-fetches torrents for the same filter, re-applies client-side pagination, and edits the current message with the latest text and keyboard for the same list page.

If the list content changes enough to reduce the total page count, the renderer clamps the page to the last available page, consistent with existing file-list behavior.

### Torrent Detail

When the user presses Refresh on a torrent detail view, the bot re-fetches torrent state by hash and re-renders the same detail view with updated state, speeds, progress, and action keyboard.

If the torrent no longer exists, the bot replaces the message with:

`This item is no longer available.`

The replacement message has no inline keyboard.

### File List

When the user presses Refresh on a file list, the bot re-fetches the torrent files for the same torrent hash and re-renders the same file page, preserving the parent list context encoded in the callback.

If the torrent or its file context can no longer be rendered, the bot replaces the message with:

`This item is no longer available.`

The replacement message has no inline keyboard.

### File Priority

When the user presses Refresh on a file-priority view, the bot re-fetches the current file list for the torrent, derives the current priority for the same file index, and re-renders the same priority selector.

If the torrent or file no longer exists, the bot replaces the message with:

`This item is no longer available.`

The replacement message has no inline keyboard.

## Keyboard Layout

Refresh placement should be consistent and visually separate from mutation and navigation actions.

- Torrent lists: keep pagination first, add a dedicated Refresh row before torrent selection rows.
- Torrent detail: add a dedicated Refresh row above Back to list.
- File list: add a dedicated Refresh row above Back.
- File priority: add a dedicated Refresh row above Back to files.

This keeps Refresh clearly associated with the current view and avoids overloading existing navigation rows.

## Handler Design

Add refresh routing in the callback dispatcher using the new `rf:` prefixes. Each route delegates to a dedicated refresh handler for its view type.

Preferred implementation shape:

- Reuse existing render helpers where possible.
- Add small parsing helpers for new callback formats.
- Add a shared helper for terminal missing-item rendering so all refresh handlers produce the same fallback text and no keyboard.

The refresh handlers should answer the callback with an empty string on success so Telegram dismisses the spinner without extra noise.

## Error Handling

Normal API failures should continue to use callback answers such as `Error: <message>` and should not replace the existing message contents.

Missing underlying view state during refresh is handled differently from ordinary API failures:

- If the refreshed torrent or file context is gone, replace the message with `This item is no longer available.`
- Remove the inline keyboard from that message.

This behavior is specific to refresh and avoids unexpected navigation into a different screen.

## Testing

Add or update tests in these areas:

- Formatter tests verifying Refresh buttons exist on all status-bearing keyboards and do not appear on non-status keyboards.
- Formatter tests verifying all new refresh callback payloads stay within Telegram's 64-byte limit under worst-case existing assumptions.
- Callback tests verifying each refresh prefix re-renders the correct current view with updated data.
- Callback tests verifying missing torrent or file cases replace the message with `This item is no longer available.` and no keyboard.

## Risks And Constraints

- Callback data length is the main protocol constraint; file-priority refresh is the tightest case and must be validated by tests.
- Refresh should not duplicate rendering logic unnecessarily; excessive duplication would make future view changes error-prone.
- qBittorrent state can change between list and detail refreshes, so handlers must treat missing items as expected runtime behavior rather than exceptional logic.

## Recommended Implementation Direction

Implement explicit per-view refresh callbacks and handlers, reuse the existing render helpers for list and files views, and add a shared terminal-state helper for missing items. This is the smallest change that matches the current codebase structure while keeping refresh semantics explicit and testable.
