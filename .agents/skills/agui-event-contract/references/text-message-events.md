# Text Message Events

来源：`docs/contracts/agui-events.md`。

## 1. 事件职责

- `TEXT_MESSAGE_START`：创建消息容器。
- `TEXT_MESSAGE_CONTENT`：追加文本 chunk。
- `TEXT_MESSAGE_END`：文本流结束。

## 2. 约束

- `TEXT_MESSAGE_CONTENT` 不得早于 START。
- `TEXT_MESSAGE_END` 不得早于 START。
- 大型结构化产物不得塞入文本 chunk。
- `messageId` 在同一条消息生命周期中保持一致。
