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

## Streaming response (SSE)

The Orchestrator client (`pkg/adk/a2a.Client.SendJSONRPCStream`) consumes the
Child Agent response as a stream of events and yields each event as it arrives,
so the Orchestrator can forward partial output to the Gateway/Frontend without
waiting for the whole task to complete.

Transport negotiation is content-type based and backward compatible:

| Server response `Content-Type` | Client behavior |
|---|---|
| `text/event-stream` | Parse SSE frames (`data: {EventDTO}\n\n`), yield each `EventDTO` as it arrives. |
| `application/json` (buffered `RunResponse`) | Compatibility mode: yield each event in `RunResponse.Events` after the full body is read. |

Rules:

- Each SSE frame `data:` payload MUST be one JSON `EventDTO` (same shape as the
  array elements in buffered `RunResponse.Events`).
- A frame whose payload is `[DONE]` (or stream EOF) terminates the stream.
- The client applies the same redaction as buffered mode: `thinking` parts are
  emptied; transport/remote error messages are sanitized before they surface.
- `SendJSONRPC` (buffered) remains supported and unchanged for callers that do
  not need streaming.

## Delta semantics (Orchestrator → Gateway)

When the Orchestrator forwards streamed agent output, each text event becomes a
`message_delta` event. The Frontend MUST treat `message_delta` as **append**
(accumulate by `messageId`), never replace. `message_start` / `message_end`
bracket one logical message; multiple `message_delta` may occur between them.

## Compatibility note

A Child Agent that returns a buffered event array after completion is still
valid (compatibility mode). The streaming SSE response is the preferred target
for new agents because it removes head-of-line latency.
