# AG-UI Events Contract

**Status:** Active
**Owner:** AgentHub
**Primary source:** Module Separation & Runtime Redesign
**Last updated:** 2026-06-08 (v1.2 plan approval — executionPath, AGENT_TURN events, cancel lifecycle fix)


## Authoritative source order

1. `docs/superpowers/specs/2026-05-26-module-separation-and-runtime-redesign.md`
2. `docs/superpowers/specs/2026-05-27-module-separation-runtime-redesign.md`
3. PDR product goals only, not old module layout
4. Sprint/UML as supporting product/demo references only

Engineering details MUST follow the module separation redesign. Old `server/` and root `agents/` are legacy reference implementations unless a task explicitly says otherwise.


## Scope

AG-UI events are the browser-facing streaming contract emitted by Gateway. Runtime translation logic lives under `pkg/runtime/agui`.

## Event types

| Type | Purpose |
|---|---|
| `RUN_STARTED` | Run accepted and stream opened. Carries plan metadata and optional `requiresConfirmation` flag. |
| `STATE_UPDATE` | Planning, active agent, retry, progress, or completion metadata. Carries HITL confirmation fields when applicable. |
| `TEXT_MESSAGE_START` | Assistant message stream begins. |
| `TEXT_MESSAGE_CONTENT` | Text delta (supports markdown, code blocks, HTML — renders progressively during streaming). |
| `TEXT_MESSAGE_END` | Assistant message stream ends. |
| `TOOL_CALL_START` | Frontend skill invocation begins. |
| `TOOL_CALL_ARGS` | Skill arguments JSON, possibly chunked. Progressive WebPreview Source updates for web-preview tools. |
| `TOOL_CALL_END` | Skill invocation complete; frontend may render/execute. |
| `TOOL_ACTION_CONFIRM` | (Legacy) Tool-level HITL confirmation. Superseded by plan-level confirmation via `STATE_UPDATE` / `RUN_STARTED` metadata. |
| `TOOL_ACTION_RESPONSE` | (Legacy) Frontend → Gateway tool-level response. Superseded by REST `POST /api/runs/{runId}/confirm`. |
| `STATE_SNAPSHOT` | Full structured state snapshot for rendering (see snapshot-event.md). |
| `STATE_DELTA` | Incremental state update to merge into existing snapshot. |
| `ACTIVITY_SNAPSHOT` | Structured activity state for plan approval, agent turns, and orchestration progress. Replaces `confirm_plan` TOOL_CALL for plan-level HITL. |
| `RUN_FINISHED` | Run completed successfully. |
| `RUN_ERROR` | Run failed. |

## Plan-level HITL confirmation (v1.1)

When `REQUIRE_PLAN_CONFIRMATION=true` is set, the Orchestrator emits confirmation events through two parallel paths:

1. **AG-UI standard tool events** (`TOOL_CALL_START` / `TOOL_CALL_ARGS` / `TOOL_CALL_END`) with tool name `confirm_plan`. This is the primary path — it expresses HITL confirmation through the AG-UI tool/interrupt model.
2. **STATE_UPDATE metadata** — retained for backward compatibility and as input to `OrchestrationCard` summary display.

Both paths carry the same plan information. The frontend `messageStore` normalizes both into a unified `PendingConfirmation` state.

### confirm_plan tool event flow (AG-UI standard)

```
TOOL_CALL_START  →  toolCallId=plan-xyz, toolCallName=confirm_plan
TOOL_CALL_ARGS   →  delta: JSON plan arguments (progressive, may arrive as single chunk)
TOOL_CALL_END    →  arguments complete; frontend renders HITLConfirm UI
```

**TOOL_CALL_START:**
```json
{
  "type": "TOOL_CALL_START",
  "runId": "run-abc123",
  "toolCallId": "plan-xyz",
  "toolCallName": "confirm_plan",
  "sender": { "type": "orchestrator", "name": "orchestrator" }
}
```

**TOOL_CALL_ARGS:**
```json
{
  "type": "TOOL_CALL_ARGS",
  "runId": "run-abc123",
  "toolCallId": "plan-xyz",
  "delta": "{\"runId\":\"run-abc123\",\"planId\":\"plan-xyz\",\"strategy\":\"sequential\",\"plannedAgents\":[\"code-agent\",\"web-agent\"],\"tasks\":[...],\"intentSummary\":\"...\",\"requiresConfirmation\":true}"
}
```

