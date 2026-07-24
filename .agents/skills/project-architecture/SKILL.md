---
name: project-architecture
description: "AgentHub 2.0 architecture boundary skill for the multi-conversation IM system, registered Agents, confirmed LLM planning, controlled context, and versioned Artifact preview."
---

# project-architecture

## Purpose

Use this Skill before any cross-module, protocol, persistence, Agent lifecycle, conversation, planning, context, Artifact, or delivery change.

AgentHub 2.0 is a multi-conversation Agent IM system. It is not a generic workflow editor and is not limited to a fixed Agent set.

## Authoritative source order

1. Current explicit user instruction.
2. `docs/pdr/AgentHub-2.0-PDR.md`.
3. `docs/contracts/project-architecture.md`.
4. The active domain Contract for the changed area.
5. JSON Schema, OpenAPI, approved ADR, implementation, and tests.
6. `docs/legacy/**`, old Sprint/UML/redesign, and v0.x/v1.x documents as historical references only.

When an old document conflicts with the PDR or an active 2.0 Contract, the active 2.0 source wins.

## Contracts to read first

Select only the Contracts relevant to the task:

```text
/conversation-contract
/planning-approval-contract
/context-management-contract
/agent-registry-contract
/data-persistence-contract
/gateway-orchestrator-contract
/a2a-agent-contract
/agent-runtime-contract
/llm-provider-contract
/artifact-contract
/web-project-preview-contract
/security-boundary-contract
/testing-review-contract
```

## Active architecture facts

```text
Frontend
  -> Gateway
  -> Orchestrator
  -> registered A2A Agents

Orchestrator
  -> LLM Planner
  -> Plan Validator
  -> User Confirmation Gate
  -> Dependency Executor
  -> Context Manager
  -> Agent Registry
  -> Result Synthesizer
```

- CodeAgent and WebAgent are built-in stable reference Agents.
- Other conforming Agents may be registered dynamically.
- Planner, Context Manager, Synthesizer, summarizer, title generator, and Preview Renderer are not Agents.
- Direct Mode invokes one eligible Agent without a multi-Agent Plan.
- Manual Multi-Agent and Auto Mode require an LLM-generated, validated, user-confirmed PlanVersion.
- Artifact is persistent and versioned; Preview binds an exact ArtifactVersion.
- Complete conversation history is persisted, while model and Agent context is selected and bounded.
- SQLite/WAL is the default 2.0 single-node persistence profile.
- Private HTTP/streaming may remain the Gateway–Orchestrator implementation; gRPC is not an implicit target.

## Module boundaries

```text
frontend                  React/TypeScript IM UI and Sandpack workspace
services/gateway          public API, auth, conversation/message access, AG-UI/SSE
services/orchestrator     planning, confirmation, context, registry, execution
services/agents/*         built-in or future managed Agent services
pkg/*                     shared contracts, adapters, storage, telemetry
scripts/**                bounded automation and evaluation helpers
server/**                 legacy unless explicitly migrated
agents/**                 legacy unless explicitly migrated
```

## Non-negotiable rules

- Frontend MUST call Gateway only.
- Gateway MUST NOT dispatch Agents or call a model provider directly.
- Orchestrator MUST be the only Agent dispatch entry.
- Multi-Agent execution MUST NOT begin before exact PlanVersion confirmation.
- Planner MUST use the eligible Registry Catalog and MUST NOT hardcode the platform Agent boundary.
- Manual Multi-Agent Mode MUST NOT add an Agent the user did not select.
- Agent registration, Agent invocation, Tool permission, Workflow, Artifact, and Preview MUST remain distinct concepts.
- Unknown Agent output or project type MUST NOT execute automatically.
- New core online services use Go unless an approved ADR changes the language boundary.
- Python is for evaluation, data processing, and bounded helper scripts by default.
- Existing official SDKs/libraries SHOULD be reused for A2A, MCP, AG-UI semantics, preview, migration, typed SQL, and telemetry.

## Required workflow

1. Identify the affected domain and active Contract.
2. Lock the allowed file scope.
3. Update Contract and Schema first when a boundary changes.
4. Implement the smallest complete slice.
5. Add deterministic positive, negative, integration, and race tests as applicable.
6. Report modified, created, deleted, and intentionally untouched files.
7. Report tests run and unresolved Contract/implementation mismatches.

## Completion checklist

- [ ] The PDR path and active Contract were used.
- [ ] Built-in Agents were not confused with all registrable Agents.
- [ ] Internal orchestration modules were not introduced as Agents.
- [ ] Unconfirmed Plan execution is impossible.
- [ ] Conversation/context and Artifact version isolation are preserved.
- [ ] Frontend, Gateway, Orchestrator, Registry, Agent, and Provider boundaries are preserved.
- [ ] No legacy source was treated as a 2.0 fact source.
- [ ] Errors, logs, events, and debug output do not expose secrets or private reasoning.
- [ ] Tests or a concrete blocker are reported.
