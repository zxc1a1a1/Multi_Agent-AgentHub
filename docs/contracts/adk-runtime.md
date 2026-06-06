# ADK Runtime Contract

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

`pkg/adk` is a pure engine package. `pkg/runtime` is the engineering framework over ADK.

## ADK owns

- `Agent` interface
- `Model` interface
- `Tool` interface
- `Plugin` interface
- `Runner`
- `Session` / `SessionService` interface
- `Event`, `Content`, `Part` types
- `pkg/adk/a2a` adapter

## ADK must not own

- concrete LLM providers
- YAML config loading
- MySQL implementation details beyond interfaces
- Skill repository/manager
- AG-UI browser translation
- business-specific Agent handlers

## Runtime owns

- YAML configuration
- model/tool/agent registries
- Anthropic/OpenAI/Proxy providers
- session implementations including SQL and memory wrapper
- context pruning
- skill system
- AG-UI translator and filters
- launcher

## Interface requirement

Every runtime session implementation must satisfy `adk.SessionService`, including `GetOrCreate` if the ADK interface requires it.
