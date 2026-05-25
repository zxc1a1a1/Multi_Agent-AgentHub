# OrchestrationPlan Schema 规则

OrchestrationPlan 是意图编排的唯一可执行计划格式。

## 必填字段

- `version`
- `planId`
- `runId`
- `conversationId`
- `planningMode`
- `strategy`
- `intentSummary`
- `tasks`
- `fallback`
- `validation`

## 规则

- `tasks` 不得为空。
- `strategy` 必须是 `single`、`ordered_parallel` 或 `sequential`。
- `validation.validated` 只能由本地校验器设置。
- `intentSummary` 是摘要，不得保存完整敏感 prompt。
- `taskId` 在同一个 plan 内必须唯一。
- `agentName` 必须来自 availableAgents。
- `capabilityIds` 必须来自目标 Agent 的声明能力。
- `expectedOutputs` 必须由目标能力支持。
- `dependsOn` 只能引用同计划内的 taskId。

## 禁止

- 让 LLM 自行声明计划已可信。
- 在 plan 中写入 API key、token、数据库连接串。
- 通过具体 Agent 名称推断能力。
