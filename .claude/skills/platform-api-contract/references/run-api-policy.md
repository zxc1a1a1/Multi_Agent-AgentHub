# Run API 策略

推荐路径：

- `POST /api/runs`
- `GET /api/runs/{runId}`
- `POST /api/runs/{runId}/cancel`

兼容路径：

- `POST /api/agui/run`
- `POST /api/agui/run/{runId}/cancel`

新增 API 优先使用 `/api/runs`。Run 创建不展开 Orchestrator 内部计划。取消 Run 不应返回 500。

## Run.status 枚举

粗粒度生命周期状态，只使用 5 值：

```text
accepted
running
completed
failed
cancelled
```

## Run.phase（可选）

细粒度当前阶段，用于展示运行细节。可选值包括：

```text
accepted
context_loaded
planning
plan_ready
dispatching
agent_task_running
agent_task_completed
agent_task_failed
retrying
fallback
aggregating
completed
failed
cancelled
```

内部阶段到 status 的映射：

| phase | status |
|---|---|
| `accepted` | `accepted` |
| `context_loaded` ~ `aggregating` | `running` |
| `completed` | `completed` |
| `failed` | `failed` |
| `cancelled` | `cancelled` |

## STATE_UPDATE.state.phase

前端 SSE 事件中的阶段提示，用于 UI 展示，不持久化。不等同于 Run.status。

## run_steps.step_type

持久化详细步骤类型（如 `planning` / `dispatch` / `agent_call` / `retry` / `aggregate`），用于审计和排障。

## 禁止

- 将内部细粒度阶段（如 `planning`、`dispatching`）写入 Run.status。
- 通过扩展 Run.status 枚举来表达运行细节。
- 把 STATE_UPDATE.state.phase 当做持久化 Run.status 使用。
