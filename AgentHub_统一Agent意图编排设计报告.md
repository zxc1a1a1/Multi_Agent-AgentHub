# AgentHub 统一 Agent 意图编排设计报告

> 面向：产品负责人、技术负责人、后端/Agent 工程师、前端集成工程师、答辩材料整理人员  
> 版本：v1.0  
> 主题：统一 Agent / Orchestrator 如何理解用户意图、编排子 Agent、转换协议并返回可视化产物  
> 结论：AgentHub v1.0 应采用“意图编排 + 状态机 + 契约化子 Agent”的混合编排方案，而不是把路由写死在 Gateway，也不是把所有逻辑压进一个巨型 prompt。

---

## 0. 报告摘要

AgentHub 的核心不是普通聊天，也不是单个代码生成 Agent，而是一个“IM 聊天式多 Agent 协作平台”。用户通过单聊或群聊发送自然语言目标，统一 Agent，也就是 Orchestrator，负责理解用户意图、构建执行计划、选择合适的子 Agent、管理多 Agent 执行过程、聚合结果，并把文本、代码、网页、文档、Diff、部署状态等产物以聊天消息和预览卡片形式返回给前端。

本报告建议 AgentHub v1.0 的统一 Agent 编排采用以下架构：

```text
用户消息
  ↓
Gateway 鉴权 / 会话 / SSE
  ↓
Orchestrator
  ├─ 1. Intent Classifier：识别任务类型、复杂度、是否 @Agent
  ├─ 2. Context Builder：构造结构化上下文，不把全量历史随意塞给所有 Agent
  ├─ 3. Agent Registry：读取 AgentCard、健康状态、capability、outputModes
  ├─ 4. Planner：生成 OrchestrationPlan
  ├─ 5. Plan Validator：校验 plan 是否只使用可用 Agent、合法 strategy、合法 artifact 类型
  ├─ 6. Execution Engine：执行 single / ordered_parallel / sequential / review_loop
  ├─ 7. Protocol Converter：A2A StreamEvent → AG-UI Event
  ├─ 8. Result Aggregator：聚合多 Agent 输出与产物
  └─ 9. Failure Handler：fallback、retry、降级、用户确认
  ↓
A2A 调用子 Agent
  ↓
AG-UI SSE 返回前端
```

这个方案与 AgentHub 当前技术路线一致：前端使用 AG-UI / SSE，Gateway 负责公开 API、鉴权、会话和 SSE 连接，Orchestrator 负责编排，子 Agent 通过 A2A + ADK Runtime 接入，产物通过 Artifact/ToolCall 映射给前端 Skills 渲染。

从赛题角度看，这个方案能同时支撑五个评价方向：

| 评价维度 | 编排方案的支撑点 |
|---|---|
| AI 协作能力 | 使用 Skills / Contract / Plan / Review 文档链路沉淀协作规范 |
| 功能完整度 | 支持单聊、群聊、@Agent、多 Agent 调度、上下文连续 |
| 生成效果质量 | 通过 Artifact → code_preview / web_preview / markdown_render 提升产物展示 |
| 代码理解度 | 可清楚解释 Gateway / Orchestrator / A2A / AG-UI / 子 Agent 边界 |
| 创新与产品感 | 群聊式多 Agent 协作、编排过程可视化、失败降级、产物内联 |

---

## 1. 输入资料与使用方式

本报告同时参考了项目内 5 个文档和外部高质量资料。

### 1.1 项目内 5 个文档映射

| 文档 | 在本报告中的作用 |
|---|---|
| `agenthub-skills-usage-report-v3.md` | 确定 17 个 Skill 的定位、边界和协作方式，尤其是 `project-architecture`、`intent-orchestration-contract`、`a2a-agent-contract`、`agui-event-contract`、`artifact-contract`、`security-boundary-contract`、`testing-review-contract`。 |
| `SPRINT-v1.0-Plan.md` | 确定 v1.0 增量目标：2+ Agent、LLM 意图编排、群聊、AgentCard Registry、健康检查、结构化多轮消息、web_preview、markdown、降级重试、Docker Demo。 |
| `UML-AgentHub系统图.md` | 确定现有系统分层：Frontend → AG-UI → Gateway → Orchestrator → A2A → Child Agents，以及 A2A → AG-UI 协议转换、单聊/群聊时序、消息生命周期。 |
| `AgentHub- 多Agent协作平台设计.pdf` | 对齐赛题要求：IM 聊天式交互、单聊、群聊协作、上下文连续、产物内联、多 Agent 接入、可运行 Demo、AI 协作开发记录。 |
| `多Agent聊天式工作台_子Agent设计与开发建议报告.md` | 提供子 Agent 分类、职责划分、上下文隔离、工具权限、Diff/Review/Test 闭环、Orchestrator 与子 Agent 的关系。 |

### 1.2 外部资料映射

