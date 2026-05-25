---
name: intent-orchestration-contract
description: "用于定义 AgentHub Orchestrator Service 中用户意图到结构化编排计划的独立契约，包括 Planner 输入、PlanningMode、通用 OrchestrationPlan、能力校验、路由优先级、single/ordered_parallel/sequential、LLM 结构化输出、fallback/retry、群聊 @mention、计划追踪和安全错误。"
---

# intent-orchestration-contract

## 1. Skill 目的

本 Skill 定义 AgentHub 中 **Intent Orchestration（意图编排）** 的通用契约。

它关注的是：

```text
用户输入 + 会话上下文 + 可用能力集合
  → PlannerInput
  → OrchestrationPlan
  → Plan Validation
  → 可执行调度决策
```

本 Skill 的目标是保证 Orchestrator Service 在面对多个子 Agent、群聊、显式 @mention、手动选择、LLM 编排、规则降级和 fallback 时，仍然使用**结构化、可校验、可追踪、可回放**的计划对象，而不是依赖临时字符串拼接或硬编码 Agent 名称。

## 2. 独立性原则

本 Skill 必须独立可读。

本 Skill 不要求读者先阅读其他 Skill 才能理解本文。

可以在边界说明中提到外部概念，但不得复制或展开其他 Skill 的详细规则。

本 Skill 不定义：

- Gateway 与 Orchestrator 的服务间 HTTP / RPC API。
- 前端实时事件字段。
- 子 Agent 的内部协议。
- 前端 Runtime Skill 参数细节。
- Artifact 完整 schema。
- 数据库表结构。
- LLM Provider SDK 封装。
- 某个具体 Agent 的实现。

## 3. 当前阶段识别

当前约束：

```text
mvpStatus = completed
activeProfile = v1-generic-orchestration
gatewayMode = separate-process
orchestratorMode = separate-process
agentCount = 2+
```

MVP v0.1 已完成，只作为历史兼容与回归测试基线。

当前阶段必须支持：

- 2+ Agent 候选。
- 单聊与群聊上下文。
- @mention / manual / direct / auto / fallback 路由模式。
- LLM Planner 与规则 fallback。
- 结构化 OrchestrationPlan。
- schema validation。
- Agent / capability / expected output 校验。
- `single` / `ordered_parallel` / `sequential`。
- fallback / retry。
- PlanTrace / SafeError。

本 Skill 不固定任何具体 Agent 名称。

## 4. MVP v0.1 Historical Profile

MVP v0.1 历史基线包括：

- 单 Agent 直接路由。
- 静态 Agent 配置。
- 最小文本流。
- 最小错误返回。
- 不启用 LLM Planner。
- 不启用多 Agent 策略。
- 不启用 fallback / retry。

这些历史规则不得继续作为当前开发禁令。

旧字段或旧 schema 可以保留兼容，但新增设计必须面向通用多 Agent 编排。

## 5. 本 Skill 负责什么

本 Skill 负责：

- PlannerInput 的通用输入结构。
- PlanningMode 的枚举和优先级。
- OrchestrationPlan 的结构。
- TaskPlan 的结构。
- AgentCapabilitySet 的抽象。
- Plan Validation 的规则。
- 路由优先级。
- `single` / `ordered_parallel` / `sequential` 策略。
- LLM Planner 结构化输出规则。
- rule / mention / manual Planner 规则。
- fallback / retry 规则。
- 群聊与 @mention 规则。
- PlanTrace 与审计摘要。
- SafeError。
- Contract Test 与 Review Checklist。

## 6. 本 Skill 不负责什么

本 Skill 不负责：

- Gateway Service 如何通过网络调用 Orchestrator Service。
- Orchestrator Service 如何调用子 Agent。
- 前端如何消费流式事件。
- 前端如何渲染 Runtime Capability。
- Artifact 如何持久化。
- 数据库 migration。
- Docker Compose 服务定义。
- 具体 LLM Provider 的请求格式。

