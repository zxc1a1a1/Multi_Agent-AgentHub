# Platform API Contract

**Status:** Active
**Owner:** AgentHub
**Primary source:** Module Separation & Runtime Redesign


## Authoritative source order

1. `docs/superpowers/specs/2026-05-26-module-separation-and-runtime-redesign.md`
2. `docs/superpowers/specs/2026-05-27-module-separation-runtime-redesign.md`
3. PDR product goals only, not old module layout
4. Sprint/UML as supporting product/demo references only

Engineering details MUST follow the module separation redesign. Old `server/` and root `agents/` are legacy reference implementations unless a task explicitly says otherwise.


## Scope

Public API is Gateway-only. It is the browser-facing surface and must not expose Orchestrator or Child Agent endpoints.

## Formal endpoints

| Method | Path | Purpose | Status |
|---|---|---|---|
| POST | `/agui/runs` | Start a run and receive SSE AG-UI stream | Target |
| POST | `/api/chat` | Compatibility wrapper over `/agui/runs` | Temporary |
| GET | `/api/conversations` | List conversations | Active |
| POST | `/api/conversations` | Create conversation | Active |
| GET | `/api/conversations/{id}/messages` | Fetch message history | Active |
| GET | `/api/agents` | List agents proxied from Orchestrator | Active |
| GET | `/health` | Gateway health | Active |

## Run request

```json
{
  "threadId": "conversation-id",
  "runId": "optional-run-id",
  "messages": [{"role":"user","content":"..."}],
  "tools": [{"name":"code_preview","description":"...","schema_json":"{}"}],
  "metadata": {"conversationType":"single|group"}
}
```

## Streaming response

`/agui/runs` returns `text/event-stream` using AG-UI events defined by `agui-events.md`.

## POST /api/runs/{runId}/confirm (Plan Approval Actions)

Extended for plan approval v1.2. See `docs/contracts/plan-approval.md`.

### Request

```json
{
  "runId": "string (required)",
  "action": "approve | revise | cancel (required)",
  "planId": "string (required)",
  "revision": 1,
  "idempotencyKey": "uuid (required)",
  "feedback": "string (required when action=revise unless selectedParticipants changed)",
  "selectedParticipants": ["agentName1", "agentName2"]
}
```

### Response

```json
{ "status": "acknowledged", "planId": "string", "revision": 2 }
```

### Action semantics

| action | Effect on SSE stream (same /api/chat connection) |
|---|---|
| `approve` | Accepts the current proposal as-is. Orchestrator resumes execution. SSE continues with `STATE_UPDATE(phase=executing)` → execution events → `RUN_FINISHED`. If participants are changed, return `PARTICIPANT_CHANGE_REQUIRES_REVISION`. |
| `revise` | Orchestrator calls original planOwner to regenerate plan (revision+1) from feedback and/or requested participant changes. SSE continues with new `confirm_plan` sequence → `STATE_UPDATE(phase=waiting_user_approval)`. |
| `cancel` | Orchestrator emits `STATE_UPDATE(phase=cancelled)` → `RUN_FINISHED(status=cancelled)`. No `RUN_ERROR`. |

### Error responses

| Status | Error code | Condition |
|---|---|---|
| 400 | `REVISION_INPUT_REQUIRED` | action=revise but both feedback and participant changes are empty |
| 400 | `REQUIRED_PARTICIPANT_MISSING` | Required participant was removed |
| 400 | `AGENT_SELECTION_CONFLICT` | (if validated at action time) |
| 404 | `NO_PENDING_PLAN` | No plan awaiting approval for this runId |
| 409 | `PLAN_REVISION_MISMATCH` | Revision in request doesn't match current |
| 409 | `IDEMPOTENCY_KEY_CONFLICT` | Same idempotencyKey, different payload |
| 409 | `PLAN_ALREADY_APPROVED` | Plan already approved/executing |
| 409 | `INVALID_RUN_STATE` | Run not in approvable state |
| 409 | `PARTICIPANT_CHANGE_REQUIRES_REVISION` | APPROVE_PLAN attempted to change selectedParticipants |
| 410 | `PLAN_EXPIRED` | Approval window expired |

### Confirm endpoint guarantees

- Returns JSON acknowledgment only. Does NOT return SSE.
- The original `POST /api/chat` SSE stream continues to carry execution/revision events.
- Gateway MUST forward the action to Orchestrator unchanged.
- Gateway MUST NOT generate plans, invoke agents, perform semantic routing, or treat changed participants in APPROVE_PLAN as executable approval.

## Rules

- Gateway owns auth, CORS, request IDs, and public error shape.
- Public API must not leak internal Orchestrator URLs or Child Agent URLs.
- `/api/chat` must not become the primary contract endpoint unless contracts are intentionally revised.
- Gateway MUST NOT generate plans or invoke agents. It only forwards.
