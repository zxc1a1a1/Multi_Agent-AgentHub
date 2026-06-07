# Snapshot Event Contract

**Status:** Active
**Owner:** AgentHub
**Scope:** `frontend/src/agui`, `services/orchestrator`, `services/gateway`

## Overview

Snapshot events carry structured, renderable state from the Orchestrator to the
Frontend. Each snapshot has a `snapshotType` that determines which Frontend
component renders it.

## Event types

### `state_snapshot` — full state snapshot

Carries a complete structured payload to replace or initialize a rendered view.

```json
{
  "type": "state_snapshot",
  "runId": "run_xxx",
  "taskId": "task_yyy",
  "snapshotType": "code|webpage|activity|form|progress",
  "payload": { }
}
```

### `state_delta` — incremental state update

Carries a partial update to be merged into the current snapshot state.

```json
{
  "type": "state_delta",
  "runId": "run_xxx",
  "taskId": "task_yyy",
  "snapshotType": "code|webpage|activity|form|progress",
  "delta": { }
}
```

## Snapshot types

| snapshotType | Description | Frontend renderer |
|---|---|---|
| `code` | Source code artifact | `CodePreview` |
| `webpage` | HTML/iframe preview | `WebPreview` |
| `activity` | Orchestration plan progress | `OrchestrationCard` |
| `form` | Structured form input (HITL) | `HITLConfirm` (or form renderer) |
| `progress` | Task progress indicator | Progress bar component |

## Snapshot registry (Frontend)

Frontend maintains a global `snapshotRegistry`:

```ts
type SnapshotRenderer = (data: SnapshotData, ctx: RenderContext) => ReactNode;
registerSnapshotRenderer(type: string, renderer: SnapshotRenderer): void;
getSnapshotRenderer(type: string): SnapshotRenderer | undefined;
```

Renderers are registered at app startup. Unknown types are silently ignored.

## Rendering flow

1. Frontend receives an AG-UI event with `state_snapshot` or `state_delta`.
2. Extract `snapshotType` from the event.
3. Look up the renderer from `snapshotRegistry`.
4. Render the component in the chat panel or right-side detail panel.
5. For `state_delta`, merge the delta into the existing snapshot state.
