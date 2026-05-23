# Debug Playbook

## 1. 目的

本文定义 AgentHub 常见问题的排障路径。

## 2. 前端没收到回复

检查：

```text
traceId
runId
messageId
agui_sse_started
agui_sse_event_emitted
agui_sse_failed
```

确认 Gateway 是否发出 SSE，前端是否接收并解析。

## 3. A2A 没返回

检查：

```text
a2a_task_started
a2a_task_stream_started
a2a_task_failed
a2aTaskId
agentName
```

确认 AgentCard、Agent health、A2A endpoint、timeout。

## 4. Artifact 没预览

检查：

```text
artifact_created
artifact_persisted
artifact_preview_mapped
tool_call_started
tool_call_finished
```

确认 artifactId、messageId、runId、toolCallId 是否关联。

## 5. LLM 超时

检查：

```text
llm_request_started
llm_stream_started
llm_request_failed
provider
model
errorCode
```

确认 timeout、rate limit、context cancellation、fallback。

## 6. 多 Agent 聚合错乱

检查：

```text
executionPlanId
stepId
strategy
dependsOn
a2aTaskId
agentName
```

确认 parallel / sequential 策略和聚合规则。

## 7. fallback 为什么触发

检查：

```text
fallback_reason
fallbackAgent
fallbackSkill
previousErrorCode
```

确认 fallback 是否在 contract 中声明。
