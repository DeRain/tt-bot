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

<!-- code-intel:start -->
# Code Intelligence — CLI + LSP Tools

Code navigation uses gopls (Go language server) and standard CLI tools. No indexing step required.

## LSP Tools (in-session, preferred)

| Action | Tool |
|--------|------|
| Find all references/callers | `lsp_find_references` |
| Jump to definition | `lsp_goto_definition` |
| Workspace symbol search | `lsp_symbols` |
| Check diagnostics | `lsp_diagnostics` |
| Rename safely | `lsp_rename` (use `lsp_prepare_rename` first) |

## CLI Tools (when LSP unavailable or for scripts)

| Action | Command |
|--------|---------|
| Call hierarchy (callers + callees) | `gopls call_hierarchy file:line:col` |
| Find interface implementations | `gopls implementation file:line:col` |
| Find references | `gopls references file:line:col` |
| Jump to definition | `gopls definition file:line:col` |
| Check for errors | `gopls check file.go` |
| Structural search (AST) | `sg -p 'pattern' -l go` |
| Content search (regex) | `rg 'pattern'` |
| File search | `fd 'pattern'` |
| Lint | `golangci-lint run` |

## Workflows

### Before Editing a Symbol
1. `lsp_find_references` — see all callers
2. If modifying an interface: `gopls implementation file:line:col`
3. For deep impact: recurse `lsp_find_references` on each caller

### Before Committing
1. `lsp_diagnostics` on changed files
2. `make gate-all` (build + lint + unit tests)
3. `make test-integration` for API-facing changes
<!-- code-intel:end -->
