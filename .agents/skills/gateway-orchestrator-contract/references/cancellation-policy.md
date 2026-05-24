# Cancellation Policy

来源：`docs/contracts/gateway-orchestrator.md`、`docs/contracts/openapi.yaml`（`/api/agui/run/{runId}/cancel`）。

## 1. MVP

- MVP 可保留轻量取消能力（必要时仅支持请求上下文取消）。
- 不要求完整分布式取消系统。

## 2. 规则

- 取消信号由 Gateway 传递到 Orchestrator context。
- Orchestrator 停止后续事件输出。
- 对前端返回安全错误或取消状态（不泄漏内部细节）。

## 3. Post-MVP

- 引入显式 run cancel endpoint 的完整编排与状态持久化。
