# Plan Approval Contract

**Status:** Active
**Owner:** AgentHub
**Primary source:** Multi-Path Plan Confirmation Design
**Last updated:** 2026-06-09 (Phase 4 — complete approval lifecycle)


## Authoritative source order

1. `docs/superpowers/specs/2026-05-26-module-separation-and-runtime-redesign.md`
2. `docs/superpowers/specs/2026-05-27-module-separation-runtime-redesign.md`
3. `docs/contracts/chat-execution-path.md`
4. `docs/contracts/participant-boundary.md`
5. `docs/contracts/agui-events.md`
6. This contract

Engineering details MUST follow the module separation redesign. Old `server/` and root `agents/` are legacy reference implementations unless a task explicitly says otherwise.


## Scope

Plan approval governs the full lifecycle from plan proposal through user confirmation to execution. It applies to all three execution paths: `single_chat`, `group_chat`, and `main_agent_orchestration`.

## Key concepts

### PendingPlan

A PendingPlan represents an execution plan that has been proposed but not yet approved, revised, or cancelled. It is bound to a single `runId` and `planId`.

| Field | Type | Description |
|---|---|---|
| `runId` | string | Run this plan belongs to |
| `planId` | string | Unique plan identifier |
| `revision` | int | Monotonic revision counter, starts at 1 |
| `executionPath` | ChatExecutionPath | `single_chat` / `group_chat` / `main_agent_orchestration` |
| `planOwner` | PlanOwner | Who owns this plan (agent / group_coordinator / main_agent) |
| `participants` | PlanParticipant[] | All participants in this plan |
| `candidateParticipants` | PlanParticipant[] | (main_agent_orchestration only) Candidates the main-agent recommends |
| `defaultSelectedParticipants` | PlanParticipant[] | (main_agent_orchestration only) Main-agent's default selection |
| `selectedParticipants` | PlanParticipant[] | User's final confirmed selection. Unset until approval; current proposal defaults live in `defaultSelectedParticipants`. |
| `steps` | ExecutionPlanStep[] | Ordered execution steps |
| `status` | PlanStatus | Current lifecycle status |
| `idempotencyKey` | string | Client-generated UUID for deduplication |
| `feedbackHistory` | PlanFeedback[] | All feedback/revision records |
| `createdAt` | timestamp | When this plan was created |
| `expiresAt` | timestamp | When the approval window closes |

### PlanFeedback

| Field | Type | Description |
|---|---|---|
| `feedbackId` | string | Unique feedback identifier |
| `planId` | string | The plan this feedback applies to |
| `revision` | int | The revision being commented on |
| `feedback` | string | User's feedback text (non-empty) |
| `createdAt` | timestamp | When the feedback was submitted |

### PlanStatus

| Value | Description |
|---|---|
| `awaiting_confirmation` | PLAN_PROPOSAL emitted, waiting for user action |
| `revising_plan` | User requested revision, new plan being generated |
| `executing` | Plan approved, execution in progress |
| `completed` | Execution finished successfully |
| `cancelled` | User cancelled the plan before execution |
| `expired` | Approval window expired without user action |
| `failed` | Execution failed with error |

## Plan lifecycle

```
awaiting_confirmation ──┬──> executing ──> completed
       │    ↑              │
       │    │              ├──> cancelled
       │    └── revise ────┘
       │                   │
       └──> expired        └──> failed
```

### Lifecycle rules

- A conversation MUST NOT have more than one active PendingPlan at a time.
- `revision` starts at 1 and increments by 1 on each REQUEST_PLAN_REVISION.
- Once a plan status is `completed`, `cancelled`, `expired`, or `failed`, it MUST NOT be executed.
- `expired` plans reach their `expiresAt` time without user action.
- `failed` plans encountered an unrecoverable error during execution.

## PLAN_PROPOSAL

A plan proposal is emitted to the frontend as an SSE event stream within the existing `/api/chat` connection.

### Event sequence

