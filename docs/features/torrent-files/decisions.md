---
title: "Torrent File Management — Decisions"
feature_id: "torrent-files"
last_updated: 2026-03-15
---

# Torrent File Management — Decisions

This document records assumptions made during specification and design, the rationale behind key design choices, and work that was explicitly deferred.

---

## Assumptions

**A-1: qBittorrent API stability**
The `GET /api/v2/torrents/files` and `POST /api/v2/torrents/filePrio` endpoints are present and stable in the qBittorrent version deployed in this project. The integration test (TEST-7, TEST-8) will catch any endpoint renames (as happened with `/pause` → `/stop` in qBittorrent v5) before the feature ships.

**A-2: Hash format is always SHA-1 (40 hex chars)**
All torrent hashes used as callback data are 40-character lowercase hex strings. This is relied upon for the callback byte-length calculations in the design. If qBittorrent ever returns shorter or longer hashes (e.g., SHA-256 in a future API version), the callback encoding must be revisited.

**A-3: File index is stable within a session**
The `index` field returned by `GET /api/v2/torrents/files` is stable and zero-based. The design passes this index directly in callback data. If qBittorrent reorders files between API calls, a priority-set action could target the wrong file. This risk is accepted at personal-bot scale; no locking or re-validation is implemented.

**A-4: File count per torrent is bounded in practice**
The design does not impose a hard upper limit on the number of files in a torrent. Pagination at 5 per page keeps any individual page manageable. File index fits in 5 digits (max 99999) for callback encoding; torrents with more than 99999 files are out of scope.

**A-5: No persistent state needed**
The feature is entirely stateless: every callback carries all context needed to re-render the correct view. This is consistent with the rest of the bot's design.

---

## Design Choices

### DC-1: Separate `fs:` and `fp:` prefixes for priority selector vs. priority set

**Decision**: Use `fs:` ("file select") to open the priority keyboard, and `fp:` ("file priority") to apply the priority. These are two distinct callback prefixes rather than a single prefix with an action sub-field.

**Rationale**: Keeps the callback router switch-statement simple and deterministic. Parsing a sub-action field adds code complexity with no byte savings at this field count.

**Alternative rejected**: A single prefix `fp:` with a sub-action byte (e.g., `fp:show:...` vs. `fp:set:...`). Rejected because it requires additional parsing logic and the sub-action eats into the 64-byte budget.

**Reference**: design.md Tradeoffs → "Callback data compactness"

---

### DC-2: Display only the last path component of the file name

**Decision**: Show only the last path component after the final `/` in the file name returned by qBittorrent. Truncate to 40 UTF-8 characters with a trailing `…` if longer.

**Rationale**: qBittorrent returns the full relative path within the torrent (e.g., `Season 1/Episode 01.mkv`). Showing full paths in Telegram button labels wastes space, increases message length, and reduces readability for deeply nested files.

**Alternative rejected**: Show the full relative path. Rejected because it easily exceeds Telegram's inline button label character limits and looks noisy.

**Reference**: design.md Tradeoffs → "File name display"

---

### DC-3: "Files" button placed as a second row below Pause/Start

**Decision**: The "Files" button is added as a new row of one button, positioned between the Pause/Start row and the Back row on the torrent detail keyboard.

**Rationale**: The existing Pause/Start buttons are control actions; Files is a navigation/inspection action. Placing them on separate rows maintains a clear visual grouping without requiring a complete keyboard redesign.

**Alternative rejected**: Add "Files" as a third button on the same row as Pause/Start. Rejected because three wide-label buttons on a single row can be cramped on narrow phone screens.

**Reference**: design.md Tradeoffs → "Files button placement"

---

### DC-4: Two-tap flow for priority change (file → priority keyboard → confirm)

**Decision**: Changing a file's priority requires two taps: one tap selects the file (opens a dedicated priority keyboard), and a second tap applies the chosen priority.

**Rationale**: Fitting four priority options as direct action buttons on each file row in the file list would produce up to 5 × 5 = 25 buttons per page, making the keyboard unreadable. The two-tap flow keeps the file list clean while still allowing quick priority changes. The priority keyboard marks the current priority with a checkmark so the user can confirm their intended change before tapping.

**Alternative rejected**: Inline priority cycling (tapping a file cycles through priorities in order). Rejected because it is non-discoverable and provides no confirmation of the change being made.

**Reference**: design.md Tradeoffs → "Priority selection UX"

---

### DC-5: Immediate priority application without undo

**Decision**: Priority changes are applied immediately when the user taps a priority option. There is no confirmation step or undo action.

**Rationale**: Consistent with the existing torrent-control feature (Pause/Start are also immediate). The priority keyboard already shows the current priority with a checkmark, giving the user enough information to make an informed choice before tapping. Undo would require storing per-user state, which conflicts with the stateless design.

**Reference**: spec.md Scope — "priority changes are applied immediately without an undo step"

---

### DC-6: `bk:fl:` back-from-file-list encodes filterChar and listPage

**Decision**: The back callback from the file list (`bk:fl:<filterChar>:<listPage>:<hash>`) encodes both the filter character and the list page so the user lands on the correct page of the torrent list they came from.

**Rationale**: Without this context, tapping Back from the file list would return the user to an arbitrary page of the torrent list, breaking navigational consistency. The byte budget (54 bytes max) comfortably accommodates these fields.

**Reference**: design.md Data Flow → "Back navigation"

---

### DC-7: Unknown priority integer displayed as "Mixed"

**Decision**: If qBittorrent returns a priority value that is not one of 0, 1, 6, or 7, `PriorityLabel` returns `"Mixed"`. This value is not offered as a setable option in the priority keyboard.

**Rationale**: qBittorrent can return priority `4` as a sentinel meaning "mixed priorities across pieces of this file." It is an informational state, not a user-settable value. Displaying it as "Mixed" is descriptive and avoids confusing the user with a raw integer. The four standard priority options remain available for any file regardless of its current priority.

**Reference**: design.md Error Handling — "Unknown priority integer from API"

---

## Deferred Work

**DEF-1: Bulk priority changes**
Changing the priority of multiple files in a single action is explicitly out of scope (spec.md Non-Goals). A future feature could add a multi-select mode to the file list keyboard.

**DEF-2: Sorting and filtering the file list**
The file list is presented in the order returned by qBittorrent (typically by file index). Sorting by name, size, or progress, and filtering by priority, are deferred. These would require encoding sort/filter state in callback data, which risks hitting the 64-byte limit.

**DEF-3: Full path display for single-file torrents**
Single-file torrents have only one file, whose name is often the same as the torrent name. Showing the full path in this case is listed as a non-goal in the spec. If the UX proves confusing in practice, a follow-up can detect single-file torrents and suppress the path truncation.

**DEF-4: Caching `ListFiles` responses**
`ListFiles` is called on every file list page render and on every priority change. For personal-bot scale this is fine. If the bot is ever used with very large multi-file torrents (thousands of files) or shared among many users, a short TTL cache keyed by torrent hash would reduce qBittorrent API load. Deferred until there is evidence of a performance problem.

**DEF-5: E2E test for full file list + priority change flow via Telegram API**
TEST-8 covers the qBittorrent API contract. A full E2E test that sends a Telegram message, taps "Files", taps a file, and taps a priority button via `bot/e2e_test.go` is listed in TASK-8 but may require additional test infrastructure. If this proves impractical in Gate 4, it can be deferred to a follow-up with a manual check recorded in verification.md.
