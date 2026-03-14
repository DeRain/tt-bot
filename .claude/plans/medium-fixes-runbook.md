# Loop Runbook: Fix Remaining MEDIUM Issues

## Pattern: Sequential with auto-commit
## Model: Sonnet (all fixes)
## Mode: Safe (full quality gates)
## Stop Condition: All 6 MEDIUMs resolved AND `make gate-all` passes

## Fix Sequence

| # | Issue | File(s) | Fix |
|---|-------|---------|-----|
| 1 | Silent Send error discard | `internal/bot/handler.go` | Add `log.Printf` at all `_ = err` sites |
| 2 | Token leak via error messages | `internal/bot/handler.go` | Sanitize URL in error before returning |
| 3 | Re-auth retry duplication | `internal/qbt/http.go` | Refactor AddTorrentFile to use doWithAuth pattern |
| 4 | UTF-8 callback truncation | `internal/formatter/format.go` | Truncate at rune boundary, not byte |
| 5 | qbt HTTPClient no timeout | `internal/qbt/http.go` | Add 30s timeout to HTTPClient |
| 6 | Sequential notifier blocking | `internal/poller/poller.go` | Dispatch notifications concurrently with WaitGroup |

## Gate Protocol

1. Sonnet agent implements fix
2. `go build ./...` — must pass
3. `go test ./... -short` — must pass
4. Auto-commit on pass

## Current State

- [ ] M-1: Silent Send error discard
- [ ] M-2: Token leak in errors
- [ ] M-3: Re-auth retry duplication
- [ ] M-4: UTF-8 callback truncation
- [ ] M-5: qbt client no timeout
- [ ] M-6: Sequential notifier
