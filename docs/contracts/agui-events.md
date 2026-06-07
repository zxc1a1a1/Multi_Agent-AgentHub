# AG-UI Events Contract

**Status:** Active
**Owner:** AgentHub
**Primary source:** Module Separation & Runtime Redesign
**Last updated:** 2026-06-07 (v1.1 HITL plan confirmation — AG-UI tool events + STATE_UPDATE metadata dual path)


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