| 外部资料 / 项目 | 对 AgentHub 的启发 |
|---|---|
| OpenAI Agents SDK | 编排可分为“LLM 决策”和“代码编排”，二者可以混合使用；handoff 和 agents-as-tools 适合不同场景。 |
| LangChain / LangGraph Multi-agent | 支持 supervisor / subagents / handoffs / router 等模式，强调主控 Agent 统一路由和上下文工程。 |
| Anthropic Building Effective Agents | 给出 prompt chaining、routing、parallelization、orchestrator-workers、evaluator-optimizer 等经典工作流模式。 |
| Google A2A / a2aproject | 说明跨框架 Agent 需要通用通信协议、AgentCard、任务和流式事件。 |
| MCP | 说明外部工具、文件、数据库、搜索、CI/CD 等能力应通过标准化工具协议/网关接入，不应让任意 Agent 直接访问全部系统。 |
| Microsoft Agent Framework / CrewAI | 强调生产级多 Agent 系统需要状态、工作流、遥测、guardrails、人工审批和可观测性。 |

---

## 2. 为什么 AgentHub 需要“意图编排”

### 2.1 仅靠硬编码路由的问题

MVP 阶段可以把所有请求都路由到 `code-agent`。但 v1.0 目标已经变成：

```text
2+ Agent
单聊 + 群聊
LLM 意图编排
AgentCard Registry
web_preview / markdown_render
parallel / sequential
失败降级
```

如果继续硬编码：

```go
if agentName == "code-agent" {
    callCodeAgent()
} else if agentName == "web-agent" {
    callWebAgent()
}
```

会出现几个问题：

1. **不能处理复杂任务**：如“做一个计数器应用，要前端页面和 Go 后端 API”，需要拆给 web-agent 与 code-agent。
2. **不能适应新增 Agent**：新增 doc-agent、review-agent、deploy-agent 后，每次都要改 Orchestrator 代码。
3. **不能解释编排过程**：答辩时难以说明为什么选这个 Agent、为什么并行或串行。
4. **不利于群聊体验**：群聊里用户可能 @Agent，也可能不 @，需要自动分派。
5. **没有 fallback**：目标 Agent 挂了就失败，无法基于 capability 找替代 Agent。

### 2.2 仅靠 LLM 自由决策的问题

如果把所有逻辑都塞进一个 Planner Prompt：

```text
你是 Orchestrator，请决定调用哪些 Agent。
```

也会有风险：

1. **Plan 不稳定**：LLM 可能返回非法 JSON。
2. **Agent 幻觉**：LLM 可能选择不存在的 agentName。
3. **越权调用**：LLM 可能要求 Agent 执行没有权限的操作。
4. **难测试**：每次输出都可能不同，单测和回归困难。
5. **难调试**：无法清楚定位是 intent 分类错、AgentCard 错、fallback 错，还是协议转换错。

### 2.3 推荐方案：混合意图编排

AgentHub 应采用：

```text
确定性规则 + LLM Planner + 本地 Schema 校验 + 状态机执行 + fallback
```

也就是说：

- 简单明确的任务，用规则直接路由。
- 显式 `@Agent`，优先按 mention 路由。
- 多意图/复杂任务，调用 LLM Planner 生成结构化 plan。
- LLM 输出必须本地 JSON Schema 校验。
- plan 中的 agent 必须来自 Registry。
- plan 中的 capability 必须来自 AgentCard。
- plan 中的 artifact/outputType 必须能被前端 Skills 支持。
- LLM 失败、超时或非法 JSON 时走 fallback。

这正好对应 OpenAI Agents SDK 中“LLM 决策”和“代码编排可以混合使用”的思路，也对应 Anthropic 的 routing / orchestrator-workers / evaluator-optimizer 组合模式。

---

## 3. 统一 Agent / Orchestrator 的定位

### 3.1 统一 Agent 不是普通子 Agent

AgentHub 中“统一 Agent”更准确地说是 Orchestrator 服务。它不是 code-agent、web-agent、doc-agent 的同级子 Agent，而是系统控制面。

它负责：

- 接收 Gateway 转来的内部请求。
- 读取会话历史和用户当前消息。
- 识别用户意图。
- 获取 Registry 中可用 Agent。
- 构建 OrchestrationPlan。
- 校验和执行计划。
- 将任务分派给子 Agent。
- 处理 A2A 流式返回。
- 转换成 AG-UI 事件。
- 聚合多 Agent 输出。
- 处理错误、超时、取消、fallback。
- 记录 trace / run / task / artifact。

### 3.2 Gateway 与 Orchestrator 必须分工清楚

推荐职责边界：

| 组件 | 负责 | 不负责 |
|---|---|---|
| Frontend | IM UI、SSE 解析、前端 Skills 渲染、消息状态 | 不直接访问 Orchestrator，不直接访问 Child Agent |
| Gateway | 鉴权、公开 REST/AG-UI API、SSE 连接、会话持久化、错误脱敏 | 不做 LLM 编排，不直接调用 LLM Provider，不直接写死子 Agent 路由 |
| Orchestrator | 意图编排、计划生成、子 Agent 调度、协议转换、聚合、fallback | 不暴露给前端，不保存真实密钥，不承担 UI 逻辑 |
| Child Agent | 执行专业任务，按 A2A 输出文本和 artifacts | 不决定全局计划，不越权调用其他 Agent |
| Data Layer | Conversation / Message / Run / Agent / Artifact 持久化 | 不承担编排判断 |

### 3.3 为什么要坚持这个边界

