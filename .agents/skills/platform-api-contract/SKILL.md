---
name: platform-api-contract
description: "AgentHub 2.0 Gateway public API contract for conversations, messages, runs, plans, registered Agents, attachments, artifacts, search, idempotency, and event replay."
---

# platform-api-contract

## Purpose

Use this Skill for browser-facing Gateway API changes.

Frontend communicates with Gateway only. Public API resources are AgentHub product resources, not Orchestrator or A2A wire objects.

## Read first

```text
/project-architecture
/conversation-contract
/planning-approval-contract
/agent-registry-contract
/data-persistence-contract
/agui-event-contract
/security-boundary-contract
/testing-review-contract
```

Primary sources:

```text
docs/contracts/platform-api.md
docs/contracts/openapi.yaml
```

## Public resource groups

```text
conversations
messages
runs
plans
agents
agent-registrations
attachments
artifacts
search
events
```

## Non-negotiable rules

- Frontend MUST call Gateway only.
- `/internal/**`, `/a2a/**`, and remote Agent endpoints MUST NOT be public Platform API.
- Public IDs use `conversationId`, `messageId`, `runId`, `planId`, `agentId`, `artifactId`.
- AG-UI `threadId` is a protocol alias for `conversationId`, not a second business identifier.
- Message creation MUST support `clientMessageId` idempotency.
- All list endpoints MUST be paginated.
- Plan confirmation MUST bind the exact `planId + version`.
- Manual Multi-Agent requests MUST preserve the user-selected Agent set.
- Agent list and Planner eligibility MUST come from the same Registry authorization policy.
- Remote Agent registration and future AgentHub-managed Agent creation MUST use different API resources.
- Agent credentials are write-only, encrypted, and never returned.
- API errors MUST be sanitized and stable.
- Compatibility endpoints such as `/api/chat` or `/agui/runs` MUST be marked deprecated and MUST NOT define new 2.0 behavior.
- OpenAPI is the machine-readable public API source.

## Allowed targets

```text
services/gateway/**
frontend/src/**
docs/contracts/platform-api.md
docs/contracts/openapi.yaml
```

Changes to Orchestrator, Registry implementation, A2A, storage Schema, or event converter require their own Skills.

## Completion checklist

- [ ] OpenAPI and prose agree.
- [ ] Public/internal boundaries are preserved.
- [ ] Object-level authorization is defined.
- [ ] Pagination and idempotency are defined.
- [ ] Plan version confirmation is enforced.
- [ ] Agent registration is distinct from managed-Agent creation.
- [ ] Secrets and internal URLs are not exposed.
- [ ] Negative and compatibility tests are reported.
