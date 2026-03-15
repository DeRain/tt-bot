---
title: "Stop and Remove Torrent Actions — Traceability Matrix"
feature_id: "torrent-remove"
status: draft
last_updated: 2026-03-15
---

# Stop and Remove Torrent Actions — Traceability Matrix

## Forward Traceability: Requirements → Design → Plan → Tests

| REQ | AC | DES | TASK | TEST / CHECK | Implementation Evidence | Status |
|-----|----|-----|------|--------------|------------------------|--------|
| REQ-1 | AC-1.1, AC-1.2 | DES-3, DES-8 | TASK-4, TASK-10 | TEST-4, TEST-10 | TODO | TODO |
| REQ-2 | AC-2.1, AC-2.2 | DES-4, DES-5, DES-8 | TASK-5, TASK-6, TASK-7, TASK-10 | TEST-5, TEST-6, TEST-7, TEST-10 | TODO | TODO |
| REQ-3 | AC-3.1, AC-3.2 | DES-1, DES-2, DES-6, DES-8 | TASK-1, TASK-2, TASK-3, TASK-8, TASK-10 | TEST-1, TEST-2, TEST-8, CHECK-1, TEST-10 | TODO | TODO |
| REQ-4 | AC-4.1, AC-4.2 | DES-1, DES-2, DES-4, DES-6, DES-8 | TASK-1, TASK-2, TASK-3, TASK-6, TASK-8, TASK-10 | TEST-1, TEST-2, TEST-6, TEST-8, CHECK-1, TEST-10 | TODO | TODO |
| REQ-5 | AC-5.1, AC-5.2 | DES-6, DES-8 | TASK-8, TASK-10 | TEST-8, TEST-10 | TODO | TODO |
| REQ-6 | AC-6.1, AC-6.2 | DES-4, DES-7, DES-8 | TASK-6, TASK-9, TASK-10 | TEST-6, TEST-9, TEST-10 | TODO | TODO |

## Backward Traceability: Tests → Plan → Design → Requirements

| TEST / CHECK | TASK | DES | REQ | AC | Status |
|--------------|------|-----|-----|----|--------|
| TEST-1 | TASK-1 | DES-1 | REQ-3, REQ-4 | AC-3.1, AC-4.1 | TODO |
| TEST-2 | TASK-2 | DES-2 | REQ-3, REQ-4 | AC-3.1, AC-3.2, AC-4.1 | TODO |
| CHECK-1 | TASK-2 | DES-2 | REQ-3, REQ-4 | AC-3.1, AC-3.2, AC-4.1 | TODO |
| TEST-3 | TASK-3 | DES-1 | REQ-3, REQ-4 | AC-3.1, AC-4.1 | TODO |
| TEST-4 | TASK-4 | DES-3 | REQ-1 | AC-1.1, AC-1.2 | TODO |
| TEST-5 | TASK-5 | DES-4 | REQ-2 | AC-2.1, AC-2.2 | TODO |
| TEST-6 | TASK-6 | DES-4 | REQ-2, REQ-4, REQ-6 | AC-2.1, AC-4.2, AC-6.1, AC-6.2 | TODO |
| TEST-7 | TASK-7 | DES-5 | REQ-2 | AC-2.1, AC-2.2 | TODO |
| TEST-8 | TASK-8 | DES-6 | REQ-3, REQ-4, REQ-5 | AC-3.1, AC-4.1, AC-5.1, AC-5.2 | TODO |
| TEST-9 | TASK-9 | DES-7 | REQ-6 | AC-6.1, AC-6.2 | TODO |
| TEST-10 | TASK-10 | DES-8 | REQ-1, REQ-2, REQ-3, REQ-4, REQ-5, REQ-6 | AC-1.1, AC-2.1, AC-3.1, AC-4.1, AC-5.1, AC-6.1 | TODO |

## AC Coverage Matrix