**TOOL_CALL_END:**
```json
{
  "type": "TOOL_CALL_END",
  "runId": "run-abc123",
  "toolCallId": "plan-xyz"
}
```

### confirm_plan arguments schema

| Field | Type | Description |
|---|---|---|
| `runId` | string | Orchestration run ID |
| `planId` | string | Plan identifier (same as `toolCallId`) |
| `strategy` | string | `single`, `ordered_parallel`, or `sequential` |
| `plannedAgents` | string[] | Agent names to be invoked |
| `tasks` | object[] | Task summaries with `taskId`, `agentName`, `content`, `dependsOn`, `priority`, `riskLevel` |
| `intentSummary` | string | Human-readable plan description |
| `requiresConfirmation` | boolean | Always `true` in this context |

### RUN_STARTED with plan preview (metadata path, backward compatible)

```json
{
  "type": "RUN_STARTED",
  "runId": "run-abc123",
  "state": {
    "phase": "awaiting_confirmation",
    "requiresConfirmation": true,
    "confirmationActionId": "plan-xyz",
    "planId": "plan-xyz",
    "strategy": "sequential",
    "intentSummary": "Generate and deploy a web app",
    "plannerSource": "llm",
    "plannerModel": "claude-sonnet-4-6",
    "taskCount": 3,
    "plannedAgents": ["code-agent", "web-agent", "deploy-agent"],
    "tasks": [
      {
        "taskId": "task-1",
        "agentName": "code-agent",
        "content": "Generate backend API code",
        "dependsOn": [],
        "priority": 1,
        "riskLevel": "low"
      }
    ]
  }
}
```

### Confirmation endpoint

**POST /api/runs/{runId}/confirm** (Frontend → Gateway → Orchestrator)

```json
{
  "runId": "run-abc123",
  "actionId": "plan-xyz",
  "confirmed": true,
  "rejectReason": ""
}
```

Gateway forwards to Orchestrator's internal endpoint:
**POST /internal/orchestrator/hitl/confirm** (Gateway → Orchestrator, with `Authorization: Bearer <internal-token>`)

### Confirmation flow

1. Orchestrator sends `RUN_STARTED` with `requiresConfirmation: true` in state.
2. Frontend shows plan preview card with agent list, task list, strategy.
3. User confirms → `POST /api/runs/{runId}/confirm` with `confirmed: true`.
4. Orchestrator resumes execution, emits another `RUN_STARTED` with `phase: "executing"`.
5. User rejects → `POST /api/runs/{runId}/confirm` with `confirmed: false` and optional `rejectReason`.
6. Orchestrator emits `RUN_FINISHED` with `status: "cancelled"`.
7. Timeout (120s default) → Orchestrator emits `RUN_FINISHED` with `status: "timeout"`.

### STATE_UPDATE confirmation metadata

During confirmation phase, `STATE_UPDATE` events MAY carry the same confirmation fields (`requiresConfirmation`, `confirmationActionId`, `plannedAgents`, `tasks`) for progressive UI updates.

## Plan Approval v1.2 (extends v1.1)

This section extends the v1.1 HITL confirmation flow with execution paths, revision, and agent turn events. See `docs/contracts/plan-approval.md` and `docs/contracts/chat-execution-path.md` for the full specification.

### confirm_plan arguments schema (v1.2 extension)

Added fields beyond v1.1:

| Field | Type | Description |
|---|---|---|
| `runId` | string | (existing) |
| `planId` | string | (existing) |
| `revision` | int | Monotonic revision counter, starts at 1 |
| `executionPath` | string | `single_chat` / `group_chat` / `main_agent_orchestration` |
| `planOwner` | object | `{type, agentName, isMainAgent}` |
| `strategy` | string | (existing) |
| `participants` | object[] | `[{agentName, required, reason, selected}]` |
| `candidateParticipants` | object[] | (main_agent_orchestration only) |
| `defaultSelectedParticipants` | object[] | (main_agent_orchestration only) |
| `tasks` | object[] | (existing) |
| `intentSummary` | string | (existing) |
| `requiresConfirmation` | boolean | (existing) |

