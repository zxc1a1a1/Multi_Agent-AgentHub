# AgentHub 架构总览

## 1. 文档目的

本文档用于把 PDR 中的系统架构落成开发阶段可执行的架构说明。  
当前文档只定义架构、服务边界和协议职责，不包含业务实现代码。

本项目采用：

```text
Frontend: React + TypeScript + AG-UI Client
Gateway: Go + AG-UI Server + REST API + Auth + Session
Orchestrator: 意图编排 + A2A 调度 + 协议转换 + 结果聚合
Child Agents: ADK Runtime + A2A + AgentCard
Data Layer: PostgreSQL + Redis + Object Storage
```

核心链路是：

```text
React IM
  → AG-UI
  → Gateway
  → Orchestrator
  → A2A
  → Child Agents
  → Artifact
  → Frontend Skills
```

---

## MVP v0.1 实施说明

AgentHub 的完整目标架构仍然遵守 PDR：Frontend、Gateway、Orchestrator、Child Agents、Data Layer 分层清晰。

MVP v0.1 是第一版最小可行产品，目标是在 2 天内跑通：

```text
用户发消息
→ Gateway
→ Orchestrator
→ code-agent
→ A2A 流式响应
→ AG-UI 流式回复
→ code_preview 代码预览
```

MVP v0.1 的阶段性简化：

- Orchestrator 嵌入 Gateway 进程。
- 后端第一版可以使用 `server/` 目录承载 Gateway + Orchestrator。
- 数据库使用 MySQL 8。
- 暂不使用 Redis 和对象存储。
- 只实现 `code-agent`。
- 只实现 `code_preview`。
- 只实现单 Agent 对话。
- 不做群聊、自建 Agent、web-agent、doc-agent、复杂意图编排和完整权限体系。

这些简化只影响 v0.1 实施范围，不改变长期 PDR 架构方向。

---

## 2. 产品形态

AgentHub 是一个以 IM 聊天为核心交互范式的多 Agent 协作平台。用户通过新建对话、发送消息、@Agent、上传文件等方式，与多个 AI Agent 协作完成代码生成、网页构建、文档编写、部署等任务。

第一版优先保证：

1. IM 聊天核心体验。
2. AG-UI 实时流式回复。
3. Gateway 网关。
4. Orchestrator 意图编排。
5. 至少两个 Child Agents。
6. A2A 调用链路。
7. Artifact 到 Frontend Skills 的映射。
8. Docker Compose 演示部署。

---

## 3. 分层架构

### 3.1 Frontend

Frontend 是用户交互层，负责：

- 对话列表
- 聊天窗口
- 消息气泡
- 流式文本渲染
- 群聊 UI
- @Agent 提及
- 前端 Skills 注册
- 产物预览组件
- ToolResult 回传

Frontend 只能连接 Gateway。

---

### 3.2 Gateway

Gateway 是对外服务入口，负责：

- REST API
- AG-UI Server endpoint
- SSE 连接
- 用户鉴权
- 会话管理
- 消息持久化
- Artifact 查询
- Agent 列表查询
- 将 Run 请求转发给 Orchestrator
- 将 Orchestrator 内部事件转为前端可消费的 AG-UI 事件

Gateway 不负责意图理解和 Agent 调度。

---

### 3.3 Orchestrator

Orchestrator 是系统大脑，负责：

- 意图理解
- AgentCard 读取
- ExecutionPlan 生成
- @Agent 直接路由
- A2A 调用 Child Agents
- single / parallel / sequential 策略执行
- 结果聚合
- fallback
- A2A → AG-UI 协议转换
- Artifact → Frontend Skill Tool Call 映射

Orchestrator 不直接暴露给 Frontend。

---

### 3.4 Child Agents

Child Agents 是具体能力提供者，负责：

- 暴露 AgentCard
- 暴露 A2A 标准端点
- 接收 A2A Task
- 流式返回文本
- 返回 Artifact
- 提供健康检查
- 遵守 ADK Runtime 约定

第一版至少实现：

```text
code-agent
web-agent
```

推荐同时准备：

```text
doc-agent
```

后续可扩展：

```text
deploy-agent
custom-agent
```

---

### 3.5 Data Layer

Data Layer 包括：

