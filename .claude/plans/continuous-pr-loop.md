# Continuous PR Loop — Safe Mode

## Date: 2026-03-15

## Loop Pattern: continuous-pr
## Mode: safe (strict quality gates)

## Feature Queue

| # | Feature ID | Title | Branch | Status |
|---|-----------|-------|--------|--------|
| 1 | uploading-list | List uploading/seeding torrents | feat/uploading-list | pending |
| 2 | torrent-files | View/manage torrent file priorities | feat/torrent-files | pending |
| 3 | torrent-detail-extra | Uploaded amount & ratio in detail | feat/torrent-detail-extra | pending |
| 4 | torrent-remove | Stop/remove torrent actions | feat/torrent-remove | pending |
| 5 | status-emojis | Human-readable statuses with emojis | feat/status-emojis | pending |

## Per-Feature Cycle

```
1. Create branch: feat/<feature-id> from main
2. Write docs (spec.md, design.md, plan.md) → Gates 1-3
3. Get user validation on docs
4. Implement via TDD → Gate 4
5. Run make gate-all && make test-integration → Gate 4
6. Update traceability + verification docs → Gate 5
7. Commit, push, create PR
8. Merge after approval → back to main
9. Next feature
```

## Safety Gates

- [ ] `make gate-all` passes before every commit
- [ ] `make test-integration` passes before PR
- [ ] No PR without user doc validation
- [ ] No merge without review

## Stop Conditions

- All 5 features completed and merged
- User requests stop
- Build failure after 3 fix attempts

## Model Routing

| Phase | Model |
|-------|-------|
| Doc writing (spec/design/plan) | Sonnet (parallel agents) |
| Implementation | Sonnet agents |
| Gate checks | Haiku |
| Review | Opus (main) |

## Mandatory Agent Prompt Requirements

Every Sonnet implementation agent prompt MUST include:
1. `make gate-all` as a required passing step
2. `make test-integration` as the FINAL step before reporting success
3. Agent MUST NOT report success unless both commands pass
4. If `make test-integration` fails, agent must fix and re-run (max 3 retries)
