# Platform API Contract

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

Public API is Gateway-only. It is the browser-facing surface and must not expose Orchestrator or Child Agent endpoints.

## Formal endpoints

| Method | Path | Purpose | Status |
|---|---|---|---|
| POST | `/agui/runs` | Start a run and receive SSE AG-UI stream | Target |
| POST | `/api/chat` | Compatibility wrapper over `/agui/runs` | Temporary |
| GET | `/api/conversations` | List conversations | Active |
| POST | `/api/conversations` | Create conversation | Active |
| GET | `/api/conversations/{id}/messages` | Fetch message history | Active |
| GET | `/api/agents` | List agents proxied from Orchestrator | Active |
| GET | `/health` | Gateway health | Active |

## Run request

```json
{
  "threadId": "conversation-id",
  "runId": "optional-run-id",
  "messages": [{"role":"user","content":"..."}],
  "tools": [{"name":"code_preview","description":"...","schema_json":"{}"}],
  "metadata": {"conversationType":"single|group"}
}
```

## Streaming response

`/agui/runs` returns `text/event-stream` using AG-UI events defined by `agui-events.md`.

## Rules

- Gateway owns auth, CORS, request IDs, and public error shape.
- Public API must not leak internal Orchestrator URLs or Child Agent URLs.
- `/api/chat` must not become the primary contract endpoint unless contracts are intentionally revised.
