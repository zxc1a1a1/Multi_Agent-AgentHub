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

## 实现（v3 A6-06）

- 共享包：`pkg/runtime/observability`。
  - `Setup(ctx, service)` 安装 W3C `traceparent`/`tracestate` propagator 与
    OTel SDK；`OTEL_EXPORTER_OTLP_ENDPOINT` 非空时启用 OTLP/HTTP 导出，
    为空时仍创建 span 以便跨服务传播，但不本地导出。
  - `Handler(service, next)` 创建 server span 并提取入站上下文。
  - `Transport(base)` 创建 client span 并注入出站 `traceparent`。
  - `AddRunID(ctx, runID)` 把 `agenthub.run_id` 写入当前 span 属性。
- 接线：Gateway、Orchestrator 与全部 service agents（code/web/document/vision/
  context/test/review/security/deploy/diff/markdown）启动时调用 `Setup`；
  HTTP server 使用 `Handler`；Gateway→Orchestrator、Orchestrator→Agent 的
  HTTP client 使用 `Transport`。
- run 属性：Gateway 在生成 runId 后、Orchestrator 在解析 runId 后写入
  `agenthub.run_id`；Agent span 通过 `traceparent` 加入同一条 trace。
- 采样：`OTEL_TRACES_SAMPLE_RATIO`（0..1，默认 1）；非法值回退 1。
- 导出协议：OTLP/HTTP protobuf，endpoint 形如 `http://collector:4318`；
  不采集 prompt 内容、工具参数或凭据。
- compose 提供 `observability` profile：`otel-collector`（OTLP 4317/4318）与
  `jaeger`（UI 16686），配置见 `deploy/otel/collector.yaml`。启用时设置
  `OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318`；未启用时保持空值，
  span 只用于传播不导出。该拓扑未在本机实际启动。
