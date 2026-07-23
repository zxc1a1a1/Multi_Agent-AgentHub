# Gateway–Orchestrator Service Contract

**Status:** Active
**Version:** AgentHub 2.0

## 1. Purpose

This Contract defines the private service boundary between Gateway and Orchestrator.

Gateway owns public HTTP/SSE mediation and user-facing object authorization.
Orchestrator owns planning, context projection, Agent Registry, Agent dispatch, and execution state.

## 2. Topology

```text
Frontend
  -> Gateway public API
  -> Orchestrator private API
  -> eligible registered Agent through A2A
```

## 3. Gateway responsibilities

- authenticate public traffic;
- authorize Conversation and child resources;
- validate public request shape;
- enforce message idempotency;
- expose Conversation, Message, Plan, Agent, Attachment, Artifact, and Event API;
- send bounded domain requests to Orchestrator;
- translate internal events to AG-UI;
- persist or coordinate persistence according to domain ownership;
- replay events;
- sanitize public errors.

Gateway MUST NOT:

- call an Agent directly;
- call an LLM directly;
- generate or modify a Plan;
- select a replacement Agent;
- expose Agent credentials or internal endpoints.

## 4. Orchestrator responsibilities

- resolve Direct/Manual/Auto mode;
- build Registry-eligible Agent Catalog;
- create and validate PlanVersions;
- enforce the confirmation gate;
- build Agent-specific ContextSnapshots;
- execute confirmed dependencies;
- dispatch Agents through A2A;
- aggregate structured outputs;
- manage Agent Registry lifecycle and health;
- emit internal ordered events;
- preserve Plan and Agent snapshots.

Orchestrator MUST NOT:

- accept browser traffic directly;
- trust public Agent IDs without Registry checks;
- execute an unconfirmed Plan;
- silently substitute an Agent after confirmation;
- return raw provider, network, or database errors.

## 5. Run paths

### Direct Run

```text
Gateway commits user Message and Run
-> ExecuteDirectRun
-> Orchestrator validates one eligible selected Agent
-> builds context
-> dispatches Agent
-> streams internal events
-> Gateway maps/persists/emits
```

No multi-Agent Plan confirmation is required.

### Multi-Agent planning

```text
Gateway commits user Message and planning Run
-> CreatePlan
-> Orchestrator builds eligible catalog/context
-> LLM Planner creates PlanVersion
-> Validator/Repair
-> Orchestrator returns plan_created and awaiting_confirmation
-> Gateway persists/exposes Plan card
```

No AgentInvocation may start.

### Confirmed execution

```text
Frontend confirms exact version
-> Gateway authorizes and forwards ExecuteConfirmedPlan
-> Orchestrator validates confirmation and current Agent availability
-> executes
-> streams events
```

### Replan

```text
execution needs material change
-> Orchestrator creates a new PlanVersion proposal
-> Run returns to awaiting_confirmation
-> Gateway exposes proposal
-> user confirms before execution resumes
```

## 6. Persistence ownership

Logical ownership follows the Data Persistence Contract.

A single-node SQLite profile may use a common storage package, but:

- service modules use explicit repositories;
- migrations have one owner;
- model/network calls do not occur inside transactions;
- multi-process writes use WAL, busy timeout, short transactions, and bounded retry.

## 7. Identity mapping

```text
conversationId    AgentHub business Conversation
runId             AgentHub Run
planId/version    exact planning proposal
invocationId      Agent invocation record
requestId/traceId correlation
```

AG-UI `threadId` equals `conversationId` as a frontend protocol alias.

A2A `contextId` is remote-Agent protocol state and MUST NOT be treated as AgentHub `conversationId`. AgentHub correlation uses explicit sanitized metadata/extension fields.

## 8. Transport profile

2.0 contract defines semantics independent of transport.

Current supported profile may be:

```text
private HTTP JSON
SSE or streamed HTTP events
service authentication
```

A future gRPC profile must preserve the same operations, states, errors, and event ordering. It requires an explicit migration Contract/ADR.

## 9. Authorization propagation

Gateway sends a sanitized execution principal:

```text
userId
workspace/tenant scope if present
conversation authorization result
requestId
traceId
```

Gateway MUST NOT forward the browser bearer token to Agents.

Agent credentials remain in Registry/credential storage and are applied only by the A2A client adapter.

## 10. Errors

Internal errors have:

```text
code
safeMessage
retryable
runId
stepId/invocationId where applicable
internalCause only in protected logs
```

Gateway maps internal errors to public ErrorResponse and AG-UI RUN_ERROR.

## 11. Required tests

- Frontend cannot reach Orchestrator;
- Gateway cannot dispatch Agent directly;
- Direct Run invokes exactly one eligible Agent;
- CreatePlan creates no AgentInvocation;
- stale or absent confirmation blocks execution;
- confirmed Agent unavailable triggers safe failure/Replan;
- cancel propagates to active Agent/Tool;
- internal events replay in order;
- user token/Agent credential not forwarded;
- public errors are sanitized;
- current HTTP profile and future transport adapters satisfy contract tests.
