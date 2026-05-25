# ID Correlation Policy

## 目的

定义 AgentHub 中跨服务、跨步骤、跨产物的统一关联 ID。

## 通用 ID

| 字段 | 说明 |
|---|---|
| `traceId` | 跨服务追踪链 |
| `requestId` | Gateway 外部请求 |
| `runId` | 一次 AgentHub run |
| `conversationId` | 会话 |
| `messageId` | 消息 |
| `planId` | 编排计划 |
| `stepId` | 编排步骤 |
| `agentTaskId` | 内部 Agent 子任务 |
| `externalTaskId` | 外部系统任务 ID |
| `artifactId` | 产物 |
| `toolCallId` | Tool Call |
| `llmRequestId` | 内部 LLM 请求 |

## 规则

- 不使用具体外部协议 ID 作为全局唯一通用字段。
- 多 Agent run 必须能通过 `agentTaskId` 区分每个子任务。
- Artifact、ToolCall、LLM 请求必须能回溯到 `runId`。
- 日志不应只依赖自然语言描述定位链路。
