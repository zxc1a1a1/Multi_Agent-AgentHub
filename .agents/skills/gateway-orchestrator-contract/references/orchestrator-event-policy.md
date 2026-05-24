# Orchestrator Event Policy

来源：`docs/contracts/gateway-orchestrator.md`、`docs/contracts/gateway-orchestrator-events.md`。

## 1. 职责

- Orchestrator 向 Gateway 输出 AG-UI 语义事件。
- Gateway 负责 SSE 包装与对外连接管理。

## 2. 禁止

- Gateway handler 直接做 A2A 调度或复杂编排。
- 透传 A2A 原始事件给前端。

## 3. 一致性

- 事件语义保持与 `agui-event-contract` 一致。
- 事件必须可被前端 reducer 消费。
