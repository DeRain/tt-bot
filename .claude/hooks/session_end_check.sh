#!/usr/bin/env bash
# Stop hook: Session-end verification checks
set -euo pipefail

PROJECT_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
WARNINGS=()

# Check for uncommitted .env files
ENV_FILES=$(git -C "$PROJECT_ROOT" ls-files --others --exclude-standard '*.env' '.env*' 2>/dev/null || true)
if [[ -n "$ENV_FILES" ]]; then
  WARNINGS+=("Untracked .env files found: $ENV_FILES")
fi

# Check for staged .env files
STAGED_ENV=$(git -C "$PROJECT_ROOT" diff --cached --name-only -- '*.env' '.env*' 2>/dev/null || true)
if [[ -n "$STAGED_ENV" ]]; then
  WARNINGS+=("WARNING: .env files are staged for commit: $STAGED_ENV")
fi

# Check for debug prints in staged Go files
STAGED_GO=$(git -C "$PROJECT_ROOT" diff --cached --name-only -- '*.go' 2>/dev/null || true)
if [[ -n "$STAGED_GO" ]]; then
  for f in $STAGED_GO; do
    FULL_PATH="$PROJECT_ROOT/$f"
    if [[ -f "$FULL_PATH" ]]; then
      DEBUG_LINES=$(grep -n 'fmt\.Print\|log\.Print\|println(' "$FULL_PATH" 2>/dev/null | grep -v '_test\.go' || true)
      if [[ -n "$DEBUG_LINES" ]]; then
        WARNINGS+=("Possible debug prints in $f: $DEBUG_LINES")
      fi
    fi
  done
fi

# Check for uncommitted changes
UNCOMMITTED=$(git -C "$PROJECT_ROOT" status --porcelain 2>/dev/null || true)
if [[ -n "$UNCOMMITTED" ]]; then
  FILE_COUNT=$(echo "$UNCOMMITTED" | wc -l | tr -d ' ')
  WARNINGS+=("$FILE_COUNT uncommitted file(s) in working tree")
fi

if [[ ${#WARNINGS[@]} -gt 0 ]]; then
  echo "[session-end] ${#WARNINGS[@]} warning(s):" >&2
  printf '  ⚠ %s\n' "${WARNINGS[@]}" >&2
fi

# Always exit 0 — Stop hooks should warn, not block
exit 0