1. **可扩展**：后续可以把 Orchestrator 从 Gateway 进程拆出去。
2. **可测试**：Gateway 测 API，Orchestrator 测 plan 和协议转换。
3. **可观测**：每次 run 都能看到 Gateway→Orchestrator→Agent 的 trace。
4. **可安全控制**：外部用户 token 不透传到 Agent；内部 service token 单独管理。
5. **答辩好解释**：系统不是“一个 handler 里写死调用”，而是分层协议架构。

---

## 4. 统一 Agent 内部组件设计

### 4.1 组件总览

```text
Orchestrator Service
├── Request Normalizer
├── Intent Classifier
├── Mention Parser
├── Context Builder
├── Agent Registry Client
├── Planner LLM Adapter
├── Plan Validator
├── Execution Engine
├── A2A Dispatcher
├── Protocol Converter
├── Result Aggregator
├── Failure / Retry Handler
├── Safety Gate
└── Trace Logger
```

### 4.2 Request Normalizer

负责把 Gateway 传入的请求转换成稳定内部格式。

输入：

```json
{
  "runId": "run_xxx",
  "threadId": "conversation_uuid",
  "messages": [],
  "tools": [],
  "conversationType": "single_or_group",
  "traceId": "trace_xxx"
}
```

输出：

```json
{
  "conversationId": "uuid",
  "runId": "run_xxx",
  "latestUserMessage": "用户当前输入",
  "frontendSkills": ["code_preview", "web_preview", "markdown_render"],
  "history": [
    {"role": "user", "content": "..."},
    {"role": "assistant", "content": "..."}
  ],
  "mode": "single"
}
```

规则：

- 校验 conversationId。
- 只读取必要历史，不把所有消息无脑塞给 Planner。
- 提取前端 tools/skills，供 Artifact 映射使用。
- 生成 traceId，贯穿后续步骤。

### 4.3 Mention Parser

负责识别用户是否显式指定 Agent。

支持形式：

```text
@code-agent 帮我改这个 Go 接口
@web-agent 生成一个登录页
@doc-agent 总结成 README
```

输出：

```json
{
  "mentionedAgents": ["web-agent"],
  "cleanedContent": "生成一个登录页"
}
```

规则：

- 如果 mention 的 agent 不存在或不健康，返回可恢复错误。
- 如果是单聊模式且 conversation 绑定 agent，默认优先 conversation agent。
- 如果是群聊模式且用户显式 @，优先 @。
- 如果没有 @，进入意图编排。

### 4.4 Intent Classifier

负责先做轻量分类，减少不必要的 LLM Planner 调用。

建议 intent 类型：

| intent | 示例 | 默认候选 Agent |
|---|---|---|
| `code_generation` | 写一个 Go HTTP server | code-agent |
| `code_review` | 帮我检查这段代码 | code-agent / review-agent |
| `web_ui_generation` | 写一个登录页面 | web-agent |
| `document_generation` | 整理成 README | doc-agent |
| `mixed_app_generation` | 做一个待办应用，前端+后端 | web-agent + code-agent |
| `debugging` | 这段报错怎么修 | code-agent / test-agent |
| `deployment` | 帮我部署 | deploy-agent |
| `unknown` | 模糊请求 | Planner 决策 |

实现策略：

```text
规则关键词
  ↓
低成本 LLM classifier
  ↓
Planner
```

### 4.5 Context Builder

Context Builder 负责构造面向 Planner 和子 Agent 的上下文包。

原则：

- 最新用户指令优先。
- 显式 @Agent 指令优先。
- Pin 消息按任务选择注入，不等于全量注入。
- Code Agent 需要目标文件、需求、约束。
- Review Agent 需要原始需求、diff、测试结果。
- Deploy Agent 需要 artifact、构建命令、目标环境。
- Planner 需要摘要，不需要所有代码正文。

Context Bundle 示例：

```json
{
  "contextBundleId": "ctx_123",
  "target": "planner",
  "latestInstruction": "做一个待办清单应用，前端+后端",
  "conversationSummary": "用户正在开发 AgentHub 示例项目",
  "requirements": [
    "需要 React 前端",
    "需要 Go 后端 API",
    "需要代码预览和网页预览"
  ],
  "constraints": [
    "不要修改 .env",
    "优先输出可预览 artifact"
  ],
  "relevantMessages": [],
  "pinnedContext": [],
  "excludedContextNotes": [
    "旧的无关聊天未注入"
  ]
}
```

### 4.6 Agent Registry Client

Registry 是 Orchestrator 的能力事实源。

每个 Agent 至少暴露：

```text
/.well-known/agent.json
/health
```

AgentCard 建议字段：

```json
{
  "name": "web-agent",
  "displayName": "Web Agent",
  "description": "生成网页、UI 原型和可预览 HTML",
  "version": "1.0.0",
  "url": "http://web-agent:8082",
  "skills": [
    {
      "id": "web_generation",
      "name": "网页生成",
      "description": "生成 HTML/CSS/JS 或 React UI 原型"
    }
  ],
  "inputModes": ["text"],
  "outputModes": ["text", "webpage", "code"],
  "capabilities": {
    "streaming": true,
    "artifactTypes": ["webpage", "code"],
    "supportsContext": true
  }
}
```

