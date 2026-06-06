# AG-UI Events Contract

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

AG-UI events are the browser-facing streaming contract emitted by Gateway. Runtime translation logic lives under `pkg/runtime/agui`.

## Event types

| Type | Purpose |
|---|---|
| `RUN_STARTED` | Run accepted and stream opened. |
| `STATE_UPDATE` | Planning, active agent, retry, progress, or completion metadata. |
| `TEXT_MESSAGE_START` | Assistant message stream begins. |
| `TEXT_MESSAGE_CONTENT` | Text delta. |
| `TEXT_MESSAGE_END` | Assistant message stream ends. |
| `TOOL_CALL_START` | Frontend skill invocation begins. |
| `TOOL_CALL_ARGS` | Skill arguments JSON, possibly chunked. |
| `TOOL_CALL_END` | Skill invocation complete; frontend may render/execute. |
| `TOOL_ACTION_CONFIRM` | Orchestrator requests HITL confirmation for a medium/high risk action. |
| `TOOL_ACTION_RESPONSE` | Frontend → Gateway: user confirmed or rejected the action. |
| `STATE_SNAPSHOT` | Full structured state snapshot for rendering (see snapshot-event.md). |
| `STATE_DELTA` | Incremental state update to merge into existing snapshot. |
| `RUN_FINISHED` | Run completed successfully. |
| `RUN_ERROR` | Run failed. |

## HITL confirmation events

### `TOOL_ACTION_CONFIRM` (Orchestrator → Frontend)

```json
{
  "type": "TOOL_ACTION_CONFIRM",
  "runId": "run-id",
  "actionId": "action-1",
  "riskLevel": "medium|high",
  "actionName": "delete-file",
  "description": "Delete /tmp/output.log",
  "parameters": { "path": "/tmp/output.log" },
  "timeoutMs": 30000
}
```

### `TOOL_ACTION_RESPONSE` (Frontend → Gateway → Orchestrator)

```json
{
  "type": "TOOL_ACTION_RESPONSE",
  "runId": "run-id",
  "actionId": "action-1",
  "confirmed": true,
  "rejectReason": ""
}
```

Frontend MUST render a confirmation dialog for `TOOL_ACTION_CONFIRM` events.
The user can confirm or reject within the `timeoutMs` window. If the timeout
expires, the action is treated as rejected.

## Common fields

```json
{
  "type": "TEXT_MESSAGE_CONTENT",
  "runId": "run-id",
  "messageId": "msg-id",
  "role": "assistant",
  "content": "delta text",
  "agentName": "code-agent",
  "metadata": {}
}
```

## Multi-agent attribution

All assistant text/tool/state events SHOULD include `agentName` when produced by a Child Agent. Frontend must render distinct attribution and not infer agent from conversation default.

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
data: {json event}


```
