# Identity Linking 规则

## 1. 目的

本文定义 Artifact 与 AgentHub 执行链路的 ID 关联规则。

Artifact 不得成为孤立对象。

## 2. 必须关联

长期 Artifact 必须关联：

```text
artifactId
conversationId
messageId
runId
```

## 3. 条件关联

当存在对应上下文时，Artifact 应关联：

```text
a2aTaskId
stepId
agentName
agentSkill
toolCallId
traceId
```

## 4. 关联语义

`conversationId`：

```text
产物所属会话。
```

`messageId`：

```text
产物归属的消息。
```

`runId`：

```text
产物所属的一次执行。
```

`a2aTaskId`：

```text
产物来源的 A2A Task。
```

`stepId`：

```text
多 Agent / 多步骤编排中的步骤。
```

`toolCallId`：

```text
产物预览所触发的 AG-UI Tool Call。
```

`traceId`：

```text
排障和日志追踪 ID。
```

## 5. MVP

MVP 至少必须保留：

```text
conversationId
messageId
runId
```

如果 A2A Task 已生成 ID，应保留：

```text
a2aTaskId
```

## 6. 禁止事项

不得：

- Artifact 没有 messageId。
- Artifact 没有 runId。
- 多 Agent 阶段 Artifact 没有 stepId。
- 预览失败但无法通过 artifactId / toolCallId 排查。
