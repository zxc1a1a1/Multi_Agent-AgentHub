---
name: gateway-orchestrator-contract
description: Use when changing Gateway-to-Orchestrator internal contracts, OrchestratorEvent mapping, run lifecycle, internal endpoints, tool results, or cancellation flow.
---

# gateway-orchestrator-contract

## 1. Skill 目的

本 Skill 用于定义 AgentHub 项目中 **Gateway ↔ Orchestrator** 的内部调用契约。

它约束 Gateway 如何把来自 Frontend 的 AG-UI Run 请求交给 Orchestrator 处理，以及 Orchestrator 如何把编排结果、A2A 调度结果、协议转换后的 AG-UI Event 返回给 Gateway。

一句话：

**Gateway 负责对外入口和连接管理，Orchestrator 负责路由 / 编排 / A2A 调度 / 协议转换；二者之间必须有清晰的内部 Contract。**

---

## 2. 适用场景

当任务涉及以下内容时，必须使用本 Skill：

- 编写或修改 Gateway 调用 Orchestrator 的逻辑。
- 编写或修改 `server/internal/orchestrator/`。
- 编写或修改独立 `orchestrator-service`。
- 设计 Gateway ↔ Orchestrator 的 Go interface。
- 设计 Gateway ↔ Orchestrator 的 internal HTTP contract。
- 设计 `OrchestratorRequest`。
- 设计 `OrchestratorResult`。
- 设计 Orchestrator 输出给 Gateway 的事件通道。
- 设计 Gateway 如何向 Frontend 转发 AG-UI SSE。
- 设计 Orchestrator 如何调用 A2A Client。
- 设计 MVP 中 Gateway 与 Orchestrator 合并进程时的模块边界。
- 设计 Post-MVP 中 Gateway 与 Orchestrator 拆分服务时的内部协议。
- Review Gateway handler 是否越界实现编排逻辑。
- Review Orchestrator 是否越界处理 HTTP、鉴权、前端连接、数据库持久化。

---

## 3. Contract 所属边界

本 Skill 只约束：

```text
Gateway ↔ Orchestrator
```

本 Skill 不约束：

```text
Frontend ↔ Gateway REST API
Frontend ↔ Gateway AG-UI Event Stream
Orchestrator ↔ Child Agent A2A
Artifact Schema
Frontend Runtime Skills Schema
ADK Runtime
```

对应关系如下：

| 通信方向 | 使用协议 / Contract | 是否由本 Skill 管 |
|---|---|---|
| Frontend → Gateway REST API | OpenAPI | 否，由 `platform-api-contract` 管 |
| Frontend ↔ Gateway AG-UI SSE | AG-UI Event Contract | 否，由 `agui-event-contract` 管 |
| Gateway ↔ Orchestrator | Go interface / internal contract | 是 |
| Orchestrator ↔ Child Agent | A2A | 否，由 `a2a-agent-contract` 管 |
| Orchestrator 输出 Artifact | Artifact Contract | 否，由 `artifact-contract` 管 |
| Artifact → Frontend Skill | Frontend Runtime Skills Contract | 否，由 `frontend-runtime-skills-contract` 管 |

---

## 4. 核心文件

本 Skill 落地后应生成或维护：

```text
docs/contracts/gateway-orchestrator.md
docs/contracts/gateway-orchestrator-events.md
```

可选维护：

```text
docs/contracts/gateway-orchestrator.schema.json
docs/contracts/gateway-orchestrator-review-checklist.md
```

MVP v0.1 阶段，核心实现可以是 Go interface，不一定需要真实 HTTP internal endpoint。

Post-MVP 阶段，如果拆分独立 Orchestrator Service，则必须将同一逻辑 Contract 映射为 internal HTTP / RPC contract。

---

## 5. 四份设计文档的优先级解释

本 Skill 必须同时遵守四类文档：

1. **PDR**：定义完整目标架构，Gateway 和 Orchestrator 是清晰分层的两个系统角色。
2. **MVP 文档**：定义 v0.1 最小实施范围，允许 Orchestrator 嵌入 Gateway 进程。
3. **UML 文档**：定义关键链路、时序、协议转换和模块关系。
4. **Skills 设计规范**：定义本 Contract 必须自建，不能交给通用后端 Skill 代替。

解释原则：

```text
PDR 决定长期方向。
MVP 决定当前范围。
UML 决定关键流程。
Skills 设计规范决定 AI 开发约束。
```

