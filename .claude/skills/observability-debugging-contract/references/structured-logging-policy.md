# Structured Logging Policy

## 目的

定义 AgentHub 服务端结构化日志格式。

## 最小字段

- `timestamp`
- `level`
- `service`
- `environment`
- `event`
- `traceId`
- `requestId`
- `runId`
- `errorCode`
- `safeMessage`
- `durationMs`

## 推荐事件命名

事件使用 dot.case，例如：

- `run.started`
- `plan.validation_failed`
- `agent_task.completed`
- `gateway.orchestrator.call_failed`
- `llm.structured_output.invalid`

## 禁止事项

- 禁止只输出自然语言日志。
- 禁止在日志中打印 API key、token、完整 prompt、数据库连接串。
- 禁止把用户原始输入作为日志主字段长期保存。
