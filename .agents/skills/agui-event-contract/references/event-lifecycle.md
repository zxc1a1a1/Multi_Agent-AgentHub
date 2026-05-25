# Event Lifecycle

## 正常流程

```text
RUN_STARTED
STATE_UPDATE*
TEXT_MESSAGE_START
TEXT_MESSAGE_CONTENT*
TEXT_MESSAGE_END
TOOL_CALL_START?
TOOL_CALL_ARGS*
TOOL_CALL_END?
RUN_FINISHED
```

## 失败流程

```text
RUN_STARTED
STATE_UPDATE*
TEXT_MESSAGE_START?
TEXT_MESSAGE_CONTENT*
STATE_UPDATE(phase=retrying)?
RUN_ERROR
```

## 规则

- `RUN_STARTED` 最多一次。
- `RUN_FINISHED` 和 `RUN_ERROR` 二选一结束 Run。
- `RUN_FINISHED` 后不得继续发送业务事件。
- `RUN_ERROR` 后不得继续发送业务事件。
- `STATE_UPDATE` 可以穿插，但不得替代消息或工具调用事件。
- 所有事件携带 `threadId`，其值为 `conversationId` 的 AG-UI 协议别名，不得视为独立会话 ID。
- AG-UI Event 来源于 `OrchestratorStreamEvent`，由 Gateway / ProtocolConverter 映射。内部事件名（snake_case，如 `message_delta`）不得直接出现在 SSE 中。
- 生命周期中的 `phase` 标注（如 `planning`、`retrying`）用于 UI 展示当前阶段，不是 Run.status。Run.status 只有 5 个值：`accepted` / `running` / `completed` / `failed` / `cancelled`。
