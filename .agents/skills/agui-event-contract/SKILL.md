---
name: agui-event-contract
description: "AgentHub 2.0 browser event contract using official AG-UI standard events plus bounded CUSTOM events for plan, artifact, preview, and context domains."
---

# agui-event-contract

## Purpose

Use this Skill for Gateway SSE, AG-UI event conversion, frontend reducers, replay, plan activity, Agent messages, Tool events, Artifact events, or preview state.

## Read first

```text
/project-architecture
/conversation-contract
/planning-approval-contract
/gateway-orchestrator-contract
/artifact-contract
/security-boundary-contract
/testing-review-contract
```

Primary sources:

```text
docs/contracts/agui-events.md
docs/contracts/agui-events.schema.json
```

## Standard events to reuse

```text
RUN_STARTED
RUN_FINISHED
RUN_ERROR
STEP_STARTED
STEP_FINISHED
TEXT_MESSAGE_START
TEXT_MESSAGE_CONTENT
TEXT_MESSAGE_END
TOOL_CALL_START
TOOL_CALL_ARGS
TOOL_CALL_END
TOOL_CALL_RESULT
STATE_SNAPSHOT
STATE_DELTA
MESSAGES_SNAPSHOT
ACTIVITY_SNAPSHOT
ACTIVITY_DELTA
CUSTOM
RAW
```

## AgentHub CUSTOM names

```text
agenthub.plan.created
agenthub.plan.awaiting_confirmation
agenthub.plan.updated
agenthub.plan.confirmed
agenthub.plan.rejected
agenthub.plan.replanned
agenthub.artifact.created
agenthub.artifact.updated
agenthub.artifact.version_conflict
agenthub.preview.building
agenthub.preview.ready
agenthub.preview.failed
agenthub.context.compacted
```

## Non-negotiable rules

- Frontend consumes Gateway events only.
- Orchestrator internal events and A2A events MUST be converted, not directly forwarded.
- Plan confirmation is a domain interrupt/API action, not a fake Tool call.
- Artifact preview is not a fake Tool call.
- Tool events describe actual Tool calls only.
- Agent identity belongs in metadata/state using stable `agentId`, `agentVersion`, and `invocationId`.
- Text deltas append between start/end.
- SSE `id` is the persisted Run event sequence.
- Reconnect uses `Last-Event-ID`.
- A Run emits one terminal outcome.
- Errors are sanitized.
- Unknown CUSTOM names are ignored or shown safely; they are never executed.

## Completion checklist

- [ ] Standard AG-UI events are reused.
- [ ] CUSTOM names are namespaced.
- [ ] Plan/Artifact/Preview events are not Tool events.
- [ ] Event ordering and replay are tested.
- [ ] Agent identity and Message IDs are stable.
- [ ] cancellation/partial failure mapping is defined.
- [ ] reducers handle duplicate replay safely.
- [ ] schema/docs/implementation agree.
