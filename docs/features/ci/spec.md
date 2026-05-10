---
title: "GitHub Actions CI Workflow"
feature_id: "ci"
status: implemented
owner: DeRain
source_files: [".github/workflows/ci.yml"]
last_updated: 2026-05-10
---

# GitHub Actions CI Workflow — Specification

## Overview

Add a GitHub Actions workflow that runs the full quality gate suite (build, lint, unit tests, arch-check, integration/E2E tests) on every push and pull request, eliminating manual gate runs before merge.

## Problem Statement

All quality gates are run manually via `make gate-all` and `make test-integration`. This is error-prone — developers forget to run the full suite, and reviewers must either trust the author or run gates themselves. There is no automated CI feedback on push or PR.

## Goals

- Automatically run `make gate-all` (build + lint + unit tests + arch-check) on every push and PR
- Automatically run integration + E2E tests against a real qBittorrent service on every push and PR
- Provide fast-fail: unit tests fail before Docker-bound integration tests waste resources
- Support local verification via `act` (GitHub Actions emulator)
- Keep secrets out of the workflow file (use GitHub Actions secrets)

## Non-Goals

- Deployment automation (CI is verify-only)
- Release/publish workflow
- Scheduled/cron runs (for now)
- Code coverage reporting to external services
- Artifact uploads
- Notification beyond GitHub's built-in status checks

## Scope

This feature covers: creating `.github/workflows/ci.yml` with two jobs (`gate` and `integration`), wiring qBittorrent as a service container, and using GitHub secrets for Telegram credentials. It does not cover adding new gate checks — only automating the existing ones.

## Requirements

- **REQ-1**: The CI workflow MUST run on every push (any branch) and every pull request targeting `main`.
- **REQ-2**: The CI workflow MUST execute the full quality gate from `docs/gates.md` Gate 4: build, lint, unit tests with coverage, and architecture check.
- **REQ-3**: The CI workflow MUST run integration and E2E tests against a real qBittorrent service with Telegram API integration.
- **REQ-4**: The integration job MUST depend on the gate job passing (fast-fail: no Docker overhead if unit tests fail).
- **REQ-5**: Secrets (TELEGRAM_BOT_TOKEN, TELEGRAM_ALLOWED_USERS) MUST be sourced from GitHub Actions secrets, never hardcoded.
- **REQ-6**: The workflow MUST be verifiable locally via `act` before pushing to GitHub.

## Acceptance Criteria

- **AC-1.1**: Pushing a commit to any branch triggers the CI workflow.
- **AC-1.2**: Creating a PR against `main` triggers the CI workflow.
- **AC-2.1**: The `gate` job runs `make build`, `make lint`, `make test`, and `make arch-check` and all pass.
- **AC-2.2**: A build failure in `gate` produces a clear, actionable error message.
- **AC-3.1**: The `integration` job starts a qBittorrent service container and runs `go test ./... -tags=integration -run "Integration|E2E"`.
- **AC-3.2**: Integration tests pass against the real qBittorrent service (healthcheck passes, login succeeds, API calls work).
- **AC-4.1**: The `integration` job is skipped when the `gate` job fails.
- **AC-5.1**: No hardcoded Telegram token or user IDs appear in the workflow file.
- **AC-6.1**: `act -j gate` passes locally (no secrets needed).
- **AC-6.2**: `act -j integration --secret TELEGRAM_BOT_TOKEN=<token> --secret TELEGRAM_ALLOWED_USERS=<ids>` passes locally.

## Quality Gates

### Gate 1: Spec Gate

This spec passes when:
- [x] All requirements are clear and unambiguous
- [x] All acceptance criteria are testable
- [x] Scope and non-goals are defined
- [x] No unresolved open questions block implementation
- [x] At least one AC exists per requirement

**Harness check command:**
```bash
grep -c "^- \*\*REQ-" docs/features/ci/spec.md  # count requirements
grep -c "^- \*\*AC-"  docs/features/ci/spec.md  # count acceptance criteria
grep -c "TODO:"        docs/features/ci/spec.md  # should be 0 for approved
```

## Risks

- **MEDIUM**: `act` runner images may differ from GitHub-hosted `ubuntu-latest` — mitigated by using GitHub runs as source of truth.
- **LOW**: qBittorrent service healthcheck timing differs between `act` and GitHub — mitigated by generous retry count (30 attempts, 2s apart).
- **LOW**: `arch-check` uses `go run ...@latest` requiring network access — mitigated by Go module proxy availability on GitHub runners.

## Open Questions

None.