```
RUN_STARTED
  state: { phase: "planning", executionPath: "..." }
STATE_UPDATE
  state: { phase: "planning", ... }
TOOL_CALL_START
  toolCallId: <planId>
  toolCallName: "confirm_plan"
TOOL_CALL_ARGS
  toolCallId: <planId>
  delta: <JSON args block — see schema below>
TOOL_CALL_END
  toolCallId: <planId>
STATE_UPDATE
  state: { phase: "waiting_user_approval", requiresConfirmation: true, ... }
```

### confirm_plan args schema

```json
{
  "runId": "string",
  "planId": "string",
  "revision": 1,
  "executionPath": "single_chat | group_chat | main_agent_orchestration",
  "planOwner": {
    "type": "agent | group_coordinator | main_agent",
    "agentName": "string",
    "isMainAgent": false
  },
  "strategy": "single | ordered_parallel | sequential",
  "intentSummary": "string",
  "participants": [
    {
      "agentName": "string",
      "required": true,
      "reason": "string",
      "selected": true
    }
  ],
  "candidateParticipants": [],
  "defaultSelectedParticipants": [],
  "tasks": [
    {
      "taskId": "string",
      "agentName": "string",
      "content": "string",
      "dependsOn": [],
      "priority": 0,
      "riskLevel": "low"
    }
  ],
  "requiresConfirmation": true,
  "expiresAt": 120000
}
```

### PATH-SPECIFIC fields

| executionPath | candidateParticipants | defaultSelectedParticipants |
|---|---|---|
| `single_chat` | empty — only one agent | empty |
| `group_chat` | empty — participants are fixed | empty |
| `main_agent_orchestration` | **REQUIRED** — relevant candidates selected from all enabled agents | **REQUIRED** — main-agent's recommended default selection |

### Critical rule

**No Agent MUST be invoked before the user approves the plan.** The only content that may be emitted before approval is the PLAN_PROPOSAL itself (plan metadata, steps, participants).

## APPROVE_PLAN

### Request

```
POST /api/runs/{runId}/confirm
```

```json
{
  "runId": "string (required)",
  "action": "approve",
  "planId": "string (required)",
  "revision": 1,
  "idempotencyKey": "uuid (required)",
  "selectedParticipants": ["agentName1", "agentName2"]
}
```

### APPROVE semantics

`APPROVE_PLAN` means: "I accept the current proposal as-is." It MUST NOT be used to change participants or question the plan.

- For `single_chat`, `selectedParticipants` may be omitted or must equal `[currentAgent]`.
- For `group_chat`, `selectedParticipants` may be omitted or must equal the fixed participant set from the current proposal.
- For `main_agent_orchestration`, `selectedParticipants` may be omitted; when omitted, the server uses `defaultSelectedParticipants`. If provided, it MUST equal `defaultSelectedParticipants` for the current proposal.
- Any participant add/remove/change in `main_agent_orchestration` MUST be sent as `REQUEST_PLAN_REVISION`, not `APPROVE_PLAN`.
- If `APPROVE_PLAN` carries a changed participant set, the backend MUST return `409 PARTICIPANT_CHANGE_REQUIRES_REVISION` and keep the plan in `awaiting_confirmation`.

### Response

```json
{ "status": "acknowledged", "planId": "...", "revision": 1 }
```

### Continuation mechanism

The SSE connection established by `POST /api/chat` remains open during the approval wait. After the confirm endpoint returns `acknowledged`, the Orchestrator confirms the signal via its internal channel and continues execution on the **same SSE stream**:

```
... (SSE connection already open from /api/chat)

--- User sends APPROVE_PLAN via POST /api/runs/{runId}/confirm ---

STATE_UPDATE { phase: "executing", requiresConfirmation: false }
TEXT_MESSAGE_START / CONTENT / END  (execution results)
RUN_FINISHED { status: "completed" }
```

**No new `POST /api/chat` call is required.** The original SSE stream carries the full lifecycle.

### Validation

