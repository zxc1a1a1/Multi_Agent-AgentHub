# Persistence ID Policy

来源：`docs/contracts/data-model.md`。

关键关联链路：

```text
conversation.id
→ message.conversationId
→ run.id / run.conversationId
→ a2aTask.runId
→ artifact.runId / artifact.messageId
→ toolCall.artifactId
```

要求：跨协议 ID（runId/threadId/taskId/toolCallId）要可追踪、可关联。
