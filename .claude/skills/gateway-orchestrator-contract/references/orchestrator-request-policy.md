# OrchestratorRequest 规则

`OrchestratorRequest` 是 Gateway 发送给 Orchestrator 的请求体。

## 设计目标

- JSON 可序列化。
- 不依赖 Go 内部类型。
- 不依赖 Gateway HTTP framework。
- 支持 2+ Agent。
- 支持单聊和群聊。
- 支持 direct / mention / auto / manual。
- 外部请求 **不得** 传入 `fallback`。`fallback` 只能由 Orchestrator 内部生成。

## 必要字段

```text
runId
conversationId
userId
conversationType
messages
history
availableAgents
selectedAgentNames
mentions
runtimeCapabilities
planningMode
traceId
requestId
deadlineMs
metadata
```

## 禁止字段

不得包含：

- HTTP request 对象。
- Gin / Echo / net/http context。
- 数据库连接。
- LLM API key。
- 用户原始 Authorization token。
- Go channel。
- 函数指针。
- 内部 service object。

## 兼容规则

旧字段 `agentName` 如需保留，只能作为 `selectedAgentNames[0]` 的 legacy alias。

新增逻辑不得只依赖 `agentName`。
