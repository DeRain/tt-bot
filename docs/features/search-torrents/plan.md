# Plan: Search Torrents

## Waves

### Wave 1: Foundation (No Dependencies)
**Task 1**: Add search types + client interface methods
- `internal/qbt/types.go`: Add `SearchResult` struct (7 JSON-tagged fields)
- `internal/qbt/client.go`: Add 5 methods to `Client` interface
- `internal/bot/handler_test.go`: Add stub methods to `mockQBTClient`
- Verification: `make build` passes

### Wave 2: Implementation (Depends on Wave 1)
**Task 2**: Implement search HTTP methods + unit tests
- `internal/qbt/http.go`: Implement `StartSearch`, `SearchStatus`, `SearchResults`, `StopSearch`, `DeleteSearch`
- `internal/qbt/http_test.go`: Unit tests with httptest.Server
- Verification: `make test ./internal/qbt/...` passes

**Task 3**: Add search result formatter functions + tests
- `internal/formatter/format.go`: Add `FormatSearchResults`, `SearchResultKeyboard`, `SearchPaginationKeyboard`, `FormatSearchStatus`
- `internal/formatter/format_test.go`: Tests for message size, keyboard callback data
- Verification: `make test ./internal/formatter/...` passes

### Wave 3: Bot Integration (Depends on Wave 2)
**Task 4**: Bot search handler + callbacks + tests
- `internal/bot/handler.go`:
  - Add `SearchState`, `SearchPrompt` structs
  - Add `searches`, `searchPrompts` maps with mutexes
  - Add `handleSearchCommand` (supports `/search <query>` and conversational mode)
  - Add `pollSearchResults` goroutine
  - Add `handleSearchPromptReply` for conversational replies
  - Add search cleanup to `evictExpired`
- `internal/bot/callback.go`:
  - Add `sr:` (select result → confirmation)
  - Add `sp:` (pagination)
  - Add `sx:` (cancel)
  - Add `ss:` (sort)
  - Add `sc:` (confirm add → category picker)
  - Add `sb:` (back to results)
- `internal/bot/commands.go`: Register `/search` command
- `internal/bot/handler_test.go`, `internal/bot/callback_test.go`: Update mock, add tests
- Verification: `make test ./internal/bot/...` passes

### Wave 4: Wiring (Depends on Wave 3)
**Task 5**: Verify build and command registration
- Verify `cmd/bot/main.go` needs no changes (qbt.Client passed as-is)
- Ensure `/search` in help output
- Run `make build && make lint && make test`
- Verification: `make gate-all` passes

### Wave 5: Tests (Depends on Wave 4)
**Task 6**: Integration tests
- `internal/qbt/http_integration_test.go`: Tests against real qBittorrent search API
- `docker-compose.test.yml`: Evaluate Jackett container addition
- Verification: `make test-integration` passes

**Task 7**: E2E tests
- `internal/bot/e2e_test.go`: Full `/search → results → select → confirm → category → add` flow
- Verification: `make test-integration` passes

## Commit Strategy

| Commit | Files | Message |
|--------|-------|---------|
| 1 | `qbt/types.go`, `qbt/client.go`, `bot/handler_test.go` | `feat(search): add SearchResult type and Client interface methods` |
| 2 | `qbt/http.go`, `qbt/http_test.go` | `feat(search): implement search HTTP methods with unit tests` |
| 3 | `formatter/format.go`, `formatter/format_test.go` | `feat(search): add search result formatting and keyboards` |
| 4 | `bot/handler.go`, `bot/callback.go`, `bot/commands.go`, `bot/*_test.go` | `feat(search): add /search command, handlers, callbacks, and tests` |
| 5 | `qbt/http_integration_test.go`, `docker-compose.test.yml` | `feat(search): add integration and E2E tests for search` |

## Updated Requirements (User Input)

1. **Conversational search**: `/search` alone → "What to search for?" → user replies. If another command is sent, prompt is abandoned.
2. **Sorting**: Results sortable by size, seeders, added date. Include added date in display.
3. **Message size**: Use existing `MaxMessageLength=4096` pattern from formatter.
4. **Error states**: Empty query, no results, timeout, Jackett not configured, non-magnet results.
5. **Two-stage selection**: Tap result → confirmation ("Add this torrent?" + "Back to list") → category picker.
6. **Search scope**: All plugins by default. Filter by plugin/provider and category.
7. **E2E**: Mock Jackett response for unit tests. Full E2E with Jackett container + public tracker in docker-compose.
