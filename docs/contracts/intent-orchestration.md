# Intent Orchestration Contract

## 目的

本文定义 AgentHub Orchestrator Service 中“用户意图 → 结构化编排计划 → 本地校验 → 可执行调度”的数据与流程契约。

## 当前 Profile

- MVP v0.1 已完成，仅作为历史基线。
- 当前支持 2+ Agent。
- Gateway 与 Orchestrator 分进程。
- 意图编排只属于 Orchestrator Service。
- 不固定任何具体 Agent 名称。

## 核心对象

- PlannerInput
- PlanningMode
- AgentCapabilitySet
- OrchestrationPlan
- TaskPlan
- PlanTrace
- SafeError

## 核心流程

```text
PlannerInput
  → Planner
  → OrchestrationPlan
  → Plan Validation
  → Execution Strategy
  → Result / SafeError
```

## 硬性规则

- 所有 Planner 都必须输出结构化计划。
- 所有计划都必须先校验后执行。
- LLM 不得直接驱动执行。
- Gateway 不得承担意图编排。
- Agent 能力判断必须基于可用能力集合，而不是具体 Agent 名称分支。
