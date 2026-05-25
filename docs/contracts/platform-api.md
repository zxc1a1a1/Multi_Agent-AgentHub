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

## MVP Historical Profile

MVP v0.1 的最小 API 只作为历史兼容与回归基线，不得继续限制 v1.0 的 Platform API 扩展。
