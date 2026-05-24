# SSE Streaming Policy

来源：`docs/contracts/agui-events.md`、`docs/skill/agui-event-contract/SKILL.md`、`docs/reports/review 2.md`。

## 1. 线协议

- 事件以 SSE `data:` 行输出。
- 每个 event block 以空行分隔（`\n\n`）。
- 每个 block 只包含一个 AG-UI 事件 JSON。

## 2. 客户端解析要求

- 按 `\n\n` 分割 event block，不按单行 `\n` 直接完成事件切分。
- 允许 chunk 半包，尾部不完整 block 留在 buffer。
- 单次读取可能包含多个 event block。

## 3. 安全要求

- 仅输出 AG-UI 事件，不透传 A2A 原始事件。
- 错误事件需脱敏。
