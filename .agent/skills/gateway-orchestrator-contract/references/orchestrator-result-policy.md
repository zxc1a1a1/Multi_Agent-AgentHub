# OrchestratorResult 规则

`OrchestratorResult` 是一次 run 的最终摘要。

## 字段

```text
runId
conversationId
status
strategy
intentSummary
messages
tasks
artifacts
toolCalls
error
startedAt
finishedAt
```

## 规则

- 一个 result 可以包含多条 assistant message。
- 一个 result 可以包含多个 task。
- 一个 result 可以包含多个 artifact 引用。
- 一个 result 可以包含多个 tool call 引用。
- result 不保存大型二进制内容。
- result 不保存完整敏感 prompt。
- result 不返回内部堆栈。
- Gateway 负责基于 result 进行持久化。

## Status

```text
completed
failed
cancelled
```

## Error

失败时必须使用 SafeError。
