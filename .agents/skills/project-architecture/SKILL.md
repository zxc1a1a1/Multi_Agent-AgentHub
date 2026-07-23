---
name: project-architecture
description: "AgentHub 2.0 architecture boundary skill for the multi-conversation IM system, confirmed LLM planning, dynamic Agent registration, context management, and artifact preview."
---

# project-architecture

## Purpose

Use this Skill before any cross-module, protocol, persistence, Agent lifecycle, conversation, planning, context, or artifact change.

AgentHub 2.0 is a **multi-conversation Agent IM system**. It is not a generic workflow editor and is not limited to a fixed set of Agents.

## Authoritative source order

1. The user's current explicit instruction.
2. The approved AgentHub 2.0 PDR and accepted follow-up decisions.
3. `docs/contracts/project-architecture.md`.
4. The active domain Contract for the changed area.
5. JSON Schema, OpenAPI, ADR, implementation, and tests.
6. Legacy Sprint, UML, redesign, and v1.x documents as historical references only.

When an old redesign document conflicts with an active 2.0 Contract, the active 2.0 Contract wins.

## Contracts to read first

Select only the contracts relevant to the task:

- `/conversation-contract`
- `/planning-approval-contract`
- `/context-management-contract`
- `/agent-registry-contract`
- `/data-persistence-contract`
- `/gateway-orchestrator-contract`
- `/a2a-agent-contract`
- `/artifact-contract`
- `/security-boundary-contract`
- `/testing-review-contract`

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
  -> Executor
  -> Context Manager
  -> Agent Registry
  -> Result Synthesizer
```

- CodeAgent and WebAgent are the default built-in reference Agents.
- Other conforming Agents may be registered dynamically.
- Planner, Context Manager, and Synthesizer are Orchestrator components, not Agents.
- Direct Mode invokes one selected Agent without a multi-Agent Plan.
- Manual Multi-Agent and Auto Mode require an LLM-generated, validated, user-confirmed PlanVersion.
- Web preview is rendered from a persistent Artifact, not modeled as a dedicated Agent.
- Complete conversation history is persisted, but only selected context is sent to a model.

## Module boundaries

```text
frontend                  IM UI, Agent selector, Plan confirmation, artifact preview
services/gateway          public API, auth, conversation/message access, SSE/AG-UI
services/orchestrator     planning, confirmation gate, context, registry, execution
services/agents/*         built-in or managed Agent services
pkg/*                     shared contracts, adapters, storage, telemetry
server/**                 legacy
agents/**                 legacy
```

## Non-negotiable rules

- Frontend MUST call Gateway only.
- Gateway MUST NOT call a Child Agent directly.
- Orchestrator MUST be the only Agent dispatch entry.
- A multi-Agent Plan MUST NOT execute before explicit confirmation.
- Planner MUST use the eligible Agent Catalog from Registry; it MUST NOT hardcode Agent names as the platform boundary.
- Manual Multi-Agent Mode MUST NOT add an Agent the user did not select.
- Registered Agent lifecycle and A2A invocation semantics MUST remain separate.
- Agent, Skill, Tool, Workflow, Artifact, and Preview MUST NOT be used as interchangeable concepts.
- SQLite is the default 2.0 single-node persistence profile; MySQL/gRPC are not mandatory targets.
- New implementation MUST NOT be added under legacy `server/**` or root `agents/**`.
- Existing protocol libraries and SDKs SHOULD be reused instead of reimplementing A2A, MCP, AG-UI, browser bundling, or database migration infrastructure.

## Required workflow

1. Identify the affected domain and active Contract.
2. Lock the allowed file scope.
3. Update Contract and Schema before implementation when a boundary changes.
4. Implement the smallest complete slice.
5. Add contract, unit, integration, and negative tests as applicable.
6. Report modified, created, deleted, and intentionally untouched files.
7. Report tests run and unresolved Contract/implementation mismatches.

## Completion checklist

- [ ] The change preserves the multi-conversation IM product model.
- [ ] Built-in Agents are not confused with the set of all registrable Agents.
- [ ] Planner/Synthesizer/Context Manager were not introduced as Agents.
- [ ] Unconfirmed Plan execution is impossible.
- [ ] Conversation and context isolation are preserved.
- [ ] Frontend, Gateway, Orchestrator, Registry, and Agent boundaries are preserved.
- [ ] No legacy source was treated as a 2.0 fact source.
- [ ] Errors and logs do not expose secrets or internal credentials.
- [ ] Tests or a concrete blocker are reported.