### Revised confirmation flow (v1.2)

1. Orchestrator emits `RUN_STARTED` with `phase: "planning"` and `executionPath`.
2. Orchestrator emits `TOOL_CALL_START/ARGS/END` with `toolCallName: "confirm_plan"` (v1.1 compatible).
3. Orchestrator emits `STATE_UPDATE` with `phase: "waiting_user_approval"`.
4. User actions (via `POST /api/runs/{runId}/confirm`):
   - **Approve** (`action: "approve"`): Orchestrator resumes execution on the **same SSE stream**. Emits `STATE_UPDATE {phase: "executing"}`, then execution events, then `RUN_FINISHED`.
   - **Revise** (`action: "revise"` with `feedback` and/or participant changes): Orchestrator invokes the **original planOwner** to regenerate the plan (revision+1). A new `confirm_plan` sequence is emitted on the **same SSE stream**. Plan enters `revising_plan` then `waiting_user_approval`.
   - **Cancel** (`action: "cancel"`): Orchestrator emits `STATE_UPDATE {phase: "cancelled"}` then `RUN_FINISHED {status: "cancelled"}`. **Does NOT emit `RUN_ERROR`.**
5. Timeout: same as v1.1 — `RUN_FINISHED {status: "timeout"}`.

### Cancel lifecycle (v1.2 fix)

**User-initiated cancellation (CANCEL_PLAN) MUST NOT emit `RUN_ERROR`.** Cancellation is a normal lifecycle transition, not a system error.

Correct event sequence for cancel:
```
STATE_UPDATE { phase: "cancelled" }
RUN_FINISHED { status: "cancelled" }
```

`RUN_ERROR` is reserved for actual system failures: planner failure, validation failure, agent dispatch failure, execution errors, etc.

### SSE continuation guarantee

The SSE connection from `POST /api/chat` remains open throughout the entire plan lifecycle (planning → waiting → approved/revised/cancelled → executing → completed). The `POST /api/runs/{runId}/confirm` endpoint returns JSON acknowledgment and signals the Orchestrator internally. **No new POST /api/chat is required** for approval, revision, or cancellation.

## AGENT_TURN events (v1.2)

For `group_chat` and `main_agent_orchestration` modes, each agent's output is wrapped in turn events for independent rendering.

### Event types

| Type | Purpose |
|---|---|
| `AGENT_TURN_STARTED` | An agent's turn begins |
| `AGENT_TURN_CONTENT` | Content delta within an agent's turn (semantically same as `TEXT_MESSAGE_CONTENT`) |
| `AGENT_TURN_FINISHED` | An agent's turn ends |

### Event schemas

**AGENT_TURN_STARTED:**
```json
{
  "type": "AGENT_TURN_STARTED",
  "runId": "run-abc123",
  "messageId": "msg-turn-1",
  "turnIndex": 0,
  "stepId": "task-1",
  "agentName": "code-agent",
  "sender": { "type": "agent", "name": "code-agent" }
}
```

**AGENT_TURN_CONTENT:**
```json
{
  "type": "AGENT_TURN_CONTENT",
  "runId": "run-abc123",
  "messageId": "msg-turn-1",
  "turnIndex": 0,
  "delta": "Here is the generated code..."
}
```

**AGENT_TURN_FINISHED:**
```json
{
  "type": "AGENT_TURN_FINISHED",
  "runId": "run-abc123",
  "messageId": "msg-turn-1",
  "turnIndex": 0,
  "agentName": "code-agent",
  "summary": "Generated HTTP server code"
}
```

### Turn rendering rules

- Each turn has a unique `turnIndex` (monotonic, starts at 0).
- Each turn has a unique `messageId` for content accumulation.
- The frontend MUST render each turn as a separate agent bubble, keyed by `messageId`.
- `turnIndex` provides ordering for the turn sequence.
- In `single_chat` mode, AGENT_TURN events are optional (text events alone suffice — one agent, one implicit turn).

## ACTIVITY_SNAPSHOT (v1.3)

`ACTIVITY_SNAPSHOT` replaces the v1.1/v1.2 `confirm_plan` TOOL_CALL events for plan-level HITL confirmation. `TOOL_CALL_START/ARGS/END` are reserved exclusively for real agent tool invocations (shell, browser, file operations, etc.).

