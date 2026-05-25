---
name: gateway-orchestrator-contract
description: "用于定义 AgentHub Gateway Service 与 Orchestrator Service 的独立进程间通信契约，包括服务职责边界、内部 API、流式事件、编排计划、多 Agent 执行、fallback/retry、取消超时、服务间鉴权、trace 传播和错误脱敏。"
---

# gateway-orchestrator-contract

## 1. Skill 目的

本 Skill 用于定义 AgentHub 中：

```text
Gateway Service ↔ Orchestrator Service
```

之间的独立进程间通信契约。

它约束：

- Gateway Service 与 Orchestrator Service 的职责边界。
- Gateway 如何通过内部 API 调用 Orchestrator。
- Orchestrator 如何以稳定的流式事件返回编排过程。
- Run 生命周期如何跨进程表达。
- OrchestrationPlan / TaskPlan 如何表达多 Agent 编排。
- single / ordered_parallel / sequential 如何执行和输出事件。
- fallback / retry 如何表达。
- 浏览器断连、用户取消、服务超时如何传播。
- requestId / traceId / runId 如何贯穿 Gateway 与 Orchestrator。
- 服务间鉴权、错误脱敏和日志边界。

一句话：

**Gateway 是对外入口；Orchestrator 是内部编排服务；两者必须分进程，通过受保护的内部 API 通信，不得通过同进程 import 或 handler 内嵌逻辑绕过服务边界。**

---

## 2. 独立性原则

本 Skill 必须独立可读，不要求读者先阅读其他 Skill。

本 Skill 只定义 Gateway 与 Orchestrator 的服务间契约，不展开前端事件协议、子 Agent 协议、Artifact Schema、数据库表结构、LLM Provider API 或具体 Agent 实现。

可以出现以下概念，但只作为内部字段或边界名使用：

- Agent
- Run
- Task
- Event
- ArtifactRef
- ToolCallRef
- RuntimeCapability

不得在本 Skill 中复制其他协议的完整字段定义。

---

## 3. 当前阶段识别

当前项目已经完成 MVP v0.1，MVP 规则仅作为历史回归基线。

当前阶段要求：

```text
Gateway Service 与 Orchestrator Service 必须分进程。
```

当前契约面向：

- 2+ Child Agent。
- 单聊与群聊。
- 手动指定、@mention、规则路由、LLM Planner 等多种 planning 来源。
- single / ordered_parallel / sequential 执行策略。
- fallback / retry。
- 多条 assistant message。
- 多个 agent task。
- 多个 artifact / tool call 引用。

本 Skill 不固定具体 Agent 名称。

禁止把 `code-agent`、`web-agent`、`doc-agent` 等具体 Agent 名称写成契约硬编码。它们只能出现在示例中。

---

## 4. MVP v0.1 Historical Profile

MVP v0.1 已完成，仅作为历史兼容和回归测试基线。

历史基线包括：

- Gateway 与 Orchestrator 可以同进程。
- Orchestrator 可以作为 `server/internal/orchestrator` 模块。
- 单 Agent 直接路由。
- 最小文本流。
- 最小 preview/tool call 映射。
- Gateway handler 可以直接调用同进程 Orchestrator 对象。

这些历史实现不得继续作为当前架构约束。

当前新增开发必须朝向：

- Gateway 独立服务。
- Orchestrator 独立服务。
- Gateway 通过 `ORCHESTRATOR_URL` 访问 Orchestrator。
- Orchestrator 暴露内部 `/health` 与内部 streaming endpoint。
- Gateway 不再 import Orchestrator 业务包执行编排。

---

## 5. 进程边界硬规则

Gateway 和 Orchestrator 必须是两个独立进程。

硬性规则：

1. Gateway 不得 import Orchestrator 的业务包来执行编排逻辑。
2. Gateway 不得在 HTTP handler 中实现 Agent 选择、任务拆解、fallback 或多 Agent 调度。
3. Orchestrator 必须通过内部网络 endpoint 被 Gateway 调用。
4. Orchestrator 必须拥有独立启动入口、独立端口和独立 `/health`。
5. Frontend 不得直接访问 Orchestrator。
6. Child Agent 不得反向依赖 Gateway 的前端 API。
7. Orchestrator 的内部 endpoint 不属于对外公开 API。
8. Gateway 与 Orchestrator 的通信必须有 timeout、trace、错误脱敏和服务间鉴权。

