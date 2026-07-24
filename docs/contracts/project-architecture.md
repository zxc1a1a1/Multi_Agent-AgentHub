# Project Architecture Contract

**Status:** Active
**Version:** AgentHub 2.0
**Product source:** `docs/pdr/AgentHub-2.0-PDR.md`
**Owner:** AgentHub

## 1. Purpose

This Contract defines the system boundary and architectural source of truth for AgentHub 2.0.

AgentHub 2.0 is:

> A multi-conversation IM system supporting direct Agent chat, user-selected multi-Agent collaboration, automatic Agent selection, LLM-generated plans requiring user confirmation, registered remote Agents, controlled context, and persistent versioned Artifacts with web preview.

It is not:

- a fixed ten-Agent showcase;
- a repository-review-only product;
- a visual workflow editor;
- an unrestricted autonomous Agent loop;
- a system in which Planner, Context Manager, Synthesizer, summarizer, or Preview Renderer is an Agent.

## 2. Source precedence

1. Current explicit user decision.
2. `docs/pdr/AgentHub-2.0-PDR.md`.
3. This architecture Contract.
4. Active domain Contracts.
5. Schema, OpenAPI, approved ADR, implementation, and tests.
6. `docs/legacy/**` and older Sprint/UML/redesign/v0.x/v1.x documents.

Legacy sources are historical evidence and MUST NOT override an active 2.0 source.

## 3. System topology

```text
Frontend
  -> Gateway
  -> Orchestrator
  -> Registered Agents
```

Expanded:

```text
Frontend
  ├── Conversation management
  ├── Chat timeline
  ├── Agent selector and registration UI
  ├── Plan confirmation
  ├── Artifact workspace
  └── Sandpack preview
        |
        | Public API + SSE/AG-UI
        v
Gateway
  ├── Authentication and object authorization
  ├── Conversation/Message/Run/Plan APIs
  ├── Agent registration API
  ├── Attachment/Artifact/Search APIs
  └── Event replay and stream mapping
        |
        | Private internal protocol
        v
Orchestrator
  ├── Mode Resolver
  ├── LLM Planner
  ├── Parser / Normalizer / Validator / Repair
  ├── Confirmation Gate
  ├── Dependency Executor
  ├── Context Manager
  ├── Agent Registry
  ├── A2A Dispatcher
  ├── Result Synthesizer
  └── Run/Event/Artifact coordination
        |
        | A2A
        v
Registered Agents
  ├── Built-in CodeAgent
  ├── Built-in WebAgent
  ├── Configured Agents
  └── Remote registered Agents
```

## 4. Product and execution resources

User-visible domain resources:

```text
User
Conversation
ConversationMember
Message
Run
Plan
PlanVersion
AgentInvocation
Attachment
Artifact
ArtifactVersion
```

Internal supporting resources:

```text
Event
ContextSnapshot
ConversationSummary
RegisteredAgent
AgentVersion
AgentHealthCheck
AgentCredential
RunAgentSnapshot
```

A Message is not an Event.
A PlanStep is not an Agent.
An Artifact is not a Tool Call.
A Skill is not an Agent.
A Preview is not an Agent or Tool.

## 5. Agent model

### Built-in

```text
CodeAgent
WebAgent
```

These are stable reference Agents, not the complete platform boundary.

### Registered

A new invocation candidate is:

```text
registered
+ enabled
+ health acceptable
+ authorized
+ valid AgentVersion
+ applicable Skill
```

Registration source may be:

```text
builtin
config
remote
```

Future platform-managed Agent resources remain separate from remote A2A registration.

### Internal modules

The following MUST NOT register as Agents:

```text
Planner
Plan Validator/Repairer
Context Manager
Result Synthesizer
Conversation Summarizer
Auto Title Generator
Preview Renderer
```

## 6. Conversation modes

### Direct

One eligible Agent, no multi-Agent Plan.

### Manual Multi-Agent

The Planner uses only the user-selected eligible Agent set. Execution requires exact PlanVersion confirmation.

### Auto

The Planner selects from the current eligible Registry Catalog. Execution requires exact PlanVersion confirmation.

A material Replan creates a new PlanVersion and requires reconfirmation.

## 7. Boundaries

### Frontend

React/TypeScript IM UI, AG-UI reducer, Plan confirmation, Artifact workspace, Sandpack Preview.

Frontend calls Gateway only.

### Gateway

Public API, auth, object authorization, Conversation/Message access, registration endpoints, AG-UI/SSE replay.

Gateway does not dispatch Agents and does not call model providers.

### Orchestrator

Planning, confirmation, context, registry, A2A dispatch, dependency execution, synthesis, and internal Run/Event state.

### Agent Runtime

Framework-neutral handler lifecycle and A2A/model/Tool adapters for built-in or future managed Agent services.

Runtime does not own Planner, Registry, Conversation, public API, or AG-UI.

### Provider layer

Use-case policy and provider/model adapters.

Provider state is optional optimization metadata, not Conversation truth.

## 8. Persistence and context

Default 2.0 single-node profile:

```text
SQLite
WAL
goose
sqlc
FTS5
```

Complete history is persisted. Model/Agent context is selected and bounded through ContextSnapshot.

MySQL and gRPC are not implicit implementation targets.

## 9. Artifacts and preview

ArtifactVersion is immutable.

User and Agent edits create new versions with conflict validation.

Web preview:

```text
WebProjectArtifact
-> exact ArtifactVersion
-> authorized mapping
-> Sandpack sandbox
```

Unknown output or project type does not execute.

## 10. Language boundary

```text
Go          core online services and adapters
TypeScript  React frontend and preview workspace
Python      evaluation, data processing, bounded helper scripts
```

A different core-service language requires an approved ADR.

## 11. Contract-first rule

For boundary changes:

```text
PDR/Contract
-> Schema/OpenAPI/ADR
-> Mock/fixture
-> implementation
-> tests
-> review
```

## 12. 2.0 initial freeze

The active Contract set is frozen as `2.0-initial` after Batch 4.

After freeze:

- wording and defect corrections update the owning Contract;
- cross-module boundary changes require an ADR and affected Contract updates;
- implementation detail does not create a new Contract by default;
- new core Skills require a non-overlapping ownership justification;
- legacy documents stay under `docs/legacy/**`.
