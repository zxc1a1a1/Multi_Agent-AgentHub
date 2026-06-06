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
  "mode": "single|sequential|parallel|ordered_parallel",
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

## Validation

- Every `agentName` must exist in Orchestrator registry and be healthy unless fallback explicitly applies.
- `dependsOn` ids must reference earlier steps for `sequential`.
- `parallel` steps must not depend on each other.
- Frontend skills must support expected artifact outputs or Orchestrator must degrade to text.
