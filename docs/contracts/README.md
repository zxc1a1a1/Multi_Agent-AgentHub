# AgentHub Active Contract Directory

**Status:** Active
**Version:** AgentHub 2.0

## Source-of-truth order

1. Current explicit user decision.
2. Approved AgentHub 2.0 PDR and accepted corrections.
3. `project-architecture.md`.
4. The active domain Contract in this directory.
5. JSON Schema, OpenAPI, ADR, implementation, and tests.
6. `docs/legacy/**` and older Sprint/UML/redesign documents as historical references only.

## Contract-first rule

```text
Contract
-> Schema/OpenAPI
-> deterministic fixture or Mock
-> implementation
-> contract/integration/security tests
-> review
```

## Active core contracts

### Product and orchestration

```text
project-architecture.md
conversation.md
planning-approval.md
context-management.md
agent-registry.md
platform-api.md
openapi.yaml
gateway-orchestrator.md
agui-events.md
```

### Agent and runtime

```text
a2a-agent-card.md
a2a-task.md
a2a-errors.md
agent-runtime.md
llm-provider.md
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
```

Workflow Skills may add implementation/review gates but do not replace domain Contracts.

## Retired active names

The following names are historical after AgentHub 2.0:

```text
intent-orchestration-contract
adk-runtime-contract
frontend-runtime-skills-contract
```

Their previous content is retained under `docs/legacy/contracts/pre-v2/`.

## Boundary reminders

- CodeAgent and WebAgent are built-in reference Agents, not the only registrable Agents.
- Planner, Synthesizer, Context Manager, summarizer, and preview renderer are not Agents.
- Multi-Agent execution requires an LLM-generated, validated, user-confirmed PlanVersion.
- Artifact is a persistent resource; preview is not a fake Tool Call.
- Complete history is persisted, while model context is selected and bounded.
- Frontend calls Gateway only.
- MySQL and gRPC are not implicit mandatory targets.