```text
PostgreSQL
Redis
Object Storage
```

PostgreSQL 负责关系型数据和 JSONB 消息内容。  
Redis 负责缓存、在线状态、Run 状态、临时事件缓冲。  
Object Storage 负责大文件、图片、PDF、压缩包、部署包等大型 Artifact。

---

## 4. 核心数据流

### 4.1 单 Agent 调用链路

```text
用户发送消息
  → Frontend 构造 AG-UI RunRequest
  → Gateway 鉴权并保存用户消息
  → Gateway 转发给 Orchestrator
  → Orchestrator 读取 AgentCard
  → Orchestrator 生成 ExecutionPlan
  → Orchestrator 通过 A2A 调用目标 Child Agent
  → Child Agent 流式返回文本和 Artifact
  → Orchestrator 转换为 AG-UI 事件
  → Gateway 通过 SSE 推给 Frontend
  → Frontend 渲染消息和预览卡片
```

### 4.2 多 Agent 协作链路

```text
用户发送复杂任务
  → Orchestrator 判断需要多个 Agent
  → 生成 parallel 或 sequential ExecutionPlan
  → 并行或串行调用多个 Child Agents
  → 收集 A2A 响应
  → 聚合结果
  → 将 Artifact 转换为 Tool Call
  → Frontend Skills 渲染多个产物卡片
```

### 4.3 交互式 Skill 链路

```text
Child Agent 请求确认或输入
  → A2A tool_call
  → Orchestrator 识别为 Frontend Skill
  → 转为 AG-UI TOOL_CALL
  → Frontend 执行 confirm_action / form_input / file_upload
  → Frontend 回传 ToolResult
  → Gateway 转发给 Orchestrator
  → Orchestrator 转换为 A2A Message
  → Child Agent 继续执行
```

---

## 5. 协议职责

### 5.1 AG-UI

AG-UI 用于 Frontend 与 Gateway 之间的实时 Agent/UI 交互：

- RUN_STARTED
- RUN_FINISHED
- RUN_ERROR
- TEXT_MESSAGE_START
- TEXT_MESSAGE_CONTENT
- TEXT_MESSAGE_END
- TOOL_CALL_START
- TOOL_CALL_ARGS
- TOOL_CALL_END
- STATE_UPDATE

AG-UI 不负责持久化资源查询。

---

### 5.2 REST API

REST API 用于持久化资源查询和管理：

- conversations
- messages
- agents
- artifacts
- custom agents

REST API 必须由 OpenAPI 描述。

---

### 5.3 Gateway-Orchestrator 内部 Contract

内部 Contract 用于：

- 创建 run
- 取消 run
- 转发 ToolResult
- 订阅内部事件
- 映射错误
- 透传 requestId / traceId

---

### 5.4 A2A

A2A 用于 Orchestrator 和 Child Agents 之间：

- AgentCard 发现
- Task 创建
- Task 流式订阅
- Task 查询
- Task 取消
- Artifact 返回

A2A 不用于 Frontend。

---

### 5.5 Artifact

Artifact 是所有非纯文本产物的统一表示：

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

大产物必须通过 Artifact 表达，不能塞进 TEXT_MESSAGE_CONTENT。

---

## 6. Mock-first 开发原则

本项目必须遵守：

```text
Contract first
Mock first
Real integration later
Review always
```

第一阶段先跑通：

```text
React mock AG-UI client
Gateway mock AG-UI server
Orchestrator mock run processor
Mock A2A code-agent
端到端 mock 链路
```

然后再逐步替换为真实 LLM、真实数据库、真实 Agent 逻辑。

---

## 7. 第一阶段里程碑

第一阶段完成标准：

```text
Frontend 发消息
  → Gateway 接收 AG-UI Run
  → Orchestrator 生成 mock ExecutionPlan
  → code-agent 通过 A2A mock 返回文本
  → Orchestrator 转换为 AG-UI 事件
  → Frontend 流式显示
```

必须看到：

1. RUN_STARTED。
2. TEXT_MESSAGE_START。
3. TEXT_MESSAGE_CONTENT。
4. TEXT_MESSAGE_END。
5. RUN_FINISHED。
6. 至少一个 Artifact → code_preview Tool Call。
