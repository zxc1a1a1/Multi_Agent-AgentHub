# Span Naming Policy

## 目的

定义 AgentHub trace span 命名和属性规则。

## 推荐 Span 名称

- `HTTP POST /api/runs`
- `HTTP POST /internal/orchestrator/runs/stream`
- `gateway.call_orchestrator`
- `gateway.forward_stream_event`
- `orchestrator.plan`
- `orchestrator.validate_plan`
- `orchestrator.execute_task`
- `orchestrator.fallback`
- `agent.call`
- `llm.generate`
- `llm.stream`
- `artifact.normalize`

## 自定义属性

AgentHub 自定义属性使用 `agenthub.*` 前缀：

- `agenthub.run_id`
- `agenthub.plan_id`
- `agenthub.step_id`
- `agenthub.agent_task_id`
- `agenthub.agent_name`
- `agenthub.tool_call_id`
- `agenthub.artifact_id`
- `agenthub.llm_request_id`
- `agenthub.error_code`

## 规则

- HTTP 通用属性优先沿用 OpenTelemetry 语义约定。
- 不把 prompt、token、用户隐私放入 span attributes。
- 高基数字段不要作为 metrics label。
