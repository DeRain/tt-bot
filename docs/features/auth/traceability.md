---
title: User Authorization — Traceability Matrix
feature_id: auth
status: implemented
last_updated: 2026-03-15
---

# User Authorization — Traceability Matrix

| Requirement | Acceptance Criteria | Design | Plan Tasks | Implementation Evidence | Verification | Status |
|-------------|---------------------|--------|------------|-------------------------|--------------|--------|
| REQ-1 | AC-1.1, AC-1.2 | DES-1, DES-2 | TASK-2, TASK-3 | internal/bot/auth.go (`IsAllowed`), internal/bot/handler.go (auth gate in `HandleUpdate`) | TEST-3, TEST-4, TEST-5 | Complete |
| REQ-2 | AC-2.1 | DES-1 | TASK-2 | internal/bot/auth.go (`IsAllowed` returns true for allowed IDs) | TEST-3 | Complete |
| REQ-3 | AC-3.1, AC-3.2 | DES-3 | TASK-1 | internal/config/config.go (comma-split, int64 parse) | TEST-1, TEST-2 | Complete |
| REQ-4 | AC-4.1 | DES-3 | TASK-1 | internal/config/config.go (minimum 1 user validation) | TEST-2 | Complete |
| REQ-5 | AC-5.1 | DES-2 | TASK-3 | internal/bot/handler.go (sends "You are not authorized" message) | TEST-5 | Complete |