1. `runId` MUST reference an existing PendingPlan → `404 PLAN_NOT_FOUND`
2. `planId` MUST match the current PendingPlan → `409 PLAN_NOT_FOUND`
3. `revision` MUST match the current PendingPlan revision → `409 PLAN_REVISION_MISMATCH`
4. Plan status MUST be `awaiting_confirmation` → `409 INVALID_RUN_STATE`
5. `idempotencyKey` MUST be present and non-empty → `400 IDEMPOTENCY_KEY_REQUIRED`
6. Idempotency: same key + same hash + status=processing → `409 CONFIRMATION_IN_PROGRESS`
7. Idempotency: same key + same hash + status=completed → `200` (cached response)
8. Idempotency: same key + different hash → `409 IDEMPOTENCY_CONFLICT`
9. `selectedParticipants` MUST be a subset of `participants` → `409 AGENT_BOUNDARY_VIOLATION`
10. `selectedParticipants` MUST include all `required: true` participants → `400 REQUIRED_PARTICIPANT_MISSING`
11. `selectedParticipants` MUST be a subset of the available boundary (`availableBoundary`) → `409 AGENT_BOUNDARY_VIOLATION`
12. For `single_chat`: `selectedParticipants` MUST contain exactly the single agent → `409 AGENT_BOUNDARY_VIOLATION`
13. For `group_chat`: `selectedParticipants` MUST be a subset of the available boundary → `409 AGENT_BOUNDARY_VIOLATION`
14. For `main_agent_orchestration`: `selectedParticipants`, if present, MUST equal the current proposal's `defaultSelectedParticipants`. Any difference → `409 PARTICIPANT_CHANGE_REQUIRES_REVISION`

### Error responses

| Status | Error code | Condition |
|---|---|---|
| 400 | `IDEMPOTENCY_KEY_REQUIRED` | idempotencyKey is missing or empty |
| 400 | `REQUIRED_PARTICIPANT_MISSING` | A required participant was removed |
| 400 | `REVISION_INPUT_REQUIRED` | action=revise but feedback is empty |
| 404 | `PLAN_NOT_FOUND` | No plan is awaiting approval for this runId |
| 409 | `PLAN_NOT_FOUND` | planId doesn't match current PendingPlan (plan was replaced) |
| 409 | `PLAN_REVISION_MISMATCH` | Revision in request doesn't match current revision |
| 409 | `INVALID_RUN_STATE` | Plan is not in `awaiting_confirmation` status (cancelled/executing/completed/etc.) |
| 409 | `IDEMPOTENCY_CONFLICT` | Same idempotencyKey with different payload |
| 409 | `CONFIRMATION_IN_PROGRESS` | A confirmation with this idempotencyKey is already being processed |
| 409 | `AGENT_BOUNDARY_VIOLATION` | selectedParticipants contains agents outside participants/boundary |
| 409 | `PARTICIPANT_CHANGE_REQUIRES_REVISION` | APPROVE_PLAN attempted to change selectedParticipants |

## REQUEST_PLAN_REVISION

### Request

```
POST /api/runs/{runId}/confirm
```

```json
{
  "runId": "string (required)",
  "action": "revise",
  "planId": "string (required)",
  "revision": 1,
  "idempotencyKey": "uuid (required)",
  "feedback": "string (optional if selectedParticipants changed)",
  "selectedParticipants": ["agentName1"]
}
```

### Response

```json
{ "status": "acknowledged", "planId": "<new-planId>", "revision": 2 }
```

### Revision dispatch rules

REVISION MUST be handled by the **original planOwner**, not a generic planner:

| executionPath | planOwner | Revision handler |
|---|---|---|
| `single_chat` | `{type:"agent", agentName:"<name>"}` | Call the **same Agent** with `mode=plan_only`, passing the original plan + feedback. Agent returns a new plan with revision+1. |
| `group_chat` | `{type:"group_coordinator"}` | Call the **group coordinator** (Orchestrator internal), passing the original plan + fixed participants + feedback. Coordinator returns a new production plan with revision+1. |
| `main_agent_orchestration` | `{type:"main_agent", agentName:"main-agent"}` | Call the **main-agent**, passing the original plan + feedback + user-requested participant changes. Main-agent returns a new PLAN_PROPOSAL with revision+1. |

### Post-revision flow

