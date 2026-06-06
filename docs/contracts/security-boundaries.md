# Security Boundaries Contract

**Status:** Active
**Owner:** AgentHub
**Primary source:** Module Separation & Runtime Redesign


## Authoritative source order

1. `docs/superpowers/specs/2026-05-26-module-separation-and-runtime-redesign.md`
2. `docs/superpowers/specs/2026-05-27-module-separation-runtime-redesign.md`
3. PDR product goals only, not old module layout
4. Sprint/UML as supporting product/demo references only

Engineering details MUST follow the module separation redesign. Old `server/` and root `agents/` are legacy reference implementations unless a task explicitly says otherwise.


## Boundary rules

- Browser talks only to Gateway.
- Gateway never exposes internal Orchestrator/Agent URLs to browser.
- Orchestrator never receives browser tokens directly; Gateway authenticates public traffic.
- Child Agent AgentCards must not leak secrets.
- ADK must not contain concrete API keys or provider secrets.
- Runtime provider configs use env references.

## Secret ban

Forbidden in contracts, AgentCards, events, logs, and skills:

```text
API keys, DSNs, bearer tokens, stack traces with env, localhost admin URLs, system prompt secrets
```
