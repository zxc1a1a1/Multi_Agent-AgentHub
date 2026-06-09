# Gateway ↔ Orchestrator Contract

**Status:** Active
**Owner:** AgentHub
**Primary source:** Module Separation & Runtime Redesign
**Last updated:** 2026-06-09 (Phase 4 — HITL confirm error passthrough)


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
- Forward request to Orchestrator **unchanged** — including `executionPath`, `selectedAgentNames`, `mentions`, `requestedPath`.
- Derive `executionPath` from agent selection fields (see `docs/contracts/chat-execution-path.md`).
- Convert Orchestrator stream to public SSE wire format.
- Persist assistant text/artifacts after stream completion.
- Own public auth/CORS/rate limit/error shape.
- **MUST NOT** generate plans, invoke agents, or perform semantic routing.
- **MUST NOT** call agents directly when Orchestrator is configured.

## Orchestrator responsibilities

- Build PlannerInput from messages/history/tools/metadata, including `executionPath`, `AllowedAgents`.
- Produce and validate OrchestrationPlan respecting participant boundaries.
- Enforce agent selection boundaries per execution path (see `docs/contracts/participant-boundary.md`).
- Execute single/sequential/parallel tasks through A2A Child Agents.
- Convert internal/A2A events to AG-UI events with `agentName`.
- Return only stream events; do not persist browser conversation state.
- Handle plan approval lifecycle: `confirm_plan` → wait → execute/revise/cancel.

## Plan approval forwarding

Gateway forwards `POST /api/runs/{runId}/confirm` requests to Orchestrator's `POST /internal/orchestrator/hitl/confirm`:
- Gateway validates the request shape (required fields).
- Gateway passes all fields (`runId`, `actionId`, `planId`, `action`, `confirmed`, `feedback`, `revision`, `rejectReason`, `idempotencyKey`, `selectedParticipants`) unchanged.
- Gateway returns the Orchestrator's JSON response to the frontend.
- Gateway **MUST NOT** interpret the action or make planning decisions.
- The original `POST /api/chat` SSE stream carries the post-action events; Gateway does not create a new stream.

### Error passthrough

Gateway MUST preserve the Orchestrator's HTTP response when confirm fails:
- **Status code**: Forward the Orchestrator's status code (not replace with generic 502).
- **Response body**: Forward the Orchestrator's error body unchanged.
- **Content-Type**: Forward the Orchestrator's Content-Type header.

Only if the Orchestrator is unreachable (connection error, not an HTTP response) does Gateway return a generic `502 Bad Gateway`.

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
- User-initiated cancel → `STATE_UPDATE(phase=cancelled)` + `RUN_FINISHED(status=cancelled)`. **Not `RUN_ERROR`.**
- No internal URL, stack trace, API key, or DSN may be exposed to browser.


## Plan approval participant-change forwarding

Gateway MUST forward `action`, `planId`, `revision`, `idempotencyKey`, `feedback`, and `selectedParticipants` unchanged. It MUST NOT reinterpret changed `selectedParticipants` in an `approve` action as approval. The Orchestrator owns the validation and MUST return `PARTICIPANT_CHANGE_REQUIRES_REVISION` when approval attempts to change participants.