目标拓扑：

```text
Frontend
  ↓ HTTP / SSE
Gateway Service
  ↓ Internal HTTP / Streaming RPC
Orchestrator Service
  ↓ Agent Client / Registry / Planner
Child Agent Services
```

---

## 6. 本 Skill 负责什么

本 Skill 负责：

- 服务职责边界。
- 服务间 API。
- 服务间鉴权。
- OrchestratorRequest。
- OrchestrationPlan。
- OrchestratorStreamEvent。
- OrchestratorResult。
- Run 生命周期。
- 多 Agent 执行规则。
- ordered_parallel 事件顺序规则。
- fallback / retry 规则。
- cancellation / timeout 跨进程传播规则。
- trace / requestId / runId 传播规则。
- Gateway handler 边界。
- Orchestrator service 边界。
- Mock-first 和 Contract Test 规则。

---

## 7. 本 Skill 不负责什么

本 Skill 不负责：

- 前端公开 REST API 的完整定义。
- 浏览器 SSE wire format 的完整规范。
- 子 Agent 协议字段。
- Agent Runtime 内部 API。
- Artifact 详细 schema。
- 前端 Runtime Capability 参数 schema。
- 数据库表结构。
- Docker Compose 文件实现。
- LLM Provider 请求格式。
- 某个具体 Agent 的业务逻辑。

如果这些信息需要被 Gateway 或 Orchestrator 使用，本 Skill 只定义它们在服务间 request / result 中的摘要字段或引用字段。

---

## 8. Gateway Service 职责

Gateway Service 是对外入口。

Gateway 负责：

- 接收前端 HTTP 请求。
- 接收和维护前端流式连接。
- 执行用户鉴权。
- 校验基础请求格式。
- 生成或透传 `requestId`、`traceId`、`runId`、`conversationId`。
- 保存用户消息。
- 查询必要历史上下文。
- 装配 `conversationType`、mentions、selectedAgents、runtimeCapabilities 等上下文。
- 构造 `OrchestratorRequest`。
- 使用 `ORCHESTRATOR_URL` 调用 Orchestrator Service。
- 将 Orchestrator 流式事件转发给 Frontend。
- 处理浏览器断连。
- 将断连、显式取消、服务超时传播给 Orchestrator。
- 持久化 `OrchestratorResult` 中的 assistant messages、task refs、artifact refs、tool call refs。
- 将内部错误转换成前端安全错误。

---

## 9. Gateway Service 禁止事项

Gateway 禁止：

- 直接选择具体 Agent。
- 直接调用 Child Agent。
- 直接调用 LLM。
- 直接生成 OrchestrationPlan。
- 直接执行 fallback / retry 决策。
- 直接解析 Child Agent 原始流。
- 直接做多 Agent 调度。
- 在 handler 中写复杂编排逻辑。
- import Orchestrator 业务包执行编排。
- 通过具体 `agentName` 写死能力判断。
- 把用户原始 Authorization token 当服务间 token 透传给 Orchestrator。
- 把内部错误堆栈直接返回前端。

判断标准：

```text
如果代码在回答“应该调用哪个 Agent、如何拆任务、如何 fallback、如何排序多 Agent 输出”，它不应该在 Gateway。
```

---

## 10. Orchestrator Service 职责

Orchestrator Service 是内部编排服务。

Orchestrator 负责：

- 独立启动。
- 暴露内部 `/health`。
- 暴露内部 run stream endpoint。
- 接收 Gateway 传入的 `OrchestratorRequest`。
- 读取用户消息、历史上下文、conversationType、mentions、runtimeCapabilities、availableAgents。
- 执行 planning。
- 生成或接收 `OrchestrationPlan`。
- 校验计划中的目标 Agent 是否可用。
- 生成一个或多个 AgentTask。
- 执行 single / ordered_parallel / sequential。
- 输出 Gateway 可转发的流式事件。
- 调用一个或多个 Child Agent。
- 聚合任务结果。
- 处理 fallback / retry。
- 生成 `OrchestratorResult`。
- 记录脱敏日志和 trace。

