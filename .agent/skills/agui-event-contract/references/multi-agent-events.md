# Multi-Agent Events

## 目的

本文定义一个 Run 中出现多个 Agent 消息时的事件归属规则。

## 核心规则

- 一个 Run 可以产生多条 assistant message。
- 每条 assistant message 必须有独立 `messageId`。
- 每条 assistant message 可以有独立 `sender`。
- `sender.name` 是普通字符串，不限定具体 Agent 名称。
- Tool Call 必须通过 `messageId` 归属到某条 message。
- 前端应按 `messageId` 聚合，不应按 Agent 名称聚合。

## ordered-parallel

v1.0 推荐 ordered-parallel：

- 策略可以展示为 parallel。
- SSE 事件可以按消息顺序输出。
- 不要求多个 Agent token 交错。
- 如果未来支持交错，必须保证每个 content 都带正确 `messageId`。
