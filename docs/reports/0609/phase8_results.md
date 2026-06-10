# Phase 8: Integration Test Report

**Test Time:** 19:10:30
**Gateway:** http://localhost:8080
**Total Cases:** 8
**Passed:** 1 / 8

---

## Environment Status

All 12 Docker containers running. Health check passed.

**Known Issues:**
1. `code-agent` plan_only returns non-JSON (plain text), causing `ORCHESTRATOR_PLAN_PARSE_FAILED`. The agent's LLM response format is incompatible with the orchestrator's JSON parser.
2. `group_chat` plan validation fails (`ORCHESTRATOR_PLAN_INVALID`) possibly due to task ordering or dependency constraints.
3. Confirm after SSE timeout returns 502 (stream already closed). This is expected behavior - confirm must be sent within the 120s window.

---

## Case 1: single_chat: React Button component
**Status:** [FAIL]
**Conversation:** `8e714fc974723c58401321f82a96fcf6`
**Run ID:** `run_1780945229820`
**Event Count:** 2

### Plan Info
| Field | Value |
|-------|-------|
| executionPath | `None` |
| planId | `None` |
| confirmActionId | `None` |
| revision | `None` |
| phase | `planning` |
| status | `None` |
| strategy | `None` |

### Assertions
- [FAIL] **execPath=single_chat**: executionPath=None (expected: single_chat)
- [FAIL] **has plan**: planId=None actionId=None
- [OK] **only code-agent in participants**: participants=[]
- [FAIL] **awaiting approval before execute**: phase=planning status=None

### Key Events (2)
- `RUN_STARTED`: {"type": "RUN_STARTED", "runId": "run_1780945229820", "sender": {"type": "agent", "name": "orchestrator"}, "author": "orchestrator", "state": {"phase": "planning"}, "stateDelta": {"phase": "planning"}}
- `RUN_ERROR`: {"type": "RUN_ERROR", "runId": "run_1780945229820", "author": "orchestrator", "error": {"code": "ORCHESTRATOR_PLAN_PARSE_FAILED", "message": "failed to parse agent plan: invalid character 'W' looking for beginning of value"}, "final": true}

---

## Case 2: single_chat: REVISE Go HTTP server
**Status:** [FAIL]
**Conversation:** `dc92c3d0d4030bda6e09897474bc6662`
**Run ID:** `run_1780945289969`
**Event Count:** 2

### Plan Info
| Field | Value |
|-------|-------|
| executionPath | `None` |
| planId | `None` |
| confirmActionId | `None` |
| revision | `None` |
| phase | `planning` |
| status | `None` |
| strategy | `None` |

### Assertions
- [FAIL] **execPath=single_chat**: executionPath=None
- [FAIL] **has plan**: planId=None actionId=None
- [FAIL] **awaiting approval**: phase=planning status=None

### Key Events (2)
- `RUN_STARTED`: {"type": "RUN_STARTED", "runId": "run_1780945289969", "sender": {"type": "agent", "name": "orchestrator"}, "author": "orchestrator", "state": {"phase": "planning"}, "stateDelta": {"phase": "planning"}}
- `RUN_ERROR`: {"type": "RUN_ERROR", "runId": "run_1780945289969", "author": "orchestrator", "error": {"code": "ORCHESTRATOR_PLAN_PARSE_FAILED", "message": "failed to parse agent plan: invalid character 'T' looking for beginning of value"}, "final": true}

---

## Case 3: group_chat: @code-agent @review-agent login
**Status:** [FAIL]
**Conversation:** `47ebecc4e7d49336285ddf7a7aaa8652`
**Run ID:** `run_1780945350089`
**Event Count:** 2

### Plan Info
| Field | Value |
|-------|-------|
| executionPath | `None` |
| planId | `None` |
| confirmActionId | `None` |
| revision | `None` |
| phase | `planning` |
| status | `None` |
| strategy | `None` |

### Assertions
- [FAIL] **execPath=group_chat**: executionPath=None
- [OK] **only code+review in participants**: participants=[]
- [FAIL] **has plan**: planId=None actionId=None