---

## 11. Orchestrator Service 禁止事项

Orchestrator 禁止：

- 直接处理前端用户登录鉴权。
- 直接暴露给 Frontend。
- 直接写浏览器 HTTP/SSE response。
- 直接持有 Gateway 的 HTTP framework context。
- 依赖 Gateway handler 类型。
- 反向调用 Gateway 的前端公开 API。
- 保存用户原始 Authorization token。
- 在结果中返回内部堆栈、密钥、私有路径或完整 system prompt。
- 使用具体 Agent 名称硬编码能力判断。

---

## 12. Service-to-Service API

Gateway 调用 Orchestrator 必须通过内部 API。

推荐最小 endpoint：

```text
GET  /health
POST /internal/orchestrator/runs/stream
POST /internal/orchestrator/runs/{runId}/cancel
```

可选 endpoint：

```text
GET  /internal/orchestrator/runs/{runId}
POST /internal/orchestrator/runs/{runId}/result
```

规则：

- `/health` 用于 Orchestrator 服务健康检查。
- `/internal/orchestrator/runs/stream` 用于创建并流式执行 run。
- `/internal/orchestrator/runs/{runId}/cancel` 用于跨进程取消。
- 所有 `/internal/*` endpoint 只允许 Gateway 或内部测试调用。
- 不得暴露给 Frontend。
- 不得写入前端公开 API 文档。
- 必须设置 timeout。
- 必须传播 trace headers。
- 必须使用服务间鉴权。

---

## 13. Service-to-Service Auth

Gateway 调 Orchestrator 必须携带服务间凭证。

v1 最小允许方案：

```text
Authorization: Bearer <internal-service-token>
```

推荐环境变量：

```text
Gateway:
- ORCHESTRATOR_URL
- ORCHESTRATOR_INTERNAL_TOKEN
- ORCHESTRATOR_TIMEOUT_MS

Orchestrator:
- ORCHESTRATOR_PORT
- INTERNAL_SERVICE_TOKEN
```

规则：

- Orchestrator 必须校验服务间 token。
- Frontend 不得持有服务间 token。
- 用户 token 不得作为服务间 token 透传。
- service token 不得写入日志。
- service token 不得写入 Dockerfile。
- service token 不得进入错误响应。
- Orchestrator 不得公网裸露。

长期可以升级为 mTLS 或更完整的服务身份机制。

---

## 14. Trace Headers

Gateway 调 Orchestrator 时必须携带请求链路信息。

推荐 headers：

```text
X-Request-Id: req_...
X-Trace-Id: trace_...
X-Run-Id: run_...
X-Conversation-Id: conv_...
X-Deadline-Ms: 120000
Authorization: Bearer <internal-service-token>
```

规则：

- `X-Trace-Id` 贯穿 Gateway、Orchestrator、Agent 调用和日志。
- `X-Request-Id` 代表本次外部请求。
- `X-Run-Id` 代表一次编排运行。
- `X-Deadline-Ms` 用于跨进程表达超时预算。
- 缺失 traceId 时，Gateway 必须生成。
- Orchestrator 不得覆盖 Gateway 传入的 traceId，除非它为空。

---

## 15. OrchestratorRequest

`OrchestratorRequest` 是 Gateway 发送给 Orchestrator 的 JSON 请求体。

推荐结构：

```json
{
  "runId": "run_001",
  "conversationId": "conv_001",
  "userId": "user_001",
  "conversationType": "group",
  "messages": [],
  "history": [],
  "availableAgents": [],
  "selectedAgentNames": [],
  "mentions": [],
  "runtimeCapabilities": [],
  "planningMode": "auto",
  "traceId": "trace_001",
  "requestId": "req_001",
  "deadlineMs": 120000,
  "metadata": {}
}
```

字段规则：

