---
name: project-architecture
description: "用于说明 AgentHub 的整体项目架构，包括 Frontend、Gateway、Orchestrator、Child Agent、ADK Runtime 和 Data Layer 的模块边界、MVP 范围以及后续演进边界。"
---

# project-architecture

## 1. Skill 目的

本 Skill 用于定义 AgentHub 项目的总体架构、服务边界、协议职责和开发约束。

AgentHub 是一个以 IM 聊天为核心交互范式的多 Agent 协作平台，整体链路必须遵守：

```text
React Frontend
  → AG-UI
Gateway Service
  → 内部通信 Contract
Orchestrator Service
  → A2A
Child Agents
  → Artifact
Frontend Runtime Skills
```

本 Skill 的目标不是直接生成具体业务代码，而是约束所有开发任务都不能偏离项目架构。

使用本 Skill 时，Coding Agent 必须优先确认：

1. 当前任务属于哪个服务层。
2. 是否违反服务边界。
3. 是否绕过规定协议。
4. 是否把不该放在某层的逻辑写错位置。
5. 是否需要同步更新 `docs/architecture` 或 `docs/contracts` 文件。

---

## 2. 适用场景

当任务涉及以下内容时，必须使用本 Skill：

- 项目目录初始化
- 服务拆分
- 架构文档编写
- Gateway / Orchestrator / Child Agent 边界判断
- 前端和后端通信方式判断
- 协议职责判断
- 全链路数据流设计
- 第一阶段项目骨架搭建
- 架构 Review
- 判断某段代码应该放在哪个服务中
- 防止把 Orchestrator 逻辑写进 Gateway
- 防止 Frontend 直接调用 Orchestrator 或 Child Agent
- 防止 Child Agent 直接返回 UI 组件

---

## 3. 项目总架构

AgentHub 的核心架构必须保持为：

```text
Frontend: React + TypeScript + AG-UI Client
Gateway: Go + AG-UI Server + REST API + Auth + Session
Orchestrator: 意图编排 + A2A 调度 + 协议转换 + 结果聚合
Child Agents: ADK Runtime + A2A + AgentCard
Data Layer: PostgreSQL + Redis + Object Storage
```

完整调用链路为：

```text
用户输入消息
  → React IM UI
  → AG-UI Run Request
  → Gateway
  → Orchestrator
  → A2A Task
  → Child Agent
  → A2A Streaming Response / Artifact
  → Orchestrator Protocol Converter
  → AG-UI Events / Tool Call
  → React Skill Executor
  → 前端 UI 组件渲染
```

---

## MVP v0.1 架构实施范围

`project-architecture` 同时约束两个层次：

1. **完整目标架构**：以 PDR 为准，长期保持 Frontend、Gateway、Orchestrator、Child Agents、Data Layer 的清晰分层。
2. **MVP v0.1 实施架构**：以 MVP 文档为准，优先在 2 天内跑通“用户发消息 → Gateway → Orchestrator → code-agent → 流式回复 → code_preview 产物预览”的最小闭环。

MVP v0.1 允许以下阶段性简化：

- Orchestrator 可以嵌入 Gateway 进程，作为 `server/internal/orchestrator/` 模块存在。
- Gateway、REST API、AG-UI Server、Orchestrator 可以先合并在 `server/` 一个 Go 服务中。
- 意图编排可以简化为直接路由到用户选择的 `code-agent`，暂不调用 LLM 生成 ExecutionPlan。
- 数据库使用 MySQL 8，暂不强制 PostgreSQL。
- 暂不使用 Redis。
- 暂不使用对象存储。
- Child Agent 第一版只实现 `code-agent`。
- Frontend Runtime Skills 第一版只实现 `code_preview`。
- 鉴权第一版可以使用固定 Token 或环境变量 Token。
- Agent 注册第一版可以写在配置文件中，不要求动态注册发现。
- MVP v0.1 不实现群聊多 Agent 协作。
- MVP v0.1 不实现自建 Agent。
- MVP v0.1 不实现 `web-agent` / `doc-agent`。
- MVP v0.1 不实现完整 Agent 市场、历史搜索、复杂错误降级和 UI 动画。

但是，MVP 简化不能破坏以下边界：