Registry 需要维护：

- AgentCard 缓存
- health 状态
- lastCheck
- capability index
- outputType index
- agent URL
- 是否可见
- 是否允许当前用户/会话调用

### 4.7 Planner LLM Adapter

Planner 只负责生成结构化 plan，不直接执行任务。

Planner Input：

```json
{
  "conversationId": "uuid",
  "runId": "run_123",
  "mode": "group",
  "latestUserMessage": "做一个待办应用，要前端页面和 Go 后端 API",
  "mentionedAgents": [],
  "frontendSkills": ["code_preview", "web_preview", "markdown_render"],
  "availableAgents": [
    {
      "name": "code-agent",
      "skills": ["code_generation", "code_review"],
      "outputModes": ["text", "code"]
    },
    {
      "name": "web-agent",
      "skills": ["web_generation", "ui_design"],
      "outputModes": ["text", "webpage", "code"]
    }
  ],
  "contextSummary": "当前是一个新任务，无已绑定代码库。"
}
```

Planner Output：

```json
{
  "intent": "生成一个包含前端页面和后端 API 的待办清单应用",
  "strategy": "ordered_parallel",
  "tasks": [
    {
      "id": "task_web",
      "agentName": "web-agent",
      "capabilityId": "web_generation",
      "taskContent": "生成待办清单应用的响应式前端页面，输出可预览 HTML。",
      "expectedArtifacts": ["webpage"],
      "dependsOn": [],
      "priority": 1
    },
    {
      "id": "task_api",
      "agentName": "code-agent",
      "capabilityId": "code_generation",
      "taskContent": "生成 Go 后端 API，包括任务增删改查接口。",
      "expectedArtifacts": ["code"],
      "dependsOn": [],
      "priority": 2
    }
  ],
  "userVisibleSummary": "我会把任务拆成前端页面和后端 API 两部分，分别交给 Web Agent 和 Code Agent。",
  "confidence": 0.86
}
```

### 4.8 Plan Validator

Plan Validator 是防止 LLM 幻觉和越权的关键。

校验项：

| 校验项 | 规则 |
|---|---|
| JSON schema | 必须符合 OrchestrationPlan schema |
| agentName | 必须来自 Registry |
| healthy | 默认只调健康 Agent |
| capabilityId | 必须属于 AgentCard |
| strategy | 只允许 `single`、`ordered_parallel`、`sequential`、`review_loop` |
| expectedArtifacts | 必须能映射到前端已注册 skill 或可降级为 text |
| taskContent | 不得包含密钥、不得要求修改 `.env` |
| dependency | dependsOn 必须引用已存在 task |
| max tasks | MVP 建议最多 3 个 task，防止过度拆分 |
| timeout | 每个 task 必须有 timeout |
| user approval | 高危操作必须要求确认 |

非法时处理：

```text
Plan invalid
  → 尝试修复一次
  → 仍失败则 fallbackPlan
  → fallback 仍失败则 RUN_ERROR + safe error
```

### 4.9 Execution Engine

执行策略：

| strategy | 含义 | v1.0 建议 |
|---|---|---|
| `single` | 只调用一个 Agent | 必做 |
| `ordered_parallel` | 逻辑上并行，UI 上按顺序展示多个 Agent 输出 | 必做 |
| `sequential` | 后一个任务依赖前一个结果 | 建议做简化版 |
| `review_loop` | Code → Review/Test → Code 修复 | 后续增强 |
| `handoff` | 一个 Agent 接管下一轮对话 | 暂不作为主路径 |

为什么建议 `ordered_parallel`：

真实 goroutine 并行会导致多个 Agent 的流式 token 交错，前端消息合并和持久化难度较高。比赛 v1.0 可以采用“并行计划、顺序展示”的折中方式：

```text
Planner 认为任务可并行
  ↓
Orchestrator 生成多个 task
  ↓
UI 展示为多 Agent 协作
  ↓
后端按顺序或受控并发执行
  ↓
每个 Agent 仍有独立 messageId / senderName / artifacts
```

这样既满足“多 Agent 协作”展示，又降低事件流混乱风险。

### 4.10 A2A Dispatcher

A2A Dispatcher 负责调用子 Agent。

输入：

```json
{
  "agentURL": "http://web-agent:8082",
  "taskId": "task_web",
  "messages": [
    {"role": "user", "content": "生成待办清单应用的响应式前端页面"}
  ],
  "contextBundleId": "ctx_web",
  "timeoutMs": 120000
}
```

输出：

```text
AsyncIterable<A2AStreamEvent>
```

事件类型：

```text
status: submitted / working / completed / failed
text: 流式文本
artifact: code / webpage / document / file
error: safe error
```

必须支持：

- timeout
- cancellation
- client reuse
- error normalization
- traceId
- agentTaskId

### 4.11 Protocol Converter

Protocol Converter 负责：

```text
A2A Event → AG-UI Event
```

映射规则：

