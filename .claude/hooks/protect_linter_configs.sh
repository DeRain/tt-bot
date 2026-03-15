#!/usr/bin/env bash
# PreToolUse hook: Block edits to linter config files
# Prevents agents from disabling rules instead of fixing code.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG="$SCRIPT_DIR/config.json"

# Read tool input from stdin
INPUT=$(cat)

# Extract the file path from the tool input
FILE_PATH=$(echo "$INPUT" | jaq -r '.tool_input.file_path // .tool_input.command // ""')

if [[ -z "$FILE_PATH" ]]; then
  exit 0
fi

# Get protected configs list
PROTECTED=$(jaq -r '.protected_configs[]' "$CONFIG" 2>/dev/null)

BASENAME=$(basename "$FILE_PATH")

while IFS= read -r config; do
  if [[ "$BASENAME" == "$config" ]]; then
    echo "BLOCKED: Editing linter config '$BASENAME' is not allowed."
    echo "Fix the code to satisfy the linter, do not modify linter configuration."
    exit 2
  fi
done <<< "$PROTECTED"

exit 0