- Frontend 仍然只能连接 Gateway。
- Gateway Handler 仍然不能直接堆叠复杂编排逻辑。
- 即使 Orchestrator 嵌入 Gateway 进程，也必须保持 `handler`、`orchestrator`、`a2a`、`converter` 的代码边界。
- Orchestrator 仍然负责 A2A 调度和 A2A → AG-UI 协议转换。
- Code-Agent 仍然必须通过 A2A 被调用。
- Code-Agent 仍然必须暴露 AgentCard。
- Code-Agent 仍然必须遵守 ADK Runtime 约定。
- Artifact 仍然必须通过 AG-UI Tool Call 映射到 Frontend Skill。
- `code` Artifact 必须映射到 `code_preview`。
- 大产物不能塞进 `TEXT_MESSAGE_CONTENT`。
- REST API 仍然只负责持久化资源查询和管理。
- AG-UI 仍然负责实时事件流。
- A2A 仍然负责 Agent 间通信。

后续版本演进方向：

- v0.2 可以恢复 LLM 意图编排和多 Agent 路由。
- v0.2 / v0.3 可以增加 `web-agent`、`doc-agent`。
- v0.3 可以增加群聊、多 Agent 并行 / 串行编排。
- v1.0 可以再拆分独立 Orchestrator Service，引入 Redis、对象存储和更完整的权限体系。

---

## 4. 服务边界总原则

AgentHub 必须拆成多个清晰服务，而不是把所有逻辑塞进一个后端服务。

推荐最小服务拆分：

```text
frontend/
gateway/
orchestrator/
agents/
  adk/
  code-agent/
  web-agent/
  doc-agent/
```

Gateway 和 Orchestrator 必须从第一版开始拆开：

```text
gateway-service
orchestrator-service
```

禁止把意图编排、A2A 调度、Agent 选择逻辑写进 Gateway handler。

---

## 5. Frontend 职责边界

### 5.1 Frontend 负责什么

Frontend 负责用户可见的交互层，包括：

- IM 聊天 UI
- 对话列表
- 聊天窗口
- 消息气泡
- 流式文本渲染
- Agent 头像和状态展示
- 群聊 UI
- @Agent 提及
- 前端 Skills 注册
- 前端 Skill 执行器
- 产物预览组件
- 用户确认弹窗
- 文件上传组件
- 代码预览
- 网页预览
- Diff 预览
- Markdown 渲染
- 部署状态卡片

Frontend 通过 AG-UI 接收实时事件，通过 REST API 查询持久化资源。

---

### 5.2 Frontend 不能做什么

Frontend 禁止：

- 直接调用 Orchestrator。
- 直接调用 Child Agent。
- 实现 A2A 协议逻辑。
- 自己决定多 Agent 编排策略。
- 自己决定 `single / parallel / sequential` 执行计划。
- 绕过 Gateway 请求后端内部服务。
- 手写和后端不一致的 REST API response 类型。
- 把 A2A Artifact 当作前端组件直接处理。
- 把 Orchestrator 的内部状态作为前端业务状态强依赖。
- 把大段产物塞进普通文本消息里渲染。

Frontend 只能连接 Gateway。

---

### 5.3 Frontend 与协议的关系

Frontend 使用：

```text
AG-UI Client
REST API Client
Frontend Runtime Skills
```

Frontend 不使用：

```text
A2A Client
ADK Runtime
AgentCard 生成逻辑
Orchestrator 内部接口
```

---

## 6. Gateway 职责边界

### 6.1 Gateway 负责什么

Gateway 是对外服务入口，负责：

- REST API
- AG-UI Server endpoint
- SSE 流式连接
- 用户鉴权
- JWT 校验
- CORS
- 会话管理
- 消息持久化
- 对话查询
- Agent 列表查询
- Artifact 查询
- Artifact 预览入口
- 前端 ToolResult 接收
- 将 Run 请求转发给 Orchestrator
- 将 Orchestrator 内部事件转换或转发为 AG-UI 事件
- `requestId / traceId` 透传

Gateway 是 Frontend 唯一能直接访问的后端服务。

---

### 6.2 Gateway 不能做什么

Gateway 禁止：

- 实现意图理解。
- 调 LLM 生成 ExecutionPlan。
- 根据用户意图选择 Agent。
- 直接调用 Child Agent。
- 实现 A2A Task 调度。
- 合并多个 Agent 的结果。
- 实现 Agent fallback 策略。
- 把 Orchestrator 的业务逻辑写进 HTTP handler。
- 依赖 React 组件细节。
- 直接返回前端组件实现细节。
- 修改 AG-UI 事件格式而不更新 `agui-event-contract`。

