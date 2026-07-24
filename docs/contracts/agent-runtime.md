# Agent Runtime Contract

**Status:** Active
**Version:** AgentHub 2.0
**Supersedes active `adk-runtime` terminology**

## 1. Purpose

Agent Runtime is the reusable execution framework used by built-in and future AgentHub-managed Agents.

It provides:

- A2A server adapter;
- handler lifecycle;
- model adapter access;
- local/MCP Tool adapter access;
- cancellation and deadlines;
- structured Message/Artifact output;
- streaming;
- deterministic Mock mode;
- health and telemetry.

It does not own:

- Conversation;
- Planner or Plan;
- Agent Registry lifecycle;
- Gateway/AG-UI;
- public API;
- global Artifact storage;
- model provider business configuration.

## 2. Layering

```text
Agent service
├── AgentCard and Skill declarations
├── business handler/prompt
└── Agent Runtime
    ├── A2A server adapter
    ├── invocation context
    ├── model adapter
    ├── Tool adapter
    ├── output encoder
    └── telemetry
```

Shared runtime packages remain domain-neutral.

## 3. Invocation context

```text
AgentInvocationContext
├── invocationId
├── runId
├── planStepId
├── agentId
├── agentVersion
├── skillId
├── deadline/cancellation
├── authorized context input
├── ArtifactRefs
├── Tool policy
├── trace context
└── safe metadata
```

Runtime does not fetch arbitrary Conversation history.

## 4. Handler contract

Conceptual interface:

```text
Handle(ctx, input, emitter) -> result/error
```

Handler may emit:

- Message start/delta/end;
- Artifact candidate/update;
- Tool activity;
- progress/status.

Handler returns one terminal outcome.

## 5. A2A adapter

- use official A2A SDK/types;
- map AgentCard, Message, Task, Artifact, stream, and cancellation;
- preserve remote/local task identifiers;
- sanitize metadata/errors;
- no AG-UI dependency in Agent service;
- built-in and registered Agents use compatible invocation semantics.

## 6. Model adapter

Runtime consumes the LLM Provider Contract.

Required:

- request use-case metadata;
- timeout/cancellation;
- streaming or buffered support;
- usage/token reporting;
- structured-output support;
- bounded retry before visible output;
- provider error normalization;
- secret redaction.

Business prompts and model selection live in Agent configuration/service.

## 7. Tool adapters

Tool sources:

```text
local trusted adapter
MCP adapter
```

Rules:

- Tool schema is explicit;
- Tool is allowlisted by Agent/Skill policy;
- arguments are validated;
- timeout/cancellation;
- output byte/item limits;
- sensitive Tool needs separate approval where policy requires;
- Tool result is untrusted model input;
- runtime does not reimplement MCP base protocol.

## 8. Output

### Message

Stable role/content and streaming lifecycle.

### Artifact

Runtime emits a candidate following Artifact Contract. Persistence and version conflict are owned by AgentHub domain services.

### Error

Structured:

```text
code
safeMessage
retryable
partialOutput
internalCause for protected logs only
```

## 9. Streaming

- start before deltas;
- ordered deltas;
- one end/terminal event;
- cancellation stops further emission;
- no full retry after visible output without resumable semantics;
- partial output remains attributable to the invocation.

## 10. Mock mode

Each built-in Agent supports deterministic Mock mode:

- no API key;
- fixed fixtures based on explicit inputs;
- stable streaming chunks;
- stable Artifact fixtures;
- controllable error/timeout cases;
- suitable for CI and Compose smoke.

Mock mode must not claim real web access or model reasoning.

## 11. Configuration

Common configuration categories:

```text
listen/public URL
Agent identity/version
model provider/model/base URL/secret reference
Tool/MCP endpoints and policy
timeouts and limits
Mock mode
telemetry endpoint/service name
```

Secrets are not part of AgentCard or logs.

## 12. Health/readiness

Health:

- process/runtime alive.

Readiness:

- handler initialized;
- required configuration valid;
- optional model/Tool dependencies classified;
- AgentCard available.

Mock profile may be ready without provider credentials. Real-model profile fails clearly if required secret/model is absent.

## 13. Telemetry

Runtime emits spans/metrics with:

```text
runId
invocationId
agentId
agentVersion
skillId
model/provider
tool name
artifact type
status/error
latency
token usage
```

No private reasoning or credential.

## 14. Errors

```text
agent_runtime_invalid_input
agent_runtime_canceled
agent_runtime_timeout
model_unavailable
model_output_invalid
tool_not_allowed
tool_input_invalid
tool_failed
tool_output_too_large
artifact_output_invalid
stream_failed
configuration_invalid
```

## 15. Required tests

- handler success/error;
- cancellation/deadline;
- streaming order;
- no duplicate retry after output;
- model structured output;
- local and MCP Tool fixtures;
- Tool allowlist;
- Artifact candidate;
- deterministic Mock;
- readiness in Mock and real profiles;
- secret/error redaction;
- race test for concurrent invocations.