| 字段 | 说明 |
|---|---|
| `runId` | 本次编排运行 ID |
| `conversationId` | 会话 ID |
| `userId` | 用户 ID 或匿名用户标识 |
| `conversationType` | `single` 或 `group` |
| `messages` | 当前请求中的用户消息 |
| `history` | Gateway 裁剪后的历史上下文 |
| `availableAgents` | 当前可用 Agent 摘要 |
| `selectedAgentNames` | 用户手动选择的 Agent 名称，可为空 |
| `mentions` | 用户输入中提到的 Agent 名称，可为空 |
| `runtimeCapabilities` | 前端当前可处理的 runtime 能力摘要 |
| `planningMode` | `direct` / `mention` / `auto` / `manual` |
| `traceId` | 链路追踪 ID |
| `requestId` | 外部请求 ID |
| `deadlineMs` | 本次编排总超时预算 |
| `metadata` | 脱敏扩展字段 |

规则：

- JSON 字段使用 camelCase。
- 不传 Go 私有类型。
- 不传 HTTP request 对象。
- 不传 Gin / Echo / net/http context。
- 不传数据库连接对象。
- 不传 LLM API key。
- 不传用户原始 Authorization token。
- `availableAgents` 是摘要，不是完整内部对象。
- `runtimeCapabilities` 表示前端能力，不等价于 Agent 能力。
- `agentName` 如需兼容，只能作为 `selectedAgentNames[0]` 的 legacy alias。

---

## 16. PlanningMode

`planningMode` 表示 Orchestrator 如何生成计划。

允许值：

```text
direct
mention
auto
manual
```

含义：

| planningMode | 含义 |
|---|---|
| `direct` | 单聊或上下文已指定目标 Agent |
| `mention` | 用户输入中包含 `@agent-name` |
| `auto` | Orchestrator 自动规划，可使用规则或 LLM |
| `manual` | 前端或用户显式选择一个或多个 Agent |

规则：

- Gateway 可以提取 mentions，但不得执行复杂编排。
- Orchestrator 必须校验 mention 是否可用。
- direct / mention / manual 也必须生成统一的 OrchestrationPlan。
- auto 可以使用 LLM，也可以使用规则 fallback。
- 本 Skill 不规定具体 Planner 算法。

---

## 17. OrchestrationPlan

`OrchestrationPlan` 是 Orchestrator 的内部编排计划。

推荐结构：

```json
{
  "planId": "plan_001",
  "intentSummary": "用户希望完成一个多步骤任务",
  "strategy": "ordered_parallel",
  "createdBy": "auto",
  "tasks": [
    {
      "taskId": "task_001",
      "agentName": "some-agent",
      "capabilityIds": ["capability_id"],
      "taskContent": "给该 Agent 的任务说明",
      "dependsOn": [],
      "expectedOutputs": ["text"]
    }
  ],
  "fallback": {
    "mode": "same_capability_alternative",
    "maxAttempts": 2
  }
}
```

规则：

- Plan 可以由 LLM、规则、mention、manual selection 生成。
- Plan 必须校验后才能执行。
- `agentName` 只作为目标标识，不用于能力推断。
- `capabilityIds` 与 `expectedOutputs` 是能力摘要，不固定具体 Agent。
- `tasks` 可以有 1 个或多个任务。
- `strategy` 必须明确。
- Plan 不得包含 API key、token、完整 system prompt 或未脱敏敏感输入。

---

## 18. Execution Strategy

允许策略：

```text
single
ordered_parallel
sequential
```

### single

一个 run 只执行一个 TaskPlan。

### ordered_parallel

语义上包含多个独立 task，UI 可展示为多 Agent 参与。

规则：

- Orchestrator 可以内部并发执行，也可以顺序执行。
- 对 Gateway 输出的事件必须保持 message 粒度可聚合。
- 不要求不同 Agent token 级交错输出。
- 如果内部并发执行，Orchestrator 必须负责排序或 messageId 隔离。
- 每个 Agent 输出必须有独立 messageId。

### sequential

多个 task 存在依赖关系。

规则：

