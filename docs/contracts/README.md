# AgentHub Active Contract Directory

**Status:** Active
**Version:** AgentHub 2.0
**Contract baseline:** `2.0-initial`
**Product source:** `docs/pdr/AgentHub-2.0-PDR.md`

## Source-of-truth order

1. Current explicit user decision.
2. `docs/pdr/AgentHub-2.0-PDR.md`.
3. `project-architecture.md`.
4. The active domain Contract in this directory.
5. JSON Schema, OpenAPI, approved ADR, implementation, and tests.
6. `docs/legacy/**` and old Sprint/UML/redesign/v0.x/v1.x documents.

## Contract-first rule

```text
PDR/Contract
-> Schema/OpenAPI/ADR
-> deterministic fixture or Mock
-> implementation
-> contract/integration/security tests
-> review
```

## Active core contracts

### Governance and architecture

```text
project-architecture.md
contract-governance.md
ai-collaboration-workflow.md
code-style.md
commit-security-review.md
```

### Product and orchestration

```text
conversation.md
planning-approval.md
context-management.md
agent-registry.md
platform-api.md
openapi.yaml
gateway-orchestrator.md
agui-events.md
```

### Agent, runtime, and model

```text
a2a-agent-card.md
a2a-task.md
a2a-errors.md
agent-runtime.md
llm-provider.md
llm-provider.schema.json
```

### Artifact and preview

```text
artifact.md
artifact.schema.json
web-project-preview.md
web-project-preview.schema.json
```

### Persistence and engineering

```text
data-model.md
security-boundaries.md
observability-debugging.md
testing-review.md
docker-compose-delivery.md
```

## Active Skill names

```text
project-architecture
ai-collaboration-workflow
conversation-contract
planning-approval-contract
context-management-contract
agent-registry-contract
platform-api-contract
gateway-orchestrator-contract
agui-event-contract
a2a-agent-contract
agent-runtime-contract
llm-provider-contract
artifact-contract
web-project-preview-contract
data-persistence-contract
security-boundary-contract
observability-debugging-contract
testing-review-contract
docker-compose-delivery
code-style-and-conventions
commit-security-review
```

Workflow and review Skills add procedure and gates; they do not replace domain Contracts.

## Retired active names

```text
intent-orchestration-contract
adk-runtime-contract
frontend-runtime-skills-contract
```

Previous content is retained under `docs/legacy/**`.

## Boundary reminders

- CodeAgent and WebAgent are built-in reference Agents, not all registrable Agents.
- Planner, Synthesizer, Context Manager, summarizer, title generator, and Preview Renderer are not Agents.
- Multi-Agent execution requires a validated and confirmed exact PlanVersion.
- Agent registration does not grant Tool permission.
- Artifact is persistent and versioned; Preview is not a fake Tool Call.
- Complete history is persisted while model context is selected and bounded.
- Provider state is not the AgentHub Conversation fact source.
- Frontend calls Gateway only.
- SQLite/WAL is the default 2.0 profile.
- MySQL and gRPC are not implicit mandatory targets.
- Core online services use Go; Python is reserved for evaluation and bounded scripts unless an ADR says otherwise.

## Freeze policy

`2.0-initial` is the baseline for implementation.

Do not add a new core Contract or Skill for an implementation detail. Cross-module boundary changes require an approved ADR and updates to the owning Contracts.
