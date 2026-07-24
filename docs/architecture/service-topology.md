# Service Topology

**Status:** Active entry document. See [PDR](../pdr/AgentHub_2.0_PDR.md) and [Compose delivery Contract](../contracts/docker-compose-delivery.md).

## 2.0 Target topology

```text
Frontend -> Gateway -> Orchestrator -> Registered A2A Agents
```

- Frontend calls Gateway only.
- Gateway exposes the public API and stream boundary; it does not call Agents or a model provider directly.
- Orchestrator is the only Agent dispatch entry and owns planning, validation, confirmation, context and registry orchestration.
- CodeAgent and WebAgent are built-in stable reference Agents.
- Remote A2A Agents are runtime-registered external resources, not mandatory Compose services.

The default 2.0 Compose Target is Frontend, Gateway, Orchestrator, CodeAgent and WebAgent. SQLite/WAL is the default single-node persistence target; MySQL and gRPC are not implicit requirements.

## Current Implementation

`docker-compose.new-arch.yml` currently defines a wider ten-Agent migration topology. This document does not change that file or claim that the five-service Target is already delivered.
