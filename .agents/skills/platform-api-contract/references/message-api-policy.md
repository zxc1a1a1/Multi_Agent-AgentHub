# Message API 策略

Message API 必须支持多 Agent、多消息、Artifact 与 ToolCall 引用。

推荐字段：`id`、`conversationId`（AG-UI 侧称 `threadId`、A2A 侧称 `metadata.threadId`）、`runId`、`senderType`、`senderId`、`senderName`、`content`、`contentFormat`、`status`、`mentions`、`artifactRefs`、`toolCallRefs`、`createdAt`、`updatedAt`。

大型 Artifact 不放入 `content`。群聊中每条 Agent message 必须能显示归属。
