# PlanTrace 规则

PlanTrace 用于说明计划如何产生、如何校验、为何选择某些目标。

## 推荐字段

- `planId`
- `runId`
- `plannerType`
- `planningMode`
- `selectedAgentNames`
- `rejectedAgentNames`
- `validationErrors`
- `fallbackAttempts`
- `createdAt`

## 规则

- PlanTrace 必须脱敏。
- 不保存 API key、token、完整 system prompt。
- 不保存未脱敏 LLM 原始长输出。
- 应能支持调试和 Demo 解释。
