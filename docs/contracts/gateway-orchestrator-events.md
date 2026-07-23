# Gateway ↔ Orchestrator Internal Events

**Status:** Active
**Version:** AgentHub 2.0

## 1. Purpose

Orchestrator emits internal domain events. Gateway persists/maps them to browser-facing AG-UI events.

Internal event names MUST NOT be emitted directly to Frontend.

## 2. Envelope

```json
{
  "eventId": "evt_1",
  "sequence": 12,
  "type": "agent_message_delta",
  "conversationId": "conv_1",
  "runId": "run_1",
  "planId": "plan_1",
  "planVersion": 1,
  "stepId": "step_1",
  "invocationId": "inv_1",
  "timestamp": "2026-01-01T00:00:00Z",
  "payload": {}
}
```

Rules:

- `(runId, sequence)` is unique and monotonic;
- event is append-only after persistence;
- payload is versioned/sanitized;
- credentials and private model reasoning are forbidden.

## 3. Event types

### Run

```text
run_started
run_status_changed
run_completed
run_partial_failure
run_failed
run_canceled
```

### Plan

```text
plan_created
plan_validation_failed
plan_awaiting_confirmation
plan_confirmed
plan_rejected
plan_replanned
```

### Execution

```text
step_started
step_completed
step_failed
step_skipped
agent_invocation_started
agent_message_start
agent_message_delta
agent_message_end
agent_invocation_completed
agent_invocation_failed
```

### Tool

```text
tool_call_started
tool_call_arguments
tool_call_result
tool_call_completed
tool_call_failed
```

### Artifact/context/preview

```text
artifact_created
artifact_updated
artifact_version_conflict
preview_building
preview_ready
preview_failed
context_compacted
```

## 4. Ordering

- `run_started` is first for a Run event stream.
- `agent_message_delta` follows `agent_message_start` for the same message.
- `agent_message_end` occurs once.
- Tool arguments/results reference an active Tool call.
- Step completion follows Step start.
- Run terminal event occurs once.
- Plan execution events do not occur before `plan_confirmed`.
- A new `plan_replanned`/`plan_awaiting_confirmation` suspends affected execution.

## 5. Gateway mapping

| Internal | AG-UI |
|---|---|
| `run_started` | `RUN_STARTED` |
| `run_completed` | `RUN_FINISHED` |
| `run_partial_failure` | `RUN_FINISHED` with result metadata |
| `run_failed` | `RUN_ERROR` |
| `run_canceled` | `RUN_FINISHED` with canceled outcome/result |
| `step_started` | `STEP_STARTED` |
| `step_completed` | `STEP_FINISHED` |
| `agent_message_start` | `TEXT_MESSAGE_START` |
| `agent_message_delta` | `TEXT_MESSAGE_CONTENT` |
| `agent_message_end` | `TEXT_MESSAGE_END` |
| Tool lifecycle | `TOOL_CALL_*` / `TOOL_CALL_RESULT` |
| Plan domain events | `CUSTOM` name `agenthub.plan.*` and/or `ACTIVITY_*` |
| Artifact events | `CUSTOM` name `agenthub.artifact.*` |
| Preview events | `CUSTOM` name `agenthub.preview.*` |
| Context compaction | `CUSTOM` name `agenthub.context.compacted` |

Tool events are for actual Tool calls. Plan confirmation and Artifact preview MUST NOT be encoded as fake Tool calls.

## 6. Replay

Gateway uses persisted `sequence` as SSE `id`.

On reconnect:

```text
read Last-Event-ID
-> replay sequence greater than cursor
-> attach live stream
```

Duplicate event IDs are ignored idempotently by Frontend.

## 7. Error behavior

Retryable internal errors may produce progress/state updates without terminating the Run.

A terminal failure emits exactly one `run_failed`.

An interrupted A2A stream after user-visible output MUST NOT be automatically replayed as duplicate text unless the Agent protocol provides resumable sequencing.

## 8. Required tests

- monotonic sequence;
- message delta ordering;
- Tool lifecycle ordering;
- no execution event before confirmation;
- cancel terminal mapping;
- partial-failure mapping;
- event replay without duplicates;
- Plan/Artifact/Preview custom event mapping;
- no raw A2A event pass-through;
- secret/error redaction.