Gateway 不应该知道某个任务应该由 `code-agent` 还是 `web-agent` 处理。

---

### 6.3 Gateway 对外接口

Gateway 对 Frontend 暴露两类接口。

#### REST API

用于持久化资源：

```text
GET    /api/conversations
POST   /api/conversations
GET    /api/conversations/{id}
PATCH  /api/conversations/{id}
DELETE /api/conversations/{id}
GET    /api/conversations/{id}/messages
GET    /api/agents
GET    /api/agents/{name}/card
POST   /api/agents/custom
PATCH  /api/agents/custom/{id}
DELETE /api/agents/custom/{id}
GET    /api/artifacts/{id}
GET    /api/artifacts/{id}/preview
```

#### AG-UI Endpoint

用于实时 Agent 交互：

```text
POST /api/agui/run
POST /api/agui/run/{runId}/cancel
POST /api/agui/run/{runId}/tool-result
```

---

## 7. Orchestrator 职责边界

### 7.1 Orchestrator 负责什么

Orchestrator 是系统的大脑，负责：

- 意图理解
- AgentCard 读取
- 根据用户消息生成 ExecutionPlan
- @Agent 直接路由
- 未指定 Agent 时的 LLM 编排
- `single / parallel / sequential` 策略执行
- A2A Client 调用
- 子 Agent 任务分发
- 多 Agent 并行调度
- 多 Agent 串行调度
- 子 Agent 结果聚合
- 子 Agent 失败 fallback
- A2A → AG-UI 协议转换
- AG-UI ToolResult → A2A Message 反向转换
- Artifact → Frontend Skill Tool Call 映射
- `activeAgent / progress / phase` 状态更新
- `traceId` 贯穿

---

### 7.2 Orchestrator 不能做什么

Orchestrator 禁止：

- 直接暴露给 Frontend。
- 直接处理浏览器鉴权。
- 直接管理用户登录态。
- 实现 React 组件。
- 保存前端 UI 局部状态。
- 直接返回 UI 组件代码。
- 绕过 Gateway 给前端推送事件。
- 自己定义 REST API response 格式。
- 绕过 A2A 调用子 Agent。
- 让子 Agent 的内部实现泄漏到前端。

---

### 7.3 Orchestrator 的输入和输出

Orchestrator 从 Gateway 接收内部请求：

```text
runId
threadId
conversationId
messages
history
frontend tools
context
requestId
traceId
```

Orchestrator 输出内部事件，由 Gateway 映射为 AG-UI 事件：

```text
RUN_STARTED
RUN_FINISHED
RUN_ERROR
TEXT_MESSAGE_START
TEXT_MESSAGE_CONTENT
TEXT_MESSAGE_END
TOOL_CALL_START
TOOL_CALL_ARGS
TOOL_CALL_END
STATE_UPDATE
```

---

## 8. Child Agent 职责边界

### 8.1 Child Agent 负责什么

Child Agent 是具体能力提供者，负责：

- 暴露 AgentCard
- 声明自身 skills
- 声明 inputModes
- 声明 outputModes
- 接收 A2A Task
- 流式返回文本
- 返回 Task 状态
- 返回 Artifact
- 处理任务取消
- 提供健康检查
- 遵守 ADK Runtime 目录和运行时约定

每个 Child Agent 都必须是 A2A Provider。

---

### 8.2 Child Agent 不能做什么

Child Agent 禁止：

- 直接访问 Frontend。
- 直接访问 Gateway 的用户 REST API。
- 直接生成 React 组件调用。
- 直接控制前端 Skill。
- 返回不符合 `artifact-contract` 的产物。
- 返回未声明 `outputModes` 的产物类型。
- 自己发明 AgentCard 字段。
- 绕过 Orchestrator 和其他 Agent 通信。
- 把大文件内容塞进流式文本。
- 在没有声明能力的情况下执行任务。

---

### 8.3 Child Agent 必须暴露的端点

每个 Child Agent 必须暴露：