如果这些内容出现在当前任务中，只能作为输入或输出摘要参与意图编排，不得在本 Skill 内展开实现细节。

## 7. 核心原则

### 7.1 结构化计划优先

Orchestrator Service 不得直接执行自然语言路由结果。

任何自动、手动、mention、规则或 LLM 产生的调度决策，都必须先转成结构化 `OrchestrationPlan`。

### 7.2 先校验，后执行

计划必须经过 schema、Agent、capability、expectedOutputs、dependsOn、fallback、安全字段校验后，才允许执行。

### 7.3 绑定能力，不绑定 Agent 名称分支

不得写：

```text
if agentName == "某个具体 Agent" then ...
```

应该写：

```text
根据 availableAgents 中声明的 capabilities / outputTypes / healthy 状态进行选择与校验。
```

### 7.4 LLM 不直接驱动执行

LLM 可以生成候选计划，但不能直接调用 Agent、工具或外部系统。

LLM 输出必须是结构化 JSON，并由 Orchestrator Service 本地校验。

### 7.5 Gateway 不承担意图编排

Gateway 和 Orchestrator 分进程。

意图编排只属于 Orchestrator Service。

Gateway 不得：

- 调用 LLM Planner。
- 生成 OrchestrationPlan。
- 根据关键词选择 Agent。
- 执行 fallback。
- 绕过 Orchestrator 直接执行调度。

## 8. PlannerInput

`PlannerInput` 是 Planner 的唯一标准输入。

推荐结构：

```json
{
  "runId": "run_001",
  "conversationId": "conv_001",
  "conversationType": "group",
  "userMessage": "用户当前输入",
  "historySummary": "可选历史摘要",
  "messages": [],
  "availableAgents": [],
  "runtimeCapabilities": [],
  "mentions": [],
  "manualSelectedAgents": [],
  "planningMode": "auto",
  "constraints": {
    "maxTasks": 4,
    "maxRetries": 1,
    "allowSequential": true,
    "allowOrderedParallel": true
  },
  "traceId": "trace_001"
}
```

规则：

- `availableAgents` 是当前可选 Agent 的摘要。
- `runtimeCapabilities` 是前端可处理能力摘要，不等价于 Agent 能力。
- `mentions` 是用户显式路由提示，不是无校验执行命令。
- `manualSelectedAgents` 是用户或 UI 显式选择，不得绕过校验。
- `historySummary` 必须可控，不得无限塞入完整历史。
- `traceId` 必须可用于排查一次计划生成过程。

## 9. PlanningMode

支持以下 PlanningMode：

| 模式 | 含义 |
|---|---|
| `direct` | 会话、请求或上游上下文已有明确目标 |
| `mention` | 用户通过 @agent-name 指定目标 |
| `manual` | 用户或 UI 手动选择一个或多个候选 Agent |
| `auto` | Planner 自动选择 Agent 和策略 |
| `fallback` | 主计划失败后的降级计划 |

规则：

- `direct` / `mention` / `manual` 也必须生成结构化 Plan。
- 任何模式都必须校验 Agent 是否存在、启用且可用。
- `mention` 优先级较高，但不是无条件执行。
- `auto` 可以由 LLM、规则或混合 Planner 实现。
- `fallback` 必须有最大次数和终止条件。

## 10. OrchestrationPlan

`OrchestrationPlan` 是意图编排结果的标准结构。

推荐结构：

```json
{
  "version": "v1",
  "planId": "plan_001",
  "runId": "run_001",
  "conversationId": "conv_001",
  "planningMode": "auto",
  "strategy": "ordered_parallel",
  "intentSummary": "用户想生成一个完整应用方案",
  "tasks": [
    {
      "taskId": "task_001",
      "agentName": "some-agent",
      "capabilityIds": ["capability_id"],
      "taskContent": "给该 Agent 的任务描述",
      "dependsOn": [],
      "expectedOutputs": ["code", "markdown"],
      "priority": 1
    }
  ],
  "aggregation": {
    "mode": "message_per_task",
    "summaryRequired": false
  },
  "fallback": {
    "mode": "same_capability_alternative",
    "maxAttempts": 1
  },
  "validation": {
    "schemaVersion": "v1",
    "validated": false
  }
}
```

