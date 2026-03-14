---
name: telegram-callback-encoding
description: "Encode Telegram inline keyboard callbacks within 64-byte limit using short prefixes"
user-invocable: false
origin: auto-extracted
---

# Telegram Callback Data Encoding

**Extracted:** 2026-03-15
**Context:** Building Telegram bots with inline keyboards

## Problem
Telegram Bot API limits callback_data to 64 bytes per button. Category names,
page numbers, and action identifiers must all fit within this constraint.
Multi-byte UTF-8 strings can be silently corrupted if truncated at a byte boundary.

## Solution
Use short prefixes with colon separator:
- `cat:<name>` — category selection (truncate name to 58 bytes, UTF-8 aware)
- `pg:all:<N>` — pagination for "all" filter
- `pg:act:<N>` — pagination for "active" filter
- `noop` — no-op for display-only buttons (e.g., "Page 2/5")

For user-provided strings, truncate at valid UTF-8 boundaries, not raw byte offsets.

## When to Use
- Building Telegram bots with inline keyboard callbacks
- Any callback data that includes user-provided text (category names, labels)
