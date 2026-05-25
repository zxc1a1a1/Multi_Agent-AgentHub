# 服务边界

Gateway Service 与 Orchestrator Service 必须是两个独立进程。

## Gateway Service

负责对外入口：

- HTTP API。
- 前端流式连接。
- 用户鉴权。
- 请求基础校验。
- 用户消息持久化。
- 历史上下文装配。
- 调用 Orchestrator 内部 API。
- 转发 Orchestrator 流式事件。
- 处理浏览器断连。
- 持久化 OrchestratorResult。

Gateway 不负责：

- Agent 选择。
- 多 Agent 编排。
- fallback / retry 决策。
- Child Agent 调用。
- LLM 调用。

## Orchestrator Service

负责内部编排：

- planning。
- OrchestrationPlan。
- AgentTask。
- single / ordered_parallel / sequential。
- fallback / retry。
- OrchestratorStreamEvent。
- OrchestratorResult。

Orchestrator 不负责：

- 前端用户登录。
- 浏览器连接管理。
- 前端公开 API。
- Gateway 持久化逻辑。

## 硬规则

- Gateway 不得 import Orchestrator 业务包。
- Orchestrator 不得直接暴露给 Frontend。
- 两者只能通过受保护的内部 API 通信。
