# OrchestratorStreamEvent 契约

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

## 通用结构

```json
{
  "type": "message_delta",
  "runId": "run_001",
  "messageId": "msg_001",
  "sender": {
    "type": "agent",
    "name": "some-agent"
  },
  "delta": "文本片段",
  "state": null,
  "toolCall": null,
  "error": null
}
```

## 规则

- 所有事件必须包含 `runId`。
- message 类事件必须包含 `messageId`。
- 多 Agent 输出不得复用 messageId。
- Gateway 不接收 Child Agent 原始流。
- Gateway 不改变事件业务语义。
- 错误必须使用 SafeError。
