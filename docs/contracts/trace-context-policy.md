# Trace Context Policy

## 目的

定义 AgentHub 在 Gateway、Orchestrator、Agent、LLM Provider 和存储等服务之间传播追踪上下文的规则。

## 标准字段

- `traceparent`
- `tracestate`

## 规则

- Gateway 必须生成或提取 trace context。
- Gateway 调 Orchestrator 必须传播 trace context。
- Orchestrator 调下游服务必须继续传播或记录 trace context。
- 业务关联 ID 不应替代 `traceparent`。
- 不得在 trace context 中保存 secret、完整 prompt 或用户隐私。