规则：

- `planId` 必须唯一。
- `runId` 必须来自当前 Run。
- `strategy` 必须是允许枚举。
- `tasks` 不得为空。
- `intentSummary` 必须是摘要，不得保存完整敏感 prompt。
- `validation.validated = true` 只能由本地校验器设置，LLM 不得自行设置可信状态。
- `capabilityIds` 必须来自目标 Agent 的 `AgentCard.skills[].id`，不得凭自然语言临时编造。

## 11. TaskPlan

`TaskPlan` 是 OrchestrationPlan 中的单个执行单元。

`TaskPlan.capabilityIds` 必须引用目标 Agent 的 `AgentCard.skills[].id`。不得凭自然语言临时生成未注册的 capabilityId，不得使用 `toolName`、`artifact.type` 或 `outputMode` 作为 capabilityId。

字段规则：

| 字段 | 规则 |
|---|---|
| `taskId` | 必须唯一 |
| `agentName` | 必须来自 `availableAgents` |
| `capabilityIds` | 必须来自目标 Agent 的 `AgentCard.skills[].id` |
| `taskContent` | 必须是给目标 Agent 的清晰任务，不得包含 secret |
| `dependsOn` | 只能引用同 Plan 内已有 taskId |
| `expectedOutputs` | 必须来自目标能力支持的输出类型 |
| `priority` | 用于 ordered_parallel 或排序，不代表安全级别 |

禁止：

- 通过 Agent 名称推断能力。
- 让 LLM 编造 `agentName`。
- 让 LLM 编造不存在的 capabilityId。
- 把完整 system prompt 或 API key 写入 `taskContent`。
- 把 `toolName` 当作 capabilityId。
- 把 `artifact.type` 当作 capabilityId。
- 把 `outputMode` 当作 capabilityId。

## 12. AgentCapabilitySet

本 Skill 使用 `AgentCapabilitySet` 抽象可用能力集合，不绑定具体 Agent 实现。

推荐结构：

```json
{
  "agentName": "some-agent",
  "healthy": true,
  "enabled": true,
  "capabilities": [
    {
      "id": "capability_id",
      "description": "能力说明",
      "inputTypes": ["text"],
      "outputTypes": ["markdown", "code"]
    }
  ]
}
```

规则：

- `agentName` 只用于标识目标。
- 能力判断必须基于 `capabilities` / `inputTypes` / `outputTypes`。
- `healthy=false` 或 `enabled=false` 的 Agent 不得被主计划选择。
- fallback 可以选择同能力的其他 healthy Agent。

## 13. Plan Validation

Plan Validation 必须在执行前完成。

校验步骤：

1. 校验 JSON schema。
2. 校验 `version`。
3. 校验 `strategy`。
4. 校验 task 数量。
5. 校验每个 `agentName` 存在。
6. 校验 Agent `enabled` / `healthy`。
7. 校验 `capabilityIds` 存在（必须来自目标 Agent 的 `AgentCard.skills[].id`）。
8. 校验 `expectedOutputs` 可被目标能力支持。
9. 校验 `dependsOn` 无环。
10. 校验 fallback 不会无限循环。
11. 校验安全字段无敏感信息。
12. 标记本地 validation 结果。

任何校验失败都不得执行该计划。

## 14. Routing Priority

推荐路由优先级（外部请求）：

```text
manual selected agents
  > explicit @mention
  > conversation direct target
  > auto planner
```

