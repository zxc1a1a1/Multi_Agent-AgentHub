---
name: observability-debugging-contract
description: "AgentHub 2.0 observability skill for OpenTelemetry traces, metrics, structured logs, event replay diagnostics, Plan/Context/Agent/Artifact correlation, redaction, and debug bundles."
---

# observability-debugging-contract

## Purpose

Use this Skill for traces, metrics, logs, debugging, correlation identifiers, event replay diagnostics, cost/latency measurements, health telemetry, or support bundles.

## Read first

```text
/project-architecture
/conversation-contract
/planning-approval-contract
/context-management-contract
/agent-registry-contract
/gateway-orchestrator-contract
/a2a-agent-contract
/artifact-contract
/security-boundary-contract
/testing-review-contract
```

Primary sources:

```text
docs/contracts/observability-debugging.md
docs/contracts/observability-debugging.schema.json
docs/contracts/metrics-policy.md
docs/contracts/observability-review-checklist.md
```

## Required correlation

```text
requestId
traceId
user/tenant reference when permitted
conversationId
messageId
runId
planId
planVersion
stepId
invocationId
agentId
agentVersion
skillId
contextSnapshotId
artifactId
artifactVersion
toolCallId
errorCode
```

## Non-negotiable rules

- Use OpenTelemetry conventions where applicable.
- Logs, traces, metrics, and events use stable IDs.
- Do not record private model reasoning.
- Do not record credentials, Authorization headers, full private prompts, or unredacted attachment content.
- High-cardinality IDs belong in traces/logs, not unrestricted metric labels.
- Event replay and live-stream diagnostics preserve Run sequence.
- Planner, Context, Registry, A2A, Tool, Artifact, Preview, and persistence failures are distinguishable.
- Cost/token fields identify use case and model without exposing secrets.
- Debug bundles are explicit, bounded, redacted, and user-authorized.
- Health checks and product telemetry do not replace one another.

## Completion checklist

- [ ] trace boundaries match service/Agent calls.
- [ ] required correlation fields are propagated.
- [ ] metrics avoid uncontrolled cardinality.
- [ ] secrets/content redaction is tested.
- [ ] Run replay and partial failure can be diagnosed.
- [ ] PlanVersion, ContextSnapshot, AgentVersion, and ArtifactVersion are visible.
- [ ] latency, token, retry, failure, and preview metrics are covered.
- [ ] debug bundle policy is explicit.
