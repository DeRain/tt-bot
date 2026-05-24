# Design: Search Torrents

## Architecture

```
User → Telegram → Bot Handler → qbt.Client → qBittorrent Web API
                                      ↓
                                jackett.py plugin
                                      ↓
                                   Jackett
                                      ↓
                              RuTracker / NNM-Club / RuTor
```

## Key Decisions

### DES-1: Async Search Polling
Search in qBittorrent is async (POST start → GET status until "Stopped" → GET results). The bot launches a goroutine for polling to avoid blocking the Telegram update loop (which is sequential in `main.go`).

### DES-2: Client-Side Pagination
Fetch all results from qBittorrent (limit=100), then paginate in Go using the `formatter` package. This matches the existing torrent list pattern and avoids complex server-side offset tracking.

### DES-3: Two-Stage Selection Flow
1. Search results page → tap result → **confirmation screen** ("Add this torrent?" + Back button)
2. Confirmation → tap "Add" → **category picker** (reuse existing flow)
3. Category picker → tap category → **AddMagnet** (reuse existing flow)

This gives users a chance to reconsider and avoids accidental adds.

### DES-4: Conversational Search with Graceful Abandonment
A new `searchPrompts` map tracks chats waiting for a query reply. When `HandleUpdate` sees a command while the chat is in prompt state, the prompt is silently cleared and the command proceeds normally.

### DES-5: Sorting
Results are sorted client-side after fetching all results. Sort buttons use callback prefixes: `ss:<jobID>:<field>` where field is `seeders`, `size`, or `date`. The current sort is stored in `SearchState`.

### DES-6: Search Scope Filtering
Plugin and category filters are applied at qBittorrent API level via `plugins` and `category` parameters on `/api/v2/search/start`.

## State Management

### SearchState
```go
type SearchState struct {
    ChatID      int64
    MessageID   int
    Query       string
    JobID       int
    Results     []qbt.SearchResult
    Total       int
    SortField   string  // "seeders" | "size" | "date"
    SortAsc     bool
    CreatedAt   time.Time
}
```

### SearchPrompt
```go
type SearchPrompt struct {
    ChatID    int64
    CreatedAt time.Time
}
```

## Callback Prefix Scheme

| Prefix | Format | Purpose |
|--------|--------|---------|
| `sr` | `sr:<jobID>:<idx>` | Select result (global index) |
| `sp` | `sp:<jobID>:<page>` | Navigate search results page |
| `sx` | `sx:<jobID>` | Cancel/close search |
| `ss` | `ss:<jobID>:<field>` | Sort results by field |
| `sc` | `sc:<jobID>:<idx>` | Confirm adding result |
| `sb` | `sb:<jobID>:<page>` | Back to search results from confirmation |

All prefixes are well under the 64-byte callback limit.

## Constants

```go
searchPollInterval   = 1 * time.Second
searchTimeout        = 30 * time.Second
searchResultsLimit   = 100
searchResultsPerPage = 8
searchTTL            = 10 * time.Minute
searchPromptTTL      = 5 * time.Minute
```

## Error Handling

**Critical rule: Jackett/qBittorrent-specific errors NEVER leak to the user.** The `qbt` package absorbs all internal errors and returns them as Go errors. The bot handler maps all errors to generic, user-friendly messages. The user never sees "Jackett", "Torznab", "nova3", Python tracebacks, or HTTP status codes.

| Scenario | Bot Response |
|----------|--------------|
| Empty query | "Usage: `/search <query>`" |
| No results | "No torrents found for '<query>'." |
| Search timeout | "Search timed out. Please try again later." |
| Search start fails (any reason: no plugins, Jackett down, auth error) | "Search unavailable. Please check your search configuration." |
| Non-magnet result | "This result doesn't have a magnet link. Try another result." |
| Generic search error | "Search failed. Please try again later." |
