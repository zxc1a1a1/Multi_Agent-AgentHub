# Debug Playbook

## 目的

定义 AgentHub 常见故障的排障入口和必查字段。

## 前端收不到流

检查：

- `requestId`
- `runId`
- Gateway `gateway.stream.opened`
- Gateway `gateway.stream.closed`
- 最后一个前端 eventType
- `errorCode`

## Gateway 到 Orchestrator 失败

检查：

- `gateway.orchestrator.call_started`
- `gateway.orchestrator.call_failed`
- `orchestratorUrl`
- `statusCode`
- `durationMs`
- `errorCode`

## 编排计划校验失败

检查：

- `planId`
- `runId`
- `planningMode`
- `PLAN_SCHEMA_INVALID`
- `PLAN_CAPABILITY_UNSUPPORTED`

## Agent 任务失败

检查：

- `agentTaskId`
- `agentName`
- `capabilityId`
- `fallbackAttempt`
- `errorCode`

## LLM 失败

检查：

- `llmRequestId`
- `providerName`
- `modelId`
- `LLM_TIMEOUT`
- `LLM_STRUCTURED_OUTPUT_INVALID`
- retry/fallback attempt

## 脱敏要求

所有排障输出不得包含 API key、token、完整 prompt、完整用户隐私输入或数据库连接串。
