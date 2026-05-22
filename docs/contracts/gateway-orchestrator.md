# Gateway-Orchestrator Internal Contract

## 1. 文档目的

本文档定义 AgentHub 中 **Gateway ↔ Orchestrator** 的内部调用契约。

该 Contract 的目标是固定：

- Gateway 如何把 Frontend 的 AG-UI Run 请求交给 Orchestrator。
- Gateway 与 Orchestrator 在 MVP v0.1 同进程模式下如何保持职责边界。
- Post-MVP 中 Orchestrator 拆成独立服务时应如何演进。
- `OrchestratorRequest`、`EventSink`、`OrchestratorResult`、错误、取消、超时、trace 的基本规则。
- Gateway handler 哪些事情可以做，哪些事情必须交给 Orchestrator。
- Orchestrator 如何通过 A2A 调用 Child Agent，并输出符合 AG-UI Contract 的事件。

一句话：

> Gateway 负责对外入口和连接管理，Orchestrator 负责路由 / 编排 / A2A 调度 / 协议转换；二者之间必须通过清晰的内部 Contract 协作。

---

## 2. 协议边界

本 Contract 只约束：

```text
Gateway ↔ Orchestrator
```

不约束：

```text
Frontend ↔ Gateway REST API
Frontend ↔ Gateway AG-UI Event Stream
Orchestrator ↔ Child Agent A2A
Artifact Schema
Frontend Runtime Skills Schema
ADK Runtime
```

对应关系：

| 通信方向 | Contract / 协议 | 是否由本文档定义 |
|---|---|---|
| Frontend ↔ Gateway REST API | OpenAPI | 否 |
| Frontend ↔ Gateway AG-UI SSE | AG-UI Event Contract | 否 |
| Gateway ↔ Orchestrator | Go interface / Internal Contract | 是 |
| Orchestrator ↔ Child Agent | A2A | 否 |
| Artifact 数据结构 | Artifact Contract | 否 |
| Artifact → Frontend Skill | Frontend Runtime Skills Contract | 否 |

---

## 3. 设计来源与解释原则

本文档同时遵守四类设计来源：

1. **PDR**：定义完整目标架构，Gateway 和 Orchestrator 是清晰分层的系统角色。
2. **MVP 文档**：定义 v0.1 最小实施范围，允许 Orchestrator 嵌入 Gateway 进程。
3. **UML 文档**：定义关键链路、时序、协议转换和活动流程。
4. **Skills 设计规范**：定义该内部 Contract 必须作为项目级 Skill 自建。

解释原则：

```text
PDR 决定长期方向。
MVP 决定当前范围。
UML 决定关键流程。
Skills 设计规范决定 AI 开发约束。
```

MVP 可以简化部署，但不能合并职责。

---

## 4. MVP v0.1 实施模式

MVP v0.1 允许 Orchestrator 嵌入 Gateway 进程，作为 `server/internal/orchestrator/` 模块存在。

推荐目录结构：

```text
server/
  internal/
    handler/
      agui.go
      conversation.go
      agent.go
    orchestrator/
      orchestrator.go
      converter.go
      types.go
    a2a/
      client.go
      types.go
    store/
      mysql.go
    config/
      config.go
```

MVP v0.1 调用关系：

```text
handler/agui.go
  → orchestrator.Process(ctx, req, history, eventSink)
  → a2a.Client.SendSubscribe(...)
  → converter.Convert(...)
  → eventSink.Emit(AGUIEvent)
  → handler/agui.go 写出 SSE
```

MVP v0.1 允许：

- Gateway 与 Orchestrator 在同一个 Go 进程中。
- Orchestrator 使用 Go interface / function call。
- Orchestrator 直接路由到用户选择的 `code-agent`。
- 暂不调用 LLM 生成 ExecutionPlan。
- Agent 注册通过配置文件写死。
- 只调 `code-agent`。
- 只处理 `code` Artifact → `code_preview`。
- 使用 MySQL 8 保存会话和消息。
- 使用固定 Token 或环境变量 Token 鉴权。

