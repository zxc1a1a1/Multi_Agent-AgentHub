# Frontend Debugging

## 目的

定义前端排障时可记录的最小字段和禁止事项。

## 可记录字段

- `requestId`
- `runId`
- `conversationId`
- `messageId`
- `toolCallId`
- `eventType`
- `streamState`
- `errorCode`

## 禁止记录

- API token
- 完整用户隐私输入
- 完整 HTML artifact
- 完整 LLM prompt

## 排障步骤

- 是否创建 `requestId / runId`？
- SSE 是否打开？
- 最后收到的 eventType 是什么？
- messageId 是否稳定？
- toolCallId 是否完整？
- RUN_ERROR 是否含 errorCode？
- 浏览器断连是否在 Gateway 中被记录？
