#!/usr/bin/env bash
# PostToolUse hook: Auto-format and lint on every file edit.
#
# Phase 1: Auto-format (silent fixes)
# Phase 2: Collect violations as JSON
# Phase 3: Delegate unfixable violations to claude subprocess
#
# Exit 0 = clean (or all fixed), Exit 2 = violations remain
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG="$SCRIPT_DIR/config.json"
PROJECT_ROOT="$(git -C "$SCRIPT_DIR" rev-parse --show-toplevel 2>/dev/null || echo "$SCRIPT_DIR/../..")"

# Read tool input from stdin
INPUT=$(cat)

# Extract the file path from the tool input
FILE_PATH=$(echo "$INPUT" | jaq -r '.tool_input.file_path // ""')

if [[ -z "$FILE_PATH" || ! -f "$FILE_PATH" ]]; then
  exit 0
fi

# Determine file type
EXT="${FILE_PATH##*.}"
BASENAME=$(basename "$FILE_PATH")

detect_language() {
  case "$EXT" in
    go)         echo "go" ;;
    sh|bash)    echo "shell" ;;
    *)
      case "$BASENAME" in
        Dockerfile*) echo "dockerfile" ;;
        *.sh)        echo "shell" ;;
        *)           echo "unknown" ;;
      esac
      ;;
  esac
}

LANG=$(detect_language)

# Check if language is enabled
is_enabled() {
  jaq -r ".languages.$1 // false" "$CONFIG" 2>/dev/null
}

if [[ $(is_enabled "$LANG") != "true" ]]; then
  exit 0
fi

VIOLATIONS=()

# ── Phase 1: Auto-format ──────────────────────────────────────────────
phase1_format() {
  case "$LANG" in
    go)
      gofmt -w "$FILE_PATH" 2>/dev/null || true
      ;;
    shell)
      if command -v shfmt &>/dev/null; then
        shfmt -w -i 2 -ci "$FILE_PATH" 2>/dev/null || true
      fi
      ;;
  esac
}

# ── Phase 2: Collect violations ───────────────────────────────────────
phase2_lint() {
  case "$LANG" in
    go)
      if command -v golangci-lint &>/dev/null; then
        # Lint the package containing the file, not the file alone
        local pkg_dir
        pkg_dir=$(dirname "$FILE_PATH")
        local output
        output=$(cd "$PROJECT_ROOT" && golangci-lint run "./${pkg_dir#"$PROJECT_ROOT"/}/..." 2>/dev/null) || true
        if [[ -n "$output" ]]; then
          while IFS= read -r line; do
            # Skip empty lines and golangci-lint summary lines
            [[ -z "$line" ]] && continue
            [[ "$line" =~ ^[0-9]+\ issues ]] && continue
            [[ "$line" =~ ^\*\  ]] && continue
            VIOLATIONS+=("$line")
          done <<< "$output"
        fi
      fi
      ;;
    shell)
      if command -v shellcheck &>/dev/null; then
        local output
        output=$(shellcheck -f gcc "$FILE_PATH" 2>/dev/null) || true
        if [[ -n "$output" ]]; then
          while IFS= read -r line; do
            VIOLATIONS+=("$line")
          done <<< "$output"
        fi
      fi
      ;;
    dockerfile)
      if command -v hadolint &>/dev/null; then
        local output
        output=$(hadolint "$FILE_PATH" 2>/dev/null) || true
        if [[ -n "$output" ]]; then
          while IFS= read -r line; do
            VIOLATIONS+=("$line")
          done <<< "$output"
        fi
      fi
      ;;
  esac
}

# ── Phase 3: Delegate to subprocess ───────────────────────────────────
phase3_delegate() {
  local count=${#VIOLATIONS[@]}
  if [[ $count -eq 0 ]]; then
    return 0
  fi

  local delegation
  delegation=$(jaq -r '.phases.subprocess_delegation // false' "$CONFIG" 2>/dev/null)
  if [[ "$delegation" != "true" ]]; then
    return 1
  fi

  # Skip delegation if env override set
  if [[ "${HOOK_SKIP_SUBPROCESS:-0}" == "1" ]]; then
    return 1
  fi

  # Select model tier based on violation count
  local threshold
  threshold=$(jaq -r '.subprocess.volume_threshold // 5' "$CONFIG" 2>/dev/null)
  local model="haiku"
  if [[ $count -gt $threshold ]]; then
    model="sonnet"
  fi

  local timeout
  timeout=$(jaq -r ".subprocess.tiers.$model.timeout // 120" "$CONFIG" 2>/dev/null)
  timeout="${HOOK_SUBPROCESS_TIMEOUT:-$timeout}"

  local max_turns
  max_turns=$(jaq -r ".subprocess.tiers.$model.max_turns // 10" "$CONFIG" 2>/dev/null)

  # Build violation text
  local violation_text
  violation_text=$(printf '%s\n' "${VIOLATIONS[@]}")

  # Spawn claude subprocess to fix
  local prompt="Fix the following linter violations in $FILE_PATH. Only fix the violations listed — do not refactor or change other code.

Violations:
$violation_text

Rules:
- Fix each violation with minimal changes
- Do not add comments explaining the fix
- Do not modify unrelated code
- Do not disable linter rules"

  if command -v claude &>/dev/null; then
    timeout "$timeout" claude -p "$prompt" --model "$model" --max-turns "$max_turns" --no-input 2>/dev/null || true
  else
    return 1
  fi

  # Re-run Phase 1+2 to verify
  VIOLATIONS=()
  phase1_format
  phase2_lint

  if [[ ${#VIOLATIONS[@]} -gt 0 ]]; then
    return 1
  fi
  return 0
}

# ── Execute ───────────────────────────────────────────────────────────
phase1_format
phase2_lint

if [[ ${#VIOLATIONS[@]} -eq 0 ]]; then
  exit 0
fi

# Try subprocess delegation
if phase3_delegate; then
  exit 0
fi

# Report remaining violations to stderr (hook system reads stderr)
echo "[hook] ${#VIOLATIONS[@]} violation(s) remain in $(basename "$FILE_PATH"):" >&2
printf '  %s\n' "${VIOLATIONS[@]}" >&2
exit 2
