# Conversation API 策略

Conversation API 必须支持单聊与群聊。

推荐字段：`id`（即内部 `conversationId`）、`title`、`conversationType`、`participants`、`lastMessageAt`、`createdAt`、`updatedAt`、`archived`。

AG-UI 事件中 `threadId` 是此 `id`（conversationId）的前端协议别名。

Participant 可以是 `user`、`agent`、`system`。

兼容期允许 `initialAgentName` 映射为 `participants[0]`，新增代码应使用 `participants`。
