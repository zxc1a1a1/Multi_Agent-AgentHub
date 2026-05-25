# Error Events

## RUN_ERROR

推荐结构：

```json
{
  "type": "RUN_ERROR",
  "runId": "run-001",
  "error": {
    "code": "AGUI_AGENT_FAILED",
    "message": "Agent 执行失败，请稍后重试",
    "retryable": true
  }
}
```

## 推荐错误码

```text
AGUI_BAD_REQUEST
AGUI_UNAUTHORIZED
AGUI_STREAM_INTERRUPTED
AGUI_PLANNING_FAILED
AGUI_AGENT_UNAVAILABLE
AGUI_AGENT_FAILED
AGUI_TOOL_ARGS_INVALID
AGUI_TOOL_UNSUPPORTED
AGUI_INTERNAL
```

## 脱敏规则

错误不得包含：

- API key
- Authorization token
- 数据库连接串
- 内部堆栈
- 完整 system prompt
- provider 原始敏感错误
- 内网拓扑

## 事件来源

`RUN_ERROR` 由 `OrchestratorStreamEvent.run_error` 经 Gateway / ProtocolConverter 映射而来。Orchestrator 内部的 `SafeError` 经脱敏后映射为 AG-UI Event 的 `error` 字段。

## fallback

如果可以恢复，应先发送：

```text
STATE_UPDATE(phase=retrying)
```

只有不可恢复时才发送 `RUN_ERROR`。
