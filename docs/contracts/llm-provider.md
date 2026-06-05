# LLM Provider Contract

**Status:** Active
**Owner:** AgentHub
**Primary source:** Module Separation & Runtime Redesign


## Authoritative source order

1. `docs/superpowers/specs/2026-05-26-module-separation-and-runtime-redesign.md`
2. `docs/superpowers/specs/2026-05-27-module-separation-runtime-redesign.md`
3. PDR product goals only, not old module layout
4. Sprint/UML as supporting product/demo references only

Engineering details MUST follow the module separation redesign. Old `server/` and root `agents/` are legacy reference implementations unless a task explicitly says otherwise.


## Scope

Concrete providers live in `pkg/runtime/model`. ADK only defines `adk.Model`.

## Providers

| Provider | Package | API style |
|---|---|---|
| Anthropic | `pkg/runtime/model/anthropic.go` | Messages API |
| OpenAI | `pkg/runtime/model/openai.go` | ChatCompletions |
| Proxy | `pkg/runtime/model/proxy.go` | OpenAI-compatible |

## Rules

- Providers are registered through Runtime model registry.
- API keys come from env vars only.
- Provider errors must be sanitized before reaching browser.
- Planner may use an LLM provider, but RulePlanner remains fallback only.
