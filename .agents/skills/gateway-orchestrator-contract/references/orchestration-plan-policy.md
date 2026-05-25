# OrchestrationPlan 规则

`OrchestrationPlan` 是 Orchestrator 的编排计划。

## 目标

- 不固定具体 Agent 名称。
- 支持 2+ Agent。
- 支持多种 planning 来源。
- 支持多种执行策略。

## 字段

```text
planId
intentSummary
strategy
createdBy
tasks
fallback
```

## TaskPlan 字段

```text
taskId
agentName
capabilityIds
taskContent
dependsOn
expectedOutputs
```

## 规则

- Plan 必须校验后执行。
- `agentName` 只是目标标识，不用于能力推断。
- `createdBy` 可以是 direct、mention、auto、manual 或 rule。
- Plan 不得包含 secret。
- taskContent 应是给目标 Agent 的脱敏任务说明。
- sequential 必须通过 dependsOn 表达依赖。