### Event schema

```json
{
  "type": "ACTIVITY_SNAPSHOT",
  "runId": "run-abc123",
  "conversationId": "conv-xyz",
  "activity": {
    "activityId": "activity-plan-xyz",
    "activityType": "plan_approval",
    "status": "awaiting_confirmation",
    "executionPath": "single_chat",
    "planId": "plan-xyz",
    "revision": 1,
    "planOwner": {
      "type": "agent",
      "agentName": "code-agent",
      "isMainAgent": false
    },
    "participants": [
      {"agentName": "code-agent", "required": true, "selected": true}
    ],
    "candidateParticipants": [],
    "defaultSelectedParticipants": [],
    "requiredParticipants": ["code-agent"],
    "title": "Code Generation Plan",
    "summary": "Generate HTTP server code with tests",
    "tasks": [
      {
        "taskId": "task-1",
        "agentName": "code-agent",
        "content": "Generate backend API code",
        "priority": 1,
        "riskLevel": "low"
      }
    ],
    "allowedActions": ["approve", "revise", "cancel"],
    "warnings": []
  }
}
```

### activityType values

| Value | Description |
|---|---|
| `plan_approval` | Plan-level HITL confirmation. The Orchestrator has produced a plan and is waiting for user action. |
| `agent_turn` | An agent's execution turn is about to start or has completed. |
| `orchestration_progress` | Orchestration progress update (task dispatch, aggregation, summary). |

### status values (for plan_approval)

| Status | Description |
|---|---|
| `awaiting_confirmation` | Plan is ready, waiting for user approve/revise/cancel. |
| `revising_plan` | User requested revision; planOwner is regenerating the plan. |
| `executing` | Plan approved, dispatching to agents. |
| `cancelled` | User cancelled the plan. |
| `completed` | Plan execution finished. |
| `failed` | Plan or execution failed with an error. |

### executionPath values

| Value | Description |
|---|---|
| `single_chat` | Single agent selected by user. PlanOwner is that agent. |
| `group_chat` | Multiple agents selected. PlanOwner is the GroupCoordinator. |
| `main_agent_orchestration` | No agent selected ("auto"). PlanOwner is the MainAgent. |

### planOwner types

| Type | executionPath | Description |
|---|---|---|
| `agent` | `single_chat` | The single selected agent owns the plan. |
| `group_coordinator` | `group_chat` | Orchestrator's internal GroupCoordinator owns the plan. |
| `main_agent` | `main_agent_orchestration` | Orchestrator's internal MainAgent owns the plan. |

### Emit rules

1. **Replaces confirm_plan TOOL_CALL**: All v1.1/v1.2 plan-level HITL confirmation MUST use `ACTIVITY_SNAPSHOT`, not `TOOL_CALL_START/ARGS/END` with `toolCallName: "confirm_plan"`.
2. **Revision emits new ACTIVITY_SNAPSHOT**: On revise, a new `ACTIVITY_SNAPSHOT` is emitted with `revision+1` and a new `activityId`/`planId`. The previous activity is superseded.
3. **Status transitions**: `awaiting_confirmation` → (on revise) `revising_plan` → `awaiting_confirmation` → (on approve) `executing` → `completed` / (on cancel) `cancelled`.
4. **Same SSE stream**: ACTIVITY_SNAPSHOT is emitted on the same SSE connection as all other events. No new POST /api/chat required.
5. **Participant boundary**: `participants` only includes `AllowedAgents`. The Orchestrator MUST NOT add agents outside the user's selected set for `group_chat`.
6. **participant change detection**: For `main_agent_orchestration`, if `APPROVE_PLAN` is received with `selectedParticipants` differing from `defaultSelectedParticipants`, the backend MUST return `PARTICIPANT_CHANGE_REQUIRES_REVISION`.

### SSE wire format

```text
event: activity_snapshot
data: {"type":"ACTIVITY_SNAPSHOT","runId":"run-abc123","activity":{...}}
```

The Gateway writes the `event:` line for backward compatibility. The `type` field in the JSON payload (UPPER_SNAKE) is authoritative.

### Backward compatibility

