# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Model Routing Enforcement (MANDATORY)

Opus MUST NOT write implementation code directly. All implementation file edits MUST be dispatched to Sonnet agents via the `model: sonnet` parameter.

### Role Separation

| Role | Model | Scope |
|------|-------|-------|
| Orchestration | Opus | Docs (`.md`), plans, memory, git ops, `.claude/` config, CLAUDE.md, review |
| Implementation | Sonnet (agents) | All code and config files |
| Gate checks | Haiku (agents) | Lint, build verification, coverage checks |

### Implementation Files (Sonnet only)

`*.go`, `*.yaml`, `*.yml`, `Dockerfile`, `docker-compose*.yml`, `Makefile`, `*.toml`, `*.json` (non-claude config), `.env*`

### Orchestration Files (Opus ok)

`*.md`, `.claude/` directory config, `CLAUDE.md`

### Enforcement

- Before editing any implementation file, Opus MUST dispatch a Sonnet agent instead
- Opus may only edit implementation files for trivial 1-line fixes after a Sonnet agent has already done the main work

### OpenCode Compatibility

This repository also supports OpenCode (DeepSeek models). See @AGENTS.md for the unified model routing table covering both tools.

## Pre-Commit Quality Gate

**See @docs/gates.md for full gate definitions and the Iterative Harness Loop Protocol.**

After Sonnet implementation agent returns, Opus MUST NOT immediately commit:

1. Opus dispatches go-reviewer agent for code review
2. Opus independently runs `make test-integration` to verify
3. Address any code review findings (dispatch Sonnet if needed)
4. **ONLY THEN**: commit, push, create PR

**The Sonnet agent's claim of "all tests pass" must be independently verified.**

## Sonnet Implementation Agent Prompts (MANDATORY)

Every Sonnet implementation agent prompt MUST include the gate requirements from @docs/gates.md — specifically the "Iterative Harness Loop Protocol" steps.

## Shared Project Instructions

@AGENTS.md contains project-wide instructions shared between Claude Code and OpenCode: architecture, build commands, testing guidelines, docs-first workflow, commit conventions, and the unified model routing table.

<!-- gitnexus:start -->
# GitNexus — Code Intelligence

This project is indexed by GitNexus as **tt-bot** (2354 symbols, 5126 relationships, 92 execution flows). Use the GitNexus MCP tools to understand code, assess impact, and navigate safely.

> If any GitNexus tool warns the index is stale, run `npx gitnexus analyze` in terminal first.

## Always Do

- **MUST run impact analysis before editing any symbol.** Before modifying a function, class, or method, run `gitnexus_impact({target: "symbolName", direction: "upstream"})` and report the blast radius (direct callers, affected processes, risk level) to the user.
- **MUST run `gitnexus_detect_changes()` before committing** to verify your changes only affect expected symbols and execution flows.
- **MUST warn the user** if impact analysis returns HIGH or CRITICAL risk before proceeding with edits.
- When exploring unfamiliar code, use `gitnexus_query({query: "concept"})` to find execution flows instead of grepping. It returns process-grouped results ranked by relevance.
- When you need full context on a specific symbol — callers, callees, which execution flows it participates in — use `gitnexus_context({name: "symbolName"})`.

## Never Do

- NEVER edit a function, class, or method without first running `gitnexus_impact` on it.
- NEVER ignore HIGH or CRITICAL risk warnings from impact analysis.
- NEVER rename symbols with find-and-replace — use `gitnexus_rename` which understands the call graph.
- NEVER commit changes without running `gitnexus_detect_changes()` to check affected scope.

## Resources

| Resource | Use for |
|----------|---------|
| `gitnexus://repo/tt-bot/context` | Codebase overview, check index freshness |
| `gitnexus://repo/tt-bot/clusters` | All functional areas |
| `gitnexus://repo/tt-bot/processes` | All execution flows |
| `gitnexus://repo/tt-bot/process/{name}` | Step-by-step execution trace |

## CLI

| Task | Read this skill file |
|------|---------------------|
| Understand architecture / "How does X work?" | `.claude/skills/gitnexus/gitnexus-exploring/SKILL.md` |
| Blast radius / "What breaks if I change X?" | `.claude/skills/gitnexus/gitnexus-impact-analysis/SKILL.md` |
| Trace bugs / "Why is X failing?" | `.claude/skills/gitnexus/gitnexus-debugging/SKILL.md` |
| Rename / extract / split / refactor | `.claude/skills/gitnexus/gitnexus-refactoring/SKILL.md` |
| Tools, resources, schema reference | `.claude/skills/gitnexus/gitnexus-guide/SKILL.md` |
| Index, status, clean, wiki CLI commands | `.claude/skills/gitnexus/gitnexus-cli/SKILL.md` |

<!-- gitnexus:end -->
