# AgentHub AG-UI Events Contract

## 1. 目的

本文是 AgentHub Frontend ↔ Gateway SSE 实时事件流的事实源。

任何发送给前端的实时事件都必须符合本文。

## 2. 当前版本

```text
version = v1.0-sprint
transport = SSE
```

## 3. 事件来源与映射

AG-UI Event 来源于内部 `OrchestratorStreamEvent`，由 Gateway / ProtocolConverter 负责映射。

| OrchestratorStreamEvent（内部，snake_case） | AG-UI Event（SSE，UPPER_SNAKE_CASE） |
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

规则：

- Orchestrator 只输出 `OrchestratorStreamEvent`（snake_case），不得直接输出 AG-UI Event 名称。
- Gateway / ProtocolConverter 负责将内部事件映射为 AG-UI Event，输出到 SSE。
- Frontend 只消费 AG-UI Event，不得直接接收 `OrchestratorStreamEvent` 或 Child Agent A2A event。
- Child Agent 原始 A2A event 不得直接透传给 Frontend。

## 4. 事件集

v1.0 必需事件：

```text
RUN_STARTED
RUN_FINISHED
RUN_ERROR

TEXT_MESSAGE_START
TEXT_MESSAGE_CONTENT
TEXT_MESSAGE_END

TOOL_CALL_START
TOOL_CALL_ARGS
TOOL_CALL_END

STATE_UPDATE
```

## 5. SSE 格式

每个事件使用一个 SSE block：

```text
event: message
data: {"type":"RUN_STARTED","runId":"run-001"}

```

规则：

- 一个 block 一个 JSON event。
- block 以空行结束。
- Gateway 写入后 flush。
- Frontend 按 `\n\n` 拆分 block。
- Frontend 处理粘包、拆包、半包。

## 6. 通用字段

| 字段 | 类型 | 说明 |
|---|---|---|
| `type` | string | 事件类型 |
| `runId` | string | Run ID |
| `threadId` | string | `conversationId` 的 AG-UI 协议别名 |
| `messageId` | string | 消息 ID |
| `toolCallId` | string | Tool Call ID |
| `timestamp` | string | RFC3339 时间 |
| `traceId` | string | 链路追踪 ID |
| `sender` | object | 消息发送者 |
| `delta` | string | 流式增量 |
| `content` | string | 旧字段兼容 |
| `state` | object | 状态对象 |
| `error` | object | 错误对象 |

## 7. Run 事件

### RUN_STARTED

```json
{
  "type": "RUN_STARTED",
  "runId": "run-001",
  "threadId": "conv-001"
}
```

### RUN_FINISHED

```json
{
  "type": "RUN_FINISHED",
  "runId": "run-001"
}
```

### RUN_ERROR

```json
{
  "type": "RUN_ERROR",
  "runId": "run-001",
  "error": {
    "code": "AGUI_RUN_FAILED",
    "message": "运行失败，请稍后重试",
    "retryable": true
  }
}
```

## 8. Text Message 事件

### TEXT_MESSAGE_START

```json
{
  "type": "TEXT_MESSAGE_START",
  "runId": "run-001",
  "threadId": "conv-001",
  "messageId": "msg-001",
  "role": "assistant",
  "sender": {
    "type": "agent",
    "name": "agent-name",
    "displayName": "Agent Display Name"
  }
}
```

### TEXT_MESSAGE_CONTENT

```json
{
  "type": "TEXT_MESSAGE_CONTENT",
  "runId": "run-001",
  "messageId": "msg-001",
  "delta": "文本片段"
}
```

兼容：

```json
{
  "type": "TEXT_MESSAGE_CONTENT",
  "runId": "run-001",
  "messageId": "msg-001",
  "content": "文本片段"
}
```

### TEXT_MESSAGE_END

```json
{
  "type": "TEXT_MESSAGE_END",
  "runId": "run-001",
  "messageId": "msg-001"
}
```

## 9. Tool Call 事件

### TOOL_CALL_START

```json
{
  "type": "TOOL_CALL_START",
  "runId": "run-001",
  "messageId": "msg-001",
  "toolCallId": "tc-001",
  "toolName": "web_preview"
}
```

### TOOL_CALL_ARGS

```json
{
  "type": "TOOL_CALL_ARGS",
  "runId": "run-001",
  "toolCallId": "tc-001",
  "delta": "{\"html\":\"<html>...</html>\"}"
}
```

兼容：

```json
{
  "type": "TOOL_CALL_ARGS",
  "runId": "run-001",
  "toolCallId": "tc-001",
  "content": "{\"html\":\"<html>...</html>\"}"
}
```

### TOOL_CALL_END

```json
{
  "type": "TOOL_CALL_END",
  "runId": "run-001",
  "toolCallId": "tc-001"
}
```

## 10. STATE_UPDATE

```json
{
  "type": "STATE_UPDATE",
  "runId": "run-001",
  "threadId": "conv-001",
  "state": {
    "phase": "planning",
    "message": "正在分析任务"
  }
}
```

兼容：

```json
{
  "type": "STATE_UPDATE",
  "runId": "run-001",
  "content": "{\"phase\":\"planning\",\"message\":\"正在分析任务\"}"
}
```

推荐 phase（UI 展示用，不等同于 Run.status）：

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

`state.phase` 是前端可见阶段提示，不持久化到 `runs.status`。Run.status 只使用 5 值粗粒度枚举：`accepted` / `running` / `completed` / `failed` / `cancelled`。

## 11. 多 Agent 消息

规则：

- 一个 Run 可以产生多条 assistant message。
- 每条 assistant message 必须有独立 `messageId`。
- 每条 assistant message 可以有独立 `sender`。
- `sender.name` 不限定具体 Agent 名称。
- Tool Call 必须通过 `messageId` 归属到消息。

## 12. ordered_parallel

v1.0 正式枚举值为 `ordered_parallel`。
legacy `parallel` / `ordered-parallel` 仅作为兼容输入别名，进入 Gateway 前归一化为 `ordered_parallel`。
UI 展示文案"并行"不等于协议字段名。

- SSE 事件可以按消息顺序输出。
- 不要求多个 Agent token 交错输出。

## 13. 错误脱敏

`RUN_ERROR.error` 不得包含：

- API key
- Authorization token
- 数据库连接串
- 内部堆栈
- 完整 system prompt
- provider 原始敏感错误
- 内网拓扑

## 14. 兼容策略

- 新实现优先使用 `delta`，兼容 `content`。
- 新实现优先使用 `state` object，兼容 `content` JSON 字符串。
- 单 Agent 单消息流程必须继续可用。
- `code_preview` Tool Call 必须继续可用。
