# A2A Invocation Contract

**Status:** Active
**Version:** AgentHub 2.0
**Protocol source:** Official A2A specification and official Go SDK

## 1. Scope

A2A is the Orchestrator↔registered Agent protocol.

It transports:

```text
Message
Task
Task status
Artifact
stream updates
cancellation
```

It does not transport AgentHub public Conversation/Plan objects as wire replacements.

## 2. Invocation input

Orchestrator converts a confirmed PlanStep or Direct Run into an A2A Message/Task request.

Input contains only:

- task instruction;
- bounded authorized context;
- approved Artifact/File/Data Parts;
- safe correlation metadata;
- protocol/extension negotiation;
- Agent-specific authentication applied by client adapter.

It MUST NOT include:

- full AgentHub Conversation history by default;
- browser bearer token;
- Registry credential value;
- unrelated Conversation content;
- hidden Planner/System prompt;
- another Agent's private trace.

## 3. Identifier mapping

```text
AgentHub conversationId   local business identifier
AgentHub runId            local execution identifier
AgentHub invocationId     local Agent call identifier
A2A task id               remote task identifier
A2A context id            remote Agent context identifier
```

A2A `contextId` is not AgentHub `conversationId`.

AgentHub correlation may use sanitized A2A metadata/extensions:

```text
agenthub.run_id
agenthub.invocation_id
agenthub.conversation_ref
agenthub.plan_step_id
traceparent or trace reference
```

Do not send user secrets in metadata.

## 4. Task lifecycle mapping

AgentHub maps official A2A Task states into invocation state.

Typical mapping:

```text
submitted/input-required/auth-required/working
completed
failed
canceled
rejected
```

Exact state names depend on the negotiated official protocol version. The adapter owns version mapping.

A terminal remote Task cannot be silently restarted as the same invocation.

## 5. Streaming

Preferred path uses official streaming operation/binding.

Stream may contain:

- Task snapshot;
- status update;
- Message;
- Artifact update/chunk.

Orchestrator converts these into internal events.

Rules:

- preserve ordering;
- append Artifact chunks according to protocol flags;
- bracket visible Message streams;
- sanitize errors;
- persist remote task/context identifiers for cancel/resubscribe;
- do not forward raw A2A events to Frontend.

Buffered compatibility is allowed when an Agent does not support streaming.

## 6. Retry

Before visible stream output:

- timeout/connection errors may use bounded retry/backoff/circuit breaker.

After visible Message or Artifact output begins:

- automatic full retry is disabled by default to avoid duplicate user-visible output;
- resumable subscribe/reconnect is preferred when the protocol/Agent supports stable sequencing;
- otherwise fail safely and preserve partial output.

## 7. Cancellation

Run cancellation propagates:

```text
Gateway cancel
-> Orchestrator context cancel
-> A2A Cancel Task when task id exists
-> local stream/subscription close
-> invocation terminal state
```

Cancellation is idempotent.

## 8. Multi-turn/input-required

If remote Task requires input or authorization:

- Orchestrator emits structured internal state;
- Gateway exposes an approved UI/API interaction;
- user response returns through Orchestrator;
- raw remote prompts do not bypass authorization or confirmation.

A material change to the AgentHub Plan may require Replan confirmation.

## 9. Artifact mapping

Official A2A Artifacts/Parts are converted to AgentHub Artifact inputs.

Known types map through Artifact Contract. Unknown types degrade to:

- structured JSON viewer;
- authorized file download;
- safe unsupported type.

Unknown remote output MUST NOT execute automatically.

## 10. Tool boundary

A remote Agent's declared capability does not grant AgentHub Tool permissions.

Agent-internal tools remain opaque unless separately exposed through an approved MCP/Tool contract.

## 11. Compatibility adapters

Existing repository custom JSON-RPC/SSE types may be retained temporarily behind an adapter.

Migration rules:

- official SDK-facing contract at the boundary;
- no new feature added only to the legacy wire shape;
- fixtures cover both legacy and official mapping;
- remove legacy path only after built-in Agent migration and compatibility tests.

## 12. Required tests

- official Message/Task invocation fixture;
- streaming status/message/artifact mapping;
- buffered compatibility;
- cancel propagation;
- input-required mapping;
- remote context ID not confused with Conversation ID;
- pre-stream retry;
- no post-stream duplicate retry;
- unknown Artifact safe fallback;
- metadata/credential redaction;
- built-in and dynamically registered Agent use the same dispatcher.
