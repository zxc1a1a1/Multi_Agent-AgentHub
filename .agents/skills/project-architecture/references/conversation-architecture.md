# Conversation Architecture

Conversation 是 AgentHub 的一级架构对象。

## 类型

- `single`：通常面向一个主要 Agent 或一个主要能力。
- `group`：允许多个 Agent participant，由 Orchestrator 统一编排。

## 规则

- Conversation 可有 participants。
- 用户可以通过 @mention 提示目标 Agent。
- @mention 是路由提示，不是绕过 Orchestrator 的直接调用。
- 一个 run 可以产生多条 Agent message。
- Agent message 必须保留 senderName / senderId。
- 前端头像与名称展示必须基于 sender 字段，而不是写死 Agent 名称。

## 群聊

群聊中 Orchestrator 负责拆解、调度和聚合。前端只展示编排过程和结果，不执行编排逻辑。