- 后续 task 可以使用前序 task 的脱敏摘要。
- 不得把完整敏感中间结果无条件传递给下游 task。
- 前序 task 失败时，应根据 fallback 策略决定是否继续。

---

## 19. OrchestratorStreamEvent

`OrchestratorStreamEvent` 是 Orchestrator 通过内部流式 endpoint 发送给 Gateway 的事件。

推荐事件类型：

```text
run_started
state_update
message_start
message_delta
message_end
tool_call_start
tool_call_args
tool_call_end
run_finished
run_error
```

推荐结构：

```json
{
  "type": "message_delta",
  "runId": "run_001",
  "messageId": "msg_001",
  "sender": {
    "type": "agent",
    "name": "some-agent"
  },
  "delta": "文本片段",
  "state": null,
  "toolCall": null,
  "error": null
}
```

规则：

- Orchestrator 只输出稳定的 `OrchestratorStreamEvent`。
- Gateway 不接收 Child Agent 原始流。
- Gateway 不改变事件业务语义。
- Gateway 可以把内部事件包装为前端传输格式。
- 所有事件必须包含 `runId`。
- message 类事件必须包含 `messageId`。
- 多 Agent 输出不得复用同一个 `messageId`。
- 错误事件必须使用脱敏 `SafeError`。

---

## 20. OrchestratorResult

`OrchestratorResult` 是 Orchestrator 对一次 run 的最终摘要。

推荐结构：

```json
{
  "runId": "run_001",
  "conversationId": "conv_001",
  "status": "completed",
  "strategy": "ordered_parallel",
  "intentSummary": "用户希望完成一个多 Agent 任务",
  "messages": [],
  "tasks": [],
  "artifacts": [],
  "toolCalls": [],
  "error": null,
  "startedAt": "2026-05-25T00:00:00Z",
  "finishedAt": "2026-05-25T00:00:03Z"
}
```

规则：

- 一个 result 可以包含多个 assistant messages。
- 一个 result 可以包含多个 tasks。
- 一个 result 可以包含多个 artifact refs。
- 一个 result 可以包含多个 tool call refs。
- result 不得返回内部堆栈。
- result 不得返回完整敏感 prompt。
- result 不得返回大对象内容，优先返回引用和摘要。
- Gateway 负责基于 result 做持久化。

---

## 21. Run Lifecycle

推荐生命周期：

```text
accepted
→ context_loaded
→ planning
→ plan_ready
→ dispatching
→ agent_task_running
→ agent_task_completed / agent_task_failed
→ retrying / fallback
→ aggregating
→ completed / failed / cancelled
```

跨服务流程：

```text
Gateway 接收请求
Gateway 鉴权和基础校验
Gateway 保存用户消息
Gateway 构造 OrchestratorRequest
Gateway 调用 Orchestrator stream endpoint
Orchestrator 输出 run_started
Orchestrator 加载上下文并规划
Orchestrator 输出 state_update
Orchestrator 执行一个或多个 task
Orchestrator 输出 message / tool / state 事件
Orchestrator 输出 run_finished 或 run_error
Gateway 持久化 OrchestratorResult
Gateway 关闭前端流
```

规则：

- 一个 run 可以产生多条 assistant message。
- 一个 run 可以执行多个 task。
- 一个 run 可以产生多个 artifact / tool call 引用。
- run 失败时必须返回安全错误。
- run 取消后不得继续输出普通事件。

---

## 22. Multi-Agent / Group Conversation Rules

多 Agent 与群聊场景必须遵守：

- Gateway 传入 `conversationType`。
- Gateway 可以传入 mentions 和 selectedAgentNames。
- Orchestrator 负责校验目标 Agent 是否可用。
- 一个 run 可以生成多个 AgentTask。
- 一个 run 可以生成多条 assistant message。
- 每条 assistant message 必须有稳定 `messageId`。
- 每条 assistant message 应包含 `sender.name`。
- Agent 名称只用于身份标识，不用于能力推断。
- 群聊中的 `@agent-name` 是路由提示，不是无校验执行命令。
- fallback 后必须记录实际执行 Agent。

