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
