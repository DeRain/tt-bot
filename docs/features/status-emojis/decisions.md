---
title: "Human-Readable Statuses with Emojis — Decisions"
feature_id: "status-emojis"
last_updated: 2026-03-15
---

# Human-Readable Statuses with Emojis — Decisions

## DEC-1: map[string]string over switch statement

**Decision**: Use a package-level `var stateLabels = map[string]string{...}` for state-to-label lookup rather than a `switch` statement inside `FormatState`.

**Rationale**:
- A map is more maintainable for 19+ entries; adding a new state is a single line with no risk of misplacing a `case` or forgetting a `break`.
- Map lookup is O(1) and idiomatic Go for static string tables.
- The map can be range-iterated in tests to verify completeness without duplicating the entry list.
- A `switch` would require the same number of lines but with more syntactic noise and no iteration capability.

**Alternatives considered**:
- `switch` statement — rejected; harder to test exhaustiveness and more verbose for a purely data-driven mapping.
- External config file (JSON/YAML) — rejected; over-engineering for a compile-time constant with no runtime override requirement.

**Consequence**: The map is unexported (`stateLabels`) to keep it an implementation detail. `FormatState` is exported to allow direct unit testing and reuse by future format functions.

---

## DEC-2: Emoji selection rationale

**Decision**: Emojis were chosen to reflect the semantic meaning of each state category rather than for strict visual consistency.

| Category         | Emoji(s) used           | Rationale                                                        |
|------------------|-------------------------|------------------------------------------------------------------|
| Error / missing  | ❌, ⚠️                   | Standard error/warning symbols widely understood                 |
| Seeding          | 🌱                       | Growth metaphor — torrent is giving back to the swarm            |
| Paused           | ⏸️                       | Universal media-player pause symbol                              |
| Queued           | 🕐                       | Clock conveys waiting                                            |
| Checking         | 🔍                       | Magnifying glass conveys inspection/verification                 |
| Force seeding    | ⏫                       | Upward fast-forward conveys forced/bypassed priority             |
| Allocating       | 💾                       | Disk icon reflects disk space being reserved                     |
| Downloading      | ⬇️                       | Downward arrow is the universal download symbol                  |
| Fetch metadata   | 🔎                       | Slightly different magnifier variant distinguishes from checking |
| Force DL         | ⏬                       | Downward fast-forward mirrors force seeding                      |
| Moving           | 📦                       | Box conveys file relocation                                      |
| Unknown          | ❓                       | Question mark is the fallback for anything unrecognised          |

**Consequence**: Emoji rendering is client-dependent. Some Telegram clients on older OS versions may show placeholder boxes for newer emoji code points. This is a cosmetic risk accepted in the spec (Non-Goals: emoji rendering is out of scope to fix).

---

## DEC-3: Fallback strategy for unknown states

**Decision**: Return `"❓ " + state` for any state string not present in `stateLabels`, including the empty string.

**Rationale**:
- The function must never panic and must never return an empty string (AC-4.1, AC-4.2).
- Surfacing the raw state alongside the ❓ prefix gives advanced users enough context to understand or report the unknown value.
- Returning an error would require every call site to handle it, adding complexity for a non-critical display concern.
- Returning a generic placeholder like `"❓ Unknown"` would obscure the actual raw value, making debugging harder.

**Consequence**: For an empty-string input the fallback is `"❓ "` — non-empty and non-panicking, satisfying AC-4.1, though cosmetically it trails a space. This is acceptable given that an empty state from the qBittorrent API is not a normal condition.

---

## DEC-4: Scope confined to internal/formatter

**Decision**: All changes are confined to the `internal/formatter` package. No new package is introduced.

**Rationale**:
- The mapping is formatter-specific — it exists solely to produce user-facing strings.
- A dedicated `statemap` or `states` package would be premature abstraction for three items (map, function, tests) that have no other consumers today.
- The formatter package already owns all display logic; placing the state map there preserves high cohesion.

**Alternatives considered**:
- New `internal/states` package — rejected; no other package needs the mapping, so the abstraction has no current benefit and adds indirection.

---

## DEC-5: FormatState is exported

**Decision**: `FormatState` is an exported function (capitalised) rather than an unexported helper.

**Rationale**:
- Exported functions can be unit-tested directly from `_test.go` files without relying on indirect assertion through the higher-level format functions.
- Future format functions (e.g., a torrent status summary view) can call `FormatState` without copy-pasting the lookup logic.
- The `stateLabels` map remains unexported — callers interact only with the function, not the raw map.

---

## Deferred Work

| Item | Rationale for Deferral |
|------|------------------------|
| Localisation / i18n of labels | Explicitly listed as a Non-Goal in spec.md. The bot serves a single operator; multi-language support is not needed. |
| Filtering or sorting by state | Out of scope per spec.md. Would require changes to `FormatTorrentList` beyond label substitution. |
| Emoji override via config | No operator request; adds complexity with no current value. Revisit if emoji rendering issues are reported. |
| Integration test for state display | `FormatState` and its call sites are pure functions with no I/O. Unit tests fully cover the AC set. Integration tests would add no additional confidence for this specific change. |
