# Project Architecture Contract

**Status:** Active  
**Version:** AgentHub 2.0  
**Owner:** AgentHub  

## 1. Purpose

This Contract defines the system boundary and architectural source of truth for AgentHub 2.0.

AgentHub 2.0 is:

> A multi-conversation IM system that supports direct Agent chat, user-selected multi-Agent collaboration, automatic Agent selection, LLM-generated plans that require user confirmation, registered remote Agents, controlled context construction, and persistent artifacts with web preview.

It is not:

- a fixed ten-Agent showcase;
- a repository-review-only product;
- a visual workflow editor;
- an unrestricted autonomous Agent loop;
- a system in which Planner, Context Manager, or Synthesizer are separate Agents.

## 2. Source precedence

1. Current explicit user decision.
2. Approved AgentHub 2.0 PDR and accepted corrections.
3. This architecture Contract.
4. Active domain Contracts.
5. Schema, OpenAPI, ADR, implementation, and tests.
6. Legacy Sprint, UML, redesign, v0.x, and v1.x documents.

Legacy documents are historical evidence. They MUST NOT override an active 2.0 Contract.

## 3. System topology

```text
Frontend
  ├── Conversation management
  ├── Chat timeline
  ├── Agent selector
  ├── Plan confirmation
  ├── Artifact workspace
  └── Web preview
        |
        | Public API + SSE/AG-UI
        v
Gateway
  ├── Authentication and authorization
  ├── Conversation and message API
  ├── Plan confirmation API
  ├── Attachment and artifact API
  └── Event stream
        |
        | Internal run protocol
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
  └── Result Synthesizer
        |
        | A2A
        v
Registered Agents
  ├── Built-in CodeAgent
  ├── Built-in WebAgent
  └── Dynamically registered conforming Agents
```

## 4. Product-level resources

The primary user-visible resources are:

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

Internal execution resources include:

```text
Event
ContextSnapshot
ConversationSummary
RegisteredAgent
AgentVersion
AgentHealthCheck
RunAgentSnapshot
```

A Message is not an Event.  
A PlanStep is not an Agent.  
An Artifact is not a Tool Call.  
A Skill is not an Agent.

## 5. Agent model

### 5.1 Built-in Agents

AgentHub 2.0 ships and validates two reference Agents:

```text
CodeAgent
WebAgent
```

They are the default built-in Agents, not the only Agents the platform may invoke.

### 5.2 Registered Agents

A conforming Agent may enter the platform through:

```text
builtin
config
remote
```

An eligible registered Agent MUST be:

```text
registered
+ enabled
+ healthy
+ authorized for the user/conversation
+ backed by a valid AgentVersion
+ equipped with a valid capability/skill declaration
```

### 5.3 Non-Agent components

The following components MUST NOT register as Agents:

- LLM Planner;
- Result Synthesizer;
- Context Manager;
- Conversation summarizer;
- title generator;
- preview renderer.

## 6. Conversation modes

### 6.1 Direct Mode

The user selects one eligible Agent.

```text
User Message -> selected Agent -> Agent response
```

Direct Mode does not require a multi-Agent Plan confirmation.

### 6.2 Manual Multi-Agent Mode

The user selects two or more eligible Agents. The LLM Planner may assign work only within that selected set.

```text
User selection -> LLM Plan -> validation -> user confirmation -> execution
```

### 6.3 Auto Mode

The LLM Planner selects from the current eligible Agent Catalog.

```text
Eligible Registry Catalog -> LLM Plan -> validation -> user confirmation -> execution
```

## 7. Planning boundary

- Planner is an Orchestrator module.
- A multi-Agent Plan is a versioned first-class resource.
- The exact confirmed PlanVersion is the only executable version.
- Editing or regenerating creates a new PlanVersion.
- A material Replan requires new confirmation.
- The internal dependency graph may represent serial, parallel, or mixed execution.
- DAG terminology is an internal scheduling detail and is not the main product UI.