```text
GET    /.well-known/agent.json
POST   /a2a/tasks/send
POST   /a2a/tasks/sendSubscribe
GET    /a2a/tasks/{id}
DELETE /a2a/tasks/{id}/cancel
GET    /health
```

---

### 8.4 预置 Child Agents

第一版至少应支持：

```text
code-agent
web-agent
doc-agent
```

后续可扩展：

```text
deploy-agent
custom-agent
```

P0 最少要跑通两个 Child Agents：

```text
code-agent
web-agent
```

---

## 9. Data Layer 职责边界

### 9.1 PostgreSQL

PostgreSQL 用于存储：

- user
- conversation
- conversation_participant
- message
- agent
- artifact

消息内容和 AgentCard 可使用 JSONB。

---

### 9.2 Redis

Redis 用于：

- 会话缓存
- 在线状态
- SSE 临时状态
- Run 状态缓存
- Agent 健康状态缓存
- 临时队列或事件缓冲

---

### 9.3 Object Storage

Object Storage 用于：

- 大文件
- 图片
- PDF
- 生成的压缩包
- 部署包
- 大型 Artifact 文件

大产物不能直接塞进数据库 text 字段，也不能塞进 AG-UI 文本流。

---

## 10. 协议职责划分

### 10.1 AG-UI

AG-UI 只负责实时 Agent / UI 交互，包括：

- Run 生命周期
- 文本流式输出
- Tool Call
- Tool Result
- 前端 Skills 调用
- 状态更新
- 错误事件
- 中断和取消

AG-UI 不负责：

- 持久化资源查询
- Agent 间通信
- 数据库分页
- AgentCard 注册
- 子 Agent 任务执行协议

---

### 10.2 REST API

REST API 负责持久化资源，包括：

- 会话列表
- 会话详情
- 消息列表
- Agent 列表
- AgentCard 查询
- 自建 Agent 管理
- Artifact 查询
- Artifact 预览
- Artifact 下载

REST API 必须由 OpenAPI 统一描述。

---

### 10.3 Gateway-Orchestrator 内部 Contract

Gateway 和 Orchestrator 的内部通信负责：

- 创建 run
- 取消 run
- 转发 ToolResult
- 返回内部事件流
- 关联 `runId / threadId / conversationId`
- 贯穿 `requestId / traceId`
- 内部错误映射
- 内部事件到 AG-UI 事件的映射

建议内部接口：

```text
POST /internal/runs
POST /internal/runs/{runId}/tool-result
POST /internal/runs/{runId}/cancel
GET  /internal/runs/{runId}/events
```

---

### 10.4 A2A

A2A 只用于 Orchestrator 和 Child Agents 之间，包括：

- AgentCard 发现
- Task 创建
- Task 流式订阅
- Task 状态查询
- Task 取消
- Artifact 返回
- Agent 能力声明

A2A 不用于 Frontend 和 Gateway。

---

### 10.5 Artifact Contract

Artifact 是所有非纯文本产物的统一格式，包括：

```text
code
webpage
diff
file
image
deploy
document
terminal
chart
```

Artifact 必须包含：

```text
id
type
title
content 或 file_url
metadata
version
message_id
conversation_id
```

Artifact 必须能映射到 Frontend Skill：

```text
code      → code_preview
webpage   → web_preview
diff      → diff_preview
file      → file_download
image     → image_preview
deploy    → deploy_status
document  → markdown_render
terminal  → terminal_output
chart     → chart_render
```

---

## 11. 目录结构规范

推荐项目最终目录：

```text
agenthub/
  frontend/
  gateway/
  orchestrator/
  agents/
    adk/
    code-agent/
    web-agent/
    doc-agent/

  docs/
    architecture/
      overview.md
      service-boundaries.md
      frontend.md
      gateway.md
      orchestrator.md
      child-agents.md

    contracts/
      openapi.yaml
      agui-events.md
      agui-events.schema.json
      gateway-orchestrator.md
      gateway-orchestrator-events.md
      a2a-agent-card.md
      a2a-task.md
      a2a-errors.md
      frontend-skills.md
      frontend-skills.schema.json
      artifact-schema.md
      artifact.schema.json
      intent-orchestration.md
      execution-plan.schema.json
      adk-runtime.md
      testing-review.md

  skills/
    project-architecture/
      SKILL.md
    platform-api-contract/
      SKILL.md
    agui-event-contract/
      SKILL.md
    gateway-orchestrator-contract/
      SKILL.md
    a2a-agent-contract/
      SKILL.md
```

