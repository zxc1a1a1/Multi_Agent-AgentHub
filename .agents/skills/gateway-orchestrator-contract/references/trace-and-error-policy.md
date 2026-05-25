# Trace 与错误脱敏规则

## Trace 字段

```text
requestId
traceId
runId
conversationId
```

## 规则

- Gateway 生成或透传 traceId。
- Gateway 调 Orchestrator 时必须带 trace headers。
- Orchestrator 日志必须包含 traceId 和 runId。
- Orchestrator 调下游时应继续传递 traceId。
- 错误日志可以保留内部细节，但用户可见错误必须脱敏。

## SafeError 禁止包含

- API key。
- Authorization token。
- service token。
- 数据库连接串。
- 内部堆栈。
- 本地绝对路径。
- 内网拓扑。
- 完整 system prompt。
- 未脱敏 LLM 原始请求或响应。

## 推荐错误码

```text
ORCHESTRATOR_BAD_REQUEST
ORCHESTRATOR_UNAUTHORIZED_SERVICE
ORCHESTRATOR_TIMEOUT
ORCHESTRATOR_CANCELLED
ORCHESTRATOR_PLANNING_FAILED
ORCHESTRATOR_AGENT_UNAVAILABLE
ORCHESTRATOR_AGENT_FAILED
ORCHESTRATOR_STREAM_INTERRUPTED
ORCHESTRATOR_INTERNAL
```