After the revision plan is generated:
1. New `planId` is assigned.
2. `revision` is incremented by 1.
3. A new `ACTIVITY_SNAPSHOT` event with the revised plan is emitted on the **same SSE stream**.
4. The frontend replaces the old plan card with the new plan (revision updated).
5. The plan returns to `awaiting_confirmation` status.

### Validation

- `feedback` MUST be non-empty (whitespace-only is treated as empty).
- If feedback is empty, return `400 REVISION_INPUT_REQUIRED`.
- `runId`, `planId`, `revision`, `idempotencyKey`, and status validations are shared with APPROVE (see shared validation helpers).
- Plan status MUST be `awaiting_confirmation`.
- `selectedParticipants`, if provided, MUST still include all `required: true` participants.

## CANCEL_PLAN

### Request

```
POST /api/runs/{runId}/confirm
```

```json
{
  "runId": "string (required)",
  "action": "cancel",
  "planId": "string (required)",
  "revision": 1,
  "idempotencyKey": "uuid (required)"
}
```

### Response

```json
{ "status": "acknowledged", "planId": "...", "revision": 1 }
```

### SSE continuation (on the existing /api/chat stream)

```
STATE_UPDATE { phase: "cancelled" }
RUN_FINISHED { status: "cancelled" }
```

### Critical rule

**CANCEL_PLAN MUST NOT emit `RUN_ERROR`.** User-initiated cancellation is a normal lifecycle transition, not a system error. The SSE stream ends with `RUN_FINISHED { status: "cancelled" }` preceded by a `STATE_UPDATE { phase: "cancelled" }`.

## Idempotency key rules

| Scenario | HTTP status | Response |
|---|---|---|
| Same `idempotencyKey` + same payload (runId, action, planId, revision, selectedParticipants, feedback) | `200` | Return first result; do NOT re-execute or re-revise |
| Same `idempotencyKey` + different payload | `409` | `{"error": "IDEMPOTENCY_KEY_CONFLICT", "message": "Same idempotencyKey used with different request parameters"}` |
| Different `idempotencyKey`, but plan already approved/executing/completed/cancelled | `409` | `{"error": "INVALID_RUN_STATE", "currentState": "executing"}` |

Idempotency keys SHOULD be client-generated UUIDs. The server MUST store the `(idempotencyKey, payload_hash, result)` tuple for at least the duration of the plan lifecycle.

## Approval before execution

**No Agent MUST be invoked for execution before the user approves the plan.** This applies to ALL execution paths:

- `single_chat`: The selected Agent MUST NOT execute before APPROVE_PLAN. It may only generate a plan via `plan_only` mode.
- `group_chat`: No participant Agent MUST be dispatched before APPROVE_PLAN.
- `main_agent_orchestration`: No child Agent MUST be dispatched before APPROVE_PLAN. Candidate agents listed in `candidateParticipants` are recommendations only, not confirmed executors. Participant changes before approval require REQUEST_PLAN_REVISION.

## Phase 1 / Phase 2 persistence note

- Phase 1: Plan state is in-memory only (current HITL behavior). No persistence required.
- Phase 2: PendingPlan remains in-memory. Idempotency keys stored in memory.
- Later phases: PendingPlan and PlanFeedback MAY be persisted to SQLite/MySQL.

## Relationship to existing HITL (REQUIRE_PLAN_CONFIRMATION)

This contract extends and replaces the existing `REQUIRE_PLAN_CONFIRMATION=true` HITL flow:

| Aspect | Status | Description |
|---|---|---|
| Confirmation trigger | **Implemented** | All execution paths (`single_chat`, `group_chat`, `main_agent_orchestration`) |
| User actions | **Implemented** | Approve / Revise / Cancel |
| Cancel behavior | **Implemented** | `STATE_UPDATE(cancelled)` + `RUN_FINISHED(cancelled)` -- no `RUN_ERROR` |
| Revision | **Implemented** | Full revision loop with feedback + feedbackHistory recording |
| Idempotency | **Implemented** | Enforced per idempotencyKey; processing record created before channel send |
| Participant selection | **Implemented** | Optional participants can be unchecked directly in approve; no revision required |
| Tombstone | **Implemented** | Terminal PendingPlan preserved after cancel/complete; channel closed |