## 8. Context boundary

- Full source history is persisted.
- The model receives only a Context Manager selection.
- Context is isolated by Conversation by default.
- Cross-conversation retrieval requires explicit user action or an explicit policy.
- Different Agents receive different context projections.
- ContextSnapshot records the selected inputs, not private model reasoning.

## 9. Artifact and preview boundary

- Agent outputs may create persistent Artifacts.
- Artifact modifications create immutable ArtifactVersions.
- A web project is represented as a `WebProjectArtifact`.
- Frontend preview uses a sandboxed renderer such as Sandpack.
- Preview is a renderer capability, not an Agent and not automatically a Tool Call.
- Unknown artifact types MUST degrade safely to structured view or download.

## 10. Service boundaries

### Frontend

MUST:

- call Gateway only;
- render Conversation, Message, Plan, AgentInvocation, and Artifact state;
- require explicit confirmation before sending a Plan execution request;
- sandbox web preview.

MUST NOT:

- call Orchestrator or registered Agents directly;
- infer authorization from AgentCard;
- execute unknown Artifact payloads.

### Gateway

MUST:

- expose the public API;
- enforce object-level authorization;
- handle idempotent message submission;
- expose event replay;
- mediate Plan confirmation and artifact access.

MUST NOT:

- perform Agent planning;
- call an Agent directly;
- silently mutate an approved Plan.

### Orchestrator

MUST:

- build Planner Catalog from Registry;
- create and validate Plans;
- block execution until confirmation;
- build Agent-specific context;
- dispatch through A2A;
- preserve Agent and Plan snapshots.

MUST NOT:

- expose its internal API to Frontend;
- bypass Registry state;
- silently replace a confirmed Agent after execution begins.

### Registered Agent

MUST:

- expose a valid AgentCard or equivalent configured definition;
- declare skills/capabilities;
- implement supported A2A invocation behavior;
- return structured errors and supported outputs.

MUST NOT:

- receive another Conversation's context;
- obtain platform secrets;
- assume its registration grants Tool permissions.

## 11. Persistence profile

AgentHub 2.0 default profile:

```text
single-node
SQLite
WAL
sqlc
goose
FTS5
```

PostgreSQL may be added as a later deployment profile behind the same repository interfaces. MySQL is not a mandatory architecture target.

Transport between Gateway and Orchestrator is contract-driven. Existing internal HTTP streaming may remain until an explicit transport migration Contract is approved. gRPC is not an implicit mandatory target.

## 12. Reuse boundary

Prefer established implementations:

- official A2A specification/SDK for Agent communication;
- official MCP SDK for Tool protocol;
- AG-UI event semantics for frontend streaming;
- Sandpack for browser frontend preview;
- sqlc for typed data access;
- goose for migrations;
- SQLite FTS5 for 2.0 full-text search;
- OpenTelemetry for traces and metrics.

AgentHub owns:

- Conversation/Message/Run/Plan domain semantics;
- Planner prompt, validation, and confirmation gate;
- Registry policy and authorization;
- context projection;
- artifact versioning;
- IM-to-Agent execution mapping;
- product-specific evaluation.

## 13. Legacy paths

```text
server/**
agents/**
```

are legacy references unless a dedicated migration task explicitly changes them.

## 14. Review blockers

A change is blocking if it:

- makes Frontend call Orchestrator or an Agent directly;
- executes a multi-Agent Plan without confirmation;
- hardcodes the platform to only CodeAgent and WebAgent;
- allows Planner to use an unregistered, disabled, unhealthy, or unauthorized Agent;
- treats Planner, Synthesizer, Context Manager, or Preview as an Agent;
- mixes Conversation history across users or Conversations;
- treats Artifact preview as arbitrary executable content;
- introduces new implementation under legacy paths;
- restores old MySQL/gRPC requirements without an approved Contract change.
