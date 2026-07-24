# LLM Provider Contract

**Status:** Active
**Version:** AgentHub 2.0
**Product source:** `docs/pdr/AgentHub-2.0-PDR.md`

## 1. Purpose

The Provider layer gives authorized backend modules a stable way to call configured language models without leaking provider-specific SDK objects or credentials into AgentHub domain contracts.

It owns:

```text
provider registry
model registry
use-case policy
provider adapter
stream normalization
structured-output normalization
usage normalization
safe error mapping
bounded retry/fallback
deterministic Mock Provider
```

It does not own:

```text
Conversation
Context selection
Plan validation
Agent Registry
A2A
Tool permission
Artifact persistence
AG-UI
public Gateway API
```

## 2. Stable use cases

```text
planner
code_agent
web_agent
synthesizer
conversation_summary
auto_title
```

A use case selects policy. It is not an Agent, Skill, Provider, or model identifier.

## 3. Request flow

```text
authorized domain caller
-> use-case policy
-> Provider Router
-> capability check
-> Provider Adapter
-> external model API
-> normalized stream/final result
```

Gateway does not call a Provider.

Built-in Agent handlers call the Provider layer through Agent Runtime or an equivalent approved backend adapter.

## 4. Provider definition

```text
ProviderDefinition
├── id
├── type
├── status
├── base_url
├── api_key_ref
├── timeout_ms
├── supports_streaming
├── supports_structured_output
├── supports_tool_use
└── metadata
```

Provider type is adapter identity, not a public API contract.

`api_key_ref` identifies an environment/secret-manager source. It never contains the secret value.

## 5. Model definition

```text
ModelDefinition
├── id
├── provider_id
├── provider_model
├── status
├── capabilities
├── max_input_tokens
├── max_output_tokens
├── cost_class
└── metadata
```

Capabilities may include:

```text
streaming
structured_output
json_schema
tool_use
vision
reasoning_output
```

`reasoning_output` means a supported safe reasoning-result field if available. AgentHub does not request, persist, or expose private chain-of-thought.

## 6. Use-case policy

```text
UseCasePolicy
├── use_case
├── model_id
├── temperature
├── max_output_tokens
├── timeout_ms
├── retry_policy
├── structured_output
├── fallback_model_ids
├── prompt_template_version
└── usage_tags
```

Rules:

- model and fallback capabilities must satisfy the use case;
- Planner requires structured output;
- structured output is validated locally;
- Summary and Auto Title receive bounded context;
- Auto Title failure does not fail the Run;
- Synthesizer does not become a Conversation Agent;
- Agent policies do not redefine AgentCard.

## 7. Local context authority

Optional provider metadata:

```text
provider_conversation_id
previous_response_id
provider_cache_key
```

This metadata is an optimization.

AgentHub's persisted Message, Summary, ArtifactRef, and ContextSnapshot remain the fact source.

Provider state loss, expiry, or provider/model switch must not make a Conversation unrecoverable.

## 8. Structured output

```text
provider response
-> extract candidate
-> parse JSON
-> validate JSON Schema/domain rules
-> accept or bounded repair/failure
```

Provider-native JSON/Schema mode improves transport reliability but does not replace local validation.

Planner output cannot create or execute a PlanVersion until Planning Validator succeeds.

## 9. Streaming

Normalized lifecycle:

```text
start
zero or more deltas
usage updates when available
one terminal end or error
```

Rules:

- stable provider request ID where available;
- ordered deltas;
- cancellation propagates;
- no emission after terminal event;
- internal provider event types do not leak into public AG-UI contracts.

## 10. Retry

Retry requires:

- classified transient error;
- idempotent/safe request boundary;
- bounded attempts and backoff;
- deadline budget;
- telemetry.

Before user-visible output, policy may retry or use a compatible fallback.

After visible output begins, the platform does not restart the whole request by default. Recovery requires provider-supported safe resume, deduplicated continuation, or a new explicit invocation.

## 11. Fallback

Fallback is allowed only when:

- enabled by use-case policy;
- candidate model is enabled;
- required capabilities match;
- privacy/data policy is compatible;
- visible output has not begun, unless safe resumable semantics exist;
- the fallback is recorded in telemetry.

Fallback cannot bypass Plan validation, Tool approval, Agent permission, or context policy.

## 12. Usage and cost

Normalize when available:

```text
provider
model
use_case
request_id
input_tokens
output_tokens
cached_input_tokens
reasoning_tokens when safely exposed as usage only
latency
status
estimated cost class/value
```

Usage metadata contains no prompt, content, secret, or private reasoning.

## 13. Mock Provider

Mock Provider:

- requires no key;
- produces deterministic buffered and streaming fixtures;
- supports structured Planner output fixtures;
- supports controlled timeout, transient, invalid-output, and cancellation cases;
- reports stable usage fixtures;
- does not claim real model reasoning or web access.

## 14. Errors

```text
provider_not_configured
provider_disabled
model_not_found
model_disabled
capability_mismatch
request_invalid
structured_output_invalid
provider_rate_limited
provider_timeout
provider_unavailable
provider_auth_failed
stream_interrupted
request_canceled
fallback_unavailable
```

Public errors are sanitized. Protected logs may retain a redacted internal cause.

## 15. Required tests

- provider/model/use-case lookup;
- disabled and capability mismatch;
- planner structured output validation;
- bounded retry;
- compatible/incompatible fallback;
- no full retry after visible output;
- cancellation and terminal ordering;
- provider state loss/switch using local context;
- usage normalization;
- secret/prompt/content redaction;
- deterministic Mock fixtures;
- Gateway cannot access Provider adapter through public path.