如果后续接入 Claude Code，建议再复制到：

```text
.claude/skills/
```

当前本地准备阶段可以先保留在：

```text
skills/
```

---

## 12. 硬性规则

Coding Agent 在本项目中必须遵守：

1. Frontend 只能连接 Gateway。
2. Gateway 对外提供 REST API 和 AG-UI。
3. Gateway 不能实现意图编排。
4. Gateway 不能直接调用 Child Agent。
5. Orchestrator 必须独立于 Gateway。
6. Orchestrator 负责意图编排、A2A 调度和结果聚合。
7. Orchestrator 不能直接暴露给 Frontend。
8. Child Agents 必须通过 A2A 被调用。
9. Child Agents 必须暴露 AgentCard。
10. Child Agents 必须遵守 ADK Runtime 约定。
11. 非文本产物必须通过 Artifact 表达。
12. Artifact 必须映射到 Frontend Skill。
13. AG-UI 只负责实时交互。
14. REST API 只负责持久化资源。
15. OpenAPI 是 REST API 唯一事实源。
16. A2A 是 Agent 间通信唯一协议。
17. 大产物不能塞进 `TEXT_MESSAGE_CONTENT`。
18. 前端不能手写后端 response 类型。
19. 所有跨服务字段变更必须先更新 Contract。
20. 先写 Contract，再写 Mock，再做真实集成。
21. 必须完整遵守 `Contract first / Mock first / Real integration later / Review always`：先定义协议和 Contract，再搭建 Mock 链路，再进行真实集成，最后始终进行人工 Review 和自动化检查。
22. REST API 的具体 request / response 字段格式不得在 `project-architecture` 中临时定义，必须由 `platform-api-contract` 和 `docs/contracts/openapi.yaml` 固定；在 `platform-api-contract` 完成前，默认遵循 PDR 中的统一响应格式 `{ "code": 0, "data": {}, "message": "success" }`，分页响应默认遵循 `{ "list": [], "total": 0, "page": 1, "pageSize": 20 }`，任何变更必须同步更新 PDR、OpenAPI、前端 API client 和 Go handler。
23. MVP v0.1 可以合并部署，但不能合并职责；即使 Orchestrator 嵌入 Gateway 进程，也必须保持模块边界。
24. MVP v0.1 优先跑通 `code-agent + code_preview` 闭环，不得提前扩展 `web-agent`、`doc-agent`、自建 Agent、群聊或复杂多 Agent 编排，除非用户明确要求进入后续版本。
25. MVP v0.1 使用 MySQL 8、固定 Token、配置文件 Agent 注册是允许的阶段性实现，不得因此修改长期 PDR 架构方向。

---

## 13. 开发阶段使用方式

### 阶段 0：PDR 转开发规范

使用本 Skill 生成：

```text
docs/architecture/overview.md
docs/architecture/service-boundaries.md
docs/contracts 文件清单
第一阶段任务拆分
```

此阶段不要写业务代码。

---

### 阶段 1：基础框架搭建

使用本 Skill 检查：

```text
frontend 是否只连 Gateway
gateway-service 是否独立
orchestrator-service 是否独立
mock code-agent 是否通过 A2A 暴露
端到端 mock 链路是否符合架构
```

---

### 阶段 2：核心功能开发

使用本 Skill 检查：

```text
Conversation API 是否属于 Gateway
AG-UI 事件是否由 Gateway 对前端输出
Agent 选择逻辑是否在 Orchestrator
Artifact 是否通过 Frontend Skill 展示
```

---

### 阶段 3：功能完善

使用本 Skill 检查：

```text
群聊是否由 Orchestrator 编排
@Agent 是否走直接路由
多 Agent 并行是否由 Orchestrator 处理
自建 Agent 是否仍然暴露 AgentCard
```

---

### 阶段 4：打磨与交付

使用本 Skill 检查：

```text
Docker Compose 是否体现服务拆分
.env.example 是否包含各服务配置
日志和 traceId 是否贯穿 Gateway 和 Orchestrator
Demo 链路是否体现 AG-UI + A2A
```

---

## 14. 必须维护的文件

使用本 Skill 时，相关变更应同步维护：

