# 数据模型总览

## 目的

本文件概述 AgentHub v1.0 的核心数据实体和关系。

## 核心原则

- 数据库是历史事实源。
- 实时事件不是事实源。
- 缓存不是事实源。
- 日志不是事实源。
- 多 Agent、群聊、产物、工具调用都必须可追溯。

## 核心实体

```text
User
  └── Conversation
        ├── ConversationParticipant
        ├── Message
        │     ├── Artifact
        │     └── ToolCall
        └── Run
              ├── RunStep
              ├── AgentTask
              ├── Message
              ├── Artifact
              └── ToolCall

Agent
  ├── AgentHealthCheck
  ├── ConversationParticipant
  ├── Message
  ├── RunStep
  ├── AgentTask
  └── Artifact
```

## v1.0 必须支持

- 一个 conversation 可以是 single 或 group。
- 一个 group conversation 可以有多个 agent participant。
- 一个 run 可以触发多个 agent task。
- 一个 run 可以产生多条 agent message。
- 一个 message 可以关联多个 artifact。
- 一个 artifact 必须能追溯到 run、message、agent。

## 禁止

- 用单个 `agent_name` 字段表达群聊全部参与者。
- 用 message content 保存大型产物。
- 用日志替代 run / step / task 记录。
- 用 JSON 字段逃避核心关系建模。
