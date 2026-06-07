---
name: gateway-orchestrator-contract
description: "Gateway-Orchestrator service boundary skill. Updated for Module Separation & Runtime Redesign."
---

# gateway-orchestrator-contract

## Purpose

Use this when changing the Gateway-Orchestrator interface. Target protocol is gRPC streaming; HTTP/SSE is compatibility only.

## Authoritative source order

1. `docs/superpowers/specs/2026-05-26-module-separation-and-runtime-redesign.md`
2. `docs/superpowers/specs/2026-05-27-module-separation-runtime-redesign.md`
3. Contracts listed in this skill
4. PDR product goals only
5. Sprint/UML as supplemental demo/product context only

## Active architecture facts

```text
pkg/adk                 pure ADK engine
pkg/runtime             runtime framework over ADK
services/gateway        public Gateway, auth, SSE, persistence
services/orchestrator   planner/router/executor/dispatcher
services/agents/*       child A2A agents
frontend                React client, Gateway-only access
```

Legacy paths are not implementation targets for new-architecture work:

```text
server/**               legacy reference only
agents/**               legacy reference only
```

## Streaming event semantics

When the Orchestrator forwards streamed agent output, each text event becomes a
`message_delta` event. The Frontend MUST treat `message_delta` as **append**
(accumulate by `messageId`), never replace. `message_start` / `message_end`
bracket one logical message; multiple `message_delta` may occur between them.

## Contracts to read first

- `docs/contracts/gateway-orchestrator.md`
- `docs/contracts/gateway-orchestrator-api.md`
- `docs/contracts/gateway-orchestrator-events.md`

## Allowed implementation targets

- `services/gateway`
- `services/orchestrator`
- `services/*/proto`

## Non-negotiable rules

- Follow the redesign plan over old PDR/Sprint directory details.
- Do not add new new-architecture work under legacy `server/` or root `agents/`.
- Do not make Frontend call Orchestrator or Child Agents directly.
- Do not put concrete LLM providers or business handlers in `pkg/adk`.
- Keep Gateway and Orchestrator as separate services.
- Treat Gateway→Orchestrator gRPC streaming as target. HTTP/SSE is temporary compatibility only unless contracts are revised.
- Treat MySQL as target persistence. SQLite is demo/profile-only unless contracts are revised.

## Required workflow

1. Identify the relevant contract files above.
2. Check whether the requested change touches cross-module fields or event lifecycles.
3. Update contract first when the boundary changes.
4. Implement only in allowed targets.
5. Add/update tests for the touched module.
6. Report changed files, tests run, and any remaining mismatch against the redesign plan.

## Completion checklist

- [ ] No stale old-path instructions were introduced.
- [ ] Contract and implementation agree.
- [ ] Public Gateway API remains separate from internal service API.
- [ ] `agentName`, `runId`, `threadId`, and `requestId/traceId` are preserved when relevant.
- [ ] Errors are sanitized and do not expose secrets or internal URLs.
- [ ] Tests or a clear blocker are reported.