```text
docs/architecture/overview.md
docs/architecture/service-boundaries.md
docs/architecture/frontend.md
docs/architecture/gateway.md
docs/architecture/orchestrator.md
docs/architecture/child-agents.md
```

涉及协议边界时，还要同步检查：

```text
docs/contracts/openapi.yaml
docs/contracts/agui-events.md
docs/contracts/gateway-orchestrator.md
docs/contracts/a2a-agent-card.md
docs/contracts/a2a-task.md
docs/contracts/frontend-skills.md
docs/contracts/artifact-schema.md
docs/contracts/intent-orchestration.md
docs/contracts/adk-runtime.md
```

---

## 15. 输出要求

当用户要求生成架构相关内容时，Coding Agent 应输出：

1. 服务边界说明。
2. 当前任务涉及哪些服务。
3. 是否需要新增或更新 Contract。
4. 数据流或时序说明。
5. 目录结构建议。
6. Mock-first 开发步骤。
7. 禁止事项。
8. Review checklist。

除非用户明确要求，否则不要直接生成业务实现代码。

---

## 16. Review Checklist

在接受任何架构、目录、代码生成结果前，必须检查：

### Frontend 检查

- Frontend 是否只调用 Gateway？
- 是否没有直接调用 Orchestrator？
- 是否没有直接调用 Child Agent？
- 是否没有实现 A2A 逻辑？
- 前端 Skills 是否只处理 UI 能力？

### Gateway 检查

- Gateway 是否只做 REST、AG-UI、鉴权、会话和转发？
- Gateway 是否没有写意图编排？
- Gateway 是否没有直接调子 Agent？
- Gateway handler 是否足够薄？
- Gateway 是否遵守 OpenAPI？

### Orchestrator 检查

- Orchestrator 是否负责 ExecutionPlan？
- Orchestrator 是否基于 AgentCard 做路由？
- Orchestrator 是否通过 A2A 调子 Agent？
- Orchestrator 是否负责 A2A → AG-UI 转换？
- Orchestrator 是否没有依赖 React 组件？

### Child Agent 检查

- Child Agent 是否暴露 AgentCard？
- Child Agent 是否暴露 A2A 端点？
- Child Agent 是否有 `/health`？
- Child Agent 是否遵守 ADK Runtime？
- Child Agent 是否只返回 text / status / artifact / tool_call？

### Artifact 检查

- 非文本产物是否使用 Artifact？
- Artifact type 是否合法？
- Artifact 是否能映射到 Frontend Skill？
- 大文件是否使用 `file_url`？
- 是否没有把大产物塞进 `TEXT_MESSAGE_CONTENT`？

### 协议检查

- AG-UI 是否只用于实时交互？
- REST API 是否只用于资源查询和管理？
- A2A 是否只用于 Agent 间通信？
- OpenAPI 是否是 REST API 唯一事实源？
- 内部 Gateway-Orchestrator Contract 是否被遵守？


## 17. v1.1 对齐补充

### 17.1 v1.1 项目级 Skills 总表

v1.1 项目级 Skills 总数为 17：

P0（13）:

```text
project-architecture
code-style-and-conventions
ai-collaboration-workflow
platform-api-contract
agui-event-contract
gateway-orchestrator-contract
a2a-agent-contract
frontend-runtime-skills-contract
artifact-contract
data-persistence-contract
intent-orchestration-contract
adk-runtime-contract
security-boundary-contract
```

P1（4）:

```text
llm-provider-contract
observability-debugging-contract
testing-review-contract
docker-compose-delivery
```

### 17.2 v1.1 新增 / 前期遗漏的 P0 支撑 Skill

以下 4 个属于 v1.1 新增或前期遗漏的 P0 支撑 Skill：

```text
code-style-and-conventions
ai-collaboration-workflow
data-persistence-contract
security-boundary-contract
```

### 17.3 Child Agent 数量的 MVP / Post-MVP 兼容说明

- PDR / Post-MVP 可以要求至少两个 Child Agents，用于验证多 Agent 能力。
- MVP v0.1 当前只强制跑通 `code-agent` 闭环。
- 不得把 Post-MVP 的“至少两个 Child Agents”误解为 MVP v0.1 必须实现。



## References

- `references/module-boundaries.md`
- `references/mvp-scope.md`
- `references/post-mvp-boundaries.md`
- `references/architecture-review-checklist.md`