---

## 23. Fallback / Retry Rules

fallback / retry 是当前通用编排能力。

推荐策略：

```text
none
same_capability_alternative
first_healthy_agent
fail_fast
```

规则：

- fallback 不得选择不可用 Agent。
- fallback 不得无限重试。
- `maxAttempts` 必须明确。
- retry / fallback 必须输出 state_update。
- fallback 后生成的 message / task / result 必须标记实际执行 Agent。
- 所有候选都失败时，run 必须进入 failed。
- 错误信息必须脱敏。

---

## 24. Cancellation / Timeout Across Processes

Gateway 与 Orchestrator 分进程后，取消语义必须跨网络传播。

Gateway 必须：

- 浏览器断连时取消本地 context。
- 如果 Orchestrator stream 正在进行，应关闭内部请求。
- 如果已有 runId，应调用 cancel endpoint 或等价取消机制。
- timeout 后不得继续向前端写普通事件。
- 取消后必须释放连接和 goroutine。

Orchestrator 必须：

- 检测内部请求断开。
- 收到 cancel 后停止未完成 task。
- 取消后不得继续启动新 task。
- 将取消信号传递给下游调用。
- 输出 cancelled 或 failed 的最终状态。
- 防止 goroutine / stream 泄漏。

推荐取消 endpoint：

```text
POST /internal/orchestrator/runs/{runId}/cancel
```

---

## 25. SafeError

所有跨服务错误必须脱敏。

推荐结构：

```json
{
  "code": "ORCHESTRATOR_AGENT_UNAVAILABLE",
  "message": "当前任务暂时无法完成，请稍后重试",
  "retryable": true,
  "details": null
}
```

禁止在错误中包含：

- API key。
- Authorization token。
- service token。
- 数据库连接串。
- 内部堆栈。
- 本地绝对路径。
- 内网拓扑。
- 完整 system prompt。
- 未脱敏 LLM 原始请求 / 响应。

推荐错误码：

```text
ORCHESTRATOR_BAD_REQUEST
ORCHESTRATOR_UNAUTHORIZED_SERVICE
ORCHESTRATOR_TIMEOUT
ORCHESTRATOR_CANCELLED
ORCHESTRATOR_PLANNING_FAILED
ORCHESTRATOR_AGENT_UNAVAILABLE
ORCHESTRATOR_AGENT_FAILED
ORCHESTRATOR_STREAM_INTERRUPTED
ORCHESTRATOR_INTERNAL
```

---

## 26. Deployment Config

Gateway 推荐环境变量：

```text
ORCHESTRATOR_URL=http://orchestrator:8090
ORCHESTRATOR_INTERNAL_TOKEN=change-me
ORCHESTRATOR_TIMEOUT_MS=120000
```

Orchestrator 推荐环境变量：

```text
ORCHESTRATOR_PORT=8090
INTERNAL_SERVICE_TOKEN=change-me
```

规则：

- Gateway 只能通过 `ORCHESTRATOR_URL` 访问 Orchestrator。
- 容器间通信必须使用 service name，不使用 localhost。
- Orchestrator 必须有独立 `/health`。
- Orchestrator 不得暴露给前端网络边界。
- 环境变量示例不得包含真实 token。

---

## 27. Mock-first Rules

在 Orchestrator Service 未完成真实实现前，可以使用 Mock Orchestrator Service。

Mock 必须：

- 作为独立进程启动。
- 暴露 `/health`。
- 暴露内部 stream endpoint。
- 校验 service token。
- 接收合法 OrchestratorRequest。
- 输出合法 OrchestratorStreamEvent。
- 支持至少一个 single run。
- 支持至少一个 ordered_parallel 示例 run。
- 支持 run_error 示例。
- 支持 cancel 示例。

Mock 禁止：

- 被 Gateway import 为同进程对象。
- 直接返回前端专用对象而不经过内部事件。
- 输出 secret。
- 写死具体 Agent 名称作为契约要求。

---

## 28. Contract Test Rules

至少应测试：

### 进程边界

