---
name: intent-orchestration-contract
description: 当定义、实现、修改或审查 AgentHub 中用户意图到 ExecutionPlan、Agent 路由、AgentCard.skills 校验、single/parallel/sequential 编排、LLM planner 或 fallback 策略时，使用本 Skill。
---

# intent-orchestration-contract

## 1. 目的

本 Skill 定义 AgentHub 意图编排的开发契约。

意图编排是指：将用户消息、会话上下文、Agent Registry 和 AgentCard.skills 转换为可验证、可执行、可追踪的 ExecutionPlan，并最终调度为 A2A Task。

本 Skill 约束的核心链路是：

```text
User Message
+ Conversation History
+ Agent Registry
+ AgentCards
→ ExecutionPlan
→ Plan Validation
→ A2A Task Dispatch
```

本 Skill 的目标是确保 Coding Agent 在开发 Orchestrator 意图编排能力时：

- 不执行非结构化自然语言计划。
- 不让 LLM 编排出不存在的 Agent。
- 不让 LLM 编排出不存在的 skill。
- 不绕过 Agent Registry。
- 不绕过 AgentCard.skills 校验。
- 不把 Gateway handler 写成复杂编排器。
- 不让 Orchestrator 直接生成前端 UI 组件。
- 不把 A2A Task 协议、Artifact schema、AG-UI 事件结构混进意图编排 contract。

## 2. 官方 / 标准约束优先级

本 Skill 没有单一的“官方意图编排规范”。

但本 Skill 依赖以下官方或事实标准：

1. A2A 官方规范  
   用于约束 AgentCard、skills、Task、Message、Artifact、Streaming、错误语义和 Agent 能力发现。

2. JSON Schema 官方规范  
   用于约束 `execution-plan.schema.json`。

3. LLM provider 的 structured output 官方能力  
   如果后续使用 LLM planner，应优先使用 provider 官方支持的结构化输出能力，并仍然在 Orchestrator 层进行 schema validation。

优先级规则：

```text
A2A 官方规范 > 本项目 A2A 落地约定
JSON Schema 官方规范 > 手写非标准校验格式
结构化 ExecutionPlan schema > LLM 自然语言 plan
```

如果项目文档与官方 A2A 最新规范冲突：

```text
官方 A2A 规范 = 长期兼容目标
项目 MVP 约束 = 当前落地范围
```

不得把 MVP 直接路由逻辑描述成长期唯一编排方式。

## 3. 适用场景

当进行以下工作时，启用本 Skill：

- 修改 Orchestrator 路由逻辑。
- 新增或修改 ExecutionPlan schema。
- 新增 `single`、`parallel`、`sequential` 编排策略。
- 新增 @Agent 手动路由。
- 新增 LLM planner。
- 新增 fallback 策略。
- 新增多 Agent 聚合逻辑。
- 校验 `task.agent` 是否存在。
- 校验 `task.skill` 是否存在于目标 AgentCard.skills。
- 修改 Agent Registry 使用方式。
- 修改计划校验、计划执行、计划失败降级逻辑。
- 审查 Orchestrator 是否执行了非结构化 plan。
- 审查 Gateway handler 是否承担了复杂编排职责。

## 4. 长期契约基线

长期架构中，Orchestrator 必须将用户意图转换为结构化 ExecutionPlan。

ExecutionPlan 必须先校验，再执行。

长期 contract 必须支持：

```text
strategy: single | parallel | sequential
```

长期 contract 必须定义：

- ExecutionPlan schema。
- Agent Registry 使用规则。
- AgentCard.skills 校验规则。
- `task.agent` 校验。
- `task.skill` 校验。
- `task.input` 规则。
- `parallel` 聚合规则。
- `sequential` 依赖规则。
- `@Agent` 手动路由优先级。
- fallback 策略。
- 失败降级规则。
- plan trace 规则。
- run / message / conversation 关联规则。

必须防止：

- LLM 编排出不存在的 Agent。
- LLM 编排出不存在的 skill。
- LLM 返回非结构化 plan。
- schema validation 失败后继续执行。
- Orchestrator 直接调用前端 Runtime Skill。
- Gateway handler 承担复杂意图编排。
- 子 Agent 自己决定全局编排策略。

