# AgentHub 服务边界说明

## 1. 总原则

AgentHub 必须保持四层服务边界：

```text
Frontend
Gateway
Orchestrator
Child Agents
```

各层只能通过规定协议通信，禁止跨层直连。

---

## MVP v0.1 边界例外

MVP v0.1 允许 Orchestrator 嵌入 Gateway 进程，但这只是部署和目录层面的简化，不代表职责合并。

即使在同一个 `server/` 中，也必须保持：

```text
server/internal/handler/        → HTTP、REST、AG-UI SSE 入口
server/internal/orchestrator/   → 路由、A2A 调度、协议转换
server/internal/a2a/            → A2A Client 类型与调用
server/internal/store/          → MySQL 持久化
```

禁止：

- 在 handler 中直接堆叠复杂编排逻辑。
- 在 handler 中直接解析 A2A Artifact 并拼装 Frontend Skill。
- 在 Frontend 中直接调用 Orchestrator。
- 在 Frontend 中直接调用 Code-Agent。
- 因为 MVP 合并进程而删除 Orchestrator 边界。

---

## 2. Frontend 边界

### 2.1 Frontend 负责

- React IM UI
- AG-UI Client
- REST API Client
- 前端 Skills 注册
- Skill 执行器
- 消息渲染
- 产物预览
- 用户交互

### 2.2 Frontend 禁止

- 直接访问 Orchestrator。
- 直接访问 Child Agent。
- 实现 A2A。
- 自己生成 ExecutionPlan。
- 自己决定 Agent 调度策略。
- 手写和 OpenAPI 不一致的 response 类型。
- 把 Artifact 当作普通文本处理。

### 2.3 Frontend 允许访问

```text
Gateway REST API
Gateway AG-UI Endpoint
```

---

## 3. Gateway 边界

### 3.1 Gateway 负责

- REST API
- AG-UI Server
- SSE 连接
- Auth / JWT
- CORS
- Session
- Conversation API
- Message API
- Agent API
- Artifact API
- ToolResult 接收
- 转发 Run 给 Orchestrator
- 转发 Orchestrator 事件给 Frontend

### 3.2 Gateway 禁止

- 实现意图理解。
- 调 LLM 生成计划。
- 选择具体 Child Agent。
- 直接调用 Child Agent。
- 实现 A2A Client。
- 聚合多个 Agent 的结果。
- 编写 fallback 策略。
- 把编排逻辑写进 handler。

### 3.3 Gateway 对外接口

REST API：

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

AG-UI：

```text
POST /api/agui/run
POST /api/agui/run/{runId}/cancel
POST /api/agui/run/{runId}/tool-result
```

### 3.4 Gateway 内部访问

Gateway 只能通过内部 Contract 访问 Orchestrator：

```text
POST /internal/runs
POST /internal/runs/{runId}/tool-result
POST /internal/runs/{runId}/cancel
GET  /internal/runs/{runId}/events
```

---

## 4. Orchestrator 边界

### 4.1 Orchestrator 负责

- 接收 Gateway 内部请求
- 读取 AgentCard
- 解析 @Agent mention
- 调 LLM 进行意图理解
- 生成 ExecutionPlan
- 通过 A2A 调 Child Agent
- 并行 / 串行 / 单 Agent 执行
- 聚合结果
- fallback
- 协议转换
- Artifact 映射 Frontend Skill

### 4.2 Orchestrator 禁止

- 对浏览器公开。
- 自己处理登录鉴权。
- 管理用户会话 UI 状态。
- 直接推送事件给 Frontend。
- 依赖 React 组件。
- 直接返回 UI 组件。
- 绕过 A2A 调用 Agent。
- 自定义 Gateway REST response 格式。

### 4.3 Orchestrator 输入

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

### 4.4 Orchestrator 输出

内部事件，最终映射为 AG-UI：

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

## 5. Child Agent 边界

### 5.1 Child Agent 负责

- AgentCard
- A2A Server
- Task Handler
- Tool 注册
- 流式文本
- Artifact 输出
- 健康检查
- 任务取消

### 5.2 Child Agent 禁止

- 直接访问 Frontend。
- 直接控制 Frontend Skills。
- 直接访问 Gateway 用户 API。
- 直接和其他 Child Agent 通信。
- 返回未声明 outputModes 的 Artifact。
- 发明不在 Contract 中的 Artifact type。
- 把大产物塞进文本流。

### 5.3 Child Agent 必须暴露

```text
GET    /.well-known/agent.json
POST   /a2a/tasks/send
POST   /a2a/tasks/sendSubscribe
GET    /a2a/tasks/{id}
DELETE /a2a/tasks/{id}/cancel
GET    /health
```

---

## 6. 边界判定规则

当不知道代码该放哪里时，按下面规则判断：

| 问题 | 放置位置 |
|---|---|
| 是否是 UI 展示？ | Frontend |
| 是否是 REST API？ | Gateway |
| 是否是鉴权 / 会话 / 消息持久化？ | Gateway |
| 是否是意图理解？ | Orchestrator |
| 是否是 Agent 选择？ | Orchestrator |
| 是否是 A2A 调用？ | Orchestrator |
| 是否是具体生成代码 / 网页 / 文档？ | Child Agent |
| 是否是 Artifact 存储查询？ | Gateway + Data Layer |
| 是否是 Artifact 产出？ | Child Agent |
| 是否是 Artifact 到 Skill 映射？ | Orchestrator |
| 是否是 Skill UI 渲染？ | Frontend |

---

## 7. 最常见错误

### 错误 1：Gateway 直接调 code-agent

错误原因：绕过 Orchestrator 和 A2A 调度。  
正确做法：Gateway 转发给 Orchestrator，Orchestrator 通过 A2A 调 code-agent。

### 错误 2：Frontend 直接请求 Orchestrator

错误原因：破坏 Gateway 统一入口。  
正确做法：Frontend 只访问 Gateway。

### 错误 3：Child Agent 返回 React 组件

错误原因：Agent 层泄漏 UI 实现。  
正确做法：Child Agent 返回 Artifact，Orchestrator 转为 Tool Call，Frontend Skill 渲染组件。

### 错误 4：把代码文件塞进 TEXT_MESSAGE_CONTENT

错误原因：大产物污染文本流。  
正确做法：代码使用 Artifact type=code，然后映射为 code_preview。

### 错误 5：未更新 Contract 就改接口字段

错误原因：破坏 Contract first。  
正确做法：先更新 docs/contracts，再更新前后端实现。

---

## 8. Review Checklist

提交前必须检查：

- Frontend 是否只连 Gateway？
- Gateway 是否没有意图编排逻辑？
- Gateway 是否没有直接调 Child Agent？
- Orchestrator 是否通过 A2A 调 Agent？
- Child Agent 是否暴露 AgentCard？
- Artifact 是否符合 contract？
- AG-UI 是否只用于实时交互？
- REST API 是否只用于资源管理？
- OpenAPI 是否同步更新？
- traceId 是否贯穿 Gateway 和 Orchestrator？