| A2A | AG-UI |
|---|---|
| status=working | TEXT_MESSAGE_START |
| text chunk | TEXT_MESSAGE_CONTENT |
| status=completed | TEXT_MESSAGE_END |
| artifact type=code | TOOL_CALL code_preview |
| artifact type=webpage | TOOL_CALL web_preview |
| artifact type=document | TOOL_CALL markdown_render |
| status/error | RUN_ERROR 或 STATE_UPDATE |
| task progress | STATE_UPDATE |

需要注意：

- artifact 不应在 text 过程中立即输出，建议缓存到 completed 后 flush。
- 每个 Agent 输出必须有独立 messageId。
- 每个 TOOL_CALL 必须有 toolCallId。
- 如果前端没有注册某个 skill，应降级为文本或普通 artifact 链接。
- 错误必须脱敏。

### 4.12 Result Aggregator

Aggregator 负责把多 Agent 输出汇总成用户可理解的结果。

例如用户要求：

```text
做一个计数器应用，要一个好看的前端页面和 Go 后端 API
```

执行结果：

```text
web-agent → webpage artifact: index.html
code-agent → code artifact: main.go
```

Aggregator 可以生成最后一条系统/Orchestrator 总结：

```text
任务已完成：

1. Web Agent 生成了可预览的计数器页面。
2. Code Agent 生成了 Go 后端 API。
3. 两个产物已分别以网页预览和代码预览卡片展示。
```

注意：

- 不要让 Aggregator 吞掉子 Agent 的原始产物。
- 聚合总结要短，重点在“做了什么”和“下一步可做什么”。
- 如果部分 Agent 失败，要明确说明部分成功/失败，不假装全部成功。

### 4.13 Failure / Retry Handler

失败类型：

| 类型 | 示例 | 处理 |
|---|---|---|
| Planner timeout | LLM 无响应 | fallback keyword router |
| Planner invalid JSON | 非法输出 | 修复一次 / fallback |
| No healthy agents | Registry 为空 | RUN_ERROR |
| Agent unavailable | 子 Agent health false | 找同 capability 替代 Agent |
| A2A error | 流断开 | retry once + safe error |
| Artifact unsupported | 前端无 skill | 降级文本 |
| Tool risk high | 部署/写文件/删文件 | human approval |

用户可见错误文案应该是：

```text
Web Agent 暂时不可用，我已尝试切换到可用 Agent，但没有找到支持 web_generation 的替代 Agent。
```

而不是：

```text
dial tcp 172.18.0.5:8082: connect: connection refused
```

---

## 5. 编排模式设计

### 5.1 单聊模式

单聊模式下用户通常选择某个 Agent：

```text
用户 ↔ code-agent
```

但底层仍建议经过 Orchestrator：

```text
Frontend → Gateway → Orchestrator → code-agent
```

原因：

- 统一鉴权和会话。
- 统一上下文打包。
- 统一 A2A→AG-UI 转换。
- 统一 artifact 映射。
- 统一 fallback 和 trace。
- 后续可以无缝加入 Review/Test/Artifact 流程。

单聊计划示例：

```json
{
  "intent": "生成 Go HTTP server",
  "strategy": "single",
  "tasks": [
    {
      "id": "task_code",
      "agentName": "code-agent",
      "capabilityId": "code_generation",
      "taskContent": "用 Go 写一个带日志中间件的 HTTP server。",
      "expectedArtifacts": ["code"]
    }
  ]
}
```

### 5.2 群聊自动编排模式

群聊中用户不指定 Agent：

```text
帮我做一个登录页面，并提供后端登录接口。
```

Orchestrator 自动拆分：

```json
{
  "intent": "生成登录页面和后端接口",
  "strategy": "ordered_parallel",
  "tasks": [
    {
      "id": "task_web",
      "agentName": "web-agent",
      "capabilityId": "web_generation",
      "taskContent": "生成登录页面，包含响应式布局和基础交互。",
      "expectedArtifacts": ["webpage"]
    },
    {
      "id": "task_code",
      "agentName": "code-agent",
      "capabilityId": "code_generation",
      "taskContent": "生成 Go 登录接口，包括请求结构、响应结构和错误处理。",
      "expectedArtifacts": ["code"]
    }
  ]
}
```

前端展示：

```text
Orchestrator：我已将任务拆为 2 个子任务：页面生成、后端接口。
Web Agent：生成登录页面...
[WebPreview 卡片]
Code Agent：生成 Go 登录接口...
[CodePreview 卡片]
Orchestrator：任务完成。
```

### 5.3 @Agent 指定模式

用户输入：

```text
@web-agent 帮我把刚才的页面改成深色主题
```

处理规则：

1. Mention Parser 识别 `web-agent`。
2. Registry 校验存在且健康。
3. 构造 single plan。
4. 只调用 web-agent。
5. 如果 web-agent 不健康，提示不可用或给出替代建议。

### 5.4 review_loop 模式

适用于代码修改闭环：

```text
Code Agent → Review Agent → Test Agent → Code Agent 修复 → Artifact Agent
```

简化 plan：

