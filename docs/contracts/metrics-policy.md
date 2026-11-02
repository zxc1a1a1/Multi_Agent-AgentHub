# Metrics Policy

## 目的

定义 AgentHub 指标命名、维度和禁止事项。

## 指标方向

指标优先覆盖：

- latency
- traffic
- errors
- saturation

## 推荐指标

- `gateway_http_requests_total`
- `gateway_http_request_duration_ms`
- `gateway_stream_disconnects_total`
- `orchestrator_runs_total`
- `orchestrator_run_duration_ms`
- `orchestrator_agent_tasks_total`
- `orchestrator_agent_task_duration_ms`
- `orchestrator_fallback_attempts_total`
- `llm_requests_total`
- `llm_request_duration_ms`
- `llm_tokens_input_total`
- `llm_tokens_output_total`
- `artifact_created_total`
- `tool_calls_total`
- `registry_health_check_failures_total`
- `dispatcher_retry_total` — dispatch retry count per attempt
- `dispatcher_breaker_state` — circuit breaker state (0=closed, 1=open, 2=half-open)
- `executor_wave_duration_ms` — DAG executor wave duration histogram
- `synthesizer_call_total` — synthesizer invocation count
- `synthesizer_call_duration_ms` — synthesizer call duration

## 允许低基数标签

- `service`
- `environment`
- `route`
- `status`
- `strategy`
- `planningMode`
- `provider`
- `model`
- `errorCode`

## 禁止标签

- `runId`
- `traceId`
- `messageId`
- `user input`
- `prompt`
- `token`
- `full URL with token`

## Routing metrics（A4-07）

`GET /internal/orchestrator/metrics/routing?window=24h`（或 `since=RFC3339`）返回：
`totalRuns/totalDecisions/selectedAgents/rejectedReasons/selectionRate/rejectionRate/loopRuns`
以及 `usage.{modelCalls,toolCalls,repairs,replans}`。`loopRuns` 以同一 run 的多条
route_decisions 计数（replan/循环代理指标）；成本以预算用量表示，不是货币金额。

## Operational metrics（A6-07）

- Orchestrator `GET /internal/orchestrator/metrics?window=24h`：
  `executions/approvals/toolCalls` 按 status 计数（低基数）。
- Gateway `GET /internal/metrics`：
  `gateway_chat_streams_total/active`、`gateway_sse_events_total`、
  `gateway_stream_write_errors_total`、`gateway_stream_disconnects_total`。
- 指标不含 runId/userId/prompt/token。

## Prometheus 与告警（A6-07）

- 三个入口均支持 `?format=prometheus`，返回 Prometheus text exposition
  （`text/plain; version=0.0.4`）：
  - Gateway `/internal/metrics` → `agenthub_gateway_*`
  - Orchestrator `/internal/orchestrator/metrics` → `agenthub_orchestrator_executions/approvals/tool_calls{status}`
  - Orchestrator `/internal/orchestrator/metrics/routing` →
    `agenthub_orchestrator_routing_{runs,decisions,multi_decision_runs}_total`、
    `..._selected_total{agent}`、`..._rejected_total{reason}`、planner 用量。
- 抓取配置：`deploy/prometheus/prometheus.yml`（`window=1h`）；规则：
  `deploy/prometheus/alerts.yml`（Gateway/Orchestrator down、run/approval 失败、
  tool unknown、SSE 写错误、路由决策缺失）。
- Orchestrator 抓取需要 `INTERNAL_SERVICE_TOKEN`，以
  `/etc/prometheus/orchestrator-token`（0600）挂载，不写入仓库。
- 状态：指标与规则文件已实现；本仓库未启动 Prometheus/Alertmanager，告警通道
  仍未配置，规则仅做 YAML 解析验证。阈值需按真实流量调整。