本契约的事实源文件是：

项目 contract 文件位于项目根目录

```text
<repo-root>/docs/contracts/intent-orchestration.md
<repo-root>/docs/contracts/execution-plan.schema.json
```

本 Skill 的辅助参考文件位于当前 Skill 目录：

```text
<current-skill-dir>/references/execution-plan-schema.md
<current-skill-dir>/references/routing-rules.md
<current-skill-dir>/references/planner-validation.md
<current-skill-dir>/references/fallback-policy.md
<current-skill-dir>/references/multi-agent-strategy.md
```

## 5. MVP 约束

MVP 阶段只实现单 Agent 直接路由。

MVP 策略：

```text
strategy = single
routing = direct
plannerLLM = disabled
```

MVP 目标 Agent 来源：

```text
req.AgentName
或 conversation.agentName
```

MVP Agent Registry：

```text
config file
```

MVP 默认目标：

```text
code-agent
```

MVP 默认 skill：

```text
code_generate
```

MVP 阶段不实现：

- LLM planner。
- `parallel`。
- `sequential`。
- fallback。
- @Agent。
- 多 Agent 聚合。
- 动态 Agent Registry。
- 复杂失败降级。
- 自动 Agent 选择。

MVP 阶段如果目标 Agent 不存在：

```text
返回 RUN_ERROR 或等价 run error
```

MVP 阶段如果 A2A 调用失败：

```text
返回 RUN_ERROR 或等价 run error
```

## 6. 阶段演进规则

### 6.1 MVP 阶段

MVP 阶段只允许：

```text
single direct routing
```

MVP 阶段不得把 direct routing 写死成长期唯一策略。

MVP 阶段不得为了“完整智能编排”提前引入 LLM planner。

MVP 阶段不得实现 parallel / sequential / fallback，除非项目负责人明确要求。

MVP 阶段不得让 Gateway handler 承担复杂编排逻辑。

### 6.2 正式开发阶段

MVP 完成后，可以按需求逐步启用更完整的意图编排能力。

新增或修改编排策略前，必须先更新：

```text
docs/contracts/intent-orchestration.md
docs/contracts/execution-plan.schema.json
```

新增 Agent 或 skill 前，必须确保：

- Agent Registry 已更新。
- AgentCard 已更新。
- AgentCard.skills 与实际 handler 能力一致。
- Orchestrator 能校验 task.agent。
- Orchestrator 能校验 task.skill。

启用 LLM planner 前，必须确保：

- planner 输出 JSON。
- planner 输出符合 `execution-plan.schema.json`。
- schema validation 失败时不得执行。
- LLM 不得创造 Agent Registry 中不存在的 Agent。
- LLM 不得创造 AgentCard.skills 中不存在的 skill。
- planner prompt 只使用 Agent Registry / AgentCard 中存在的能力。

### 6.3 多 Agent 阶段

多 Agent 阶段可以逐步支持：

- @Agent 手动路由。
- `single`。
- `parallel`。
- `sequential`。
- `parallel` 聚合。
- `sequential` 依赖。
- fallbackAgent。
- 部分失败降级。
- plan trace。
- run step 追踪。
- 多 Artifact 聚合。
- 多 Agent 回复汇总。

## 7. 本 Skill 负责

本 Skill 负责：

- ExecutionPlan schema 规则。
- `single` / `parallel` / `sequential` 策略规则。
- Agent Registry 使用规则。
- AgentCard.skills 校验规则。
- `task.agent` 校验规则。
- `task.skill` 校验规则。
- @Agent 手动路由优先级。
- LLM planner 结构化输出规则。
- planner validation 规则。
- fallback 策略。
- 多 Agent 聚合规则。
- 失败降级规则。
- MVP direct routing 规则。
- 正式开发阶段编排能力扩展规则。

## 8. 本 Skill 不负责

本 Skill 不负责：

