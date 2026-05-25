# STATE_UPDATE Events

## 目的

`STATE_UPDATE` 用于展示运行过程状态，例如编排、分派、当前 Agent、重试、fallback。

## 推荐结构

```json
{
  "type": "STATE_UPDATE",
  "runId": "run-001",
  "state": {
    "phase": "planning",
    "message": "正在分析任务"
  }
}
```

## phase 与 Run.status 的区别

`state.phase` 是前端可见的阶段提示，用于 UI 展示，**不等同于持久化 Run.status**。

- `Run.status`：粗粒度生命周期状态（`accepted` / `running` / `completed` / `failed` / `cancelled`），持久化到 `runs.status`。
- `state.phase`：前端可见 UI 提示，不持久化。需要展示时使用 `STATE_UPDATE`，不应扩展 Run.status 枚举。
- `Run.phase`：可选内部细粒度阶段（如 `context_loaded`、`aggregating`），位于 Run 对象中，不在 status 字段。
- `run_steps.step_type`：持久化详细步骤类型（如 `planning` / `dispatch` / `agent_call` / `retry` / `aggregate`），用于审计。

## phase

推荐枚举：

```text
accepted
planning
dispatching
agent_streaming
tool_calling
retrying
finished
failed
```

## 兼容

旧实现可以把 JSON 字符串放在 `content` 中。

前端兼容：

```ts
const state = event.state ?? safeParseJSON(event.content)
```

## 规则

- `STATE_UPDATE` 携带 `threadId`，其值为 `conversationId` 的 AG-UI 协议别名。
- `STATE_UPDATE` 由 `OrchestratorStreamEvent.state_update` 经 Gateway / ProtocolConverter 映射而来。
- `STATE_UPDATE` 不得替代文本消息。
- `STATE_UPDATE` 不得替代 Tool Call。
- `message` 必须可展示。
- 不得泄漏内部错误、密钥或堆栈。
