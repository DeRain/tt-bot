---
title: Configuration Loading -- Design Decisions
feature_id: config
status: accepted
last_updated: 2026-03-15
---

# Configuration Loading -- Design Decisions

## DEC-1: Environment variables vs. configuration file

**Context:** The bot needs runtime configuration for Telegram credentials, qBittorrent connection details, and operational parameters. Two common approaches: environment variables or a configuration file (YAML, TOML, JSON).

**Options considered:**

| Option | Pros | Cons |
|--------|------|------|
| Environment variables | Docker/K8s native; no file to manage; 12-factor app compliant; secret-manager friendly | No nesting; no complex structures; type parsing required |
| Configuration file | Structured; supports comments; IDE-friendly | Requires volume mounts in Docker; secrets in plaintext on disk; extra dependency for parsing |

**Decision:** Environment variables.

**Rationale:** tt-bot runs in Docker via `docker-compose.yml`. Environment variables are the standard mechanism for injecting secrets and configuration into containers. The configuration surface is flat (six scalar values plus one list), making a config file unnecessary complexity. Environment variables also integrate naturally with Docker secrets, CI/CD systems, and `.env` files.

## DEC-2: Fail-fast (first error) vs. accumulate all errors

**Context:** When multiple environment variables are missing or invalid, `Load()` could either return on the first error or collect all validation errors and return them together.

**Options considered:**

| Option | Pros | Cons |
|--------|------|------|
| Fail-fast (first error) | Simple implementation; clear single error message; standard Go idiom | User may need multiple runs to fix all issues |
| Accumulate errors | User sees all problems at once | More complex code; multi-error formatting; non-standard Go pattern |

**Decision:** Fail-fast on first error.

**Rationale:** The configuration has only six variables. In practice, either all are set (via `.env` or docker-compose) or the deployment is fundamentally misconfigured. The fail-fast approach keeps the code simple, follows idiomatic Go error handling, and produces a single clear error message. The sequential validation order (token, users, URL, username, password, poll interval) is deterministic, so the user always knows which variable to fix next.

## DEC-3: Value type (struct) vs. pointer type

**Context:** `Load()` could return either `Config` (value) or `*Config` (pointer). This affects mutability, nil-safety, and API semantics.

**Options considered:**

| Option | Pros | Cons |
|--------|------|------|
| Value type `Config` | Immutable by default (copy semantics); no nil checks needed; thread-safe sharing | Copies slice header (AllowedUsers), though this is negligible |
| Pointer type `*Config` | Avoids copy overhead for large structs; conventional for mutable objects | Enables accidental mutation; requires nil checks; shared mutable state risk |

**Decision:** Value type (`Config` struct returned by value).

**Rationale:** The `Config` struct is small (five strings, one slice, one duration -- under 200 bytes). Copy overhead is negligible and happens exactly once (at startup). Returning by value communicates immutability intent: callers get their own copy and cannot accidentally mutate configuration used by other components. This aligns with the project's immutability-first coding style and eliminates an entire class of concurrency bugs.

## DEC-4: Comma-separated string vs. JSON array for user IDs

**Context:** `TELEGRAM_ALLOWED_USERS` contains multiple int64 values. The encoding format affects usability and parsing complexity.

**Options considered:**

| Option | Pros | Cons |
|--------|------|------|
| Comma-separated | Simple; human-readable; easy to type in `.env` files and CLI; no quoting needed | Custom parsing; whitespace handling required |
| JSON array | Standard format; native type support; no custom parser | Requires quoting in shell/env (`'[1,2,3]'`); JSON dependency; more error-prone in `.env` files |

**Decision:** Comma-separated string.

**Rationale:** Environment variables are strings. A comma-separated list is the most natural encoding for a flat list of IDs in an env var. It is easy to read, edit, and type in `.env` files, `docker-compose.yml`, and shell commands without worrying about JSON quoting or escaping. The custom parser is 20 lines and handles edge cases (whitespace, empty tokens) gracefully. JSON would add friction for the common case (one to five user IDs) without meaningful benefit.

## DEC-5: time.ParseDuration for poll interval

**Context:** `POLL_INTERVAL` needs to be parsed from a string to a `time.Duration`. Options range from a plain integer (seconds) to a full duration string.

**Options considered:**

| Option | Pros | Cons |
|--------|------|------|
| `time.ParseDuration` (Go duration string) | Stdlib; expressive (`30s`, `2m`, `1h`); no custom code | Users must know Go duration syntax |
| Integer seconds | Simplest parsing | Less expressive; easy to confuse units |
| ISO 8601 duration | Standard format | No stdlib support; requires third-party library |

**Decision:** `time.ParseDuration` with Go duration string format.

**Rationale:** Go's `time.ParseDuration` is zero-dependency, well-documented, and expressive. Users can write `30s`, `2m`, `1h30m` naturally. The format is intuitive even for non-Go developers. The default of `30s` is applied when the variable is empty, requiring zero additional parsing logic. ISO 8601 would add a dependency for no practical benefit in this context.
