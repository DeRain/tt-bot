# Feature: Search Torrents

## Overview
Add a `/search` command that lets users search for torrents across configured qBittorrent search plugins (Jackett-backed indexers like RuTracker, NNM-Club, RuTor). Results are shown with inline keyboard pagination. Tapping a result adds it to the existing `PendingTorrent` → category picker → `AddMagnet` flow.

## Requirements

### REQ-1: Search Command
Users can initiate search via `/search <query>` or conversational `/search` → prompt → reply.

### REQ-2: Search Results Display
Results show: truncated title, size, seeders, leechers, added date, tracker source. Paginated at 8 results per page. Message stays under Telegram 4096-char limit.

### REQ-3: Search Result Sorting
Results can be sorted by: size, seeders count, added date. Default sort: by seeders (descending).

### REQ-4: Search Scope
Search all configured plugins by default. Allow filtering by specific plugin/provider and by category.

### REQ-5: Result Selection Flow
Tapping a result shows: "Add this torrent?" confirmation + "Back to list" button. Confirmation proceeds to category picker.

### REQ-6: Graceful State Handling
If user starts conversational search but sends another command before replying, the search prompt is abandoned without error.

### REQ-7: Search State Cleanup
Search jobs in qBittorrent are stopped and deleted after results are fetched. In-memory search state has a 10-minute TTL with background cleanup.

### REQ-8: Error Handling
Handle: empty query, no results, search timeout, Jackett plugin not configured, non-magnet results.

## Acceptance Criteria

- AC-1.1: `/search ubuntu` starts search and shows "Searching..." message
- AC-1.2: `/search` alone prompts "What to search for?"
- AC-1.3: Replying to prompt with query starts search
- AC-1.4: Sending another command while in search prompt abandons prompt
- AC-2.1: Results displayed with title, size, seeders, leechers, date, source
- AC-2.2: Pagination works with Prev/Next buttons
- AC-2.3: Message never exceeds 4096 chars
- AC-3.1: Sort buttons present (size, seeders, date)
- AC-3.2: Default sort is seeders descending
- AC-4.1: Search all plugins by default
- AC-4.2: Can filter by plugin and category
- AC-5.1: Tap result → confirmation message with "Add" and "Back" buttons
- AC-5.2: Tap "Add" → category picker → add torrent
- AC-5.3: Tap "Back" → return to search results
- AC-6.1: Search prompt abandoned on other command with no error
- AC-7.1: qBittorrent search jobs cleaned up after use
- AC-7.2: In-memory search state evicted after 10 minutes
- AC-8.1: Empty query shows usage hint
- AC-8.2: No results shows "No torrents found"
- AC-8.3: Timeout shows "Search timed out"
- AC-8.4: Jackett not configured shows "Search unavailable"
