# Orchestration Plan Contract

**Status:** Active
**Owner:** AgentHub
**Primary source:** Module Separation & Runtime Redesign


## Authoritative source order

1. `docs/superpowers/specs/2026-05-26-module-separation-and-runtime-redesign.md`
2. `docs/superpowers/specs/2026-05-27-module-separation-runtime-redesign.md`
3. PDR product goals only, not old module layout
4. Sprint/UML as supporting product/demo references only

Engineering details MUST follow the module separation redesign. Old `server/` and root `agents/` are legacy reference implementations unless a task explicitly says otherwise.


## Shape

```json
{
  "mode": "single|sequential|ordered_parallel",
  "intent": "brief user intent",
  "steps": [
    {
      "id": "step-1",
      "agentName": "code-agent",
      "input": "task for this agent",
      "dependsOn": [],
      "expectedOutputs": ["text", "code"]
    }
  ]
}
```

## Strategies

| Strategy | Description |
|---|---|
| `single` | Exactly 1 task, dispatched directly. |
| `ordered_parallel` | 2+ tasks with no mutual dependencies, executed in priority order (serial). |
| `sequential` | 1+ tasks, connected via `dependsOn` edges. Executor topologically sorts into waves; same-wave tasks run concurrently; later waves wait for upstream results. |

## Sequential (DAG) semantics

- Tasks declare upstream dependencies via `dependsOn` (array of `taskId`).
- The dependency graph MUST be acyclic (validated by Kahn's algorithm).
- Upstream task output is injected into downstream `taskContent` via `{{deps.{taskId}.output}}` placeholders.
- If an upstream task fails, dependent downstream tasks are skipped with `ORCHESTRATOR_SKIPPED_DEP`.
- Independent branches continue execution regardless of failures in other branches.
- Task limit: default 5, configurable via `ORCHESTRATOR_MAX_TASKS` env.

## Validation

- Every `agentName` must exist in Orchestrator registry and be healthy unless fallback explicitly applies.
- `dependsOn` ids must reference other taskIds in the plan (no dangling refs, no self-loops).
- `ordered_parallel` tasks must not have `dependsOn`.
- The `dependsOn` graph must contain no cycles.
- Frontend skills must support expected artifact outputs or Orchestrator must degrade to text.
