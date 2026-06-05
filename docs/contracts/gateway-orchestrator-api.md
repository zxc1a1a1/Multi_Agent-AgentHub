# Gateway ↔ Orchestrator API Detail

**Status:** Active
**Owner:** AgentHub
**Primary source:** Module Separation & Runtime Redesign


## Authoritative source order

1. `docs/superpowers/specs/2026-05-26-module-separation-and-runtime-redesign.md`
2. `docs/superpowers/specs/2026-05-27-module-separation-runtime-redesign.md`
3. PDR product goals only, not old module layout
4. Sprint/UML as supporting product/demo references only

Engineering details MUST follow the module separation redesign. Old `server/` and root `agents/` are legacy reference implementations unless a task explicitly says otherwise.


## ExecuteRequest

Fields:

| Field | Required | Description |
|---|---:|---|
| `thread_id` | yes | Conversation/session id from Gateway. |
| `run_id` | yes | Run id for correlation. |
| `messages[]` | yes | Current user-turn messages. |
| `history[]` | no | Bounded message history from Gateway store. |
| `tools[]` | no | Frontend declared runtime skills/capabilities. |
| `metadata` | no | `conversationType`, `requestId`, `traceId`, user metadata. |

## AGUIEvent

The Orchestrator stream uses AG-UI-compatible events, not database models.

Required cross-service fields:

```text
type, run_id, message_id, content, tool_call_id, tool_name, role, error, agent_name, metadata
```

## ListAgents

Gateway must proxy agent list from Orchestrator to avoid direct Frontend access to Child Agents.
