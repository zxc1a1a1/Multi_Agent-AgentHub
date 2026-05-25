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
planningMode       # 外部请求: direct / mention / auto / manual；内部 fallback plan: fallback
createdBy          # direct / mention / auto / manual / rule / fallback
tasks
fallback           # fallback.mode: none / same_capability_alternative / lower_risk_plan / single_agent_fallback / fail_fast
parentPlanId       # fallback plan 必须关联原 plan
fallbackOf         # 指向原 plan 的引用
reason             # 触发 fallback 的原因摘要
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
- `createdBy` 可以是 direct、mention、auto、manual、rule 或 fallback。
- `createdBy = fallback` 表示此 plan 是 Orchestrator 内部生成的降级计划，不由外部请求直接创建。
- fallback plan 必须通过 `parentPlanId` 或 `fallbackOf` 关联原 plan。
- Plan 不得包含 secret。
- taskContent 应是给目标 Agent 的脱敏任务说明。
- sequential 必须通过 dependsOn 表达依赖。