`fallback` 不在路由优先级排序中。它是 Orchestrator 在主计划失败、风险过高或健康检查失败后的内部降级行为：
- fallback plan 由 Orchestrator 内部生成，不由 Gateway 或 Frontend 传入。
- fallback plan 必须关联原 plan（`parentPlanId` / `fallbackOf`）。
- fallback plan 仍需通过 Plan Validation。

规则：

- 高优先级输入也必须校验。
- 目标 Agent 不存在或不可用时，不得盲目执行。
- 用户显式选择多个 Agent 时，可以生成 `ordered_parallel` 或 `sequential`。
- 无显式选择时，`auto` Planner 可以选择一个或多个 Agent。

## 15. Strategy

支持策略：

| strategy | 含义 |
|---|---|
| `single` | 一个 task |
| `ordered_parallel` | 多个独立 task，语义上可并行，但输出按稳定顺序聚合 |
| `sequential` | 多个有依赖 task，必须按 `dependsOn` 执行 |

兼容规则：

```text
legacy parallel = ordered_parallel
```

禁止：

- 在没有稳定 message/task 隔离的情况下做 token 级交错输出。
- 在 `sequential` 中忽略 `dependsOn`。
- 在 `ordered_parallel` 中让输出顺序不可预测。

## 16. LLM Planner Rules

LLM Planner 是当前可用 Planner 类型之一，但不是唯一 Planner。

规则：

- LLM Planner 输出必须是结构化 JSON。
- 优先使用 provider 原生 structured output。
- 不管 provider 是否声称严格输出，都必须本地 schema validation。
- schema validation 失败不得执行。
- LLM Planner 不得直接调用 Agent。
- LLM Planner 不得返回自然语言计划后让代码猜测执行。
- LLM Planner prompt 只能包含可选 Agent / capability 摘要，不得包含 secret。
- LLM 原始输出不得直接作为执行依据。
- 必要时只保存脱敏摘要和 PlanTrace。

## 17. Rule / Mention / Manual Planner Rules

非 LLM Planner 也必须输出结构化 OrchestrationPlan。

规则：

- rule planner 可以用于 fallback 和低成本路由。
- mention planner 只能把 @mention 转为候选目标。
- manual planner 只能把用户选择转为候选目标。
- 所有 Planner 输出都必须统一走 Plan Validation。
- 不得为不同 Planner 维护多套执行路径。

## 18. Fallback / Retry Rules

支持的 fallback mode：

| mode | 含义 |
|---|---|
| `none` | 不 fallback，失败即失败 |
| `same_capability_alternative` | 选择具备相同能力的其他健康 Agent |
| `lower_risk_plan` | 降级为更简单、更低风险计划 |
| `single_agent_fallback` | 降级为单 task 计划 |
| `fail_fast` | 快速失败并返回安全错误 |

规则：

- retry 必须有最大次数。
- fallback 必须有最大次数。
- fallback 不得无限循环。
- fallback 不得选择已失败且不可恢复的目标。
- fallback 后必须重新校验计划。
- fallback 后实际执行 Agent 必须可追踪。
- 所有 Agent 都失败时必须返回 SafeError。

## 19. Group Conversation / Mention Rules

群聊与 @mention 规则：

- 群聊中一个用户输入可以生成多个 task。
- @mention 是路由提示，不是无校验执行命令。
- 多个 @mention 可以生成 `ordered_parallel` 或 `sequential`。
- 无 mention 时，auto planner 可以选择一个或多个 Agent。
- 每个 task 必须记录 `agentName`。
- fallback 后必须记录实际执行 Agent。
- 同一个 Run 可以有多条 Agent 回复。

## 20. PlanTrace / Audit

推荐 PlanTrace：

```json
{
  "planId": "plan_001",
  "runId": "run_001",
  "plannerType": "llm_planner",
  "planningMode": "auto",
  "selectedAgentNames": ["some-agent"],
  "rejectedAgentNames": [],
  "validationErrors": [],
  "fallbackAttempts": 0,
  "createdAt": "2026-05-25T00:00:00Z"
}
```

规则：

