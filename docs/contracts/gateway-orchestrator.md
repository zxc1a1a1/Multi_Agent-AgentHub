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
- fallback / retry。
- OrchestratorStreamEvent。
- OrchestratorResult。

## 禁止

- Gateway 直接调用 Child Agent。
- Gateway 直接调用 LLM。
- Gateway 直接生成 OrchestrationPlan。
- Orchestrator 直接写前端响应。
- Frontend 直接访问 Orchestrator。
- 通过具体 Agent 名称硬编码能力判断。
