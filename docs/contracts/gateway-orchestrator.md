# Gateway-Orchestrator 服务间契约

## 目的

本文定义 AgentHub 中 Gateway Service 与 Orchestrator Service 的独立进程间通信契约。

## 核心原则

- Gateway 和 Orchestrator 必须分进程。
- Gateway 是对外入口。
- Orchestrator 是内部编排服务。
- Gateway 只能通过内部 API 调用 Orchestrator。
- Orchestrator 不得直接暴露给 Frontend。
- 不固定具体 Agent 名称。
- MVP v0.1 仅作为历史基线。

## 服务拓扑

```text
Frontend
  ↓ HTTP / SSE
Gateway Service
  ↓ Internal HTTP / Streaming RPC
Orchestrator Service
  ↓ Agent Client / Planner / Registry
Child Agent Services
```

## Gateway 职责

- 外部 HTTP / SSE。
- 用户鉴权。
- 请求基础校验。
- 用户消息持久化。
- 上下文装配。
- OrchestratorRequest 构造。
- 调用 Orchestrator。
- 转发 OrchestratorStreamEvent。
- 取消 / 超时传播。
- OrchestratorResult 持久化。

## Orchestrator 职责

- 内部 `/health`。
- 内部 stream endpoint。
- planning。
- OrchestrationPlan。
- 多 Agent task。
- single / ordered_parallel / sequential。
- fallback / retry（内部降级，非外部请求 planningMode）。
- OrchestratorStreamEvent。
- OrchestratorResult。

## Run.status 与 phase

Run.status 是粗粒度生命周期状态，只使用 5 值：

```text
accepted
running
completed
failed
cancelled
```

内部阶段（phase）到 Run.status 的映射：

| 内部阶段 | Run.status |
|---|---|
| `accepted` | `accepted` |
| `context_loaded` / `planning` / `plan_ready` / `dispatching` | `running` |
| `agent_task_running` / `agent_task_completed` | `running` |
| `agent_task_failed`（可恢复） / `retrying` / `fallback` | `running` |
| `aggregating` | `running` |
| `completed` | `completed` |
| `failed` / `agent_task_failed`（不可恢复） | `failed` |
| `cancelled` | `cancelled` |

规则：

- 内部阶段不等同于 Public API Run.status。
- 不得将细粒度阶段写入 Run.status。
- `STATE_UPDATE.state.phase` 用于 UI 展示，不等同于持久化 Run.status。
- `run_steps.step_type` 用于持久化详细步骤类型。

## ID 映射

- `conversationId` 是 AgentHub 内部会话主标识。内部持久化、追踪、Artifact links、Run 关联优先使用 `conversationId`。
- AG-UI `threadId` 是 `conversationId` 的前端事件协议别名，仅用于兼容前端事件模型。
- A2A `metadata.threadId` 是 `conversationId` 的 Child Agent 协议别名，仅用于兼容 A2A 任务协议。
- Gateway 构造 `OrchestratorRequest` 时使用 `conversationId`。
- Orchestrator 调用 A2A 时通过 `metadata.threadId` 传递，值等于 `conversationId`。
- 不得把 `threadId` 解释为不同于 `conversationId` 的独立业务概念，不得新增第三套会话 ID。

## 事件映射

Orchestrator 输出内部 `OrchestratorStreamEvent`（snake_case），由 Gateway / ProtocolConverter 映射为 AG-UI Event（UPPER_SNAKE_CASE）后通过 SSE 发给 Frontend。

| OrchestratorStreamEvent（内部） | AG-UI Event（SSE 前端） |
|---|---|
| `run_started` | `RUN_STARTED` |
| `state_update` | `STATE_UPDATE` |
| `message_start` | `TEXT_MESSAGE_START` |
| `message_delta` | `TEXT_MESSAGE_CONTENT` |
| `message_end` | `TEXT_MESSAGE_END` |
| `tool_call_start` | `TOOL_CALL_START` |
| `tool_call_args` | `TOOL_CALL_ARGS` |
| `tool_call_end` | `TOOL_CALL_END` |
| `run_finished` | `RUN_FINISHED` |
| `run_error` | `RUN_ERROR` |

规则：

- Orchestrator 只输出 `OrchestratorStreamEvent`，不得直接输出 AG-UI Event 名称。
- Gateway / ProtocolConverter 负责命名映射和 SSE 格式包装。
- Frontend 只消费 AG-UI Event。
- Child Agent A2A event 不得直接透传给 Frontend。
- `OrchestratorStreamEvent` 事件名不得直接作为 AG-UI Event 名称出现在 SSE 中。

## 禁止

- Gateway 直接调用 Child Agent。
- Gateway 直接调用 LLM。
- Gateway 直接生成 OrchestrationPlan。
- Orchestrator 直接写前端响应。
- Frontend 直接访问 Orchestrator。
- 通过具体 Agent 名称硬编码能力判断。
