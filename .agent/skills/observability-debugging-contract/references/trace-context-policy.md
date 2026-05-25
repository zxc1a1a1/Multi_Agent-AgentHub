# Trace Context Policy

## 目的

定义 AgentHub 跨服务追踪上下文的传播规则。

## 标准传播字段

跨 HTTP 服务边界优先使用：

- `traceparent`
- `tracestate`

## AgentHub 业务关联字段

- `requestId`
- `runId`
- `conversationId`
- `messageId`
- `planId`
- `stepId`
- `agentTaskId`
- `artifactId`
- `toolCallId`
- `llmRequestId`

## 规则

- Gateway 接收外部请求时，如果没有 trace context，必须创建新的 trace。
- Gateway 调 Orchestrator 必须传播 trace context。
- Orchestrator 调 Agent、LLM Provider、存储或 Registry 时必须继续传播或记录 trace context。
- `traceId` 不是凭证，不得用于鉴权。
- `tracestate` 中不得包含 secret、token、完整 prompt 或用户隐私。
- 日志至少包含 `traceId` 和 `runId`。
