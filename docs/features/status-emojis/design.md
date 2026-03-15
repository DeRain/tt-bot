---
title: "Human-Readable Statuses with Emojis — Design"
feature_id: status-emojis
status: draft
depends_on_spec: "docs/features/status-emojis/spec.md"
last_updated: 2026-03-15
---

# Human-Readable Statuses with Emojis — Design

## Overview

The implementation is a pure, stateless addition to `internal/formatter`. A package-level map holds the 19 known state-to-label entries. A single exported function `FormatState` performs the lookup with a graceful fallback. The two existing format functions are updated to call `FormatState` in place of the raw `t.State` interpolation.

## Architecture

All changes are confined to `internal/formatter/formatter.go` (or a new sibling file in that package if the file exceeds the 800-line threshold). No new packages, interfaces, or external dependencies are introduced.

```
internal/formatter/
  formatter.go          ← existing; gains stateLabels map + FormatState func;
                          FormatTorrentList and FormatTorrentDetail updated
```

## Data Flow

1. qBittorrent API returns a `qbt.Torrent` with a raw `State` string (e.g. `"stalledUP"`).
2. `FormatTorrentList` or `FormatTorrentDetail` is called with the torrent slice/item.
3. Each call site replaces `t.State` with `FormatState(t.State)`.
4. `FormatState` looks up the state in `stateLabels`. On hit, returns the label string (emoji + text). On miss, returns `"❓ " + state`.
5. The composed message string is returned to the caller unchanged in all other respects.

## Interfaces

```go
// FormatState maps a raw qBittorrent state string to a human-readable label
// with an emoji prefix. If the state is not recognised, it returns "❓ <state>".
// It never returns an empty string and never panics.
func FormatState(state string) string
```

`FormatState` is exported so it can be unit-tested directly and reused by future format functions without duplication.

The `stateLabels` map is unexported (package-level `var`):

```go
var stateLabels = map[string]string{
    "error":               "❌ Error",
    "missingFiles":        "⚠️ Missing Files",
    "uploading":           "🌱 Seeding",
    "pausedUP":            "⏸️ Paused (Seeding)",
    "queuedUP":            "🕐 Queued (Seeding)",
    "stalledUP":           "🌱 Seeding (stalled)",
    "checkingUP":          "🔍 Checking",
    "forcedUP":            "⏫ Force Seeding",
    "allocating":          "💾 Allocating",
    "downloading":         "⬇️ Downloading",
    "metaDL":              "🔎 Fetching Metadata",
    "pausedDL":            "⏸️ Paused (Downloading)",
    "queuedDL":            "🕐 Queued (Downloading)",
    "stalledDL":           "⬇️ Downloading (stalled)",
    "checkingDL":          "🔍 Checking",
    "forcedDL":            "⏬ Force Downloading",
    "checkingResumeData":  "🔍 Checking",
    "moving":              "📦 Moving",
    "unknown":             "❓ Unknown",
}
```

## Data/Storage Impact

None. The mapping is a compile-time constant. No database, file, or network access is involved.

## Error Handling

`FormatState` has no error return. Any unrecognised state — including an empty string — produces `"❓ " + state` (or just `"❓ "` for an empty input, which is still non-empty and non-panicking). This satisfies AC-4.1 and AC-4.2.

## Security Considerations

The state string originates from the qBittorrent API response, which is an internal trusted service. The string is used only as a map key and concatenated into a Telegram message. No injection risk exists; Telegram messages are plain text / MarkdownV2 escaped by the existing formatter.

## Performance Considerations

Map lookup is O(1). The map is initialised once at package load. There is no measurable performance impact.

## Tradeoffs

| Decision | Alternative | Rationale |
|----------|-------------|-----------|
| Single exported `FormatState` function | Inline map lookup at each call site | Avoids duplication; enables direct unit testing; future call sites reuse without copy-paste |
| Package-level unexported map | `switch` statement | Map is more maintainable for 19+ entries; constant-time lookup; easy to extend |
| Fallback `"❓ " + state` | Return empty string or error | Graceful degradation; user sees something meaningful; no caller error handling required |
| Keep all changes in `formatter` package | New `statemap` package | Feature is small and formatter-specific; a separate package would be over-engineering |

## Risks

- Emoji bytes increase message length. The longest label (`⏸️ Paused (Downloading)`) is ~24 chars, well within Telegram's 4096-char limit. No pagination impact.

## Design Items

- **DES-1**: Add `FormatState(state string) string` function to `internal/formatter`.
  - Satisfies: REQ-1, REQ-4
  - Covers: AC-1.1, AC-1.2, AC-1.3, AC-4.1, AC-4.2

- **DES-2**: Add package-level `stateLabels map[string]string` with all 19 documented state mappings.
  - Satisfies: REQ-1, REQ-2
  - Covers: AC-1.3, AC-2.1, AC-2.2

- **DES-3**: Update `FormatTorrentList` and `FormatTorrentDetail` to call `FormatState(t.State)` in place of `t.State`.
  - Satisfies: REQ-3
  - Covers: AC-3.1, AC-3.2

## Quality Gates

### Gate 2: Design Gate

This design passes when:
- [x] Every REQ-* from spec.md is addressed by at least one DES-*
- [x] Every AC-* from spec.md is covered by at least one DES-*
- [x] Risks and tradeoffs are documented
- [x] No DES-* exists without a linked REQ-*

**Harness check command:**
```bash
spec_reqs=$(grep -oP 'REQ-\d+' docs/features/status-emojis/spec.md | sort -u)
design_reqs=$(grep -oP 'REQ-\d+' docs/features/status-emojis/design.md | sort -u)
comm -23 <(echo "$spec_reqs") <(echo "$design_reqs")  # should be empty
```

## Requirement Mapping

| Design Item | Satisfies | Covers |
|-------------|-----------|--------|
| DES-1 | REQ-1, REQ-4 | AC-1.1, AC-1.2, AC-1.3, AC-4.1, AC-4.2 |
| DES-2 | REQ-1, REQ-2 | AC-1.3, AC-2.1, AC-2.2 |
| DES-3 | REQ-3 | AC-3.1, AC-3.2 |
