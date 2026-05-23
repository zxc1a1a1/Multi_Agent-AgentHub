# Trace Context 规则

## 1. 目的

本文定义 AgentHub 中跨服务、跨协议、跨 Agent 的 trace context 和 ID 关联规则。

## 2. 长期 ID 集合

长期关键 ID：

```text
traceId
requestId
conversationId
messageId
runId
executionPlanId
stepId
a2aTaskId
agentName
agentSkill
artifactId
toolCallId
llmRequestId
```

## 3. MVP 必需 ID

MVP 阶段至少需要：

```text
requestId
runId
conversationId
messageId
agentName
a2aTaskId
toolCallId
```

## 4. 传播规则

HTTP 请求必须携带 requestId 或生成 requestId。

Gateway 必须生成或接收 traceId，并在后续调用中传播。

A2A Client 调用子 Agent 时，必须传播：

```text
traceId
runId
messageId
a2aTaskId
agentName
```

AG-UI SSE 输出中必须能关联：

```text
runId
messageId
toolCallId
```

Artifact 处理必须能关联：

```text
artifactId
runId
messageId
a2aTaskId
```

正式开发阶段优先支持 W3C Trace Context：

```text
traceparent
tracestate
```

## 5. 禁止事项

不得：

- 各服务各自生成无法关联的 traceId。
- A2A 调用丢失 runId。
- Artifact 没有 messageId / runId 关联。
- Tool Call 没有 toolCallId。
- 多 Agent task 没有 stepId。