| AC | Requirement | Covered by DES | Verified by TEST / CHECK | Status |
|----|-------------|----------------|--------------------------|--------|
| AC-1.1 | REQ-1 | DES-3, DES-8 | TEST-4, TEST-10 | TODO |
| AC-1.2 | REQ-1 | DES-3 | TEST-4 | TODO |
| AC-2.1 | REQ-2 | DES-4, DES-5, DES-8 | TEST-5, TEST-7, TEST-10 | TODO |
| AC-2.2 | REQ-2 | DES-4, DES-5 | TEST-5, TEST-7 | TODO |
| AC-3.1 | REQ-3 | DES-1, DES-2, DES-6, DES-8 | TEST-2, TEST-8, CHECK-1, TEST-10 | TODO |
| AC-3.2 | REQ-3 | DES-2 | CHECK-1 | TODO |
| AC-4.1 | REQ-4 | DES-1, DES-2, DES-6, DES-8 | TEST-2, TEST-8, CHECK-1, TEST-10 | TODO |
| AC-4.2 | REQ-4 | DES-4 | TEST-6 | TODO |
| AC-5.1 | REQ-5 | DES-6, DES-8 | TEST-8, TEST-10 | TODO |
| AC-5.2 | REQ-5 | DES-6 | TEST-8 | TODO |
| AC-6.1 | REQ-6 | DES-4, DES-7, DES-8 | TEST-6, TEST-9, TEST-10 | TODO |
| AC-6.2 | REQ-6 | DES-4, DES-7 | TEST-6, TEST-9 | TODO |

## Design Item Coverage

| DES | Satisfies REQ | Covers AC | Implemented by TASK | Status |
|-----|---------------|-----------|---------------------|--------|
| DES-1 | REQ-3, REQ-4 | AC-3.1, AC-4.1 | TASK-1, TASK-3 | TODO |
| DES-2 | REQ-3, REQ-4 | AC-3.1, AC-3.2, AC-4.1 | TASK-2 | TODO |
| DES-3 | REQ-1 | AC-1.1, AC-1.2 | TASK-4 | TODO |
| DES-4 | REQ-2, REQ-6 | AC-2.1, AC-2.2, AC-4.2, AC-6.1, AC-6.2 | TASK-5, TASK-6 | TODO |
| DES-5 | REQ-2 | AC-2.1, AC-2.2 | TASK-7 | TODO |
| DES-6 | REQ-3, REQ-4, REQ-5 | AC-3.1, AC-4.1, AC-5.1, AC-5.2 | TASK-8 | TODO |
| DES-7 | REQ-6 | AC-6.1, AC-6.2 | TASK-9 | TODO |
| DES-8 | REQ-1, REQ-2, REQ-3, REQ-4, REQ-5, REQ-6 | AC-1.1, AC-2.1, AC-3.1, AC-4.1, AC-5.1, AC-6.1 | TASK-10 | TODO |

## Task Completion Status

| TASK | Derived from DES | Implements REQ | Verification Target | Affected Files | Status |
|------|-----------------|----------------|---------------------|----------------|--------|
| TASK-1 | DES-1 | REQ-3, REQ-4 | TEST-1 | `internal/qbt/client.go` | TODO |
| TASK-2 | DES-2 | REQ-3, REQ-4 | TEST-2, CHECK-1 | `internal/qbt/http.go`, `internal/qbt/http_test.go` | TODO |
| TASK-3 | DES-1 | REQ-3, REQ-4 | TEST-3 | `internal/bot/handler_test.go` | TODO |
| TASK-4 | DES-3 | REQ-1 | TEST-4 | `internal/formatter/format.go`, `internal/formatter/format_test.go` | TODO |
| TASK-5 | DES-4 | REQ-2 | TEST-5 | `internal/formatter/format.go` | TODO |
| TASK-6 | DES-4 | REQ-2, REQ-6 | TEST-6 | `internal/formatter/format.go` | TODO |
| TASK-7 | DES-5 | REQ-2 | TEST-7 | `internal/bot/callback.go`, `internal/bot/callback_test.go` | TODO |
| TASK-8 | DES-6 | REQ-3, REQ-4, REQ-5 | TEST-8 | `internal/bot/callback.go`, `internal/bot/callback_test.go` | TODO |
| TASK-9 | DES-7 | REQ-6 | TEST-9 | `internal/bot/callback.go`, `internal/bot/callback_test.go` | TODO |
| TASK-10 | DES-8 | REQ-1, REQ-2, REQ-3, REQ-4, REQ-5, REQ-6 | TEST-10 | `internal/bot/callback.go` | TODO |
| TASK-11 | DES-1 through DES-8 | REQ-1 through REQ-6 | `make gate-all`, `make test-integration` | `docs/features/torrent-remove/traceability.md`, `docs/features/torrent-remove/verification.md` | TODO |
