# AgentHub 2.0 Platform API Contract

**Status:** Active
**Owner:** Gateway
**Machine-readable source:** `docs/contracts/openapi.yaml`

## 1. Purpose

Platform API is the public HTTP/SSE contract between Frontend and Gateway.

It exposes AgentHub product resources:

```text
Conversation
Message
Run
Plan and PlanVersion
Registered Agent
Agent Registration
Attachment
Artifact and ArtifactVersion
Event stream
```

It does not expose Orchestrator, A2A, remote Agent, or provider APIs.

## 2. Public boundary

Allowed public paths:

```text
/api/**
/health
```

Deprecated compatibility paths may remain temporarily:

```text
/agui/runs
/api/chat
```

They MUST delegate to the 2.0 domain path and MUST NOT become the source for new features.

Forbidden public paths:

```text
/internal/**
/orchestrator/**
/a2a/**
/.well-known/agent-card.json
/.well-known/agent.json
remote Agent endpoints
```

## 3. Authentication and authorization

- Public API uses Gateway authentication.
- Every resource read/write enforces object-level authorization.
- Conversation authorization is inherited by Messages, Runs, Plans, Attachments, and Artifacts.
- Agent visibility and invocation authorization come from Agent Registry policy.
- Agent credentials are accepted only through write-only fields and are never returned.
- Token/query-string authentication is forbidden.

## 4. Common conventions

### IDs

```text
conversationId
messageId
runId
planId
agentId
registrationId
attachmentId
artifactId
```

AG-UI `threadId` is a protocol alias whose value equals `conversationId`. It is not an independent business resource.

### Pagination

List endpoints use cursor pagination:

```text
pageSize
pageToken
nextPageToken
```

Default and maximum page size are implementation configuration and MUST be bounded.

### Idempotency

Message creation requires:

```text
clientMessageId
```

Plan confirmation requires:

```text
idempotencyKey
planId
version
```

A duplicate idempotent request returns the existing result.

### Errors

```json
{
  "error": {
    "code": "plan_version_stale",
    "message": "The plan version is no longer current.",
    "requestId": "req_...",
    "details": {}
  }
}
```

Public errors MUST NOT expose stack traces, internal URLs, credentials, provider payloads, or raw database errors.

## 5. Conversation API

Required capabilities:

```text
create
list
get
rename/update settings
pin/unpin
archive/restore
soft delete
search
list Messages
send Message
```

Conversation modes:

```text
direct
manual_multi
auto
```

Response modes:

```text
separate
synthesize
```

Conversation create/update accepts Agent IDs, not display names.

Direct Mode requires one eligible Agent.
Manual Multi-Agent Mode requires at least two selected eligible Agents.
Auto Mode may omit Agent IDs.

## 6. Message and Run API

Sending a Message:

```text
POST /api/conversations/{conversationId}/messages
```

Request includes:

- `clientMessageId`;
- text content;
- attachment references;
- optional mode/Agent override;
- active Run behavior.

Response is `202 Accepted` with the committed user Message and created Run.

Default active Run behavior is product configuration. Supported explicit values:

```text
queue
cancel_current
human_input
```

Run API supports:

- get state;
- cancel;
- replay/subscribe to events.

## 7. Plan API

Multi-Agent planning creates a PlanVersion in `awaiting_confirmation`.

Required operations:

```text
get Plan
create edited version
confirm exact version
reject
regenerate
```

Rules:

- confirmation binds exact `planId + version`;
- stale versions return conflict;
- editing/regeneration creates a new version;
- unconfirmed Plans cannot execute;
- response contains no Agent credentials or hidden Registry metadata.

## 8. Agent and registration API

### Agent read API

```text
GET /api/agents
GET /api/agents/{agentId}
GET /api/agents/{agentId}/card
```

The public Agent summary is sanitized.

It may include:

```text
id
name
description
source
status
health
version
visibility
skills
input/output modes
streaming
```

It MUST NOT include:

```text
credentials
secret headers
internal admin endpoints
unredacted health errors
private system prompts
```

### Remote registration

```text
POST /api/agent-registrations
GET /api/agent-registrations/{registrationId}
```

Request submits an AgentCard URL or Agent base URL and optional write-only credential.

Registration may be asynchronous.

### Agent lifecycle

```text
refresh
check
enable
disable
remove
```

These operations require ownership/operator authorization.

### Managed Agent boundary

Future AgentHub-hosted prompt/model/tool Agents use:

```text
/api/managed-agents
```

They MUST NOT reuse remote registration semantics.

## 9. Attachment API

Required operations:

```text
upload
get metadata/content subject to authorization
delete
```

Upload is multipart and bounded by MIME, extension, and size policy.

An Attachment is not automatically trusted model context.

## 10. Artifact API

Required operations:

```text
list Conversation Artifacts
get Artifact
list versions
restore version
```

Version restore creates a new version or explicit restoration record; it does not rewrite historical versions.

Detailed Artifact and preview structure is governed by the Artifact Contract.

## 11. Event API

```text
GET /api/runs/{runId}/events
```

- returns SSE;
- authenticates the Run owner;
- supports `Last-Event-ID`;
- replays persisted missing events before live subscription;
- closes or continues according to Run lifecycle;
- emits AG-UI standard events plus approved AgentHub `CUSTOM` events.

## 12. Compatibility

`/agui/runs` and `/api/chat` may remain during migration.

They MUST:

- create the same Message/Run domain objects;
- use the same authorization;
- use the same Registry eligibility;
- use the same Plan confirmation gate;
- be marked deprecated in OpenAPI;
- have removal criteria.

## 13. Required tests

- unauthorized/forbidden object access;
- Conversation pagination and lifecycle;
- duplicate `clientMessageId`;
- Direct/Manual/Auto validation;
- stale Plan confirmation;
- unconfirmed execution rejection;
- Agent list authorization;
- remote registration secret redaction;
- registration ownership checks;
- event replay with `Last-Event-ID`;
- attachment type/size rejection;
- Artifact version authorization;
- deprecated endpoint parity.