```json
{
  "intent": "实现并验证消息组件改动",
  "strategy": "review_loop",
  "tasks": [
    {"id": "explore", "agentName": "explorer-agent", "capabilityId": "codebase_explore"},
    {"id": "code", "agentName": "code-agent", "capabilityId": "code_generation", "dependsOn": ["explore"]},
    {"id": "review", "agentName": "review-agent", "capabilityId": "code_review", "dependsOn": ["code"]},
    {"id": "test", "agentName": "test-agent", "capabilityId": "test_run", "dependsOn": ["code"]},
    {"id": "artifact", "agentName": "artifact-agent", "capabilityId": "artifact_manifest", "dependsOn": ["review", "test"]}
  ]
}
```

v1.0 竞赛阶段不一定全部实现，但架构上应预留。

---

## 6. OrchestrationPlan 契约建议

### 6.1 TypeScript 定义

```ts
type PlanningMode = "auto" | "mention" | "single" | "group";

type OrchestrationStrategy =
  | "single"
  | "ordered_parallel"
  | "sequential"
  | "review_loop";

type PlannerInput = {
  traceId: string;
  runId: string;
  conversationId: string;
  planningMode: PlanningMode;
  conversationType: "single" | "group";
  latestUserMessage: string;
  mentionedAgents: string[];
  frontendSkills: string[];
  contextSummary: string;
  availableAgents: AgentCapabilitySet[];
};

type AgentCapabilitySet = {
  agentName: string;
  displayName: string;
  description: string;
  healthy: boolean;
  skills: {
    id: string;
    name: string;
    description: string;
    tags?: string[];
  }[];
  inputModes: string[];
  outputModes: string[];
  artifactTypes: string[];
};

type OrchestrationPlan = {
  planId: string;
  intent: string;
  strategy: OrchestrationStrategy;
  tasks: TaskPlan[];
  userVisibleSummary: string;
  confidence: number;
  fallbackUsed?: boolean;
};

type TaskPlan = {
  id: string;
  agentName: string;
  capabilityId: string;
  taskContent: string;
  expectedArtifacts: string[];
  dependsOn: string[];
  priority: number;
  timeoutMs: number;
  riskLevel: "low" | "medium" | "high";
};
```

### 6.2 JSON Schema 重点

必须强制：

- `strategy` enum。
- `tasks` 最少 1 个，最多建议 3 个。
- `agentName` 字符串，但后续由 Validator 校验是否存在。
- `capabilityId` 必填。
- `riskLevel` 必填。
- `expectedArtifacts` 必须为数组。
- `timeoutMs` 必须在安全范围内，例如 5 秒到 180 秒。

### 6.3 PlanTrace

为了答辩和 debug，建议每次 plan 保存 trace：

```json
{
  "traceId": "trace_001",
  "runId": "run_001",
  "plannerMode": "llm",
  "inputHash": "sha256...",
  "availableAgents": ["code-agent", "web-agent"],
  "rawPlannerOutput": "{...}",
  "validatedPlan": "{...}",
  "fallbackReason": null,
  "createdAt": "2026-05-25T12:00:00Z"
}
```

注意：`rawPlannerOutput` 不能包含真实密钥或过长用户私密文件内容，必要时只保存 hash 或脱敏摘要。

---

## 7. 数据模型建议

### 7.1 runs

记录一次用户请求的整体执行。

```sql
CREATE TABLE runs (
  id CHAR(36) PRIMARY KEY,
  conversation_id CHAR(36) NOT NULL,
  trace_id VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL,
  strategy VARCHAR(32),
  intent TEXT,
  planner_mode VARCHAR(32),
  error_code VARCHAR(64),
  safe_error TEXT,
  started_at DATETIME NOT NULL,
  completed_at DATETIME NULL
);
```

### 7.2 run_steps

记录每个子任务。

```sql
CREATE TABLE run_steps (
  id CHAR(36) PRIMARY KEY,
  run_id CHAR(36) NOT NULL,
  task_id VARCHAR(64) NOT NULL,
  agent_name VARCHAR(128) NOT NULL,
  capability_id VARCHAR(128),
  status VARCHAR(32) NOT NULL,
  input_summary TEXT,
  output_summary TEXT,
  started_at DATETIME NULL,
  completed_at DATETIME NULL,
  error_code VARCHAR(64),
  safe_error TEXT
);
```

### 7.3 agents

记录 Registry 可见 Agent。

```sql
CREATE TABLE agents (
  name VARCHAR(128) PRIMARY KEY,
  display_name VARCHAR(128),
  description TEXT,
  url TEXT NOT NULL,
  version VARCHAR(64),
  healthy BOOLEAN NOT NULL DEFAULT FALSE,
  last_check_at DATETIME NULL,
  agent_card_json JSON,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);
```

### 7.4 artifacts

记录产物。

```sql
CREATE TABLE artifacts (
  id CHAR(36) PRIMARY KEY,
  run_id CHAR(36) NOT NULL,
  message_id CHAR(36),
  agent_name VARCHAR(128),
  type VARCHAR(64) NOT NULL,
  title VARCHAR(255),
  content MEDIUMTEXT NULL,
  content_ref TEXT NULL,
  metadata JSON,
  created_at DATETIME NOT NULL
);
```

### 7.5 context_bundles

记录上下文包。

