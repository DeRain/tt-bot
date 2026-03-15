# Pull Request Checklist

Every PR in this repository MUST include the following traceability information.

## PR Description Template

```markdown
## Feature

- **Feature ID**: <feature-id>
- **Feature docs**: docs/features/<feature-id>/

## Traceability

- **Implements**: REQ-1, REQ-2, ...
- **Plan tasks**: TASK-1, TASK-2, ...
- **Acceptance criteria covered**: AC-1.1, AC-2.1, ...

## Verification Evidence

- [ ] Unit tests pass (`make test`)
- [ ] Lint clean (`make lint`)
- [ ] Build passes (`make build`)
- [ ] Integration tests pass (if applicable: `make test-integration`)
- [ ] traceability.md updated with implementation evidence
- [ ] verification.md updated with test results

## Out-of-Scope Changes

<!-- List any changes not traceable to REQ-*/TASK-*. Justify each. -->

## Remaining Gaps

<!-- List any AC-* not yet verified. Explain plan to close. -->
```

## Commit Message Guidance

Include feature ID and task ID where practical:

```
feat(<feature-id>): TASK-N <description>

Implements REQ-N: <requirement summary>
```

Examples:
```
feat(auth): TASK-2 implement whitelist authorizer

Implements REQ-1: restrict bot access to allowed user IDs
```

```
fix(add-torrent): TASK-6 fix TTL eviction race condition

Fixes REQ-5: pending torrents must expire after 5 minutes
```

## Validation Rules

A PR reviewer MUST check:

1. **Traceability**: Every changed file can be linked to a TASK-* and REQ-*.
2. **Completeness**: All listed AC-* have verification evidence.
3. **Scope**: No untraced changes exist without justification.
4. **Gates**: `make gate-all` passes.
5. **Docs**: traceability.md and verification.md are updated.

## Flagging Rules

Flag the PR if:
- A changed file cannot be linked to a TASK-* and REQ-*.
- A requirement has no implementation evidence.
- An acceptance criterion has no verification item.
- Implementation introduces behavior not described by any requirement (scope expansion).
- Tests exist but do not validate stated acceptance criteria (weak coverage).
