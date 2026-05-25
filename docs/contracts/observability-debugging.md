# AgentHub Observability Debugging Contract

## 目的

本文档是 AgentHub 可观测性与排障的事实源契约，定义 Trace Context、关联 ID、结构化日志、span、metrics、错误码、脱敏、debug dump 和跨服务排障规则。

## 当前阶段

- MVP v0.1 已完成，仅作为历史基线。
- Gateway 与 Orchestrator 必须分进程。
- 当前支持 2+ Agent、群聊、编排、fallback、丰富产物和多 LLM 请求。
- 本契约不绑定任何具体 Agent 名称。

## 核心要求

1. 每个外部请求必须有 `requestId`。
2. 每个 run 必须有 `runId`。
3. 跨服务必须传播 `traceparent` 或等价 trace context。
4. 日志必须是结构化 JSON。
5. 错误必须有稳定 `errorCode` 和脱敏 `safeMessage`。
6. debug dump 必须可控、脱敏、可关闭。
7. metrics label 不得包含高基数或敏感字段。
8. Gateway ↔ Orchestrator 分进程调用必须可排障。

## 最小日志字段

- `timestamp`
- `level`
- `service`
- `event`
- `traceId`
- `requestId`
- `runId`
- `errorCode`
- `durationMs`

## 禁止记录

- API key
- token
- Authorization header
- 数据库连接串
- 对象存储签名 URL
- 完整 prompt
- 完整 LLM 原始请求 / 响应
- 完整用户隐私输入