### Key Events (2)
- `RUN_STARTED`: {"type": "RUN_STARTED", "runId": "run_1780945350089", "sender": {"type": "agent", "name": "orchestrator"}, "author": "orchestrator", "state": {"phase": "planning"}, "stateDelta": {"phase": "planning"}}
- `RUN_ERROR`: {"type": "RUN_ERROR", "runId": "run_1780945350089", "author": "orchestrator", "error": {"code": "ORCHESTRATOR_PLAN_INVALID", "message": "Orchestration plan validation failed"}, "final": true}

---

## Case 4: group_chat: AGENT_SELECTION_CONFLICT
**Status:** [PASS]
**Conversation:** `5fe10973bdbd65336c473c0458fdbc8f`
**Run ID:** `run_1780945410204`
**Event Count:** 2

### Plan Info
| Field | Value |
|-------|-------|
| executionPath | `None` |
| planId | `None` |
| confirmActionId | `None` |
| revision | `None` |
| phase | `planning` |
| status | `None` |
| strategy | `None` |

### Assertions
- [OK] **no plan (rejected)**: planId=None actionId=None
- [OK] **RUN_ERROR present**: RUN_ERROR=True
- [OK] **error code = AGENT_SELECTION_CONFLICT**: Checking for AGENT_SELECTION_CONFLICT in error code

### Key Events (2)
- `RUN_STARTED`: {"type": "RUN_STARTED", "runId": "run_1780945410204", "sender": {"type": "agent", "name": "orchestrator"}, "author": "orchestrator", "state": {"phase": "planning"}, "stateDelta": {"phase": "planning"}}
- `RUN_ERROR`: {"type": "RUN_ERROR", "runId": "run_1780945410204", "author": "orchestrator", "error": {"code": "AGENT_SELECTION_CONFLICT", "message": "selectedAgentNames and mentions have no overlap"}, "final": true}

---

## Case 5: auto: Go HTTP server (main_agent_orchestration)
**Status:** [FAIL]
**Conversation:** `394bb01bb3dead5e48ec73291135f111`
**Run ID:** `run_1780945410273`
**Event Count:** 12

### Plan Info
| Field | Value |
|-------|-------|
| executionPath | `main_agent_orchestration` |
| planId | `plan_1780945410273` |
| confirmActionId | `plan_1780945410273` |
| revision | `1` |
| phase | `awaiting_confirmation` |
| status | `awaiting_confirmation` |
| planOwner | `{"type": "main_agent", "agentName": "main-agent", "isMainAgent": true}` |
| strategy | `None` |
| title | Auto-orchestrated plan with 1 agent(s) |
| summary | Auto-orchestrated plan with 1 agent(s) |

**participants:**
- [R][S] `code-agent` - ?
- [O][ ] `document-agent` - ?

**candidateParticipants:**
- `code-agent` - ?
- `document-agent` - ?

**defaultSelectedParticipants:** ['code-agent']

**tasks (1):**
- `task-1` -> `code-agent`: Write a Go HTTP server.

**allowedActions:** ['approve', 'revise', 'cancel']

### Assertions
- [OK] **execPath=main_agent_orch**: executionPath=main_agent_orchestration
- [OK] **has plan**: planId=plan_1780945410273 actionId=plan_1780945410273
- [OK] **has candidate recommendations**: candidates=2, defaults=1
- [OK] **awaiting approval**: phase=awaiting_confirmation status=awaiting_confirmation
- [FAIL] **confirm accepted (200)**: confirm_status=502

### Confirm Action
- **HTTP 502**: status=?
- **Error:** `HITL confirmation failed` - ?

