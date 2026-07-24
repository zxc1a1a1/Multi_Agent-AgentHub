---
name: llm-provider-contract
description: "AgentHub 2.0 model-provider contract skill for use-case routing, provider/model registries, structured output, streaming, retries, fallback, usage, redaction, and deterministic Mock mode."
---

# llm-provider-contract

## Purpose

Use this Skill for model provider adapters, model registry, use-case policy, structured output, provider streaming, retry/fallback, token usage, provider conversation state, Mock Provider, or model credential changes.

## Read first

```text
/project-architecture
/context-management-contract
/planning-approval-contract
/agent-runtime-contract
/security-boundary-contract
/observability-debugging-contract
/testing-review-contract
```

Primary sources:

```text
docs/contracts/llm-provider.md
docs/contracts/llm-provider.schema.json
docs/contracts/llm-provider-security.md
docs/contracts/llm-provider-review-checklist.md
```

References:

```text
references/use-case-policy.md
references/provider-adapter-policy.md
references/retry-streaming-policy.md
references/secret-redaction-policy.md
references/review-checklist.md
```

## Stable use cases

```text
planner
code_agent
web_agent
synthesizer
conversation_summary
auto_title
```

A use case is platform policy, not an Agent identity or Provider endpoint.

## Boundary

```text
domain module
-> use-case policy
-> Provider Router
-> Provider Adapter
-> external model API
```

Gateway does not call a model provider.

Planner, built-in Agent handlers, Synthesizer, Summary, and Auto Title may call the provider layer through explicit use-case policy.

## Non-negotiable rules

- Provider Conversation or response ID is an optimization, not the AgentHub Conversation fact source.
- Local ContextSnapshot remains sufficient to reconstruct a request after provider switch or state loss.
- Planner structured output is parsed and validated locally before it can create a PlanVersion.
- Provider/model capabilities are declared and checked before routing or fallback.
- Retry is bounded and classified.
- Full request retry after user-visible streaming begins is forbidden unless the provider supports safe resumable semantics.
- Fallback before visible output requires compatible capability and use-case policy.
- API keys come from secret references, never ordinary persisted fields.
- Credentials, Authorization headers, full private prompts, private model reasoning, and unredacted content do not enter logs/events.
- The platform requests final answers and structured outputs, not private chain-of-thought.
- Mock Provider is deterministic and requires no production credential.
- Provider-specific SDK objects do not leak into domain, API, Event, Artifact, or AgentCard contracts.

## Completion checklist

- [ ] Provider/model/use-case policy is explicit.
- [ ] capability and structured-output validation are local.
- [ ] timeout/retry/fallback and visible-stream boundary are tested.
- [ ] provider state loss/switch is recoverable from local context.
- [ ] token/usage metadata is normalized.
- [ ] secrets and content redaction are tested.
- [ ] deterministic Mock mode exists.
- [ ] Gateway and public browser boundary remain provider-free.
