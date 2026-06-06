# A2A Task Contract

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

A2A is the Orchestrator↔Child Agent protocol. The ADK adapter lives in `pkg/adk/a2a`.

## Target endpoints

```text
GET  /health
GET  /.well-known/agent.json
POST /              JSON-RPC / A2A task endpoint
```

If streaming endpoint is implemented separately, it must be explicitly documented as A2A streaming compatibility.

## Message mapping

| ADK | A2A |
|---|---|
| `Content.Role` | message role |
| `TextPart` | text part |
| `ToolCallPart` | task/tool call event |
| `ToolResultPart` | tool result message |
| `Artifact` | artifact / output part |

## Execution

Child Agent owns its Runner/session for task execution. Orchestrator only sends tasks and consumes stream/result events.

## Current gap

If current A2A implementation returns an event array after completion, mark it as compatibility. Target is streaming event delivery from Child Agent to Orchestrator.