- PlanTrace 用于调试、审计和 Demo 解释。
- PlanTrace 不得保存 API key、token、完整 system prompt。
- PlanTrace 不得保存未脱敏 LLM 原始长输出。
- PlanTrace 应能说明为什么选择某些 Agent。

## 21. SafeError

推荐结构：

```json
{
  "code": "PLAN_VALIDATION_FAILED",
  "message": "编排计划校验失败",
  "retryable": false,
  "details": {
    "reason": "agent_not_available"
  }
}
```

错误规则：

- 用户可见错误必须脱敏。
- 内部日志可以记录更多上下文，但不得记录 secret。
- LLM 原始错误不得直接返回给用户。
- plan validation 失败应返回明确错误码。

推荐错误码：

```text
PLANNER_INPUT_INVALID
PLANNER_LLM_FAILED
PLANNER_OUTPUT_INVALID_JSON
PLAN_SCHEMA_INVALID
PLAN_VALIDATION_FAILED
PLAN_AGENT_NOT_FOUND
PLAN_AGENT_UNHEALTHY
PLAN_CAPABILITY_NOT_FOUND
PLAN_OUTPUT_UNSUPPORTED
PLAN_DEPENDENCY_CYCLE
PLAN_FALLBACK_EXHAUSTED
PLAN_INTERNAL_ERROR
```

## 22. Contract-first 规则

修改意图编排行为前，必须先更新：

```text
docs/contracts/intent-orchestration.md
docs/contracts/orchestration-plan.md
docs/contracts/orchestration-plan.schema.json
```

如果保留历史 `execution-plan` 命名，必须明确它是兼容别名，不得与 `orchestration-plan` 分叉。

## 23. Contract Test 规则

至少应测试：

- LLM 返回非法 JSON。
- LLM 返回不存在的 Agent。
- LLM 返回不存在的 capability。
- Agent unhealthy。
- expectedOutputs 不支持。
- dependsOn 成环。
- ordered_parallel 多 task。
- sequential 依赖顺序。
- @mention 不存在 Agent。
- fallback 成功。
- fallback 失败。
- plan 中包含疑似 secret。

## 24. Review Checklist

Review 时必须检查：

- 是否没有写死具体 Agent 名称。
- 是否没有通过 agentName 推断能力。
- 是否支持 2+ Agent 候选。
- 是否支持单聊与群聊。
- 是否定义 PlannerInput。
- 是否定义 PlanningMode。
- 是否定义 OrchestrationPlan。
- 是否所有 Planner 都输出结构化计划。
- 是否本地 schema validation。
- 是否校验 Agent enabled / healthy。
- 是否校验 capabilityIds。
- 是否校验 expectedOutputs。
- 是否校验 dependsOn 无环。
- 是否支持 `single` / `ordered_parallel` / `sequential`。
- fallback 是否有最大次数。
- fallback 是否重新校验。
- LLM 是否不能直接执行计划。
- Gateway 是否不承担意图编排。
- plan / trace / error 是否脱敏。
- `capabilityIds` 是否来自目标 Agent 的 `AgentCard.skills[].id`。
- 是否没有把 `toolName`、`artifact.type`、`outputMode` 当作 capabilityId。

## 25. 完成定义

本 Skill 视为完成，当且仅当：

- `SKILL.md` 独立可读。
- MVP v0.1 已降级为历史基线。
- 当前契约支持 2+ Agent。
- 不固定任何具体 Agent 名称。
- Gateway 与 Orchestrator 分进程的职责边界明确。
- 意图编排只属于 Orchestrator Service。
- PlannerInput / PlanningMode / OrchestrationPlan / TaskPlan / AgentCapabilitySet 明确。
- Plan Validation 明确。
- LLM Planner 结构化输出与本地校验规则明确。
- fallback / retry 明确。
- 群聊与 @mention 规则明确。
- PlanTrace / SafeError 明确。
- references 和 docs/contracts 同步更新。
