# Text Message Events

## 事件

```text
TEXT_MESSAGE_START
TEXT_MESSAGE_CONTENT
TEXT_MESSAGE_END
```

## TEXT_MESSAGE_START

开始一条 assistant message。

必须包含：

- `type`
- `runId`
- `messageId`
- `role`
- `sender`

## TEXT_MESSAGE_CONTENT

追加文本片段。

新实现优先使用：

```json
{"delta":"文本片段"}
```

兼容读取：

```json
{"content":"文本片段"}
```

前端聚合：

```ts
const chunk = event.delta ?? event.content ?? ''
```

## TEXT_MESSAGE_END

结束一条 assistant message。

## 规则

- `messageId` 是文本聚合主键。
- 多 Agent 场景必须使用独立 `messageId`。
- `TEXT_MESSAGE_CONTENT` 只承载文本，不承载大型产物。
- Markdown 文本可以通过 `TEXT_MESSAGE_CONTENT` 传输。
