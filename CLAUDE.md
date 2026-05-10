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

## RAG and Search Tools

Use **claude-context** MCP server (`@zilliz/claude-context-mcp`) as the semantic search tool for the codebase. Fall back to built-in tools (Grep, Glob, Read) when claude-context returns insufficient results.

### claude-context Availability

- **On session start**, if claude-context is unavailable, attempt to reconnect via `/mcp` before proceeding.
- **During workload**, if a claude-context call fails with a connection error, reconnect via `/mcp` and retry once. If reconnection fails, fall back to built-in tools (Grep, Glob, Read) and notify the user.
- **Dependencies**: Requires Milvus (`docker compose up milvus-standalone`) and Ollama running locally.

### claude-context Tools

| Tool | Purpose |
|------|---------|
| `search_code` | Semantic search using natural language queries. Requires absolute path. |
| `index_codebase` | Index a codebase directory for semantic search. |
| `get_indexing_status` | Check indexing progress for a codebase directory. |
| `clear_index` | Clear the search index for a codebase directory. |

### Tool Selection Guide

| Task | Primary Tool | Fallback |
|------|-------------|----------|
| Find code related to a concept | `search_code` | Grep |
| Find a symbol definition | `search_code` | Grep |
| Find all references to a symbol | `search_code` | Grep |
| Explore a module's structure | `search_code` | Glob + Read |
| Find files by name/pattern | Glob | — |
| Exact text/regex search | Grep | `search_code` |
| Read full file content | Read | — |

### Fallback Chain

1. **Always try `search_code` first** for code discovery.
2. **Fall back to Grep** for exact text/regex matches or when `search_code` returns insufficient results.
3. **Fall back to Glob** for file name/pattern discovery.
4. **Fall back to Read** for full file content when needed.

### Token-Efficiency Rules

1. Use `search_code` as the primary discovery tool.
2. Use Grep/Glob when you need exact text matches or file patterns.
3. Read full file bodies only when search results are insufficient.
4. Avoid repeated scans of unchanged areas.
5. Batch parallelizable reads/searches — do not serialize independent commands.
6. If a command fails, diagnose once, pivot strategy, continue — no blind retry loops.

### Anti-Patterns

1. Reading entire files without searching first.
2. Running independent commands sequentially when they can be parallelized.
3. Repeating failed commands without changing inputs or approach.
4. Skipping `search_code` and jumping straight to Grep/Glob/Read.
5. Never trust a single empty result — verify with Grep before concluding something has no references.

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