---

## 6. MVP v0.1 实施模式

MVP v0.1 中，Orchestrator 允许嵌入 Gateway 进程，作为 `server/internal/orchestrator/` 模块存在。

推荐目录：

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

MVP v0.1 的调用关系：

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
- 不调用 LLM 生成 ExecutionPlan。
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
- Orchestrator 直接访问 React UI 或前端组件。

---

## 7. Post-MVP 完整模式

Post-MVP 可以将 Orchestrator 拆分为独立服务：

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

拆分后必须保持：

- 逻辑字段不变。
- `OrchestratorRequest` 语义不变。
- Orchestrator 输出事件语义不变。
- AG-UI Event 仍然遵守 `agui-event-contract`。
- A2A 调用仍然遵守 `a2a-agent-contract`。
- Gateway 仍然不做编排。
- Orchestrator 仍然不暴露给 Frontend。
- 内部接口必须有服务间鉴权、traceId、timeout、错误映射。

如果 Post-MVP 使用 internal HTTP，推荐路径形态：

```text
POST /internal/orchestrator/runs
POST /internal/orchestrator/runs/{runId}/cancel
```

注意：

- 这些 internal endpoint 不属于 Frontend REST API。
- 不能写进 `docs/contracts/openapi.yaml` 的 Frontend Platform API 中。
- 应由 `gateway-orchestrator-contract` 独立维护。
- 不允许 Frontend 调用 `/internal/*`。

---

## 8. Gateway 职责

Gateway 在本 Contract 中负责：

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
13. 统一日志、requestId、traceId。
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

## 9. Orchestrator 职责

Orchestrator 在本 Contract 中负责：

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

## 10. MVP v0.1 必须支持的内部调用流程

MVP v0.1 的最小流程：

