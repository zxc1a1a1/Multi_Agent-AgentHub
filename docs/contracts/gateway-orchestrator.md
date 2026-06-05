# Gateway ↔ Orchestrator Contract

**Status:** Active
**Owner:** AgentHub
**Primary source:** Module Separation & Runtime Redesign


## Authoritative source order

1. `docs/superpowers/specs/2026-05-26-module-separation-and-runtime-redesign.md`
2. `docs/superpowers/specs/2026-05-27-module-separation-runtime-redesign.md`
3. PDR product goals only, not old module layout
4. Sprint/UML as supporting product/demo references only

Engineering details MUST follow the module separation redesign. Old `server/` and root `agents/` are legacy reference implementations unless a task explicitly says otherwise.


## Boundary

Gateway and Orchestrator are separate deployable services.

```text
services/gateway  --gRPC streaming-->  services/orchestrator
```

## Target protocol

- Target: gRPC server streaming or bidirectional streaming as defined by `services/gateway/proto/orchestrator.proto` and `services/orchestrator/proto/orchestrator.proto`.
- Temporary compatibility: HTTP POST + SSE is allowed only under explicit compatibility notes and must not be documented as the target architecture.

## Gateway responsibilities

- Validate public request.
- Save user message and load bounded history.
- Forward request to Orchestrator.
- Convert Orchestrator stream to public SSE wire format.
- Persist assistant text/artifacts after stream completion.
- Own public auth/CORS/rate limit/error shape.

## Orchestrator responsibilities

- Build PlannerInput from messages/history/tools/metadata.
- Produce and validate OrchestrationPlan.
- Execute single/sequential/parallel tasks through A2A Child Agents.
- Convert internal/A2A events to AG-UI events with `agentName`.
- Return only stream events; do not persist browser conversation state.

## Required gRPC operations

```protobuf
service OrchestratorService {
  rpc Execute(ExecuteRequest) returns (stream AGUIEvent);
  rpc ListAgents(ListAgentsRequest) returns (ListAgentsResponse);
}
```

## Failure rules

- Orchestrator unavailable → Gateway returns HTTP 503 or AG-UI `RUN_ERROR` if stream already started.
- Child Agent failure → Orchestrator emits `STATE_UPDATE` retry/fallback when possible, then `RUN_ERROR` if all attempts fail.
- No internal URL, stack trace, API key, or DSN may be exposed to browser.
