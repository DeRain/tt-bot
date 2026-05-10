---
title: "Architecture Governance (arch-go) — Design"
feature_id: "arch-go"
status: implemented
depends_on_spec: "docs/features/arch-go/spec.md"
last_updated: 2026-05-10
---

# Architecture Governance (arch-go) — Design

## Overview

Add an `arch-go.yml` configuration file encoding the current 6-package dependency structure as dependency rules, a `make arch-check` target using `go run` for zero-dependency CLI invocation, and wire it into `make gate-all`. Update quality gate documentation and templates to reference architecture checks.

## Architecture

### Dependency Graph (Current State)

```
cmd/bot
  ├── internal/config    (stdlib only)
  ├── internal/bot       → internal/qbt, internal/formatter
  ├── internal/poller    → internal/qbt
  └── internal/qbt       (stdlib only)

internal/bot      → internal/qbt, internal/formatter
internal/formatter → internal/qbt
internal/poller   → internal/qbt
internal/qbt      (stdlib only — leaf)
internal/config   (stdlib only — leaf)
```

> The `**.internal.**` glob in Rule 1 matches all sub-packages (config, qbt, bot, formatter, poller). This is intentional — cmd is the composition root and may wire any internal package.

### Rules Encoding

| Rule | Package Pattern | Constraint Type | Constraint |
|------|----------------|-----------------|------------|
| 1 | `**.cmd.**` | shouldOnlyDependsOn.internal | `**.internal.**` |
| 2 | `**.internal.qbt` | shouldNotDependsOn.internal | `**.internal.**` |
| 3 | `**.internal.config` | shouldNotDependsOn.internal | `**.internal.**` |
| 4 | `**.internal.formatter` | shouldOnlyDependsOn.internal | `**.internal.qbt` |
| 5 | `**.internal.poller` | shouldOnlyDependsOn.internal | `**.internal.qbt` |
| 6 | `**.internal.bot` | shouldOnlyDependsOn.internal | `**.internal.qbt`, `**.internal.formatter` |

### Docker Test Configuration
The `docker-compose.test.yml` passes `TELEGRAM_BOT_TOKEN` and `TELEGRAM_ALLOWED_USERS` from the host environment to the integration test container. This enables `TestRegisterCommands_Integration` to call the Telegram `setMyCommands` API. Use a dedicated test bot token in CI to avoid exposing production credentials.

### Thresholds

- **compliance**: 100% — all rules must pass
- **coverage**: 90% — allows 1 uncovered package out of ~10 total

### Integration Points

```
arch-go.yml           → Rules definition (repo root)
Makefile              → arch-check target, gate-all dependency chain
docs/gates.md         → Gate 4 criteria updated
docs/pr-checklist.md  → Verification evidence updated
docs/features/_templates/plan.md → Quality gates section updated
```

## Data Flow

### Architecture Check Flow

1. Developer runs `make arch-check` (or `make gate-all`)
2. Make invokes `go run github.com/arch-go/arch-go/v2@latest`
3. Go downloads arch-go binary if not cached, then executes it
4. arch-go reads `arch-go.yml` from the current directory
5. arch-go parses the module's packages via `go list` and builds an import graph
6. Each dependency rule is evaluated against the import graph
7. Results are printed to stdout; exit code 0 if compliance >= threshold

### Gate Integration Flow

```
make gate-all
  ├── go build ./...          (build)
  ├── golangci-lint run       (lint)
  ├── go test ./... -short    (unit tests)
  └── go run .../arch-go      (architecture)  ← NEW
```

## Interfaces

### arch-go.yml (version 1 schema)

No code interfaces — purely configuration. The `arch-go.yml` file is the contract between the project and the arch-go tool.

### Makefile Targets

```makefile
arch-check:
	go run github.com/arch-go/arch-go/v2@latest

gate-all: build lint test arch-check
```

## Data/Storage Impact

None. No new files beyond `arch-go.yml` and feature docs. No state stored.

## Error Handling

- arch-go non-zero exit: Make halts, developer sees rule violation details on stdout
- Network unavailable on first `go run`: Go build toolchain error, same as any missing dependency
- `arch-go.yml` missing or malformed: arch-go reports YAML parse error, exits non-zero

## Security Considerations

- `go run @latest` fetches from GitHub — standard Go module proxy security model
- No secrets in `arch-go.yml` — it's pure architecture rules
- No runtime impact — arch-go is a static analysis tool, not imported into the binary

## Performance Considerations

- First run: ~10-15s to download and compile arch-go
- Subsequent runs: ~500ms (cached binary, reads `arch-go.yml`, runs `go list`, evaluates rules)
- No impact on build or test times beyond the gate-all chain

## Tradeoffs

| Decision | Alternative | Rationale |
|----------|-------------|-----------|
| `go run` CLI | `go install` + PATH | `go run` is self-contained, no PATH setup, no global tool pollution |
| `@latest` version | Pinned version | Descriptive rules rarely break; `@latest` is simpler for a small project |
| Descriptive rules only | Prescriptive rules | Prescriptive rules would require code refactoring; separate feature |
| No go.mod dependency | Add as dev dependency | Keeps dependency graph clean; architecture check is a CI concern, not a build concern |
| 90% coverage threshold | 100% coverage | Allows one uncovered package (e.g., future utility) without blocking |

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| arch-go API change on major version | Low | Low | `@latest` follows v2.x; major bumps require `v3@latest` explicitly |
| New package not covered by rules | Low | Low | 90% threshold allows margin; add rules for new packages in their own feature PR |

## Design Items

- **DES-1**: `arch-go.yml` configuration with 6 dependency rules and threshold settings
  - Satisfies: REQ-1, REQ-5
  - Covers: AC-1.1, AC-1.2, AC-1.3, AC-5.1

- **DES-2**: `make arch-check` target using `go run` for zero-dependency CLI invocation
  - Satisfies: REQ-2
  - Covers: AC-2.1

- **DES-3**: `make gate-all` target dependency chain updated to include `arch-check`
  - Satisfies: REQ-3
  - Covers: AC-3.1, AC-3.2

- **DES-4**: Quality gate documentation and templates updated
  - Satisfies: REQ-4
  - Covers: AC-4.1, AC-4.2, AC-4.3

- **DES-5**: `docker-compose.test.yml` passes Telegram env vars to test container
  - Satisfies: REQ-6
  - Covers: AC-6.1

## Quality Gates

### Gate 2: Design Gate

This design passes when:
- [ ] Every REQ-* from spec.md is addressed by at least one DES-*
- [ ] Every AC-* from spec.md is covered by at least one DES-*
- [ ] Risks and tradeoffs are documented
- [ ] No DES-* exists without a linked REQ-*

**Harness check command:**
```bash
spec_reqs=$(grep -oP 'REQ-\d+' docs/features/arch-go/spec.md | sort -u)
design_reqs=$(grep -oP 'REQ-\d+' docs/features/arch-go/design.md | sort -u)
comm -23 <(echo "$spec_reqs") <(echo "$design_reqs")  # should be empty
```

## Requirement Mapping

| Design Item | Satisfies | Covers |
|-------------|-----------|--------|
| DES-1 | REQ-1, REQ-5 | AC-1.1, AC-1.2, AC-1.3, AC-5.1 |
| DES-2 | REQ-2 | AC-2.1 |
| DES-3 | REQ-3 | AC-3.1, AC-3.2 |
| DES-4 | REQ-4 | AC-4.1, AC-4.2, AC-4.3 |
| DES-5 | REQ-6 | AC-6.1 |
