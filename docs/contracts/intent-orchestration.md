# Intent Orchestration Contract

版本：v0.1-mvp  
适用项目：AgentHub - 多 Agent 协作平台  
适用阶段：MVP + 后续正式开发演进  
事实源文件：

```text
docs/contracts/intent-orchestration.md
docs/contracts/execution-plan.schema.json
```

## 1. 目的

本文定义 AgentHub 意图编排的项目级契约。

意图编排是指：将用户消息、会话上下文、Agent Registry 和 AgentCard.skills 转换为可验证、可执行、可追踪的 ExecutionPlan，并最终调度为 A2A Task。

核心链路：

```text
User Message
+ Conversation History
+ Agent Registry
+ AgentCards
→ ExecutionPlan
→ Plan Validation
→ A2A Task Dispatch
```

## 2. 官方 / 标准约束优先级

本契约依赖以下官方或事实标准：

- A2A 官方规范：用于约束 AgentCard、skills、Task、Message、Artifact 和 Streaming。
- JSON Schema 官方规范：用于约束 `execution-plan.schema.json`。
- LLM provider structured output 官方能力：用于约束后续 LLM planner 输出。

优先级：

```text
A2A 官方规范 > 本项目 A2A 落地约定
JSON Schema 官方规范 > 手写非标准校验格式
结构化 ExecutionPlan schema > LLM 自然语言 plan
```

## 3. Contract first 规则

任何新增、修改或删除意图编排策略、ExecutionPlan、LLM planner、fallback 或多 Agent 编排前，必须先更新：

```text
docs/contracts/intent-orchestration.md
docs/contracts/execution-plan.schema.json
```

未更新 contract 的实现变更不得接受。

## 4. 阶段演进规则

### 4.1 MVP 阶段

MVP 阶段只实现：

```text
single direct routing
```

MVP 阶段配置：

```text
strategy = single
routing = direct
plannerLLM = disabled
targetAgent = req.AgentName 或 conversation.agentName
agentRegistry = config file
targetSkill = code_generate
```

MVP 阶段不实现：

- LLM planner。
- parallel。
- sequential。
- fallback。
- @Agent。
- 多 Agent 聚合。
- 动态 Agent Registry。
- 复杂失败降级。

### 4.2 正式开发阶段

MVP 完成后，可以逐步启用：

- @Agent 手动路由。
- LLM planner。
- parallel。
- sequential。
- fallback。
- 多 Agent 聚合。
- plan trace。
- run step 追踪。

启用前必须先更新本文和 schema。

## 5. ExecutionPlan

MVP 最小 ExecutionPlan：

```json
{
  "version": "v0.1",
  "strategy": "single",
  "reason": "Direct routing to selected code-agent.",
  "tasks": [
    {
      "id": "task_1",
      "agent": "code-agent",
      "skill": "code_generate",
      "input": {
        "message": "帮我写一个 Go HTTP 服务器"
      },
      "dependsOn": [],
      "requiredArtifacts": ["code"]
    }
  ],
  "fallback": null
}
```

ExecutionPlan 必须：

- 是 JSON。
- 符合 `execution-plan.schema.json`。
- 通过 Agent Registry 校验。
- 通过 AgentCard.skills 校验。
- 通过 strategy 校验。
- 校验通过后才能执行。

## 6. 路由优先级

长期路由优先级：

```text
1. 用户显式 @Agent
2. 当前 conversation 默认 Agent
3. 用户创建对话时选择的 Agent
4. LLM planner 基于 AgentCard 选择
5. fallback Agent
```

MVP 阶段只使用：

```text
req.AgentName
或 conversation.agentName
```

## 7. Agent / skill 校验

`task.agent` 必须来自 Agent Registry。

`task.skill` 必须来自目标 Agent 的 AgentCard.skills。

校验流程：

```text
ExecutionPlan
→ schema validation
→ task.agent exists
→ AgentCard loaded
→ task.skill exists in AgentCard.skills
→ execute A2A Task
```

## 8. LLM planner

MVP 阶段不启用 LLM planner。

正式开发阶段如果启用，必须：

- 输出 JSON。
- 符合 schema。
- 通过本地 validation。
- 只选择 Agent Registry 中存在的 Agent。
- 只选择 AgentCard.skills 中存在的 skill。
- 不直接执行自然语言 plan。

## 9. 编排策略

MVP 只实现：

```text
single
```

正式开发阶段可以扩展：

```text
parallel
sequential
```

启用前必须定义并发限制、依赖规则、聚合规则、失败策略和 trace 规则。

## 10. fallback

MVP 不实现 fallback。

正式开发阶段启用 fallback 前，必须定义：

- fallbackAgent。
- fallbackSkill。
- fallback 条件。
- fallback 最大次数。
- fallback trace。
- fallback 失败后的最终错误。

## 11. 禁止事项

不得：

- 执行自然语言 plan。
- 执行 schema validation 失败的 plan。
- 执行包含未知 Agent 的 plan。
- 执行包含未知 skill 的 plan。
- 在 MVP 阶段启用 LLM planner。
- 在 MVP 阶段启用 parallel / sequential。
- 在 MVP 阶段启用 fallback。
- 让 Gateway handler 承担复杂编排。
- 让 Orchestrator 直接调用前端 Runtime Skill。
- 把 MVP direct routing 规则扩展成长期唯一策略。
