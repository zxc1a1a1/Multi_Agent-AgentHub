# Identity Linking Policy

## 目的

Artifact 必须可追溯到它的上下文和来源。

## 必填关联

```text
artifactId
links.conversationId
links.messageId
links.runId
```

## 推荐来源字段

```text
source.agentName
source.taskId
source.stepId
```

## 多 Agent 场景

多 Agent 或群聊中：

- 同一 run 可以产生多个 message。
- 同一 message 可以产生多个 Artifact。
- 不同 Agent 可以生成同名文件。
- Artifact ID 不得由 title 单独派生。
- source.agentName 用于展示和调试，不用于判断 Artifact 类型。

## 禁止

- 用 filename 当 artifactId。
- 用 messageId 当 artifactId。
- 用 agentName 决定 artifact.type。
- 缺少 runId。
