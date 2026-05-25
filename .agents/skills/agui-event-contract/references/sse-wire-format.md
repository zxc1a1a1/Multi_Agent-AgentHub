# SSE Wire Format

## 目的

本文定义 AgentHub AG-UI 事件通过 SSE 传输时的 wire format。

## 基本格式

```text
event: message
data: {"type":"RUN_STARTED","runId":"run-001"}

```

## 会话标识

AG-UI 事件中的 `threadId` 是 `conversationId` 的前端协议别名。SSE 传输层不改变此映射关系，`threadId` 不得视为独立于 `conversationId` 的第二套会话 ID。

SSE 中只允许出现 AG-UI Event 名称（UPPER_SNAKE_CASE，如 `TEXT_MESSAGE_CONTENT`）。内部 `OrchestratorStreamEvent` 名称（snake_case，如 `message_delta`）和 Child Agent A2A event 不得直接出现在 SSE 中。

## 必须遵守

- 一个 SSE event block 只承载一个 JSON event。
- event block 使用空行分隔。
- Gateway 必须在每个 event block 后 flush。
- `data:` 必须是合法 JSON 字符串。
- 不输出 JSON array。
- 不在注释行承载业务状态。

## 前端解析要求

Frontend 必须：

- 按 `\n\n` 拆分 event block。
- 处理一个 network chunk 内多个 event。
- 处理一个 event 被拆成多个 network chunk。
- 保留最后一个 incomplete buffer。
- 支持多行 `data:`。
- malformed JSON 不得导致页面崩溃。

## 禁止

- 不得按单个 `\n` 假设事件结束。
- 不得假设一次 `reader.read()` 就是一条完整事件。
- 不得把多个业务事件合并为一个 JSON array。