### Key Events (12)
- `RUN_STARTED`: {"type": "RUN_STARTED", "runId": "run_1780945410273", "sender": {"type": "agent", "name": "orchestrator"}, "author": "orchestrator", "state": {"phase": "planning"}, "stateDelta": {"phase": "planning"}}
- `STATE_UPDATE`: {"type": "STATE_UPDATE", "runId": "run_1780945410273", "author": "orchestrator", "state": {"executionPath": "main_agent_orchestration", "intentSummary": "Auto-orchestrated plan with 1 agent(s)", "phase": "planning", "planId": "plan_1780945410273", "p
- `ACTIVITY_SNAPSHOT`: {"type": "ACTIVITY_SNAPSHOT", "runId": "run_1780945410273", "sender": {"type": "agent", "name": "orchestrator"}, "author": "orchestrator", "activity": {"activityId": "plan_1780945410273", "activityType": "plan_approval", "status": "awaiting_confirmat
- `STATE_UPDATE`: {"type": "STATE_UPDATE", "runId": "run_1780945410273", "author": "orchestrator", "state": {"confirmationActionId": "plan_1780945410273", "executionPath": "main_agent_orchestration", "intentSummary": "Auto-orchestrated plan with 1 agent(s)", "phase": 
- `STATE_UPDATE`: {"type": "STATE_UPDATE", "runId": "run_1780945410273", "author": "orchestrator", "state": {"confirmationActionId": "plan_1780945410273", "heartbeat": true, "phase": "awaiting_confirmation", "requiresConfirmation": true}, "stateDelta": {"confirmationA
- `STATE_UPDATE`: {"type": "STATE_UPDATE", "runId": "run_1780945410273", "author": "orchestrator", "state": {"confirmationActionId": "plan_1780945410273", "heartbeat": true, "phase": "awaiting_confirmation", "requiresConfirmation": true}, "stateDelta": {"confirmationA
- `STATE_UPDATE`: {"type": "STATE_UPDATE", "runId": "run_1780945410273", "author": "orchestrator", "state": {"confirmationActionId": "plan_1780945410273", "heartbeat": true, "phase": "awaiting_confirmation", "requiresConfirmation": true}, "stateDelta": {"confirmationA
- `STATE_UPDATE`: {"type": "STATE_UPDATE", "runId": "run_1780945410273", "author": "orchestrator", "state": {"confirmationActionId": "plan_1780945410273", "heartbeat": true, "phase": "awaiting_confirmation", "requiresConfirmation": true}, "stateDelta": {"confirmationA
- `STATE_UPDATE`: {"type": "STATE_UPDATE", "runId": "run_1780945410273", "author": "orchestrator", "state": {"confirmationActionId": "plan_1780945410273", "heartbeat": true, "phase": "awaiting_confirmation", "requiresConfirmation": true}, "stateDelta": {"confirmationA
- `STATE_UPDATE`: {"type": "STATE_UPDATE", "runId": "run_1780945410273", "author": "orchestrator", "state": {"confirmationActionId": "plan_1780945410273", "heartbeat": true, "phase": "awaiting_confirmation", "requiresConfirmation": true}, "stateDelta": {"confirmationA
- `STATE_UPDATE`: {"type": "STATE_UPDATE", "runId": "run_1780945410273", "author": "orchestrator", "state": {"confirmationActionId": "plan_1780945410273", "heartbeat": true, "phase": "awaiting_confirmation", "requiresConfirmation": true}, "stateDelta": {"confirmationA
- `RUN_ERROR`: {"type": "RUN_ERROR", "runId": "run_1780945410273", "author": "orchestrator", "error": {"code": "ORCHESTRATOR_CONFIRM_TIMEOUT", "message": "Plan confirmation timed out after 120s. Please retry or select a specific agent."}, "final": true}

---

## Case 6: auto: Login (React frontend + Go backend)
**Status:** [FAIL]
**Conversation:** `4454b659112aa6107291078cae4e2795`
**Run ID:** `run_1780945530332`
**Event Count:** 12

### Plan Info
| Field | Value |
|-------|-------|
| executionPath | `main_agent_orchestration` |
| planId | `plan_1780945530333` |
| confirmActionId | `plan_1780945530333` |
| revision | `1` |
| phase | `awaiting_confirmation` |
| status | `awaiting_confirmation` |
| planOwner | `{"type": "main_agent", "agentName": "main-agent", "isMainAgent": true}` |
| strategy | `None` |
| title | Auto-orchestrated plan with 2 agent(s) |
| summary | Auto-orchestrated plan with 2 agent(s) |

**participants:**
- [R][S] `web-agent` - ?
- [R][S] `code-agent` - ?

**candidateParticipants:**
- `web-agent` - ?
- `code-agent` - ?

**defaultSelectedParticipants:** ['web-agent', 'code-agent']

**tasks (2):**
- `task-1` -> `web-agent`: Build a login feature: React frontend page, Go backend login API.
- `task-2` -> `code-agent`: Build a login feature: React frontend page, Go backend login API.

**allowedActions:** ['approve', 'revise', 'cancel']

### Assertions
- [OK] **execPath=main_agent_orch**: executionPath=main_agent_orchestration
- [OK] **has plan**: planId=plan_1780945530333 actionId=plan_1780945530333
- [OK] **recommends code/web-agent**: candidates=['web-agent', 'code-agent'], defaults=['web-agent', 'code-agent']
- [FAIL] **participants have reason**: Checking reason field on participants/candidates
- [FAIL] **confirm accepted (200)**: confirm_status=502

### Confirm Action
- **HTTP 502**: status=?
- **Error:** `HITL confirmation failed` - ?

### Key Events (12)
- `RUN_STARTED`: {"type": "RUN_STARTED", "runId": "run_1780945530332", "sender": {"type": "agent", "name": "orchestrator"}, "author": "orchestrator", "state": {"phase": "planning"}, "stateDelta": {"phase": "planning"}}
- `STATE_UPDATE`: {"type": "STATE_UPDATE", "runId": "run_1780945530332", "author": "orchestrator", "state": {"executionPath": "main_agent_orchestration", "intentSummary": "Auto-orchestrated plan with 2 agent(s)", "phase": "planning", "planId": "plan_1780945530333", "p
- `ACTIVITY_SNAPSHOT`: {"type": "ACTIVITY_SNAPSHOT", "runId": "run_1780945530332", "sender": {"type": "agent", "name": "orchestrator"}, "author": "orchestrator", "activity": {"activityId": "plan_1780945530333", "activityType": "plan_approval", "status": "awaiting_confirmat
- `STATE_UPDATE`: {"type": "STATE_UPDATE", "runId": "run_1780945530332", "author": "orchestrator", "state": {"confirmationActionId": "plan_1780945530333", "executionPath": "main_agent_orchestration", "intentSummary": "Auto-orchestrated plan with 2 agent(s)", "phase": 
- `STATE_UPDATE`: {"type": "STATE_UPDATE", "runId": "run_1780945530332", "author": "orchestrator", "state": {"confirmationActionId": "plan_1780945530333", "heartbeat": true, "phase": "awaiting_confirmation", "requiresConfirmation": true}, "stateDelta": {"confirmationA
- `STATE_UPDATE`: {"type": "STATE_UPDATE", "runId": "run_1780945530332", "author": "orchestrator", "state": {"confirmationActionId": "plan_1780945530333", "heartbeat": true, "phase": "awaiting_confirmation", "requiresConfirmation": true}, "stateDelta": {"confirmationA
- `STATE_UPDATE`: {"type": "STATE_UPDATE", "runId": "run_1780945530332", "author": "orchestrator", "state": {"confirmationActionId": "plan_1780945530333", "heartbeat": true, "phase": "awaiting_confirmation", "requiresConfirmation": true}, "stateDelta": {"confirmationA
- `STATE_UPDATE`: {"type": "STATE_UPDATE", "runId": "run_1780945530332", "author": "orchestrator", "state": {"confirmationActionId": "plan_1780945530333", "heartbeat": true, "phase": "awaiting_confirmation", "requiresConfirmation": true}, "stateDelta": {"confirmationA
- `STATE_UPDATE`: {"type": "STATE_UPDATE", "runId": "run_1780945530332", "author": "orchestrator", "state": {"confirmationActionId": "plan_1780945530333", "heartbeat": true, "phase": "awaiting_confirmation", "requiresConfirmation": true}, "stateDelta": {"confirmationA
- `STATE_UPDATE`: {"type": "STATE_UPDATE", "runId": "run_1780945530332", "author": "orchestrator", "state": {"confirmationActionId": "plan_1780945530333", "heartbeat": true, "phase": "awaiting_confirmation", "requiresConfirmation": true}, "stateDelta": {"confirmationA
- `STATE_UPDATE`: {"type": "STATE_UPDATE", "runId": "run_1780945530332", "author": "orchestrator", "state": {"confirmationActionId": "plan_1780945530333", "heartbeat": true, "phase": "awaiting_confirmation", "requiresConfirmation": true}, "stateDelta": {"confirmationA
- `RUN_ERROR`: {"type": "RUN_ERROR", "runId": "run_1780945530332", "author": "orchestrator", "error": {"code": "ORCHESTRATOR_CONFIRM_TIMEOUT", "message": "Plan confirmation timed out after 120s. Please retry or select a specific agent."}, "final": true}

---

## Case 7: auto: Remove test-agent from plan
**Status:** [FAIL]
**Conversation:** `d1b8bf2a97c5062b18827be1e15f7d9d`
**Run ID:** `run_1780945650388`
**Event Count:** 12

### Plan Info
| Field | Value |
|-------|-------|
| executionPath | `main_agent_orchestration` |
| planId | `plan_1780945650388` |
| confirmActionId | `plan_1780945650388` |
| revision | `1` |
| phase | `awaiting_confirmation` |
| status | `awaiting_confirmation` |
| planOwner | `{"type": "main_agent", "agentName": "main-agent", "isMainAgent": true}` |
| strategy | `None` |
| title | Auto-orchestrated plan with 1 agent(s) |
| summary | Auto-orchestrated plan with 1 agent(s) |

**participants:**
- [R][ ] `web-agent` - ?

**candidateParticipants:**
- `web-agent` - ?

**defaultSelectedParticipants:** ['web-agent']

**tasks (1):**
- `task-1` -> `web-agent`: Build login feature with tests.

**allowedActions:** ['approve', 'revise', 'cancel']

### Assertions
- [OK] **execPath=main_agent_orch**: executionPath=main_agent_orchestration
- [OK] **has plan**: planId=plan_1780945650388 actionId=plan_1780945650388
- [FAIL] **confirm accepted (200)**: confirm_status=502

### Confirm Action
- **HTTP 502**: status=?
- **Error:** `HITL confirmation failed` - ?

### Key Events (12)
- `RUN_STARTED`: {"type": "RUN_STARTED", "runId": "run_1780945650388", "sender": {"type": "agent", "name": "orchestrator"}, "author": "orchestrator", "state": {"phase": "planning"}, "stateDelta": {"phase": "planning"}}
- `STATE_UPDATE`: {"type": "STATE_UPDATE", "runId": "run_1780945650388", "author": "orchestrator", "state": {"executionPath": "main_agent_orchestration", "intentSummary": "Auto-orchestrated plan with 1 agent(s)", "phase": "planning", "planId": "plan_1780945650388", "p
- `ACTIVITY_SNAPSHOT`: {"type": "ACTIVITY_SNAPSHOT", "runId": "run_1780945650388", "sender": {"type": "agent", "name": "orchestrator"}, "author": "orchestrator", "activity": {"activityId": "plan_1780945650388", "activityType": "plan_approval", "status": "awaiting_confirmat
- `STATE_UPDATE`: {"type": "STATE_UPDATE", "runId": "run_1780945650388", "author": "orchestrator", "state": {"confirmationActionId": "plan_1780945650388", "executionPath": "main_agent_orchestration", "intentSummary": "Auto-orchestrated plan with 1 agent(s)", "phase": 
- `STATE_UPDATE`: {"type": "STATE_UPDATE", "runId": "run_1780945650388", "author": "orchestrator", "state": {"confirmationActionId": "plan_1780945650388", "heartbeat": true, "phase": "awaiting_confirmation", "requiresConfirmation": true}, "stateDelta": {"confirmationA
- `STATE_UPDATE`: {"type": "STATE_UPDATE", "runId": "run_1780945650388", "author": "orchestrator", "state": {"confirmationActionId": "plan_1780945650388", "heartbeat": true, "phase": "awaiting_confirmation", "requiresConfirmation": true}, "stateDelta": {"confirmationA
- `STATE_UPDATE`: {"type": "STATE_UPDATE", "runId": "run_1780945650388", "author": "orchestrator", "state": {"confirmationActionId": "plan_1780945650388", "heartbeat": true, "phase": "awaiting_confirmation", "requiresConfirmation": true}, "stateDelta": {"confirmationA
- `STATE_UPDATE`: {"type": "STATE_UPDATE", "runId": "run_1780945650388", "author": "orchestrator", "state": {"confirmationActionId": "plan_1780945650388", "heartbeat": true, "phase": "awaiting_confirmation", "requiresConfirmation": true}, "stateDelta": {"confirmationA
- `STATE_UPDATE`: {"type": "STATE_UPDATE", "runId": "run_1780945650388", "author": "orchestrator", "state": {"confirmationActionId": "plan_1780945650388", "heartbeat": true, "phase": "awaiting_confirmation", "requiresConfirmation": true}, "stateDelta": {"confirmationA
- `STATE_UPDATE`: {"type": "STATE_UPDATE", "runId": "run_1780945650388", "author": "orchestrator", "state": {"confirmationActionId": "plan_1780945650388", "heartbeat": true, "phase": "awaiting_confirmation", "requiresConfirmation": true}, "stateDelta": {"confirmationA
- `STATE_UPDATE`: {"type": "STATE_UPDATE", "runId": "run_1780945650388", "author": "orchestrator", "state": {"confirmationActionId": "plan_1780945650388", "heartbeat": true, "phase": "awaiting_confirmation", "requiresConfirmation": true}, "stateDelta": {"confirmationA
- `RUN_ERROR`: {"type": "RUN_ERROR", "runId": "run_1780945650388", "author": "orchestrator", "error": {"code": "ORCHESTRATOR_CONFIRM_TIMEOUT", "message": "Plan confirmation timed out after 120s. Please retry or select a specific agent."}, "final": true}

---

## Case 8: duplicate click: idempotency
**Status:** [FAIL]
**Conversation:** `3027e0785ed6fb253dc29dfaf6705dbd`
**Run ID:** `run_1780945770435`
**Event Count:** 2

### Plan Info
| Field | Value |
|-------|-------|
| executionPath | `None` |
| planId | `None` |
| confirmActionId | `None` |
| revision | `None` |
| phase | `planning` |
| status | `None` |
| strategy | `None` |

### Assertions
- [FAIL] **execPath=single_chat**: executionPath=None

### Key Events (2)
- `RUN_STARTED`: {"type": "RUN_STARTED", "runId": "run_1780945770435", "sender": {"type": "agent", "name": "orchestrator"}, "author": "orchestrator", "state": {"phase": "planning"}, "stateDelta": {"phase": "planning"}}
- `RUN_ERROR`: {"type": "RUN_ERROR", "runId": "run_1780945770435", "author": "orchestrator", "error": {"code": "ORCHESTRATOR_PLAN_PARSE_FAILED", "message": "failed to parse agent plan: invalid character 'T' looking for beginning of value"}, "final": true}

---

## Summary

| # | Case | execPath | planId | Confirm | Result |
|---|------|----------|--------|---------|--------|
| 1 | single_chat: React Button component | `None` | N | None | [FAIL] |
| 2 | single_chat: REVISE Go HTTP server | `None` | N | None | [FAIL] |
| 3 | group_chat: @code-agent @review-age | `None` | N | None | [FAIL] |
| 4 | group_chat: AGENT_SELECTION_CONFLIC | `None` | N | None | [OK] |
| 5 | auto: Go HTTP server (main_agent_or | `main_agent_orchestration` | Y | 502 | [FAIL] |
| 6 | auto: Login (React frontend + Go ba | `main_agent_orchestration` | Y | 502 | [FAIL] |
| 7 | auto: Remove test-agent from plan | `main_agent_orchestration` | Y | 502 | [FAIL] |
| 8 | duplicate click: idempotency | `None` | N | None | [FAIL] |

---

*Report generated: 19:10:30*
*Environment: Docker Compose new-arch, all 12 agents healthy*