```sql
CREATE TABLE context_bundles (
  id CHAR(36) PRIMARY KEY,
  run_id CHAR(36) NOT NULL,
  target_agent_name VARCHAR(128),
  summary TEXT,
  requirements_json JSON,
  constraints_json JSON,
  relevant_messages_json JSON,
  relevant_artifacts_json JSON,
  created_at DATETIME NOT NULL
);
```

---

## 8. 前端交互与编排可视化

### 8.1 STATE_UPDATE 事件

Orchestrator 应向前端发送编排状态：

```json
{
  "type": "STATE_UPDATE",
  "runId": "run_123",
  "state": {
    "phase": "planning",
    "message": "正在分析任务并选择合适的 Agent"
  }
}
```

```json
{
  "type": "STATE_UPDATE",
  "runId": "run_123",
  "state": {
    "phase": "dispatching",
    "activeAgent": "web-agent",
    "message": "已分派给 Web Agent 生成页面"
  }
}
```

```json
{
  "type": "STATE_UPDATE",
  "runId": "run_123",
  "state": {
    "phase": "retrying",
    "failedAgent": "web-agent",
    "fallbackAgent": "code-agent",
    "message": "Web Agent 暂不可用，正在切换到 Code Agent"
  }
}
```

### 8.2 用户看到什么

不要把全部内部日志刷进聊天流。建议分层展示：

| 信息 | 默认展示 | 可展开详情 |
|---|---:|---:|
| 正在编排 | 是 | 是 |
| 选择了哪些 Agent | 是 | 是 |
| 每个 Agent 的完整 trace | 否 | 是 |
| A2A 原始事件 | 否 | debug 模式 |
| LLM 原始 plan | 否 | 管理/开发模式 |
| 安全拦截 | 是 | 是 |
| fallback 说明 | 是 | 是 |

### 8.3 消息归属

每个 Agent 回复必须有：

```text
messageId
senderType = "agent"
senderName = "web-agent"
runId
agentTaskId
```

否则前端无法稳定展示多 Agent 群聊。

---

## 9. 安全边界

### 9.1 密钥与 token

- `.env` 不进入 Git。
- 前端不接触真实 LLM key。
- 子 Agent 不接收用户 token。
- A2A 内部鉴权使用 service token 或内网信任 + gateway 策略。
- AgentCard 不允许包含 secret、system prompt、内部敏感 URL。

### 9.2 Planner 安全

- Planner 只能输出 plan，不能直接触发工具。
- Planner 输出必须 schema validation。
- Planner 不能选择不存在 Agent。
- Planner 不能越权要求部署、删文件、修改 `.env`。
- 高风险 task 必须进入 approval。

### 9.3 Artifact 安全

- `web_preview` 必须 sandbox。
- HTML 不得默认获得同源敏感权限。
- Markdown 渲染默认禁用危险 HTML。
- 大产物走 contentRef，不要全部塞进 message.content。
- 文件下载需校验 content type 和 size。

### 9.4 Tool 安全

所有工具调用都应有：

```json
{
  "toolCallId": "tool_123",
  "agentName": "deploy-agent",
  "riskLevel": "high",
  "requiresApproval": true,
  "argsHash": "sha256...",
  "status": "blocked"
}
```

---

## 10. 可观测性与调试

每次 run 需要贯穿以下 ID：

```text
requestId
traceId
runId
conversationId
messageId
agentTaskId
toolCallId
artifactId
llmRequestId
```

日志建议字段：

```json
{
  "level": "info",
  "component": "orchestrator",
  "traceId": "trace_123",
  "runId": "run_123",
  "phase": "planning",
  "plannerMode": "llm",
  "availableAgents": ["code-agent", "web-agent"],
  "strategy": "ordered_parallel",
  "durationMs": 432
}
```

排障路径：

| 现象 | 优先看 |
|---|---|
| 前端没流式输出 | AG-UI client、SSE、Gateway flush |
| Agent 没被调用 | Orchestrator plan、Registry health |
| Planner 乱选 Agent | AgentCard、prompt、Plan Validator |
| web_preview 不显示 | Artifact type、converter、frontendSkills |
| 多 Agent 消息混在一起 | messageId / senderName / STATE_UPDATE |
| 测试环境跑不起来 | docker-compose、healthcheck、smoke-test |

---

## 11. 测试矩阵

### 11.1 单元测试

| 模块 | 测试 |
|---|---|
| Mention Parser | `@web-agent`、非法 @、多个 @ |
| Intent Classifier | code/web/doc/mixed/unknown |
| Planner Parser | 合法 JSON、markdown 包裹 JSON、非法 JSON |
| Plan Validator | 不存在 Agent、非法 capability、非法 strategy |
| Registry | AgentCard parse、health false、URL missing |
| Converter | code → code_preview、webpage → web_preview、document → markdown_render |
| Failure Handler | Planner 超时、Agent 失败、fallback 成功/失败 |

### 11.2 Contract 测试

- AgentCard schema。
- OrchestrationPlan schema。
- A2A StreamEvent schema。
- AG-UI Event schema。
- ArtifactDraft schema。
- REST `/api/agents` response schema。

### 11.3 集成测试

- 单聊 code-agent。
- 单聊 web-agent。
- 群聊 mixed task。
- @Agent 直接路由。
- web-agent down → fallback。
- Planner invalid JSON → fallback。
- 前端无 web_preview skill → 降级。
- Docker compose 全链路。

