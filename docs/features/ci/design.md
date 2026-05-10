---
title: "GitHub Actions CI Workflow — Design"
feature_id: "ci"
status: implemented
depends_on_spec: "docs/features/ci/spec.md"
last_updated: 2026-05-10
---

# GitHub Actions CI Workflow — Design

## Overview

Create a single GitHub Actions workflow file (`.github/workflows/ci.yml`) with two jobs: `gate` for fast compile/lint/test/arch checks, and `integration` for Docker-based integration + E2E tests. The integration job uses a qBittorrent service container and Telegram secrets from GitHub.

## Architecture

The workflow is a single file with two dependent jobs:

```
.github/workflows/ci.yml
├── trigger: push (any branch) + pull_request (main)
├── job: gate
│   ├── checkout@v4
│   ├── setup-go@v5 (1.26)
│   ├── install golangci-lint v2.12.2
│   ├── make build
│   ├── make lint
│   ├── make test        ← unit tests with -short -cover
│   └── make arch-check  ← arch-go v2 via go run @latest
└── job: integration (needs: gate)
    ├── services: qbittorrent (linuxserver/qbittorrent:latest)
    ├── checkout@v4
    ├── setup-go@v5 (1.26)
    ├── clean stale lockfile
    ├── wait-for-qbittorrent (curl health check loop)
    └── go test ./... -tags=integration -run "Integration|E2E"
```

## Data Flow

1. Developer pushes to any branch → GitHub triggers the workflow
2. `gate` job runs: checkout → install Go → install linter → build → lint → unit tests → arch check
3. If `gate` passes → `integration` job starts
4. `integration`: qBittorrent service container starts with healthcheck
5. Integration tests connect to qBittorrent at `localhost:8080`, use Telegram API via secrets
6. Both jobs report pass/fail as GitHub status checks on the commit/PR

## Design Items

- **DES-1**: Single workflow file with two dependent jobs to match the existing Makefile gate structure.
- **DES-2**: `gate` job replicates `make gate-all` (build → lint → test → arch-check) as discrete steps for clearer failure attribution.
- **DES-3**: `integration` job uses GitHub Actions `services:` for qBittorrent instead of `docker-compose` to avoid Docker-in-Docker complexity.
- **DES-4**: Integration job env vars override test defaults: `QBITTORRENT_URL=http://localhost:8080` (tests default to `:18080`), `QBITTORRENT_PASSWORD=adminadmin` (tests default to `""`).
- **DES-5**: Telegram secrets referenced via `${{ secrets.TELEGRAM_BOT_TOKEN }}` and `${{ secrets.TELEGRAM_ALLOWED_USERS }}` — never hardcoded.
- **DES-6**: `needs: gate` on integration job ensures fast-fail: if unit tests fail, Docker doesn't even start.

## Environment Variables

| Variable | Source | CI Value | Test Default | Why Overridden |
|---|---|---|---|---|
| `QBITTORRENT_URL` | Hardcoded in workflow | `http://localhost:8080` | `http://localhost:18080` | CI maps 8080:8080 (no port remap) |
| `QBITTORRENT_USERNAME` | Hardcoded | `admin` | `admin` | Same as default — explicit for clarity |
| `QBITTORRENT_PASSWORD` | Hardcoded | `adminadmin` | `""` | Test default is empty; qBittorrent image requires password |
| `TELEGRAM_BOT_TOKEN` | GitHub secret | `${{ secrets.* }}` | N/A (skip if unset) | Must not be hardcoded |
| `TELEGRAM_ALLOWED_USERS` | GitHub secret | `${{ secrets.* }}` | N/A (skip if unset) | Must not be hardcoded |

## Interfaces

No Go interfaces are defined — this is a pure CI configuration feature. The workflow reads from existing Makefile targets and test infrastructure.

## Requirement Mapping

| Requirement | Design Items |
|---|---|
| REQ-1 (run on push/PR) | DES-1 (trigger config) |
| REQ-2 (gate suite) | DES-2 (gate job) |
| REQ-3 (integration tests) | DES-3 (services), DES-4 (env overrides) |
| REQ-4 (fast-fail) | DES-6 (needs: gate) |
| REQ-5 (secrets) | DES-5 (GitHub secrets) |
| REQ-6 (act verification) | DES-1 (standard workflow, act-compatible) |

## Risks and Tradeoffs

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| `act` image differences from GitHub ubuntu-latest | Medium | Low | GitHub is source of truth; `act` is dev convenience |
| qBittorrent container slow to start | Low | Medium | 30 retries × 2s = 60s timeout; generous start period |
| `golangci-lint` version drift between local and CI | Low | Medium | Pin v2.12.2 in workflow install script |
| Integration test password change in qBittorrent image | Low | High | Monitor linuxserver/qbittorrent release notes |
