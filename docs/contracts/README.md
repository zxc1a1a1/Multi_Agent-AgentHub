# AgentHub Contract Directory

**Status:** Active

This directory contains the active contract surface for the new AgentHub architecture.

## Source-of-truth order

1. `docs/superpowers/specs/2026-05-26-module-separation-and-runtime-redesign.md`
2. `docs/superpowers/specs/2026-05-27-module-separation-runtime-redesign.md`
3. PDR product goals only
4. Sprint/UML supplemental notes only

## Current implementation target

```text
pkg/adk                 Pure ADK engine
pkg/runtime             Runtime framework over ADK
services/gateway        Frontend entry, auth, SSE, persistence
services/orchestrator   Planning and execution brain
services/agents/*       Child Agents exposed over A2A
frontend                React client that talks only to Gateway
```

Legacy directories are not active contract targets:

```text
server/                 legacy reference only
agents/                 legacy reference only
```

## Contract-first rule

All cross-service fields or lifecycle changes must update contracts first, then implementation, then tests.

```text
Contract → Implementation → Contract Test / Smoke Test → Review
```

## Current core contracts

- `project-architecture.md`
- `platform-api.md` and `openapi.yaml`
- `gateway-orchestrator.md`
- `agui-events.md`
- `a2a-agent-card.md` and `a2a-task.md`
- `adk-runtime.md`
- `intent-orchestration.md`, `planner-input.md`, `orchestration-plan.md`
- `frontend-runtime-skills.md`
- `artifact-schema.md`
- `data-model.md`, `mysql-schema.md`, `storage-profiles.md`
- `docker-compose-delivery.md`
