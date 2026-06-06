# New Architecture Source of Truth

**Status:** Active
**Owner:** AgentHub
**Primary source:** Module Separation & Runtime Redesign


## Authoritative source order

1. `docs/superpowers/specs/2026-05-26-module-separation-and-runtime-redesign.md`
2. `docs/superpowers/specs/2026-05-27-module-separation-runtime-redesign.md`
3. PDR product goals only, not old module layout
4. Sprint/UML as supporting product/demo references only

Engineering details MUST follow the module separation redesign. Old `server/` and root `agents/` are legacy reference implementations unless a task explicitly says otherwise.


## Purpose

This contract prevents future work from drifting back to the old `server/` + root `agents/` layout.

## Active module boundaries

| Module | Path | Responsibility | Forbidden responsibility |
|---|---|---|---|
| ADK engine | `pkg/adk` | Agent, Model, Tool, Plugin, Runner, Session interfaces, Event types, A2A adapter | YAML config, concrete LLM providers, business handlers |
| Runtime framework | `pkg/runtime` | Config, registries, providers, session implementations, pruning, skills, AG-UI translator, launcher | Business-specific Agent logic |
| Gateway | `services/gateway` | Public HTTP/SSE entry, auth, conversation/message persistence, Orchestrator client | Planning, agent routing, tool execution |
| Orchestrator | `services/orchestrator` | Planner, router, executor, dispatcher, A2A calls, event conversion | Frontend API, browser auth, message storage |
| Child Agents | `services/agents/*` | Agent-specific behavior, A2A server, AgentCard | Frontend protocol, Gateway persistence |
| Frontend | `frontend` | React UI, AG-UI event handling, skill rendering | Direct Orchestrator/Agent calls |

## Legacy path rule

New development MUST NOT target these paths unless explicitly doing migration/removal:

```text
server/**
agents/**
```

## Current known gaps

- `SQLSessionService` must satisfy `adk.SessionService`, including `GetOrCreate`.
- `pkg/runtime/registry/agent.go` must implement Agent Factory Registry and YAML-to-Agent construction.
- Gateway→Orchestrator target protocol is gRPC streaming. HTTP/SSE is temporary compatibility only.
- Canonical Docker topology is six services: frontend, gateway, orchestrator, code-agent, web-agent, mysql.
