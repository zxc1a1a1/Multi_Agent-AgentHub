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
- `data-model.md`, `mysql-schema.md`, `storage-profiles.md`, `redis-usage.md`
- `runtime-harness.md`, `memory.md`, `tool-broker.md`, `event-replay.md`
- `event-bus.md` and `event-bus.schema.json` (status: `draft-frozen`; envelope/streams/consumer groups/ack/DLQ/ordering/idempotency are defined here. Redis-side queue behaviour is NOT implemented — see `event-bus.md` §0 and §9; `redis-usage.md` §9 and `event-replay.md` §4 note where the old per-run stream remains the compatibility/rollback path.)
- `docker-compose-delivery.md`

## v2 decisions

- `docs/architecture/optimization-architecture.md`
- `docs/architecture/optimization-technology-selection.md`
- `docs/architecture/adr/ADR-V2-01-mysql-redis.md` … `ADR-V2-06-event-replay.md`
- `docs/plans/agenthub-redis-sql-harness-plan.md`