- A2A HTTP endpoint 细节。
- A2A Task envelope 完整定义。
- A2A streaming 事件结构。
- AG-UI 事件名称。
- AG-UI 事件字段结构。
- 前端 Runtime Skill 注册。
- React Component 实现。
- Artifact schema。
- Artifact 存储策略。
- ADK Runtime API。
- 子 Agent handler 实现。
- LLM provider 具体封装。
- 数据库完整 DDL。
- Gateway-Orchestrator 内部 API 细节。
- 通用 Go / TypeScript 代码风格。

涉及以上内容时，只引用相关 contract，不在本 Skill 中重新定义。

## 9. Contract first 规则

任何新增或修改意图编排行为前，必须先更新 contract。

必须优先更新：

```text
docs/contracts/intent-orchestration.md
docs/contracts/execution-plan.schema.json
```

未更新 contract 的实现变更不得接受。

不得先改 Orchestrator 编排实现，再补 contract。

如果修改 Agent / skill 校验，还必须检查：

```text
docs/contracts/a2a-agent-card.md
docs/contracts/a2a-task.md
docs/contracts/adk-runtime.md
docs/contracts/agent-config.md
```

如果修改 Artifact 预期输出，还必须检查：

```text
docs/contracts/artifact-schema.md
docs/contracts/artifact.schema.json
```

## 10. 必须更新的文件

修改意图编排 contract 时，必须优先更新：

```text
docs/contracts/intent-orchestration.md
docs/contracts/execution-plan.schema.json
```

如果涉及 MVP 实现，可能影响：

```text
server/internal/orchestrator/orchestrator.go
server/internal/orchestrator/converter.go
server/internal/a2a/client.go
server/internal/config/config.go
```

如果涉及 Agent Registry，可能影响：

```text
server/internal/config/config.go
docs/contracts/agent-config.md
docs/contracts/a2a-agent-card.md
```

如果涉及 LLM planner，可能影响：

```text
docs/contracts/llm-provider.md
server/internal/orchestrator/planner.go
```

## 11. ExecutionPlan 规则

ExecutionPlan 是 Orchestrator 执行计划的唯一结构化表达。

ExecutionPlan 必须是 JSON。

ExecutionPlan 必须通过 schema validation。

MVP 最小结构：

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

长期结构可包含：

- `planId`
- `runId`
- `conversationId`
- `messageId`
- `traceId`
- `version`
- `strategy`
- `reason`
- `tasks`
- `aggregation`
- `fallback`
- `approvalRequired`
- `createdAt`

详细规则见：

```text
references/execution-plan-schema.md
```

## 12. Agent Registry 规则

`task.agent` 必须来自 Agent Registry。

Agent Registry 可以来自：

- MVP 配置文件。
- AgentCard discovery。
- 数据库。
- 服务注册表。
- 静态配置 + 健康检查。

MVP 阶段使用：

```text
config file
```

正式开发阶段，Agent Registry 必须能提供：

- Agent name。
- Agent URL。
- AgentCard。
- health status。
- supported skills。
- inputModes。
- outputModes。
- enabled / disabled 状态。

不存在的 Agent 不得执行。

## 13. AgentCard.skills 校验规则

`task.skill` 必须来自目标 Agent 的 AgentCard.skills。

规则：

1. 先解析 ExecutionPlan。
2. 校验 `task.agent` 是否存在。
3. 读取目标 Agent 的 AgentCard。
4. 校验 `task.skill` 是否存在于 `AgentCard.skills`。
5. 只有校验通过后，才能调度 A2A Task。

不得允许 LLM planner 自造 skill。

不得允许 Orchestrator 调用 AgentCard 未声明的能力。

## 14. 路由优先级

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

详细规则见：

```text
references/routing-rules.md
```

## 15. 编排策略规则

### 15.1 single

`single` 表示只执行一个目标 Agent 任务。

MVP 阶段只允许：

```text
strategy = single
```

### 15.2 parallel

`parallel` 表示多个任务可以并行执行。

正式开发阶段启用前，必须定义：

- 并发限制。
- 聚合规则。
- 失败处理。
- 超时策略。
- Artifact 合并规则。

### 15.3 sequential

`sequential` 表示任务按依赖顺序执行。

