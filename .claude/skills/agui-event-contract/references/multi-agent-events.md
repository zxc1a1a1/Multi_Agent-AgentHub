# Multi-Agent Events

## 目的

本文定义一个 Run 中出现多个 Agent 消息时的事件归属规则。

## 事件来源

多 Agent 消息的 AG-UI Event 由 Orchestrator 内部的 `OrchestratorStreamEvent` 经 Gateway / ProtocolConverter 映射而来。Orchestrator 负责按 `messageId` 隔离不同 Agent 的输出，Gateway 只做命名映射。

Child Agent 原始 A2A event 不得绕过 Orchestrator 直接透传给 Frontend。

## 核心规则

- 一个 Run 可以产生多条 assistant message。
- 每条 assistant message 必须有独立 `messageId`。
- 每条 assistant message 可以有独立 `sender`。
- `sender.name` 是普通字符串，不限定具体 Agent 名称。
- Tool Call 必须通过 `messageId` 归属到某条 message。
- 前端应按 `messageId` 聚合，不应按 Agent 名称聚合。

## ordered_parallel

v1.0 正式枚举值为 `ordered_parallel`。
legacy `parallel` / `ordered-parallel` 仅作为兼容输入别名，进入 Gateway 前归一化为 `ordered_parallel`。
UI 展示文案"并行"不等于协议字段 `parallel`。

- SSE 事件可以按消息顺序输出。
- 不要求多个 Agent token 交错。
- 如果未来支持交错，必须保证每个 content 都带正确 `messageId`。