MVP v0.1 不允许：

- 在 Gateway handler 中直接写 A2A 调度逻辑。
- 在 Gateway handler 中直接写 A2A → AG-UI 协议转换逻辑。
- 在 Gateway handler 中直接解析 Artifact 并拼装 `code_preview` Tool Call。
- 因为同进程而删除 Orchestrator 模块边界。
- Frontend 直接调用 Orchestrator。
- Frontend 直接调用 Child Agent。
- Orchestrator 直接写 HTTP response。
- Orchestrator 直接管理 SSE 连接。
- Orchestrator 直接做用户鉴权。

---

## 5. Post-MVP 完整模式

Post-MVP 可将 Orchestrator 拆分为独立服务：

```text
gateway-service
orchestrator-service
```

长期调用关系：

```text
Gateway
  → internal HTTP / RPC
  → Orchestrator Service
  → A2A
  → Child Agents
```

如果拆分为独立服务，推荐 internal endpoint：

```text
POST /internal/orchestrator/runs
POST /internal/orchestrator/runs/{runId}/cancel
```

要求：

- `/internal/*` 不得暴露给 Frontend。
- internal endpoint 不得写入 Frontend REST API 的 `docs/contracts/openapi.yaml`。
- internal endpoint 必须有服务间鉴权。
- internal endpoint 必须支持 `traceId`。
- internal endpoint 必须支持 timeout。
- Orchestrator 输出的事件语义仍然必须符合 `agui-event-contract`。
- A2A 调用仍然只发生在 Orchestrator 边界内。

---

## 6. Gateway 职责

Gateway 负责：

1. 接收 Frontend 的 `POST /api/agui/run`。
2. 执行鉴权。
3. 校验基础请求格式。
4. 生成或透传 `runId`、`threadId`、`traceId`。
5. 保存用户消息。
6. 查询会话历史。
7. 构造 `OrchestratorRequest`。
8. 创建事件通道或 `EventSink`。
9. 调用 Orchestrator。
10. 将 Orchestrator 输出的 AG-UI Event 写成 SSE。
11. 处理客户端断开、取消、超时。
12. 在运行结束后保存 Agent 回复和 Artifact 引用。
13. 统一日志、`requestId`、`traceId`。
14. 将 Orchestrator 错误映射为 AG-UI `RUN_ERROR` 或统一错误响应。

Gateway 不负责：

- LLM 意图分析。
- ExecutionPlan 生成。
- A2A 调度。
- A2A Stream 解析。
- A2A → AG-UI 协议转换。
- Artifact → Frontend Skill 映射。
- 多 Agent 结果聚合。
- Child Agent 选择策略。
- 直接调用 Child Agent。
- 直接调用 LLM。

---

## 7. Orchestrator 职责

Orchestrator 负责：

1. 接收 `OrchestratorRequest`。
2. 读取用户消息、历史消息、前端声明的 Tools / Skills。
3. MVP v0.1 中直接路由到指定 `code-agent`。
4. Post-MVP 中执行意图分析和 ExecutionPlan 生成。
5. 通过 A2A Client 调用 Child Agent。
6. 接收 A2A Stream Event。
7. 调用 ProtocolConverter 将 A2A 事件转换为 AG-UI Event。
8. 缓存 Artifact，并在 completed 后 flush 为 Tool Call。
9. 将 AG-UI Event 推送给 Gateway 的 `EventSink`。
10. 输出 `RUN_STARTED`、`TEXT_MESSAGE_*`、`TOOL_CALL_*`、`RUN_FINISHED`、`RUN_ERROR`。
11. 返回 `OrchestratorResult`，供 Gateway 持久化 Agent 回复和 Artifact 引用。
12. 保持可取消、可超时、可追踪。

Orchestrator 不负责：