正式开发阶段启用前，必须定义：

- `dependsOn`。
- 依赖失败处理。
- 中间结果传递。
- 取消策略。
- trace 规则。

详细规则见：

```text
references/multi-agent-strategy.md
```

## 16. LLM planner 规则

MVP 阶段不启用 LLM planner。

正式开发阶段如果启用 LLM planner，必须满足：

- 输出必须是 JSON。
- 输出必须符合 `execution-plan.schema.json`。
- schema validation 失败不得执行。
- LLM 不得创造不存在的 Agent。
- LLM 不得创造不存在的 skill。
- LLM 不得绕过 Agent Registry。
- LLM 不得直接返回可执行代码作为 plan。
- planner prompt 必须只包含允许被选择的 Agent 和 skills。
- planner 输出必须可审计。
- planner 输出必须关联 `traceId` 或等价追踪字段。

如果 provider 支持 structured outputs，应优先使用结构化输出。

即使 provider 支持 structured outputs，Orchestrator 仍必须执行本地 schema validation。

详细规则见：

```text
references/planner-validation.md
```

## 17. fallback 规则

MVP 阶段不实现 fallback。

正式开发阶段启用 fallback 前，必须定义：

- fallbackAgent。
- fallbackSkill。
- fallback 条件。
- fallback 最大次数。
- fallback 是否继承上下文。
- fallback 是否继承 partial artifacts。
- fallback 失败后的最终错误。
- fallback trace 规则。

详细规则见：

```text
references/fallback-policy.md
```

## 18. 错误处理规则

意图编排必须安全处理以下错误：

- plan JSON 解析失败。
- schema validation 失败。
- strategy 不支持。
- task.agent 不存在。
- task.skill 不存在。
- AgentCard 缺失。
- Agent disabled。
- Agent health check failed。
- A2A 调用失败。
- task 超时。
- parallel 部分失败。
- sequential 依赖失败。
- fallback 失败。

MVP 阶段简化错误：

```text
target agent not found → RUN_ERROR
A2A call failed → RUN_ERROR
```

错误信息不得包含：

- stack trace。
- API key。
- token。
- system prompt。
- LLM 原始敏感输出。
- 内部服务地址。
- 内部文件路径。

## 19. 禁止事项

Coding Agent 不得：

- 执行非结构化自然语言 plan。
- 在未更新 contract 的情况下新增编排策略。
- 让 LLM planner 直接驱动执行。
- 让 LLM 创造不存在的 Agent。
- 让 LLM 创造不存在的 skill。
- 跳过 `execution-plan.schema.json` 校验。
- 跳过 Agent Registry 校验。
- 跳过 AgentCard.skills 校验。
- 在 MVP 阶段启用 LLM planner。
- 在 MVP 阶段启用 parallel / sequential。
- 在 MVP 阶段启用 fallback。
- 让 Gateway handler 写复杂编排逻辑。
- 让 Orchestrator 直接调用 React Component。
- 让 Orchestrator 直接生成前端 Runtime Skill 参数。
- 把 Artifact schema 写进本 Skill。
- 把 A2A Task 协议写进本 Skill。
- 把 MVP direct routing 规则扩展成长期唯一策略。

## 20. 相关契约

本 Skill 只引用以下契约，不重新定义它们：

- `a2a-agent-contract`：负责 AgentCard、AgentCard.skills、A2A Task、Streaming 和错误语义。
- `adk-runtime-contract`：负责子 Agent Runtime、Task handler、AgentCard 生成和 Runtime API。
- `artifact-contract`：负责 Artifact schema、类型、生命周期、存储和预览映射。
- `gateway-orchestrator-contract`：负责 Gateway 与 Orchestrator 的内部 run 协议。
- `frontend-runtime-skills-contract`：负责前端 Runtime Skill 注册、参数校验、组件映射和 ToolResult。
- `llm-provider-contract`：负责 LLM provider、prompt、结构化输出、重试和降级。
- `security-boundary-contract`：负责鉴权、安全边界、危险操作确认和敏感信息保护。
- `observability-debugging-contract`：负责 traceId、runId、stepId、日志和调试规则。