```text
1. Gateway 接收 POST /api/agui/run
2. Gateway 鉴权
3. Gateway 保存用户消息
4. Gateway 查询最近历史消息
5. Gateway 构造 OrchestratorRequest
6. Gateway 创建 EventSink
7. Gateway 调用 Orchestrator.Process
8. Orchestrator 输出 RUN_STARTED
9. Orchestrator 选择 code-agent
10. Orchestrator 调用 A2A sendSubscribe
11. Orchestrator 接收 A2A status/text/artifact/completed
12. Orchestrator 转换为 AG-UI Event
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
- `cancel` 完整实现。
- 多路并发 Agent 聚合。

---

## 11. OrchestratorRequest 规范

MVP v0.1 推荐的逻辑结构：

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

字段含义：

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

## 12. EventSink 规范

MVP v0.1 推荐使用事件通道或接口：

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

## 13. OrchestratorResult 规范

Orchestrator 完成后应返回结构化结果，供 Gateway 持久化。

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

字段含义：

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

## 14. 错误模型

内部错误推荐结构：

```go
type OrchestratorError struct {
    Code    string
    Message string
    Cause   error
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

## 15. Context、取消与超时

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

MVP v0.1 可以只支持浏览器断开触发取消，不强制实现完整 `/api/agui/run/{runId}/cancel`。

---

## 16. Trace 与日志

Gateway ↔ Orchestrator 内部调用必须支持追踪字段：

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

## 17. AG-UI Event 输出规则

Orchestrator 输出给 Gateway 的事件必须遵守 `agui-event-contract`。

MVP v0.1 必须事件链：

```text
RUN_STARTED
TEXT_MESSAGE_START
TEXT_MESSAGE_CONTENT*
TEXT_MESSAGE_END
TOOL_CALL_START
TOOL_CALL_ARGS
TOOL_CALL_END
RUN_FINISHED
```

错误时：

```text
RUN_ERROR
```

规则：

- `TEXT_MESSAGE_CONTENT` 只传文本 chunk。
- 大代码、大网页、大文件不能塞进 `TEXT_MESSAGE_CONTENT`。
- Artifact 必须映射为 `TOOL_CALL_*` 或 Artifact Contract。
- `code` Artifact 必须映射为 `code_preview`。
- `TOOL_CALL_ARGS` 可以分片，但必须通过 `toolCallId` 聚合。
- Gateway 不应该修改 Orchestrator 输出的事件语义。
- Gateway 可以添加 SSE 包装，但不能改变 event payload 字段。

---

## 18. A2A 调用边界

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

A2A 调用规则由 `a2a-agent-contract` 细化，本 Skill 只规定：

- A2A Client 属于 Orchestrator 边界。
- A2A Stream Event 不得直接透传给 Gateway / Frontend。
- A2A Event 必须先经过 ProtocolConverter。
- A2A Artifact 必须经过 Artifact → Frontend Skill 映射。
- A2A 错误必须转换为 OrchestratorError 或 AG-UI `RUN_ERROR`。

---

## 19. ProtocolConverter 边界

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

## 20. Gateway Handler 边界

Gateway handler 可以做：

```text
HTTP parse
auth
request validate
DB save user message
DB load history
create eventSink
call orchestrator
write SSE
persist result
```

Gateway handler 不可以做：

```text
intent planning
A2A client call
A2A stream parse
artifactBuffer
artifact → code_preview mapping
multi-agent orchestration
result aggregation
LLM call
```

判断标准：

如果代码在回答“应该调用哪个 Agent / 如何拆任务 / 如何处理 A2A artifact / 如何转换成 Tool Call”，它应该在 Orchestrator 或 Converter 中，不应该在 Gateway handler 中。

---

## 21. MVP v0.1 直接路由规则

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

## 22. Post-MVP 编排规则

Post-MVP 中，Orchestrator 可以扩展：

- LLM 意图分析。
- AgentCard 读取。
- ExecutionPlan 生成。
- single / parallel / sequential 策略。
- 多 Agent 结果聚合。
- fallback / retry。
- 群聊 activeAgent 状态切换。
- `STATE_UPDATE` 编排状态输出。

但扩展时必须保持：

- Gateway-Orchestrator Contract 向后兼容。
- Gateway handler 不承担编排。
- Orchestrator 输出仍然是 AG-UI Event。
- A2A 调用仍然只在 Orchestrator 中。
- ExecutionPlan schema 由 `intent-orchestration-contract` 定义。
- AgentCard schema 由 `a2a-agent-contract` 定义。

---

## 23. 内部 HTTP 模式规范

如果 Post-MVP 将 Orchestrator 拆成独立服务，internal HTTP contract 推荐：

```text
POST /internal/orchestrator/runs
```

Request body 逻辑等价于：

```text
OrchestratorRequest
```

Response 有两种可选模式：

### 模式一：Gateway 仍然负责 SSE，Orchestrator 返回内部事件流

```text
Gateway → Orchestrator internal stream
Orchestrator → Gateway AG-UI event stream
Gateway → Frontend SSE
```

### 模式二：Gateway 通过消息通道接收事件

```text
Gateway → Orchestrator start run
Orchestrator → internal event bus
Gateway → subscribe events → Frontend SSE
```

无论哪种模式：

- Frontend 不得直接连接 Orchestrator。
- Orchestrator internal stream 不等于 AG-UI public endpoint。
- 内部接口必须有 service token。
- 内部接口必须支持 traceId。
- 内部接口必须支持 timeout。
- 内部接口不能写入 `docs/contracts/openapi.yaml` 的 Frontend API 范围。

---

## 24. Mock-first 规则

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

## 25. Contract Test 规则

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

## 26. 安全规则

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

## 27. 与其他 Skills 的协作

### 27.1 与 project-architecture

`project-architecture` 定义服务边界。  
本 Skill 细化 Gateway 与 Orchestrator 的内部边界。

如果发现 Gateway handler 写了编排逻辑，必须拒绝。

---

### 27.2 与 platform-api-contract

`platform-api-contract` 只管 Frontend ↔ Gateway REST API。  
本 Skill 不允许把 `/internal/orchestrator/*` 写成前端 API。

---

### 27.3 与 agui-event-contract

Orchestrator 通过本 Contract 输出的事件必须遵守 `agui-event-contract`。  
本 Skill 不重新定义 AG-UI event schema，只引用其语义。

---

### 27.4 与 a2a-agent-contract

Orchestrator 调用 Child Agent 的具体 A2A request / response / AgentCard 由 `a2a-agent-contract` 定义。  
本 Skill 只规定 A2A 调用属于 Orchestrator 边界。

---

### 27.5 与 intent-orchestration-contract

MVP v0.1 中不实现复杂意图编排。  
Post-MVP 的 ExecutionPlan、TaskPlan、routing strategy 由 `intent-orchestration-contract` 定义。

---

### 27.6 与 artifact-contract

Artifact 的字段、类型、metadata 由 `artifact-contract` 定义。  
本 Skill 只规定 Artifact 不能直接塞进文本流，必须通过 Converter 映射为 Tool Call 或 Artifact 引用。

---

### 27.7 与 frontend-runtime-skills-contract

`code_preview`、`web_preview` 等前端 Skill 参数由 `frontend-runtime-skills-contract` 定义。  
本 Skill 只规定 Orchestrator / Converter 必须按照已注册 Skill 构造 Tool Call。

---

## 28. 硬性规则

Coding Agent 在处理 Gateway ↔ Orchestrator 相关任务时必须遵守：

1. Gateway 对外，Orchestrator 对内编排。
2. Gateway handler 不允许写复杂编排逻辑。
3. Gateway handler 不允许直接调用 Child Agent。
4. Orchestrator 不允许直接暴露给 Frontend。
5. Orchestrator 不允许直接写 SSE response。
6. Orchestrator 不允许直接处理用户鉴权。
7. Orchestrator 是唯一允许调用 A2A Client 的模块。
8. ProtocolConverter 属于 Orchestrator 边界。
9. A2A Event 不得直接透传给 Frontend。
10. Orchestrator 输出给 Gateway 的事件必须符合 `agui-event-contract`。
11. Gateway 负责将 AG-UI Event 写成 SSE。
12. Gateway 负责保存用户消息和 Agent 回复。
13. Orchestrator 不直接查询会话列表。
14. MVP v0.1 可以把 Orchestrator 嵌入 Gateway 进程，但不能合并职责。
15. MVP v0.1 必须优先跑通 `code-agent + code_preview`。
16. MVP v0.1 不提前实现复杂多 Agent 编排，除非用户明确要求。
17. Post-MVP 拆分独立 Orchestrator Service 时，内部接口不能暴露给 Frontend。
18. Internal contract 不属于 `docs/contracts/openapi.yaml` 的 Frontend REST API。
19. 所有内部调用必须携带或生成 `runId`、`threadId`、`traceId`。
20. 所有错误必须可映射为 `RUN_ERROR`。
21. 必须遵守 `Contract first / Mock first / Real integration later / Review always`。

---

## 29. 必须维护的文件

使用本 Skill 时，至少需要维护：

```text
skills/gateway-orchestrator-contract/SKILL.md
docs/contracts/gateway-orchestrator.md
docs/contracts/gateway-orchestrator-events.md
```

根据需要维护：

```text
docs/contracts/gateway-orchestrator.schema.json
docs/contracts/gateway-orchestrator-review-checklist.md
server/internal/orchestrator/types.go
server/internal/orchestrator/orchestrator.go
server/internal/orchestrator/converter.go
server/internal/a2a/client.go
server/internal/handler/agui.go
```

MVP v0.1 阶段不要求马上生成业务代码。  
如果用户只要求 Contract，则不要创建 Go 实现。

---

## 30. 输出要求

当用户要求设计 Gateway ↔ Orchestrator Contract 时，Coding Agent 必须输出：

1. 当前属于 MVP 同进程模式还是 Post-MVP 独立服务模式。
2. Gateway 职责。
3. Orchestrator 职责。
4. `OrchestratorRequest` 字段。
5. EventSink / EventChan 规则。
6. `OrchestratorResult` 字段。
7. 错误模型。
8. context cancellation / timeout 规则。
9. traceId / runId / threadId 追踪规则。
10. AG-UI Event 输出边界。
11. A2A 调用边界。
12. Gateway handler 禁止事项。
13. MVP v0.1 直接路由规则。
14. Post-MVP 扩展规则。
15. Review Checklist。

除非用户明确要求，不要直接生成 Gateway / Orchestrator 业务实现代码。

---

## 31. Review Checklist

在接受任何 Gateway ↔ Orchestrator 设计或实现前，必须检查：

### 文件与 Contract

- 是否有 `docs/contracts/gateway-orchestrator.md`？
- 是否有 `docs/contracts/gateway-orchestrator-events.md`？
- 是否明确 MVP 同进程模式？
- 是否明确 Post-MVP 独立服务模式？
- 是否没有把 internal contract 写进 Frontend OpenAPI？

### Gateway 边界

- Gateway 是否只负责 HTTP / SSE / auth / session / persistence？
- Gateway 是否没有做意图编排？
- Gateway 是否没有直接调用 Child Agent？
- Gateway 是否没有直接解析 A2A Artifact？
- Gateway 是否通过 Orchestrator 调用 A2A？
- Gateway 是否负责把 AG-UI Event 写成 SSE？

### Orchestrator 边界

- Orchestrator 是否接收 `OrchestratorRequest`？
- Orchestrator 是否输出 AG-UI Event？
- Orchestrator 是否通过 A2A Client 调 Child Agent？
- Orchestrator 是否没有直接写 HTTP response？
- Orchestrator 是否没有直接管理前端连接？
- Orchestrator 是否没有直接做用户鉴权？
- Orchestrator 是否没有直接依赖 React UI？

### MVP v0.1 检查

- 是否支持 Orchestrator 嵌入 Gateway 进程？
- 是否保持 handler / orchestrator / a2a / converter 模块边界？
- 是否直接路由到 `code-agent`？
- 是否不要求 LLM ExecutionPlan？
- 是否只要求 `code` Artifact → `code_preview`？
- 是否没有提前实现群聊和复杂多 Agent？

### 事件与协议

- Orchestrator 输出事件是否符合 `agui-event-contract`？
- A2A Event 是否没有直接暴露给 Frontend？
- Artifact 是否没有塞进 `TEXT_MESSAGE_CONTENT`？
- `RUN_ERROR` 是否覆盖失败场景？
- context cancellation 是否能停止 A2A 调用？

### 安全与可观测性

- 是否有 `traceId`？
- 是否有 `runId`？
- 是否有 `threadId`？
- 是否不打印 token / API key？
- Post-MVP internal endpoint 是否不暴露给 Frontend？
- 错误信息是否不泄漏内部堆栈？

---

## 32. 完成定义

本 Skill 视为完成，当且仅当：

```text
skills/gateway-orchestrator-contract/SKILL.md
```

已经明确：

- Gateway 与 Orchestrator 的职责边界。
- MVP v0.1 同进程 Go interface 模式。
- Post-MVP 独立服务 internal contract 模式。
- Gateway → Orchestrator 的请求字段。
- Orchestrator → Gateway 的事件输出方式。
- OrchestratorResult。
- 错误模型。
- 取消与超时。
- traceId / runId / threadId。
- AG-UI / A2A / REST 的边界。
- 硬性规则。
- Review Checklist。

正式落地时还应生成：

```text
docs/contracts/gateway-orchestrator.md
docs/contracts/gateway-orchestrator-events.md
```


## 33. v1.1 对齐补充

### 33.1 MVP v0.1 简化保留

- MVP v0.1 中，Orchestrator 可以在 Gateway 同进程内输出 AG-UI-compatible event，以降低实现复杂度。
- Gateway 负责 SSE 包装和对外推送。
- 这只是 MVP 简化，不代表长期唯一架构。

### 33.2 Post-MVP / v1.1 长期标准

- Post-MVP / v1.1 标准中，Orchestrator 应输出内部 `OrchestratorEvent`。
- Gateway 负责将 `OrchestratorEvent` 映射成 AG-UI Event。
- Gateway 仍不承担复杂编排。
- Orchestrator 仍不直接暴露给 Frontend。

### 33.3 v1.1 推荐 internal endpoints

```text
POST /internal/runs
GET  /internal/runs/{runId}/events
POST /internal/runs/{runId}/tool-result
POST /internal/runs/{runId}/cancel
```

兼容说明：`/internal/orchestrator/runs` 可作为早期语义化备选或兼容路径；v1.1 推荐标准路径为 `/internal/runs*`。

### 33.4 tool-result / cancel 与 approval.required

- MVP v0.1 可暂不实现 tool-result / cancel 的完整 HTTP 化接口。
- Post-MVP 引入交互式 Frontend Runtime Skill、confirm_action、取消运行后，应补齐这些 internal endpoints。
- `approval.required` 表示 Orchestrator 需要用户确认高危操作。
- `confirm_action` 参数与 ToolResult 由后续 `frontend-runtime-skills-contract` 细化。
- 高危操作安全策略由后续 `security-boundary-contract` 细化。
- 审批持久化由后续 `data-persistence-contract` 细化。



## References

- `references/orchestrator-event-policy.md`
- `references/run-lifecycle.md`
- `references/cancellation-policy.md`
- `references/gateway-agui-mapping.md`
- `references/gateway-orchestrator-review-checklist.md`
