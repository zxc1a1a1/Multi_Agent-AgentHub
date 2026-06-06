# Gateway ↔ Orchestrator Events

**Status:** Active
**Owner:** AgentHub
**Primary source:** Module Separation & Runtime Redesign


## Authoritative source order

1. `docs/superpowers/specs/2026-05-26-module-separation-and-runtime-redesign.md`
2. `docs/superpowers/specs/2026-05-27-module-separation-runtime-redesign.md`
3. PDR product goals only, not old module layout
4. Sprint/UML as supporting product/demo references only

Engineering details MUST follow the module separation redesign. Old `server/` and root `agents/` are legacy reference implementations unless a task explicitly says otherwise.


## Event sequence

Typical single-agent run:

```text
RUN_STARTED
STATE_UPDATE {phase:"planning"}
STATE_UPDATE {activeAgent:"code-agent"}
TEXT_MESSAGE_START
TEXT_MESSAGE_CONTENT ...
TEXT_MESSAGE_END
TOOL_CALL_START / TOOL_CALL_ARGS / TOOL_CALL_END
RUN_FINISHED
```

Multi-agent run:

```text
RUN_STARTED
STATE_UPDATE {phase:"planning", mode:"parallel|sequential"}
STATE_UPDATE {activeAgent:"web-agent"}
... web-agent text/tool events ...
STATE_UPDATE {activeAgent:"code-agent"}
... code-agent text/tool events ...
RUN_FINISHED
```

## Ordering rules

- Every `TEXT_MESSAGE_CONTENT` must follow a `TEXT_MESSAGE_START` for the same message id.
- Every `TOOL_CALL_ARGS` and `TOOL_CALL_END` must reference an active `TOOL_CALL_START`.
- `RUN_FINISHED` is emitted once per run by Orchestrator, then Gateway closes SSE.
- `RUN_ERROR` terminates the run unless marked retryable in metadata.
