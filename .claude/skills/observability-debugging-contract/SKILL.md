---
name: observability-debugging-contract
description: 当定义、实现、修改或审查 AgentHub 中 traceId/runId 传播、结构化日志、错误码、指标、脱敏、调试手册、跨协议排障或 OpenTelemetry 接入时，使用本 Skill。
---

# observability-debugging-contract

## 1. 目的

本 Skill 定义 AgentHub 的可观测性与排障开发契约。

本 Skill 约束的核心链路是：

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

目标：

- 让每个 run、message、A2A task、Artifact、Tool Call、LLM request 都能被关联。
- 让前端、Gateway、Orchestrator、子 Agent、LLM Provider 和 Artifact 链路可排障。
- 统一 traceId、requestId、runId、messageId、a2aTaskId、artifactId、toolCallId。
- 统一结构化日志、错误码、指标和脱敏策略。
- 避免只有自然语言日志，无法定位跨服务问题。
- 避免日志、错误、debug dump 泄漏 API key、token、prompt secret 或用户敏感内容。

## 2. 官方标准优先级

涉及分布式追踪、日志关联、指标和上下文传播时，以官方标准为优先参考。

优先级：

```text
OpenTelemetry / W3C Trace Context 官方规范
> AgentHub 可观测性 contract
> 各服务自行发明的日志字段
```

官方标准参考方向：

- W3C Trace Context：`traceparent`、`tracestate` 等跨服务追踪上下文。
- OpenTelemetry：traces、metrics、logs、context propagation。
- OpenTelemetry Semantic Conventions：通用 span、metric、attribute 命名语义。

AgentHub 可以定义项目内 ID、事件名、错误码和排障手册，但不得和官方 trace context 传播规则冲突。

## 3. 文件位置说明

### 项目级 Contract 文件

以下文件属于 AgentHub 项目仓库，是项目级事实源：

```text
<repo-root>/docs/contracts/observability-debugging.md
<repo-root>/docs/contracts/observability-debugging.schema.json
```

### 当前 Skill 的参考文件

以下文件属于当前 Coding Agent Skill：

```text
<current-skill-dir>/references/trace-context.md
<current-skill-dir>/references/structured-logging.md
<current-skill-dir>/references/error-taxonomy.md
<current-skill-dir>/references/metrics-policy.md
<current-skill-dir>/references/redaction-policy.md
<current-skill-dir>/references/debug-playbook.md
<current-skill-dir>/references/span-naming.md
<current-skill-dir>/references/frontend-debugging.md
```

如果当前 Skill 安装在 Claude Code 项目目录中，则 `<current-skill-dir>` 通常是：

```text
<repo-root>/.claude/skills/observability-debugging-contract
```

## 4. 适用场景

当进行以下工作时，启用本 Skill：

- 新增或修改 traceId / requestId / runId / messageId 传播。
- 新增或修改结构化日志。
- 新增或修改错误码。
- 新增或修改 metrics。
- 新增或修改 OpenTelemetry span。
- 新增或修改 debug endpoint / debug dump。
- 新增或修改日志脱敏规则。
- 排查 AG-UI SSE、A2A、Artifact、LLM、前端 Runtime Skill 问题。
- 接入多 Agent parallel / sequential / fallback。
- 审查日志是否泄漏 secret、prompt、token 或用户敏感内容。

## 5. 长期契约基线

长期架构中，AgentHub 必须能通过统一 ID 串联完整执行链路。

长期必须定义：

- Trace Context 传播规则。
- Request ID 规则。
- Run ID 规则。
- Conversation ID / Message ID 规则。
- ExecutionPlan ID / stepId 规则。
- A2A Task ID 规则。
- Artifact ID 规则。
- Tool Call ID 规则。
- LLM Request ID 规则。
- Span 命名规则。
- 结构化日志规则。
- 错误码规则。
- Metrics 规则。
- Redaction / 脱敏规则。
- Debug playbook。
- 前端调试规则。
- SSE / A2A / Artifact / LLM 排障规则。

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

## 6. MVP 约束

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

MVP 最小日志示例：