- 对外暴露 Frontend API。
- 直接处理浏览器连接。
- 直接写 SSE response。
- 执行用户登录鉴权。
- 管理前端状态。
- 保存用户消息。
- 查询会话列表。
- 直接操作 React 组件。
- 返回普通 REST response 给 Frontend。

---

## 8. MVP v0.1 最小内部流程

```text
1. Gateway 接收 POST /api/agui/run
2. Gateway 鉴权
3. Gateway 保存用户消息
4. Gateway 查询最近历史消息
5. Gateway 构造 OrchestratorRequest
6. Gateway 创建 EventSink / EventChan
7. Gateway 调用 Orchestrator.Process
8. Orchestrator 输出 RUN_STARTED
9. Orchestrator 选择 code-agent
10. Orchestrator 调用 A2A sendSubscribe
11. Orchestrator 接收 A2A status / text / artifact / completed
12. Orchestrator 调用 ProtocolConverter 转换 AG-UI Event
13. Gateway 将 AG-UI Event 通过 SSE 写给 Frontend
14. Orchestrator 返回 OrchestratorResult
15. Gateway 保存 Agent 回复和 Artifact
16. Gateway 关闭 SSE
```

MVP v0.1 不要求：

- 多 Agent ExecutionPlan。
- LLM 意图分析。
- 群聊 Agent 选择。
- 失败自动切换备用 Agent。
- 分布式 Orchestrator Service。
- 复杂任务状态持久化。
- 完整 cancel 实现。
- 多路并发 Agent 聚合。

---

## 9. OrchestratorRequest

MVP v0.1 推荐逻辑结构：

```go
type OrchestratorRequest struct {
    RunID     string
    ThreadID  string
    UserID    string
    AgentName string

    Messages []AGUIMessage
    History  []Message

    Tools   []AGUITool
    Context map[string]any

    TraceID   string
    RequestID string
}
```

字段说明：

| 字段 | 说明 |
|---|---|
| `RunID` | 单次 AG-UI run 的 ID |
| `ThreadID` | 对话 / conversation ID |
| `UserID` | 当前用户 ID；MVP 可用固定用户 |
| `AgentName` | MVP 直接路由目标，默认 `code-agent` |
| `Messages` | 本次 Run 携带的消息 |
| `History` | Gateway 查询到的历史上下文 |
| `Tools` | Frontend 声明的可用 Skills，如 `code_preview` |
| `Context` | 额外上下文，如 mentions、conversationType |
| `TraceID` | 跨服务追踪 ID |
| `RequestID` | Gateway 请求 ID |

规则：

- `RunID` 必须存在。
- `ThreadID` 必须存在。
- MVP v0.1 中 `AgentName` 应明确指向 `code-agent` 或由 Gateway 设置默认值。
- `Tools` 必须传给 Orchestrator，用于判断前端是否支持 `code_preview`。
- `History` 由 Gateway 提供，Orchestrator 不直接查询会话数据库。
- 字段命名对外 JSON 使用 camelCase；Go 内部结构可以使用 PascalCase，但 json tag 必须统一。

---

## 10. EventSink / EventChan

推荐接口：

```go
type EventSink interface {
    Emit(ctx context.Context, event AGUIEvent) error
}
```

或：

```go
type EventChan chan<- AGUIEvent
```

规则：

- Orchestrator 只向 `EventSink` 输出 AG-UI Event。
- `AGUIEvent` 必须遵守 `agui-event-contract`。
- Orchestrator 不直接调用 `c.SSEvent`。
- Orchestrator 不直接持有 Gin `Context`。
- EventSink 必须支持客户端取消和 context 超时。
- Gateway 负责把 EventSink 中的事件转成 SSE。
- EventSink 中不能传 A2A 原始事件给 Frontend。
- EventSink 中不能传 Gateway-Orchestrator 私有调试对象给 Frontend。

---

## 11. OrchestratorResult

推荐结构：

