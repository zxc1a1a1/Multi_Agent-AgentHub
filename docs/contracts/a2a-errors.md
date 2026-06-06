# A2A Error Contract

**Status:** Active
**Owner:** AgentHub
**Primary source:** Module Separation & Runtime Redesign


## Authoritative source order

1. `docs/superpowers/specs/2026-05-26-module-separation-and-runtime-redesign.md`
2. `docs/superpowers/specs/2026-05-27-module-separation-runtime-redesign.md`
3. PDR product goals only, not old module layout
4. Sprint/UML as supporting product/demo references only

Engineering details MUST follow the module separation redesign. Old `server/` and root `agents/` are legacy reference implementations unless a task explicitly says otherwise.


## Error classes

| Code | Meaning | Retry |
|---|---|---:|
| `agent_unavailable` | Child Agent cannot be reached | yes |
| `agent_card_invalid` | AgentCard failed validation | no |
| `task_rejected` | Child Agent rejected malformed task | no |
| `task_failed` | Handler/Runner failed | maybe |
| `stream_interrupted` | A2A stream broke mid-run | yes |

## Propagation

A2A errors must be sanitized before becoming AG-UI `RUN_ERROR`. Internal URL, stack trace, API key, and DSN leakage is forbidden.