```json
{
  "level": "info",
  "event": "run_started",
  "traceId": "trace_123",
  "requestId": "req_123",
  "runId": "run_123",
  "conversationId": "conv_123",
  "messageId": "msg_123",
  "agentName": "code-agent"
}
```

MVP 阶段不要求：

- OpenTelemetry Collector。
- Jaeger。
- Prometheus。
- Grafana。
- distributed tracing backend。
- complex dashboards。
- cost analytics。
- alert rules。

## 7. 阶段演进规则

### MVP 阶段

只实现：

```text
lightweight IDs
structured JSON logs
safe error codes
redaction
```

不得只打自然语言日志。

不得没有 runId / requestId / messageId。

不得在日志中暴露 secret、token、prompt secret、完整 provider 原始错误。

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

### 多 Agent 阶段

必须支持：

- executionPlanId。
- stepId。
- parallel task trace。
- sequential dependency trace。
- fallback trace。
- partial failure trace。
- multi artifact trace。
- final aggregation trace。

## 8. 本 Skill 负责

本 Skill 负责：

- Trace Context 传播规则。
- AgentHub ID 关联规则。
- 结构化日志字段。
- 错误码 taxonomy。
- Metrics 命名和采集规则。
- Span 命名和属性规则。
- 日志 / 错误 / debug dump 脱敏规则。
- 前端调试规则。
- SSE / A2A / Artifact / LLM 排障手册。
- MVP 轻量日志规则。
- P1 OpenTelemetry 演进规则。

## 9. 本 Skill 不负责

本 Skill 不负责：

- Orchestrator 如何选择 Agent。
- ExecutionPlan schema。
- A2A Task 协议。
- AG-UI 事件结构。
- Frontend Runtime Skill 参数 schema。
- React Component 实现。
- Artifact schema。
- LLM Provider Adapter 具体实现。
- 数据库完整 DDL。
- 安全权限策略的完整定义。
- Docker Compose 交付规则。
- 通用 Go / TypeScript 代码风格。

涉及以上内容时，只引用相关 contract，不在本 Skill 中重新定义。

## 10. Contract first 规则

任何新增或修改可观测性字段、错误码、日志格式、metrics、span、debug dump 前，必须先更新：

```text
<repo-root>/docs/contracts/observability-debugging.md
<repo-root>/docs/contracts/observability-debugging.schema.json
```

未更新 contract 的实现变更不得接受。

## 11. 核心规则

### Trace Context

跨 HTTP / SSE / A2A / LLM 调用时，必须传播 trace context 或等价 traceId。

正式开发阶段优先支持 W3C Trace Context：

```text
traceparent
tracestate
```

### Structured Logging

日志必须结构化。

不得只打自然语言日志。

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

### Error Taxonomy

错误必须有稳定错误码。

用户可见错误必须使用 safeMessage。

不得把 stack trace、provider raw error、secret 或内部路径直接给用户。

### Metrics

指标必须可聚合、可命名、可按 service / agent / route / provider 维度分析。

MVP 可先不接 metrics backend，但事件命名要预留。

### Redaction

日志、错误、debug dump、metrics label 不得包含敏感信息。

禁止出现：

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

## 12. 禁止事项

Coding Agent 不得：

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
- 引入 OpenTelemetry 时破坏 W3C Trace Context 传播。
- 未更新 contract 就新增错误码或日志事件。

## 13. 相关契约

本 Skill 只引用以下契约，不重新定义它们：

- `intent-orchestration-contract`：负责 ExecutionPlan、strategy、fallback 和多 Agent 编排。
- `adk-runtime-contract`：负责子 Agent Runtime、Task handler、`ctx.StreamText`、`ctx.AddArtifact`。
- `artifact-contract`：负责 Artifact schema、生命周期、存储和预览映射。
- `llm-provider-contract`：负责 Provider Adapter、streaming normalization、retry、rate limit 和 fallback。
- `a2a-agent-contract`：负责 A2A Task、AgentCard、Streaming 和错误语义。
- `agui-event-contract`：负责 AG-UI 事件名称和事件结构。
- `security-boundary-contract`：负责 secret、权限、沙箱和敏感信息保护。
- `data-persistence-contract`：负责数据库、Redis、对象存储和迁移策略。