### 11.4 E2E / Demo 测试

Demo 必跑：

```text
1. 新建 code-agent 单聊 → 生成 Go 代码 → code_preview
2. 新建 web-agent 单聊 → 生成登录页 → web_preview
3. 新建群聊 → “前端页面 + 后端 API” → 两个 Agent 回复
4. @web-agent 修改页面风格
5. 刷新页面后历史消息仍在
```

---

## 12. 分阶段实现建议

### 12.1 第一阶段：v1.0 最小可演示

目标：把赛题最核心的“多 Agent 调度 + IM 体验 + 产物预览”跑通。

实现：

- Registry 读取 code-agent、web-agent。
- Planner 支持 single / ordered_parallel。
- Mention Parser 支持 @Agent。
- Orchestrator 调用多个 Agent。
- A2A → AG-UI 转换支持 code/webpage。
- 前端展示多 Agent 头像、senderName、web_preview。
- Planner 失败 fallback 到关键词规则。

### 12.2 第二阶段：质量闭环

目标：让代码类任务形成闭环。

实现：

- Review Agent。
- Test Agent。
- Diff artifact。
- review_loop strategy。
- run_steps / trace。
- 错误降级 UI。
- smoke-test 覆盖多 Agent。

### 12.3 第三阶段：自建 Agent 与高级产物

目标：体现创新和产品感。

实现：

- Custom Agent Builder。
- Document Agent。
- Deploy Agent。
- Version Agent。
- Artifact history。
- 代码二次编辑和局部修改。

---

## 13. 与五个项目文档的对应关系

| 报告章节 | 对应文档 |
|---|---|
| 统一 Agent 定位、Gateway/Orchestrator 边界 | `agenthub-skills-usage-report-v3.md`、`UML-AgentHub系统图.md` |
| v1.0 目标与实现阶段 | `SPRINT-v1.0-Plan.md` |
| IM、群聊、产物内联、评分点 | `AgentHub- 多Agent协作平台设计.pdf` |
| 子 Agent 编排与职责边界 | `多Agent聊天式工作台_子Agent设计与开发建议报告.md` |
| A2A→AG-UI 协议转换 | `UML-AgentHub系统图.md`、`agenthub-skills-usage-report-v3.md` |
| 安全、测试、交付 | `agenthub-skills-usage-report-v3.md`、`SPRINT-v1.0-Plan.md` |

---

## 14. 外部参考资料

1. OpenAI Agents SDK - Agent orchestration  
   https://openai.github.io/openai-agents-python/multi_agent/

2. OpenAI Agents SDK - Guardrails  
   https://openai.github.io/openai-agents-python/guardrails/

3. OpenAI API Docs - Agents  
   https://developers.openai.com/api/docs/guides/agents

4. LangChain Docs - Multi-agent  
   https://docs.langchain.com/oss/python/langchain/multi-agent

5. LangChain Docs - Subagents / Supervisor Pattern  
   https://docs.langchain.com/oss/python/langchain/multi-agent/subagents-personal-assistant

6. LangGraph Supervisor Reference  
   https://reference.langchain.com/python/langgraph-supervisor

7. Anthropic - Building Effective Agents  
   https://www.anthropic.com/research/building-effective-agents

8. Google Developers Blog - Announcing Agent2Agent Protocol  
   https://developers.googleblog.com/en/a2a-a-new-era-of-agent-interoperability/

9. a2aproject/A2A GitHub  
   https://github.com/a2aproject/A2A

10. Model Context Protocol Introduction  
    https://modelcontextprotocol.io/docs/getting-started/intro

11. Anthropic - Introducing the Model Context Protocol  
    https://www.anthropic.com/news/model-context-protocol

12. Microsoft Agent Framework Overview  
    https://learn.microsoft.com/en-us/agent-framework/overview/

13. Microsoft Agent Framework Workflows  
    https://learn.microsoft.com/en-us/agent-framework/workflows/

14. CrewAI Introduction  
    https://docs.crewai.com/en/introduction

15. CrewAI Crews  
    https://docs.crewai.com/en/concepts/crews

---

## 15. 最终建议

AgentHub 的统一 Agent 不应该被实现成一个“会调用 LLM 的 Gateway handler”，也不应该被实现成一个“万能 prompt”。正确方向是：

```text
Orchestrator = 状态机 + LLM Planner + Agent Registry + Plan Validator + A2A Dispatcher + AG-UI Converter + Result Aggregator + Safety/Fallback
```

v1.0 推荐先交付：

```text
code-agent + web-agent + LLM 意图编排 + @Agent + ordered_parallel + web_preview + code_preview + markdown_render + fallback
```

后续再增强：

```text
review_loop + diff + test + deploy + version + custom agent
```

答辩时可以用一句话解释：

> AgentHub 的 Orchestrator 先通过意图编排把用户自然语言目标转换成结构化 OrchestrationPlan，再基于 AgentCard Registry 选择健康且具备对应能力的子 Agent，通过 A2A 执行任务，并把子 Agent 的流式输出与 Artifact 转换成 AG-UI 事件返回前端，从而实现 IM 聊天式的多 Agent 协作。
