# Intent Orchestration Contract

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

Planning and execution live only in `services/orchestrator`.

## Components

```text
internal/planner      intent → OrchestrationPlan
internal/router       agent registry and capability match
internal/executor     single/sequential/parallel execution
internal/dispatcher   A2A calls to Child Agents
internal/converter    A2A/ADK events → AG-UI events
```

## Planning mode

| Mode | Meaning |
|---|---|
| `single` | One Child Agent handles the task. |
| `sequential` | Ordered dependent steps. |
| `parallel` | Independent steps may run concurrently. |
| `ordered_parallel` | Temporary demo-safe mode: independent intent, deterministic ordered output. |

## Planner sources

- Target: LLM planner using AgentCards and history.
- Fallback: RulePlanner based on capabilities/keywords.

RulePlanner may exist but must not be documented as the final intelligent planning implementation.