```go
type OrchestratorResult struct {
    RunID      string
    ThreadID   string
    Status     string

    AssistantMessage string
    Artifacts        []ArtifactRef

    AgentName string
    TaskID    string

    StartedAt  time.Time
    FinishedAt time.Time
}
```

字段说明：

| 字段 | 说明 |
|---|---|
| `Status` | `completed` / `failed` / `cancelled` |
| `AssistantMessage` | Agent 最终文本回复 |
| `Artifacts` | 产物引用或元数据 |
| `AgentName` | 实际执行的 Agent |
| `TaskID` | A2A Task ID |
| `StartedAt` / `FinishedAt` | 运行时间 |

规则：

- Gateway 负责把 `AssistantMessage` 和 `Artifacts` 保存为消息记录。
- Orchestrator 可以在运行中聚合文本，但不直接写数据库。
- Artifact 具体 schema 由 `artifact-contract` 定义。
- MVP v0.1 中 Artifact 可先以内联 metadata 或 JSON 字段持久化，但不能塞进 `TEXT_MESSAGE_CONTENT`。
- 如果运行失败，`Status` 必须为 `failed`，并且已经或即将输出 `RUN_ERROR`。

---

## 12. 错误模型

推荐结构：

```go
type OrchestratorError struct {
    Code      string
    Message   string
    Cause     error
    Retryable bool
}
```

推荐错误码：

```text
ORCHESTRATOR_INVALID_REQUEST
ORCHESTRATOR_AGENT_NOT_FOUND
ORCHESTRATOR_A2A_CONNECT_FAILED
ORCHESTRATOR_A2A_STREAM_FAILED
ORCHESTRATOR_CONVERTER_FAILED
ORCHESTRATOR_TIMEOUT
ORCHESTRATOR_CANCELLED
ORCHESTRATOR_INTERNAL
```

错误处理规则：

- Orchestrator 内部错误不能直接泄漏堆栈给 Frontend。
- Gateway 应将可展示错误映射为 AG-UI `RUN_ERROR`。
- 如果 `Orchestrator.Process` 返回错误，但尚未输出 `RUN_ERROR`，Gateway 必须补发 `RUN_ERROR`。
- 如果已输出 `RUN_ERROR`，Gateway 不应重复发送冲突的错误事件。
- MVP v0.1 不强制自动 fallback 到备用 Agent。
- Post-MVP 可以通过 `STATE_UPDATE` 表示 retrying / fallback 状态。

---

## 13. Context、取消与超时

Gateway 调用 Orchestrator 时必须传入 `context.Context`。

规则：

- 浏览器断开 SSE 时，Gateway 必须取消 context。
- 用户取消 run 时，Gateway 必须取消 context 或调用 cancel contract。
- Orchestrator 必须把 context 传给 A2A Client。
- A2A Client 必须支持 context 取消。
- LLM / Agent 长时间无响应时应触发 timeout。
- timeout 应转成 `RUN_ERROR` 或 `Status=failed`。
- 不能产生 goroutine 泄漏。
- 不能在 context cancelled 后继续向 EventSink 写事件。
- MVP v0.1 可以只支持浏览器断开触发取消，不强制实现完整 `/api/agui/run/{runId}/cancel`。

---

## 14. Trace 与日志

Gateway ↔ Orchestrator 内部调用必须支持：

```text
traceId
requestId
runId
threadId
taskId
agentName
```

规则：

- Gateway 生成或透传 `traceId`。
- Gateway 将 `traceId` 写入 `OrchestratorRequest`。
- Orchestrator 调 A2A 时继续透传。
- 日志中必须包含 `runId` 和 `traceId`。
- 日志中不能打印 Authorization token。
- 日志中不能打印完整敏感 prompt 或 API key。
- 错误日志必须能定位 Gateway、Orchestrator、A2A、Child Agent 哪一层失败。

---

## 15. A2A 调用边界

Orchestrator 是唯一允许调用 Child Agent A2A endpoint 的系统角色。

Gateway 不允许直接调用：

