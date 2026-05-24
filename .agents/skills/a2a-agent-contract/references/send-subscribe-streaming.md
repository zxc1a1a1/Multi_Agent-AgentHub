# sendSubscribe Streaming

来源：`docs/contracts/a2a-task.md`。

## 1. MVP 流式事件

- `status: working`
- `text`
- `artifact`
- `status: completed`
- `status: failed`

## 2. 映射边界

- A2A 事件先到 Orchestrator。
- 经 converter 后再输出 AG-UI。
- 不可将 A2A 原始事件直接透传前端。