- Gateway 与 Orchestrator 是否独立启动。
- Gateway 是否通过 URL 调用 Orchestrator。
- Gateway 是否没有 import Orchestrator 业务包。
- Frontend 是否不能直接访问 Orchestrator。

### 内部 API

- `/health` 是否可用。
- stream endpoint 是否校验 service token。
- 请求是否 JSON 可序列化。
- trace headers 是否透传。
- timeout 是否生效。

### 编排

- direct / mention / auto / manual 是否能生成统一 OrchestrationPlan。
- single 是否能执行。
- ordered_parallel 是否能输出多 message。
- sequential 是否能表达依赖。
- fallback 是否能输出 state_update。

### 取消与错误

- 浏览器断连是否关闭内部请求。
- cancel endpoint 是否停止 run。
- run_error 是否脱敏。
- service token 是否不出现在日志或错误中。

---

## 29. Review Checklist

Review Gateway ↔ Orchestrator 变更时必须检查：

### 进程边界

- Gateway 和 Orchestrator 是否是两个独立服务？
- Gateway 是否没有 import Orchestrator 业务包？
- Orchestrator 是否有独立 main / health / port？
- Frontend 是否不能直接访问 Orchestrator？
- Child Agent 是否不反向依赖 Gateway 前端 API？

### 内部 API

- Gateway 是否通过 `ORCHESTRATOR_URL` 调用？
- 是否有 service-to-service auth？
- 是否有 timeout？
- 是否传播 requestId / traceId / runId？
- 请求/响应是否 JSON 可序列化？
- 是否没有传 HTTP context / DB handle / Go channel？

### Gateway

- 是否只负责外部 HTTP / SSE / auth / persistence / request assembly？
- 是否没有 Agent 选择逻辑？
- 是否没有 fallback 决策？
- 是否没有直接调用 Child Agent？
- 是否没有解析 Child Agent 原始事件？

### Orchestrator

- 是否负责 planning / task / multi-agent / fallback？
- 是否不处理用户登录？
- 是否不写浏览器响应？
- 是否输出稳定 stream events？
- 是否返回 OrchestratorResult？

### 通用性

- 是否没有固定具体 Agent 名称？
- 是否支持 2+ Agent？
- 是否支持 single / ordered_parallel / sequential？
- 是否没有通过 agentName 推断能力？

### 取消超时

- 浏览器断连是否传播到 Orchestrator？
- Gateway timeout 是否关闭内部请求？
- 是否有 cancel endpoint 或等价机制？
- Orchestrator 是否停止未完成 task？
- 是否没有 goroutine / stream 泄漏？

### 安全

- service token 是否不进日志？
- 用户 token 是否不当作 service token？
- Orchestrator 是否不公网暴露？
- 错误是否脱敏？

---

## 30. 完成定义

本 Skill 视为完成，当且仅当：

- 明确 Gateway 与 Orchestrator 必须分进程。
- 明确 Gateway Service 和 Orchestrator Service 职责边界。
- 明确服务间 API。
- 明确服务间鉴权。
- 明确 OrchestratorRequest。
- 明确 OrchestrationPlan。
- 明确 OrchestratorStreamEvent。
- 明确 OrchestratorResult。
- 明确 Run 生命周期。
- 明确 single / ordered_parallel / sequential。
- 明确 fallback / retry。
- 明确 cancellation / timeout 跨进程规则。
- 明确 trace / safe error 规则。
- 明确不固定具体 Agent 名称。
- 明确 MVP v0.1 仅作为历史基线。
- 明确 Review Checklist。

---

## References

- `references/service-boundary.md`
- `references/service-to-service-api.md`
- `references/orchestrator-request-policy.md`
- `references/orchestrator-stream-event-policy.md`
- `references/orchestrator-result-policy.md`
- `references/orchestration-plan-policy.md`
- `references/multi-agent-execution-policy.md`
- `references/fallback-retry-policy.md`
- `references/cancellation-timeout-policy.md`
- `references/service-auth-policy.md`
- `references/trace-and-error-policy.md`
- `references/deployment-config-policy.md`
- `references/gateway-orchestrator-review-checklist.md`
