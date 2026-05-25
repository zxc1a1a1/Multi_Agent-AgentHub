# OrchestratorStreamEvent 规则

`OrchestratorStreamEvent` 是 Orchestrator 发送给 Gateway 的内部流式事件。

## 事件类型

```text
run_started
state_update
message_start
message_delta
message_end
tool_call_start
tool_call_args
tool_call_end
run_finished
run_error
```

## 通用字段

```text
type
runId
messageId
sender
delta
state
toolCall
error
phase
```

`state_update` 事件中的 `state.phase` 用于前端展示当前阶段，**不等同于持久化 Run.status**。`state.phase` 是 UI 提示（如 `planning`、`agent_streaming`），Run.status 是粗粒度生命周期状态（`accepted` / `running` / `completed` / `failed` / `cancelled`）。

## 事件映射

`OrchestratorStreamEvent` 是内部事件（snake_case），不得直接作为 AG-UI Event 输出。Gateway / ProtocolConverter 负责映射：

| OrchestratorStreamEvent | AG-UI Event (SSE) |
|---|---|
| `run_started` | `RUN_STARTED` |
| `state_update` | `STATE_UPDATE` |
| `message_start` | `TEXT_MESSAGE_START` |
| `message_delta` | `TEXT_MESSAGE_CONTENT` |
| `message_end` | `TEXT_MESSAGE_END` |
| `tool_call_start` | `TOOL_CALL_START` |
| `tool_call_args` | `TOOL_CALL_ARGS` |
| `tool_call_end` | `TOOL_CALL_END` |
| `run_finished` | `RUN_FINISHED` |
| `run_error` | `RUN_ERROR` |

## 规则

- 所有事件必须包含 `runId`。
- message 类事件必须包含 `messageId`。
- 多 Agent 输出不得复用同一个 `messageId`。
- Gateway 不得接收 Child Agent 原始流。Child Agent A2A event 必须由 Orchestrator 转为 OrchestratorStreamEvent。
- Gateway 不得改变事件业务语义。
- Gateway / ProtocolConverter 负责将 OrchestratorStreamEvent 映射为 AG-UI Event，输出到 SSE。
- OrchestratorStreamEvent 事件名不得直接作为 AG-UI Event 名称出现在 SSE 中。
- 错误事件必须使用 SafeError。

## 有序并行

ordered_parallel 场景中可以有多个 message 流。

规则：

- 每个 Agent 输出有独立 messageId。
- 事件可按 message 粒度顺序输出。
- 不要求不同 Agent token 级交错。
- 如果内部并发，Orchestrator 负责排序或隔离。