```text
/a2a/tasks/send
/a2a/tasks/sendSubscribe
/a2a/tasks/{id}
/a2a/tasks/{id}/cancel
/.well-known/agent.json
```

MVP v0.1 中，Orchestrator 调用：

```text
POST /a2a/tasks/sendSubscribe
```

目标 Agent：

```text
code-agent
```

规则：

- A2A Client 属于 Orchestrator 边界。
- A2A Stream Event 不得直接透传给 Gateway / Frontend。
- A2A Event 必须先经过 ProtocolConverter。
- A2A Artifact 必须经过 Artifact → Frontend Skill 映射。
- A2A 错误必须转换为 OrchestratorError 或 AG-UI `RUN_ERROR`。

---

## 16. ProtocolConverter 边界

ProtocolConverter 属于 Orchestrator 边界。

它负责：

- A2A `status: working` → AG-UI `TEXT_MESSAGE_START`
- A2A `text` → AG-UI `TEXT_MESSAGE_CONTENT`
- A2A `artifact` → 缓存 Artifact
- A2A `status: completed` → `TEXT_MESSAGE_END` → flush artifacts → `TOOL_CALL_*` → `RUN_FINISHED`
- A2A `status: failed` → `RUN_ERROR`
- `code` Artifact → `code_preview` Tool Call

它不负责：

- 写 SSE。
- 保存 DB。
- 处理用户鉴权。
- 查询会话列表。
- 渲染前端组件。
- 定义 Artifact schema。
- 定义 Frontend Skill schema。

---

## 17. MVP v0.1 直接路由规则

MVP v0.1 不调用 LLM 做意图编排，使用直接路由。

直接路由输入来源：

- 用户新建对话时选择的 Agent。
- `AGUIRunRequest.agentName`。
- Gateway 根据 conversation 绑定的 `agentName` 设置。
- 默认 `code-agent`。

规则：

- 如果没有指定 Agent，MVP 可以默认 `code-agent`。
- 如果指定了不存在的 Agent，返回 `ORCHESTRATOR_AGENT_NOT_FOUND` 并输出 `RUN_ERROR`。
- MVP 不实现多 Agent parallel / sequential。
- MVP 不实现 Agent fallback。
- MVP 不实现 LLM ExecutionPlan。
- 这些能力保留给 `intent-orchestration-contract` 和 Post-MVP。

---

## 18. Post-MVP 编排扩展

Post-MVP 中，Orchestrator 可以扩展：

- LLM 意图分析。
- AgentCard 读取。
- ExecutionPlan 生成。
- single / parallel / sequential 策略。
- 多 Agent 结果聚合。
- fallback / retry。
- 群聊 activeAgent 状态切换。
- `STATE_UPDATE` 编排状态输出。

扩展时必须保持：

- Gateway-Orchestrator Contract 向后兼容。
- Gateway handler 不承担编排。
- Orchestrator 输出仍然是 AG-UI Event。
- A2A 调用仍然只在 Orchestrator 中。
- ExecutionPlan schema 由 `intent-orchestration-contract` 定义。
- AgentCard schema 由 `a2a-agent-contract` 定义。

---

## 19. Mock-first 规则

在真实 A2A / LLM 完成前，允许使用 mock Orchestrator。

Mock Orchestrator 必须：

- 接收真实形态的 `OrchestratorRequest`。
- 输出符合 `agui-event-contract` 的 AG-UI Event。
- 能模拟文本流。
- 能模拟 `code` Artifact → `code_preview` Tool Call。
- 能模拟 `RUN_ERROR`。
- 不返回未定义事件。
- 不绕过 EventSink。
- 不把 A2A mock event 直接发给 Frontend。

推荐 MVP mock 事件链：

```text
RUN_STARTED
TEXT_MESSAGE_START
TEXT_MESSAGE_CONTENT
TEXT_MESSAGE_CONTENT
TEXT_MESSAGE_END
TOOL_CALL_START
TOOL_CALL_ARGS
TOOL_CALL_END
RUN_FINISHED
```

