# Frontend Aggregation

## 聚合主键

| 数据 | 主键 |
|---|---|
| Run 状态 | `runId` |
| 文本消息 | `messageId` |
| Tool Call 参数 | `toolCallId` |
| 对话状态 | `threadId` |

## 必须兼容字段

```ts
const textDelta = event.delta ?? event.content ?? ''
const argsDelta = event.delta ?? event.content ?? ''
const state = event.state ?? safeParseJSON(event.content)
```

## 处理要求

- 未知事件不崩溃。
- 未知工具不崩溃。
- malformed JSON 不崩溃。
- `RUN_ERROR` 停止 loading。
- `RUN_FINISHED` 停止 loading。
- per-conversation streaming 状态独立。
- 多 Agent 消息按 `messageId` 独立渲染。
