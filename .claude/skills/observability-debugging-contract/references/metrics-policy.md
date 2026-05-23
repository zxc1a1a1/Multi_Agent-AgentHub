# Metrics 规则

## 1. 目的

本文定义 AgentHub 指标命名和采集规则。

MVP 阶段可以先不接 metrics backend，但事件和字段要预留。

## 2. 推荐指标

```text
agenthub_run_started_total
agenthub_run_finished_total
agenthub_run_failed_total
agenthub_run_duration_ms
agenthub_a2a_task_started_total
agenthub_a2a_task_failed_total
agenthub_a2a_task_duration_ms
agenthub_llm_request_total
agenthub_llm_request_failed_total
agenthub_llm_latency_ms
agenthub_llm_tokens_total
agenthub_artifact_created_total
agenthub_artifact_failed_total
agenthub_agui_sse_events_total
agenthub_tool_call_failed_total
```

## 3. 推荐标签

低基数字段：

```text
service
environment
agentName
agentSkill
provider
model
status
errorCode
```

## 4. 禁止标签

不得作为 metrics label：

```text
userId
messageId
runId
traceId
prompt
rawError
apiKey
token
fullURLWithSecret
```

高基数字段应进入 trace / log，不应进入 metrics label。

## 5. 禁止事项

不得：

- 在 metric label 中写敏感信息。
- 用高基数字段做 label。
- 指标名称每个服务各自发明。
- 没有 errorCode 就统计失败。
