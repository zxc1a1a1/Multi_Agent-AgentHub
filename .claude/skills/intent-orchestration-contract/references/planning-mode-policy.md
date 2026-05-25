# PlanningMode 规则

## 外部请求 planningMode（OrchestratorRequest）

Gateway 传入 Orchestrator 的外部请求只允许：

- `direct` — 用户直接指定单个目标 Agent
- `mention` — 用户通过 @mention 提到 Agent
- `manual` — 用户手动选择多个 Agent
- `auto` — Orchestrator 自动规划

外部请求 **不得** 传入 `fallback`。`fallback` 不是前端或 Gateway 可选的请求模式。

## 内部计划 planningMode（OrchestrationPlan）

Orchestrator 内部生成的 OrchestrationPlan 可使用：

- `direct` / `mention` / `manual` / `auto` — 与外部请求一致的来源标记
- `fallback` — Orchestrator 在主计划失败、风险过高或健康检查失败后内部生成的降级计划

## fallback plan 规则

- `fallback` 只能由 Orchestrator 内部生成，不得由 Gateway 或 Frontend 传入。
- fallback plan 必须关联原 plan，推荐字段：
  - `parentPlanId` — 原计划 ID
  - `fallbackOf` — 指向原 plan 的引用
  - `reason` — 触发 fallback 的原因摘要
- fallback plan 仍然必须通过 Plan Validation。
- `OrchestrationPlan.createdBy` 对应 fallback 时为 `fallback`。

## 优先级

```text
manual > mention > direct > auto
```

`fallback` 不在路由优先级排序中，它是计划失败后的降级行为，不是路由优先级的一级。

## 规则

- 所有 PlanningMode 都必须生成 OrchestrationPlan。
- 所有 PlanningMode 都必须走 Plan Validation。
- mention / manual 不得绕过 Agent 健康状态校验。
- fallback 必须有最大次数和终止条件。
