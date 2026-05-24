# Plan: Search Torrent Download

## Waves

### Wave 1: Specification (No Dependencies)

**Task 1**: Write `docs/features/search-torrent-download/spec.md`
- Requirements: REQ-1 through REQ-8
- Acceptance criteria: AC-1.1 through AC-8.1
- Verification: README references updated

### Wave 2: Download Safety Functions — TDD (Depends on Wave 1)

**Task 2**: Write `internal/bot/download_test.go` (TDD red phase)
- `TestIsPublicHostname_RejectsLoopback` — 127.0.0.1, 127.0.0.2, ::1
- `TestIsPublicHostname_RejectsPrivate` — 10.x.x.x, 172.16-31.x.x, 192.168.x.x
- `TestIsPublicHostname_RejectsLinkLocal` — 169.254.x.x
- `TestIsPublicHostname_AcceptsPublic` — 8.8.8.8, 1.1.1.1
- `TestIsPublicHostname_InvalidHostname` — empty, spaces
- `TestDownloadSearchTorrent_Success` — happy path, verify returned bytes
- `TestDownloadSearchTorrent_HTMLRejected` — text/html Content-Type rejected
- `TestDownloadSearchTorrent_TooLarge` — body >10 MB rejected
- `TestDownloadSearchTorrent_Non200` — HTTP 404 rejected
- `TestDownloadSearchTorrent_Timeout` — aggressive timeout rejected
- `TestDownloadSearchTorrent_TooManyRedirects` — chain >5 hops rejected
- `TestDownloadSearchTorrent_HTTPSDowngrade` — HTTPS to HTTP downgrade rejected
- `TestNewDownloadClient_Timeout` — 15s timeout asserted
- `TestNewDownloadClient_RedirectPolicy` — redirect limit + downgrade asserted
- Verification: `go test ./internal/bot/... -run TestIsPublicHostname -short -v` compiles and fails (red)

**Task 3**: Implement `internal/bot/download.go`
- `isPublicHostname(host string) error` — SSRF guard
- `newDownloadClient() *http.Client` — 15s timeout, CheckRedirect (5 hops, no HTTPS→HTTP)
- `downloadSearchTorrent(ctx, client, url) ([]byte, error)` — full download pipeline
- `downloadSearchTorrentFn` — package-level var for test injection
- Verification: `go test ./internal/bot/... -run 'TestIsPublicHostname|TestDownloadSearchTorrent|TestNewDownloadClient' -short -v` passes

### Wave 3: Callback Integration — TDD (Depends on Wave 2)

**Task 4**: Update `internal/bot/callback_test.go` (TDD red phase)
- `TestCallback_SearchConfirmCallback_WithMagnet` — existing, must still pass (regression guard)
- `TestCallback_SearchConfirmCallback_NoMagnet_ShowsError` — existing, ftp:// rejection
- `TestCallback_SearchConfirmCallback_MagnetUnchanged` — existing, magnet regression
- `TestCallback_SearchConfirmCallback_HTTPDownload_Success` — inject downloadSearchTorrentFn, verify FileData stored
- `TestCallback_SearchConfirmCallback_HTTPDownload_Failure` — inject failing fn, verify error message
- `TestCallback_SearchConfirmCallback_HTTPDownload_SSRFReject` — 10.0.0.1 URL, verify SSRF error
- Verification: `go test ./internal/bot/... -run TestCallback_SearchConfirmCallback -short -v` compiles and some fail (red)

**Task 5**: Modify `internal/bot/callback.go`
- `handleSearchConfirmCallback` — branch on `result.FileURL`:
  - `magnet:?` → existing flow (unchanged)
  - `http://` or `https://` → call `downloadSearchTorrentFn`, store `PendingTorrent{FileData, FileName}`, show category keyboard
  - otherwise → "no valid download link" error
- Verification: `go test ./internal/bot/... -run TestCallback_SearchConfirmCallback -short -v` passes

### Wave 4: Quality Gate (Depends on Wave 3)

**Task 6**: Create design.md, plan.md, traceability.md
- These three files
- Verification: files exist with correct format

**Task 7**: Run `make gate-all` (build, lint, test, coverage, arch-check, mutation-test-pr)
- Ensure `go build ./...` passes
- Ensure `golangci-lint run` passes (gocyclo, errcheck, etc.)
- Ensure `go test ./... -short -cover` passes with >80% bot package coverage
- Ensure arch-go rules pass
- Verification: `make gate-all` exits 0

**Task 8**: Run `make test-integration`
- Integration tests against real qBittorrent in Docker
- E2E tests for search result selection flow
- Verification: `make test-integration` exits 0

### Wave 5: Verification Documentation (Depends on Wave 4)

**Task 9**: Write `docs/features/search-torrent-download/verification.md`
- Evidence from `make gate-all` and `make test-integration`
- Assertion tracker: each AC mapped to test name and pass/fail
- Coverage report for `internal/bot/download.go`
- Verification: All ACs marked PASS

## Dependency Graph

```
TASK-1 (spec)
   │
   ▼
TASK-2 (download_test.go) ──► TASK-3 (download.go)
                                        │
                                        ▼
                              TASK-4 (callback_test.go) ──► TASK-5 (callback.go)
                                                                  │
                                                                  ├── TASK-6 (design docs)
                                                                  │
                                                                  ▼
                                                          TASK-7 (gate-all)
                                                                  │
                                                                  ▼
                                                          TASK-8 (test-integration)
                                                                  │
                                                                  ▼
                                                          TASK-9 (verification.md)
```

## Commit Strategy

| Commit | Files | Message |
|--------|-------|---------|
| 1 | `docs/features/search-torrent-download/spec.md` | `feat(search-torrent-download): TASK-1 add spec.md` |
| 2 | `internal/bot/download_test.go` | `feat(search-torrent-download): TASK-2 add download safety tests (TDD red)` |
| 3 | `internal/bot/download.go` | `feat(search-torrent-download): TASK-3 implement download safety functions` |
| 4 | `internal/bot/callback_test.go` | `feat(search-torrent-download): TASK-4 add callback download tests (TDD red)` |
| 5 | `internal/bot/callback.go` | `feat(search-torrent-download): TASK-5 add HTTP download path to search confirm callback` |
| 6 | `docs/features/search-torrent-download/design.md`, `plan.md`, `traceability.md` | `feat(search-torrent-download): TASK-6 add design docs` |
| 7 | `docs/features/search-torrent-download/verification.md` | `feat(search-torrent-download): TASK-9 add verification.md` |
