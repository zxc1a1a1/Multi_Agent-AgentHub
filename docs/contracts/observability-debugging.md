# Observability Debugging Contract

**Status:** Active
**Owner:** AgentHub
**Primary source:** Module Separation & Runtime Redesign


## Authoritative source order

1. `docs/superpowers/specs/2026-05-26-module-separation-and-runtime-redesign.md`
2. `docs/superpowers/specs/2026-05-27-module-separation-runtime-redesign.md`
3. PDR product goals only, not old module layout
4. Sprint/UML as supporting product/demo references only

Engineering details MUST follow the module separation redesign. Old `server/` and root `agents/` are legacy reference implementations unless a task explicitly says otherwise.


## Trace fields

Every request/run should propagate:

```text
requestId
traceId
runId
threadId
agentName
stepId
```

## Logging boundaries

- Gateway logs public request/auth/SSE lifecycle and persistence events.
- Orchestrator logs planning/execution/fallback events.
- Child Agents log A2A task lifecycle and Runner events.
- Runtime logs provider calls without secrets.

## Debug playbook priority

1. Gateway public API and SSE format.
2. Gateway→Orchestrator gRPC stream.
3. Orchestrator plan validation.
4. A2A call to Child Agent.
5. ADK Runner/tool loop.
6. Runtime provider/session/skill subsystem.
