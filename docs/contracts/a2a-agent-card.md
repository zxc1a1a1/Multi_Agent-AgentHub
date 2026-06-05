# A2A AgentCard Contract

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

Child Agents under `services/agents/*` expose AgentCard metadata through the ADK A2A adapter in `pkg/adk/a2a`.

## Required endpoint

```text
GET /.well-known/agent.json
```

## Required fields

```json
{
  "name": "code-agent",
  "description": "Code generation agent",
  "version": "0.1.0",
  "url": "http://code-agent:8081",
  "streaming": true,
  "skills": [{"id":"code_generate","name":"code_generate"}],
  "inputModes": ["text"],
  "outputModes": ["text", "code"]
}
```

## Security rules

AgentCard must never expose API keys, DSNs, internal admin/debug paths, system prompt secrets, or local filesystem paths.

## Registry

Orchestrator owns active Agent discovery. Gateway may expose a sanitized list through `/api/agents` but must not let Frontend call Child Agents directly.
