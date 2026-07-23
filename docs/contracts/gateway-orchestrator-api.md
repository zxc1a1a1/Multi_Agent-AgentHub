# Gateway ↔ Orchestrator Internal API

**Status:** Active
**Version:** AgentHub 2.0
**Visibility:** Private service API

## 1. Common envelope

Every request carries:

```json
{
  "requestId": "req_...",
  "traceId": "trace_...",
  "principal": {
    "userId": "user_...",
    "tenantId": null
  }
}
```

The principal is sanitized authorization context, not the browser bearer token.

## 2. Execute Direct Run

```text
POST /internal/v2/runs/direct
```

Request:

```json
{
  "conversationId": "conv_1",
  "runId": "run_1",
  "triggerMessageId": "msg_1",
  "agentId": "agent_1",
  "agentVersion": null,
  "skillId": null,
  "contextInput": {
    "currentMessageId": "msg_1",
    "attachmentIds": [],
    "activeArtifactRefs": []
  },
  "responseMode": "separate"
}
```

Rules:

- validates Agent eligibility;
- snapshots AgentVersion;
- creates one AgentInvocation;
- streams internal events;
- does not generate a multi-Agent Plan.

## 3. Create Plan

```text
POST /internal/v2/runs/plan
```

Request:

```json
{
  "conversationId": "conv_1",
  "runId": "run_1",
  "triggerMessageId": "msg_1",
  "mode": "manual_multi",
  "selectedAgentIds": ["web-agent", "code-agent"],
  "responseMode": "synthesize",
  "contextInput": {
    "currentMessageId": "msg_1",
    "attachmentIds": [],
    "activeArtifactRefs": []
  }
}
```

Response/event result:

```json
{
  "planId": "plan_1",
  "version": 1,
  "status": "awaiting_confirmation",
  "validationResult": {}
}
```

Manual Mode may use only `selectedAgentIds`.
Auto Mode ignores an empty selected set and uses eligible Registry Catalog.

No AgentInvocation is created.

## 4. Execute Confirmed Plan

```text
POST /internal/v2/runs/{runId}/execute
```

Request:

```json
{
  "planId": "plan_1",
  "version": 1,
  "confirmedBy": "user_1",
  "confirmationId": "confirm_1",
  "idempotencyKey": "idem_1"
}
```

Rules:

- validate exact version and confirmation;
- reject stale version;
- recheck Agent authorization/availability;
- store Run Agent snapshots;
- stream execution events;
- never silently replace an unavailable Agent.

## 5. Replan

```text
POST /internal/v2/runs/{runId}/replan
```

Request:

```json
{
  "basePlanId": "plan_1",
  "baseVersion": 1,
  "reasonCode": "agent_unavailable",
  "feedback": "Use another eligible research Agent."
}
```

Response is a new PlanVersion in `awaiting_confirmation`.

## 6. Cancel Run

```text
POST /internal/v2/runs/{runId}/cancel
```

Request:

```json
{
  "reason": "user_requested"
}
```

Cancellation is idempotent and propagates to pending Steps, active A2A tasks, and Tool calls where supported.

## 7. Get Run

```text
GET /internal/v2/runs/{runId}
```

Returns internal state needed by Gateway recovery. It MUST NOT expose credentials or raw provider errors.

## 8. Eligible Agent Catalog

```text
GET /internal/v2/agents/catalog
```

Query scope includes user and optional Conversation.

Response is the sanitized Registry projection used by Agent selector and Planner.

## 9. Registry management

Gateway forwards authorized operations:

```text
POST   /internal/v2/agent-registrations
GET    /internal/v2/agent-registrations/{registrationId}
POST   /internal/v2/agents/{agentId}/refresh
POST   /internal/v2/agents/{agentId}/check
POST   /internal/v2/agents/{agentId}/enable
POST   /internal/v2/agents/{agentId}/disable
DELETE /internal/v2/agents/{agentId}
```

Credentials use a protected request field and are never echoed.

## 10. Common response/error

```json
{
  "error": {
    "code": "plan_version_stale",
    "safeMessage": "The plan version is no longer current.",
    "retryable": false,
    "runId": "run_1"
  }
}
```

Internal diagnostic details are logged only after redaction.

## 11. Idempotency

- Direct Run keyed by `runId`.
- Plan creation keyed by `runId + planningAttempt`.
- Plan execution keyed by `planId + version + idempotencyKey`.
- Cancel is idempotent.
- Registration requests may use request idempotency metadata.

## 12. Compatibility

Existing endpoints may be adapted internally during migration, but must preserve these semantics. Compatibility routes MUST NOT bypass confirmation, Registry authorization, or context isolation.
