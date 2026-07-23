# AG-UI Events Contract

**Status:** Active
**Version:** AgentHub 2.0
**Emitter:** Gateway
**Consumer:** Frontend

## 1. Purpose

Gateway exposes AgentHub Run progress through AG-UI-compatible SSE events.

AgentHub reuses standard AG-UI events for lifecycle, messages, tools, state, steps, and activity. Product-specific Plan, Artifact, Preview, and Context signals use namespaced `CUSTOM` events.

## 2. Standard event families

### Lifecycle

```text
RUN_STARTED
RUN_FINISHED
RUN_ERROR
STEP_STARTED
STEP_FINISHED
```

### Messages

```text
TEXT_MESSAGE_START
TEXT_MESSAGE_CONTENT
TEXT_MESSAGE_END
MESSAGES_SNAPSHOT
```

### Tools

```text
TOOL_CALL_START
TOOL_CALL_ARGS
TOOL_CALL_END
TOOL_CALL_RESULT
```

### State and activity

```text
STATE_SNAPSHOT
STATE_DELTA
ACTIVITY_SNAPSHOT
ACTIVITY_DELTA
```

### Special

```text
CUSTOM
RAW
```

`RAW` is reserved for controlled compatibility adapters. Frontend MUST NOT execute its payload.

## 3. Base identity

Where applicable:

```text
threadId       = conversationId
runId          = AgentHub Run ID
messageId      = AgentHub Message ID
stepName       = PlanStep ID or stable display value
```

Agent-specific metadata:

```json
{
  "agenthub": {
    "agentId": "code-agent",
    "agentVersion": "2.0.0",
    "invocationId": "inv_1",
    "planId": "plan_1",
    "planVersion": 1,
    "stepId": "step_1"
  }
}
```

Do not rely on display `agentName` as identity.

## 4. SSE serialization and replay

Each event frame uses persisted Run sequence:

```text
id: 42
data: {"type":"TEXT_MESSAGE_CONTENT", ...}
```

Reconnect:

```text
Last-Event-ID: 42
```

Gateway replays events with sequence greater than 42 and then attaches the live stream.

Frontend reducers MUST be idempotent by event ID.

## 5. Message streaming

Sequence:

```text
TEXT_MESSAGE_START
TEXT_MESSAGE_CONTENT*
TEXT_MESSAGE_END
```

Rules:

- content deltas append;
- one logical Message uses one `messageId`;
- Agent identity is stable for the stream;
- final committed Message must agree with accumulated content or a later snapshot;
- separate response mode may create multiple Agent Messages;
- synthesize mode preserves Agent activity and emits a final synthesized Message.

## 6. Tool events

Tool events represent actual Tool calls.

Sequence:

```text
TOOL_CALL_START
TOOL_CALL_ARGS*
TOOL_CALL_END
TOOL_CALL_RESULT
```

Plan confirmation, Artifact preview, and UI code rendering MUST NOT be represented as Tool calls.

Tool arguments/results are treated as untrusted and sanitized before display.

## 7. Plan events

Use `CUSTOM`:

```json
{
  "type": "CUSTOM",
  "name": "agenthub.plan.awaiting_confirmation",
  "value": {
    "planId": "plan_1",
    "version": 1,
    "goal": "Research and update a page",
    "selectedAgentIds": ["web-agent", "code-agent"],
    "steps": [],
    "responseMode": "synthesize",
    "allowedActions": ["confirm", "edit", "regenerate", "reject"]
  }
}
```

Other names:

```text
agenthub.plan.created
agenthub.plan.updated
agenthub.plan.confirmed
agenthub.plan.rejected
agenthub.plan.replanned
```

Frontend confirmation is submitted through Platform API against the exact version.

`ACTIVITY_SNAPSHOT`/`ACTIVITY_DELTA` may render a structured plan/execution timeline, but they do not replace Plan persistence.

## 8. Agent execution activity

PlanStep execution may use:

```text
STEP_STARTED
STEP_FINISHED
ACTIVITY_SNAPSHOT
ACTIVITY_DELTA
```

Activity data includes stable IDs, status, safe summaries, and progress. It does not include private model reasoning.

## 9. Artifact events

```text
agenthub.artifact.created
agenthub.artifact.updated
agenthub.artifact.version_conflict
```

Value contains only authorized metadata and references, not uncontrolled executable content.

## 10. Preview events

```text
agenthub.preview.building
agenthub.preview.ready
agenthub.preview.failed
```

Preview events identify:

```text
artifactId
artifactVersion
renderer
safe error code
```

Frontend fetches authorized Artifact content separately or from an approved state payload.

## 11. Context compaction

```text
agenthub.context.compacted
```

Value may include:

```text
summaryId
coveredFromSequence
coveredToSequence
estimatedTokensBefore
estimatedTokensAfter
```

It MUST NOT include hidden prompts or private reasoning.

## 12. Run termination

### Success

```text
RUN_FINISHED
```

### Partial failure

`RUN_FINISHED` with a safe result/outcome showing `partial_failure`.

### Cancellation

`RUN_FINISHED` with canceled outcome/result.

### Failure

```text
RUN_ERROR
```

A Run has exactly one terminal outcome.

## 13. Error safety

`RUN_ERROR` contains only:

```text
code
message
runId/threadId where supported
safe metadata
```

No stack trace, internal URL, credential, provider body, or database detail.

## 14. Compatibility

Legacy event types such as custom `AGENT_TURN_*`, `TOOL_ACTION_CONFIRM`, `TOOL_ACTION_RESPONSE`, or `STATE_UPDATE` may be read by a migration adapter, but new 2.0 production flow uses official event families and namespaced `CUSTOM`.

## 15. Required tests

- standard message accumulation;
- duplicate replay;
- plan CUSTOM event reducer;
- Activity snapshot/delta application;
- Tool event lifecycle;
- Artifact and Preview events;
- cancel/partial failure terminal behavior;
- no fake confirm-plan Tool call;
- no raw A2A event pass-through;
- sanitized RUN_ERROR;
- unknown CUSTOM event safe handling.
