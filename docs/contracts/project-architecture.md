# Project Architecture Contract

**Status:** Active
**Owner:** AgentHub
**Primary source:** Module Separation & Runtime Redesign


## Authoritative source order

1. `docs/superpowers/specs/2026-05-26-module-separation-and-runtime-redesign.md`
2. `docs/superpowers/specs/2026-05-27-module-separation-runtime-redesign.md`
3. PDR product goals only, not old module layout
4. Sprint/UML as supporting product/demo references only

Engineering details MUST follow the module separation redesign. Old `server/` and root `agents/` are legacy reference implementations unless a task explicitly says otherwise.


## System topology

```text
Frontend ──AG-UI/SSE──> Gateway ──gRPC stream──> Orchestrator ──A2A──> Child Agents
                                  │
                                  └── MySQL persistence
```

## Current modules

```text
pkg/adk
pkg/runtime
services/gateway
services/orchestrator
services/agents/code-agent
services/agents/web-agent
frontend
```

## Rules

1. Frontend talks only to Gateway.
2. Gateway persists conversations and streams AG-UI events, but does not plan or route Agent tasks.
3. Orchestrator plans, validates, executes, and dispatches to Child Agents.
4. Child Agents are independently deployable A2A services.
5. ADK must remain pure engine: no concrete provider, no project business logic.
6. Runtime implements providers/registries/config/skills/translator over ADK.
7. Legacy `server/` and root `agents/` are reference-only.

## Profiles

| Profile | Purpose | DB | Gateway↔Orchestrator |
|---|---|---|---|
| Target | Architecture contract | MySQL | gRPC streaming |
| Demo compatibility | Temporary | SQLite allowed | HTTP/SSE allowed only if marked compatibility |

## Review blockers

A change is blocking if it:

- Imports Orchestrator into Gateway as in-process logic.
- Makes Frontend call Orchestrator or Child Agent directly.
- Adds concrete LLM providers under `pkg/adk`.
- Adds new work to legacy `server/` or root `agents/` without a migration task.
