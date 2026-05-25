# AgentHub Platform API Contract

## 定位

Platform API 是 AgentHub Frontend 与 Gateway Service 之间的公开 HTTP API 契约。

Gateway Service 是公开 API 的唯一入口。Orchestrator Service 与 Gateway Service 分进程，但 Orchestrator 的内部 API 不属于 Platform API。

## 事实源

`docs/contracts/openapi.yaml` 是 Frontend ↔ Gateway API 的唯一事实源。

## 公开资源

v1 Platform API 面向以下通用资源：Conversation、Message、Agent Summary、Run、Artifact。

所有资源必须是通用平台概念，不得绑定具体 Agent 名称。

## 公开 API 边界

允许：`/api/**`。

禁止公开：`/internal/**`、`/orchestrator/**`、`/a2a/**`、`/.well-known/agent.json`、`/a2a/tasks/**`。

## Gateway Handler 边界

Gateway Handler 负责 HTTP、鉴权、权限、请求校验、response envelope 和日志。

Gateway Handler 不负责 Agent 选择、LLM 调用、Orchestrator 内部计划生成或 Child Agent 调用。

## Agent 状态分离

`AgentSummary.status` 与 `AgentSummary.health` 分离：

- `status`：生命周期/启用状态（`enabled` / `disabled` / `experimental` / `deprecated`）。
- `health`：当前健康状态（`healthy` / `degraded` / `unhealthy` / `unknown`）。
- `disabled` 属于 `status`，不属于 `health`。
- AgentSummary 不得暴露内部 healthcheck 细节（如原始探针 latency、连续失败次数、探针 endpoint）。

## capabilityId 边界

`CapabilitySummary.id` 是 `AgentCard.skills[].id` 的公开摘要投影，非独立 ID。`capabilityIds` 的事实源是 `AgentCard.skills[].id`。

禁止：
- 将 `toolName` 当作 capabilityId。
- 将 `artifact.type` 当作 capabilityId。
- 将 `outputMode` 当作 capabilityId。
- 凭自然语言临时生成未注册的 capabilityId。

## ID 映射

- `conversationId` 是 AgentHub 内部会话主标识。Platform API 资源路径使用 `conversationId`（如 `/api/conversations/{conversationId}`）。
- AG-UI `threadId` 是 `conversationId` 的前端事件协议别名，仅用于兼容前端事件模型。
- A2A `metadata.threadId` 是 `conversationId` 的 Child Agent 协议别名，仅用于兼容 A2A 任务协议。
- 不得把 `threadId` 解释为不同于 `conversationId` 的独立业务概念，不得新增第三套会话 ID。

## MVP Historical Profile

MVP v0.1 的最小 API 只作为历史兼容与回归基线，不得继续限制 v1.0 的 Platform API 扩展。
