# AgentHub 2.0 Architecture Overview

**Status:** Active entry document. Product facts come from [AgentHub 2.0 PDR](../pdr/AgentHub_2.0_PDR.md); domain rules come from the [Active Contract index](../contracts/README.md).

## Status

**2.0 Contract 已冻结，业务实现迁移中。** Current Implementation must not be described as completed 2.0 Target unless implementation and tests establish that fact.

## 2.0 Target

```text
Frontend -> Gateway -> Orchestrator -> Registered A2A Agents
```

The Orchestrator contains the LLM Planner, Plan Validator, User Confirmation Gate, Context Manager, Agent Registry and Result Synthesizer. These are internal platform modules, not Agents.

CodeAgent and WebAgent are built-in stable reference Agents. The platform Target supports runtime registration of additional A2A-conforming Agents; it is not a fixed ten-Agent architecture.

Direct invokes one eligible Agent. Manual Multi-Agent and Auto require an LLM-generated, locally validated and user-confirmed exact PlanVersion before execution. Manual Multi-Agent may not add an Agent that the user did not select.

SQLite/WAL is the default 2.0 single-node persistence target. MySQL and gRPC are not implicit mandatory targets. The default Target Compose has Frontend, Gateway, Orchestrator, CodeAgent and WebAgent; remote Agents register at runtime.

## Current Implementation

The repository currently retains migration-era services and a `docker-compose.new-arch.yml` topology with ten specialist Agent containers. This is an implementation transition, not the 2.0 target statement. Historical materials are retained under [docs/legacy](../legacy/).
