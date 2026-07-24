---
name: agent-runtime-contract
description: "AgentHub 2.0 contract skill for Agent handler lifecycle, A2A/MCP/LLM adapters, cancellation, streaming, Tool policy, structured output, Artifact emission, deterministic mocks, and telemetry."
---

# agent-runtime-contract

## Purpose

Use this Skill for built-in or managed Agent runtime lifecycle, request handling, model adapter use, MCP/local Tool adapters, cancellation, streaming, structured output, Artifact emission, deterministic Mock mode, or Agent telemetry.

This Skill replaces `/adk-runtime-contract`.

## Read first

```text
/project-architecture
/agent-registry-contract
/a2a-agent-contract
/context-management-contract
/llm-provider-contract
/artifact-contract
/security-boundary-contract
/observability-debugging-contract
/testing-review-contract
```

Primary sources:

```text
docs/contracts/agent-runtime.md
docs/contracts/agent-runtime-review-checklist.md
```

## Runtime boundary

```text
A2A server adapter
-> Agent handler
-> model and/or Tool adapters
-> structured Message/Artifact result
-> A2A stream
```

## Non-negotiable rules

- Runtime does not own Conversation, Planner, Registry, AG-UI, or public API.
- Official A2A/MCP SDKs define protocol behavior.
- Runtime MUST NOT define a second incompatible AgentCard or MCP wire model.
- Context is supplied by Orchestrator according to Context Contract.
- cancellation and deadlines propagate to model and Tool calls.
- Tool access is allowlisted by Agent/Skill policy.
- Tool output and model output are untrusted and bounded.
- Artifact output follows Artifact Contract.
- Mock mode is deterministic and requires no production credential.
- secrets are injected through configuration/secret storage and never emitted.
- streaming has stable Message/Artifact lifecycle and safe terminal errors.
- built-in and managed Agents expose health and AgentCard through a common adapter.
- runtime packages remain domain-neutral; business-specific prompts/handlers live with the Agent service.

## Completion checklist

- [ ] lifecycle and ownership boundaries are respected.
- [ ] official adapter types are used.
- [ ] cancellation, timeout, and streaming tests exist.
- [ ] Tool permissions and output limits are enforced.
- [ ] Mock mode is deterministic.
- [ ] Artifact output and error mapping are contract-compliant.
- [ ] secrets and provider payloads are redacted.
- [ ] Agent telemetry includes stable invocation identifiers.
