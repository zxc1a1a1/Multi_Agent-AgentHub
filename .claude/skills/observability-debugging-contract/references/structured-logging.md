# 结构化日志规则

## 1. 目的

本文定义 AgentHub 结构化日志字段和日志事件规则。

## 2. 基础字段

日志必须是结构化 JSON。

基础字段：

```text
timestamp
level
service
environment
event
traceId
requestId
runId
conversationId
messageId
agentName
agentSkill
errorCode
safeMessage
```

## 3. 常见事件

推荐事件名：

```text
run_started
run_finished
run_failed
message_saved
orchestration_started
execution_plan_created
execution_plan_failed
a2a_task_started
a2a_task_stream_started
a2a_task_finished
a2a_task_failed
artifact_created
artifact_persisted
artifact_preview_mapped
artifact_failed
agui_sse_started
agui_sse_event_emitted
agui_sse_failed
llm_request_started
llm_stream_started
llm_request_finished
llm_request_failed
tool_call_started
tool_call_finished
tool_call_failed
```

## 4. 日志等级

```text
debug
info
warn
error
```

MVP 默认不输出过多 debug 日志。

生产环境 debug 日志必须可关闭。

## 5. 禁止事项

不得：

- 只打自然语言日志。
- 日志字段名每个服务不一致。
- 在日志中写 API key、token、prompt secret。
- 在用户可见错误中写 stack trace。
- 把完整 LLM raw request 写入日志。
