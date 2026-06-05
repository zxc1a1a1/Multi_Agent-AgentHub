# Planner Input Contract

**Status:** Active
**Owner:** AgentHub
**Primary source:** Module Separation & Runtime Redesign


## Authoritative source order

1. `docs/superpowers/specs/2026-05-26-module-separation-and-runtime-redesign.md`
2. `docs/superpowers/specs/2026-05-27-module-separation-runtime-redesign.md`
3. PDR product goals only, not old module layout
4. Sprint/UML as supporting product/demo references only

Engineering details MUST follow the module separation redesign. Old `server/` and root `agents/` are legacy reference implementations unless a task explicitly says otherwise.


## PlannerInput

```json
{
  "messages": [{"role":"user","content":"..."}],
  "history": [{"role":"assistant","content":"...","senderName":"code-agent"}],
  "agents": [{"name":"code-agent","skills":["code_generate"],"outputModes":["text","code"]}],
  "tools": [{"name":"code_preview"}],
  "metadata": {"conversationType":"single|group"}
}
```

## Rules

- Planner must never inspect Gateway DB directly.
- Planner must not call Child Agents.
- Planner output must be validated before execution.
