---
name: gateway-orchestrator-contract
description: "AgentHub 2.0 internal Gateway-Orchestrator contract for Direct Run, Plan creation, confirmed execution, Replan, cancellation, Registry operations, internal events, and transport independence."
---

# gateway-orchestrator-contract

## Purpose

Use this Skill when changing the private Gateway–Orchestrator boundary.

## Read first

```text
/project-architecture
/conversation-contract
/planning-approval-contract
/context-management-contract
/agent-registry-contract
/data-persistence-contract
/agui-event-contract
/security-boundary-contract
/testing-review-contract
```

Primary sources:

```text
docs/contracts/gateway-orchestrator.md
docs/contracts/gateway-orchestrator-api.md
docs/contracts/gateway-orchestrator-events.md
```

## Core operations

```text
ExecuteDirectRun
CreatePlan
ExecuteConfirmedPlan
CreateReplan
CancelRun
GetRun
ListEligibleAgents
Register/Refresh/Check/Enable/Disable/Remove Agent
```

## Non-negotiable rules

- Gateway and Orchestrator remain separate service boundaries.
- Frontend MUST NOT call Orchestrator.
- Gateway MUST NOT call a registered Agent or LLM directly.
- Orchestrator MUST NOT execute an unconfirmed multi-Agent PlanVersion.
- Gateway confirmation MUST include exact `planId + version`.
- Orchestrator MUST use Registry eligibility and Agent snapshots.
- Context input is bounded; full history MUST NOT be blindly forwarded.
- Internal events are not AG-UI events; Gateway performs mapping.
- Current internal HTTP/streaming may remain the 2.0 profile.
- gRPC is an optional future transport, not an implicit requirement.
- User bearer tokens and Agent credentials MUST NOT be forwarded as generic metadata.
- Cancellation and request/trace IDs MUST propagate.
- A material Replan returns a new PlanVersion and waits for confirmation.

## Allowed targets

```text
services/gateway/**
services/orchestrator/**
pkg/contracts/**
docs/contracts/gateway-orchestrator*.md
```

Protocol migration or persistence implementation changes require explicit additional scope.

## Completion checklist

- [ ] Direct and planned Run paths are separate.
- [ ] Confirmation and stale-version behavior are explicit.
- [ ] Internal/public event mapping is explicit.
- [ ] Registry management operations are authenticated and sanitized.
- [ ] Context, authorization, cancellation, and trace propagation are defined.
- [ ] No direct Gateway→Agent or Frontend→Orchestrator path exists.
- [ ] Tests and compatibility gaps are reported.