---

## 20. Contract Test 规则

Gateway-Orchestrator Contract 至少应验证：

- Gateway 是否构造了合法 `OrchestratorRequest`。
- `RunID` / `ThreadID` / `Tools` / `History` 是否正确传递。
- Orchestrator 是否通过 EventSink 输出事件。
- Orchestrator 是否不直接写 HTTP response。
- Gateway 是否把事件写成 SSE。
- A2A 原始事件是否没有直接暴露给 Frontend。
- Orchestrator 错误是否映射为 `RUN_ERROR`。
- context cancellation 是否能停止 Orchestrator。
- MVP 直接路由是否选择 `code-agent`。
- `code` Artifact 是否通过 Converter 转为 `code_preview`。

---

## 21. 安全规则

Gateway ↔ Orchestrator Contract 必须遵守：

- 不传递 Authorization 原始 token，除非内部协议明确需要。
- 不在日志中输出 token、API key、完整 system prompt。
- 不允许 Frontend 访问 `/internal/*`。
- 内部服务模式必须有 service-to-service 鉴权。
- Orchestrator 不直接信任 Frontend 传来的 AgentName，必须经过 Gateway / config 校验。
- Tools / Skills 必须来自前端声明并经过白名单校验。
- Artifact 内容在输出给 Frontend Skill 前要遵守 Artifact / Frontend Runtime Skills Contract。
- 错误信息不能泄漏内部路径、密钥、堆栈。

---

## 22. 禁止事项

禁止：

- Gateway handler 写复杂编排逻辑。
- Gateway handler 直接调用 Child Agent。
- Gateway handler 直接解析 A2A Stream。
- Gateway handler 直接处理 Artifact → `code_preview` 映射。
- Orchestrator 直接暴露给 Frontend。
- Orchestrator 直接写 SSE response。
- Orchestrator 直接做用户鉴权。
- Orchestrator 直接保存用户消息。
- A2A Event 直接透传给 Frontend。
- internal contract 写进 `docs/contracts/openapi.yaml`。
- Frontend 调用 `/internal/*`。
- MVP 阶段提前实现复杂多 Agent 编排。
- 错误信息泄漏 token、API key、堆栈。
- 跳过 Contract Review 直接写实现。

---

## 23. 完成定义

本 Contract 视为完成，当：

```text
docs/contracts/gateway-orchestrator.md
docs/contracts/gateway-orchestrator-events.md
```

已经明确：

- Gateway 与 Orchestrator 的职责边界。
- MVP v0.1 同进程 Go interface 模式。
- Post-MVP 独立服务 internal contract 模式。
- Gateway → Orchestrator 的请求字段。
- Orchestrator → Gateway 的事件输出方式。
- `OrchestratorResult`。
- 错误模型。
- 取消与超时。
- `traceId` / `runId` / `threadId`。
- AG-UI / A2A / REST 的边界。
- 硬性禁止事项。
- Review Checklist。


## 24. v1.1 对齐补充

### 24.1 MVP 与长期模型的兼容

- MVP v0.1 允许 Orchestrator 输出 AG-UI-compatible event，Gateway 负责 SSE 包装。
- Post-MVP / v1.1 推荐 Orchestrator 输出 `OrchestratorEvent`，再由 Gateway 映射为 AG-UI Event。

### 24.2 v1.1 推荐 internal endpoints

```text
POST /internal/runs
GET  /internal/runs/{runId}/events
POST /internal/runs/{runId}/tool-result
POST /internal/runs/{runId}/cancel
```

说明：`/internal/orchestrator/runs` 为兼容或备选路径，推荐新实现采用 `/internal/runs*`。

### 24.3 approval.required 关系

`approval.required` 用于高危操作确认流程；confirm_action 参数、ToolResult、安全策略与审批持久化分别由后续对应 Skill 细化。

