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
