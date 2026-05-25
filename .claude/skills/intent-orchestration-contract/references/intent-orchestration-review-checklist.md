# Intent Orchestration Review Checklist

## 通用性

- 是否没有写死具体 Agent 名称？
- 是否没有通过 agentName 推断能力？
- 是否支持 2+ Agent 候选？
- 是否支持单聊与群聊？

## Planner

- 是否定义 PlannerInput？
- 是否定义 PlanningMode？
- 所有 Planner 是否输出结构化 OrchestrationPlan？
- LLM 输出是否本地 schema validation？
- LLM 是否不能直接执行计划？

## Plan

- 是否定义 OrchestrationPlan？
- taskId 是否唯一？
- strategy 是否受限枚举？
- ordered_parallel 是否输出稳定？
- sequential 是否校验 dependsOn？

## 校验

- agentName 是否存在？
- Agent 是否 enabled / healthy？
- capabilityIds 是否存在？
- expectedOutputs 是否支持？
- dependsOn 是否无环？
- fallback 是否不会无限循环？
- fallback.mode 是否为正式枚举：none / same_capability_alternative / lower_risk_plan / single_agent_fallback / fail_fast？
- same_capability_alternative 是否按 healthy 优先选择？
- 外部请求 planningMode 是否不含 fallback？
- fallback plan 是否由 Orchestrator 内部生成？
- fallback plan 是否关联原 plan（parentPlanId / fallbackOf）？

## 安全

- plan 是否不含 secret？
- error 是否脱敏？
- trace 是否可审计？
- Gateway 是否不承担意图编排？
