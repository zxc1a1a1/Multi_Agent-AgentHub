# Observability Debugging Contract

版本：v0.1-p1  
适用项目：AgentHub - 多 Agent 协作平台  
适用阶段：MVP 轻量埋点 + P1 正式开发演进  
事实源文件：

```text
<repo-root>/docs/contracts/observability-debugging.md
<repo-root>/docs/contracts/observability-debugging.schema.json
```

## 1. 目的

本文定义 AgentHub 的项目级可观测性与排障契约。

核心链路：

```text
React
→ Gateway
→ Orchestrator
→ A2A Client
→ Child Agent / ADK Runtime
→ LLM Provider
→ Artifact
→ AG-UI SSE
→ Frontend Runtime Skill
```

## 2. 官方标准优先

涉及分布式追踪、日志关联、指标和上下文传播时，以官方标准为优先参考。

```text
OpenTelemetry / W3C Trace Context 官方规范
> AgentHub 可观测性 contract
> 各服务自行发明的日志字段
```

AgentHub 可以定义项目内 ID、事件名、错误码和排障手册，但不得和官方 trace context 传播规则冲突。

## 3. Contract first 规则

任何新增或修改可观测性字段、错误码、日志格式、metrics、span、debug dump 前，必须先更新：

```text
<repo-root>/docs/contracts/observability-debugging.md
<repo-root>/docs/contracts/observability-debugging.schema.json
```

未更新 contract 的实现变更不得接受。

## 4. 阶段演进规则

### MVP 阶段

MVP 阶段不要求完整 OpenTelemetry Collector、Jaeger、Prometheus 或 Grafana。

MVP 阶段必须至少支持：

```text
requestId
runId
conversationId
messageId
agentName
a2aTaskId
toolCallId
errorCode
structured JSON logs
safe error message
redaction
```

### P1 / 正式开发阶段

逐步启用：

- OpenTelemetry traces。
- OpenTelemetry metrics。
- OpenTelemetry logs correlation。
- W3C `traceparent` / `tracestate` propagation。
- Span naming conventions。
- HTTP semantic attributes。
- A2A semantic attributes。
- LLM semantic attributes。
- AG-UI SSE metrics。
- dashboard rules。
- debug dump。
- incident checklist。

## 5. Trace Context

长期关键关联字段：

```text
traceId
requestId
conversationId
messageId
runId
executionPlanId
stepId
a2aTaskId
agentName
agentSkill
artifactId
toolCallId
llmRequestId
```

正式开发阶段优先支持：

```text
traceparent
tracestate
```

## 6. Structured Logging

日志必须是结构化 JSON。

最小字段：

```text
timestamp
level
service
environment
event
traceId
requestId
runId
conversationId
messageId
errorCode
safeMessage
```

不得只打自然语言日志。

## 7. Error Taxonomy

错误必须有稳定错误码。

用户可见错误必须使用 safeMessage。

不得把 stack trace、provider raw error、secret 或内部路径直接给用户。

## 8. Metrics

推荐指标包括：

```text
agenthub_run_started_total
agenthub_run_finished_total
agenthub_run_failed_total
agenthub_run_duration_ms
agenthub_a2a_task_duration_ms
agenthub_llm_latency_ms
agenthub_llm_tokens_total
agenthub_artifact_created_total
agenthub_agui_sse_events_total
agenthub_tool_call_failed_total
```

Metrics label 不得包含高基数字段或敏感内容。

## 9. Redaction

日志、错误、debug dump、metrics label 不得包含：

```text
API key
access token
refresh token
system prompt
完整 LLM raw request
完整用户隐私输入
数据库连接串
对象存储签名 URL
内部绝对路径
```

## 10. 禁止事项

不得：

- 只打自然语言日志。
- 没有 requestId / runId / messageId 就执行主链路。
- 前端、Gateway、Agent 各自发明不关联的 ID。
- 多 Agent 并行时丢失 stepId / a2aTaskId。
- Artifact 失败但没有 artifactId / messageId / runId 关联。
- 把 API key、token、prompt secret 写入日志。
- 把 Provider 原始错误直接返回给用户。
- 把完整 prompt 或完整用户隐私输入写入 debug dump。
- 把 stack trace 直接暴露给用户。
- 在 metrics label 中写入高基数字段或敏感内容。