- **v1.1/v1.2 confirm_plan TOOL_CALL events are DEPRECATED** for plan-level HITL. New runs MUST use `ACTIVITY_SNAPSHOT`.
- Frontends MAY keep a legacy fallback path for `TOOL_CALL_END` with `toolCallName: "confirm_plan"` to replay old conversation history. New code paths MUST use `ACTIVITY_SNAPSHOT` as the primary signal.
- `STATE_UPDATE` confirmation metadata (v1.1 backward-compatible path) remains valid as supplementary state, but `ACTIVITY_SNAPSHOT` is the authoritative plan approval event.

## v1.1/v1.2 confirm_plan TOOL_CALL — DEPRECATED

The sections below (§ Plan-level HITL confirmation v1.1, § Plan Approval v1.2) are retained for historical reference and backward compatibility with old conversation replays. **New implementations MUST use `ACTIVITY_SNAPSHOT` (§ above) for plan-level HITL confirmation.** `TOOL_CALL_START/ARGS/END` events are reserved exclusively for real agent tool invocations.

## Streaming behavior requirements

### Progressive code blocks

During streaming (`TEXT_MESSAGE_CONTENT`), the frontend MUST:
- Render code fences progressively: detect open ` ``` ` blocks and render their content as styled code blocks even before the closing fence arrives.
- Continue rendering non-code text as plain `whitespace-pre-wrap` with a pulsing cursor.

After streaming completes, the full Markdown renderer handles syntax highlighting.

### Progressive WebPreview

During `TOOL_CALL_ARGS` for web-preview tools (`web_preview`, `generate_html_snippet`), the frontend SHOULD:
- Parse partial JSON to extract `html`/`content` fields.
- Update the WebPreview Source tab progressively so users can watch HTML being generated.
- The Preview tab renders once sufficient HTML structure is detected.

### Tool output streaming

Tool call argument accumulation MUST happen per `toolCallId`/`id`, and:
- Arguments MUST be kept separate per tool call (no cross-contamination).
- Preview updates MUST NOT block text streaming.
- Artifact deltas (`artifact.delta`) MUST update preview state without interrupting the text stream.

### Multi-agent message separation

Messages from different agents MUST be separated by `messageId`:
- A new `messageId` signals a new agent message bubble.
- `sender.name` / `sender.type` / `agentName` MUST be used for attribution.
- The frontend MUST NOT infer agent identity from content analysis.

## Common fields

```json
{
  "type": "TEXT_MESSAGE_CONTENT",
  "runId": "run-id",
  "messageId": "msg-id",
  "taskId": "task-1",
  "role": "assistant",
  "delta": "delta text",
  "sender": {
    "type": "agent",
    "name": "code-agent"
  },
  "metadata": {}
}
```

**Field conventions:**
- `delta` is the preferred field for incremental text content (AG-UI v1.0).
- `content` and `text` are accepted as legacy fallbacks.
- `sender` object is preferred over bare `agentName`/`author` strings.
- `taskId` links content to a specific orchestration task.

## Multi-agent attribution

All assistant text/tool/state events SHOULD include `agentName` and/or `sender` when produced by a Child Agent. Frontend must render distinct attribution and not infer agent from conversation default.

## Skill mapping

Artifacts and ToolCalls map to frontend skills:

| Artifact / tool | Frontend skill |
|---|---|
| `code` | `code_preview` |
| `webpage` | `web_preview` |
| `document` | `markdown_render` |
| `file` | `file_download` |
| `diff` | `diff_preview` |
| `terminal` | `terminal_output` |

## Wire format

SSE block:

```text
event: <event-type>
data: {json event}


```

The `event:` line is optional; the `type` field in the JSON payload is authoritative. The Gateway writes `event:` lines for backward compatibility with older SSE clients.


### Auto participant-change event rule

For `main_agent_orchestration`, participant modifications are not approval. If the frontend sends `APPROVE_PLAN` with a `selectedParticipants` set that differs from the current proposal default, the backend MUST return `PARTICIPANT_CHANGE_REQUIRES_REVISION` and MUST NOT emit execution events. The correct flow is `REQUEST_PLAN_REVISION` followed by a new `confirm_plan` sequence on the existing SSE stream.
