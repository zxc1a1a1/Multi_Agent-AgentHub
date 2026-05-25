# OrchestratorResult 规则

`OrchestratorResult` 是一次 run 的最终摘要。

## 字段

```text
runId
conversationId
status
strategy
intentSummary
messages
tasks
artifacts
toolCalls
error
startedAt
finishedAt
```

## 规则

- 一个 result 可以包含多条 assistant message。
- 一个 result 可以包含多个 task。
- 一个 result 可以包含多个 artifact 引用。
- 一个 result 可以包含多个 tool call 引用。
- result 不保存大型二进制内容。
- result 不保存完整敏感 prompt。
- result 不返回内部堆栈。
- Gateway 负责基于 result 进行持久化。

## Status

`status` 是粗粒度生命周期状态，对外使用 5 值枚举：

```text
accepted
running
completed
failed
cancelled
```

`phase` 是可选细粒度当前阶段（内部字段），用于追踪内部进度。内部阶段（如 `planning`、`dispatching`、`retrying`）不得写入 `status`。

内部阶段 → status 映射：

| 内部阶段 | status |
|---|---|
| `accepted` | `accepted` |
| `context_loaded` / `planning` / `plan_ready` / `dispatching` | `running` |
| `agent_task_running` / `agent_task_completed` | `running` |
| `agent_task_failed`（可恢复） / `retrying` / `fallback` | `running` |
| `aggregating` | `running` |
| `completed` | `completed` |
| `failed` / `agent_task_failed`（不可恢复） | `failed` |
| `cancelled` | `cancelled` |

## Error

失败时必须使用 SafeError。
