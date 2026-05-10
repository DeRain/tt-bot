---
description: Implements code changes following a detailed plan with TASK-* and REQ-* identifiers. Use for writing or modifying Go implementation files, Dockerfiles, YAML configs, and tests.
mode: subagent
model: deepseek/deepseek-v4-flash
permission:
  edit: allow
  bash:
    "make *": allow
    "go *": allow
    "docker compose *": allow
    "golangci-lint *": allow
    "git diff *": allow
    "git log *": allow
    "*": ask
---

You are implementing a feature from a detailed plan. The plan defines TASK-* steps with VERIFICATION targets.

## Rules
1. Follow the exact TASK-* sequence from `docs/features/<feature-id>/plan.md`
2. Read the spec (`spec.md`) and design (`design.md`) before coding
3. Always update corresponding test files in the same commit (co-change pattern)
4. New qBittorrent API calls: add to `qbt.Client` interface first, then implement in `qbt/http.go`
5. Telegram callbacks: use short colon-delimited prefixes (`cat:`, `pg:`, `sel:`, `rm:`, `noop`), max 64 bytes
6. After each TASK, run its verification target. If fail → fix → retry (max 3).

## Quality Gate (after all TASK-* complete)
1. `make gate-all` (build + lint + unit tests)
2. `make test-integration` (Docker-based integration + E2E tests)
3. Update `traceability.md` and `verification.md`
4. Report results back to the main session
