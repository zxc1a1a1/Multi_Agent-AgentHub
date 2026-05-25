# PlanningMode 规则

支持：

- `direct`
- `mention`
- `manual`
- `auto`
- `fallback`

## 优先级

```text
manual > mention > direct > auto > fallback
```

## 规则

- 所有 PlanningMode 都必须生成 OrchestrationPlan。
- 所有 PlanningMode 都必须走 Plan Validation。
- mention / manual 不得绕过 Agent 健康状态校验。
- fallback 必须有最大次数和终止条件。
