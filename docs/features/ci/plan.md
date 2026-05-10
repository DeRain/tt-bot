---
title: "GitHub Actions CI Workflow — Plan"
feature_id: "ci"
status: implemented
depends_on_design: "docs/features/ci/design.md"
last_updated: 2026-05-10
---

# GitHub Actions CI Workflow — Plan

## Overview

Three sequential tasks: create the workflow file, verify with `act`, update README docs. The workflow YAML is self-contained — no Go code changes needed.

## Preconditions

- `act` v0.2.88 installed locally
- Docker available (v29.4.2)
- `golangci-lint` v2.12.2 available (installed locally, installed in CI via script)
- Go 1.26.1 (used by `setup-go@v5` in CI)
- `testdata/qbt-config/` exists with qBittorrent test config
- Telegram secrets configured in GitHub repo settings
- `arch-go.yml` exists (protected file — not modified by this feature)
- `docker-compose.test.yml` exists (protected file — not modified by this feature)

## Task Sequence

- **TASK-1**: Create `.github/workflows/ci.yml` with `gate` job (build, lint, unit tests, arch-check)
  - Derived from: DES-1, DES-2
  - Implements: REQ-1, REQ-2
  - Impacts: `.github/workflows/ci.yml` (new file)
  - Verification: CHECK-1 (`act -j gate`)
  - Gate: 4

- **TASK-2**: Add `integration` job to `.github/workflows/ci.yml` (qBittorrent service, integration + E2E tests)
  - Derived from: DES-3, DES-4, DES-5, DES-6
  - Implements: REQ-3, REQ-4, REQ-5
  - Impacts: `.github/workflows/ci.yml` (extend)
  - Verification: CHECK-2 (`act -j integration`)
  - Gate: 4

- **TASK-3**: Update README.md with CI documentation and local `act` usage
  - Derived from: DES-1
  - Implements: REQ-6
  - Impacts: `README.md`
  - Verification: CHECK-3 (manual review)
  - Gate: 4

- **TASK-4**: Full workflow verification with `act push` and GitHub push
  - Derived from: DES-1
  - Implements: REQ-6
  - Impacts: None (verification only)
  - Verification: CHECK-4 (`act push`), CHECK-5 (GitHub Actions run)
  - Gate: 5

## File Manifest

| File | Action | Description |
|---|---|---|
| `.github/workflows/ci.yml` | CREATE | CI workflow definition |
| `.github/workflows/` | CREATE (dir) | Workflow directory |
| `README.md` | UPDATE | Add CI section |

## Dependency Graph

```
TASK-1 (gate job) ──┐
                     ├──> TASK-3 (README) ──> TASK-4 (verify)
TASK-2 (integration) ┘
```

TASK-1 and TASK-2 are in the same file and can be created together; TASK-3 is a documentation follow-up. TASK-4 is the verification gate.
