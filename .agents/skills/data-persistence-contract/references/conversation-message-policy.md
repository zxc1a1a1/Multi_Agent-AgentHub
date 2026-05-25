# Conversation 与 Message 规则

## Conversation

`conversationId` 是 AgentHub 内部会话主标识。DB `conversations.id` 存储此值。
AG-UI `threadId` 和 A2A `metadata.threadId` 是其协议别名，不得视为第二套独立会话 ID。

`conversations` 表表达会话本体。

关键字段：

```text
id
user_id
title
conversation_type
primary_agent_name
status
is_pinned
is_archived
created_at
updated_at
deleted_at
```

## Conversation Type

```text
single
group
```

规则：

- single 可以有一个默认 Agent。
- group 必须使用 `conversation_participants` 表表达参与者。
- `primary_agent_name` 不是群聊参与者事实源。

## Participants

`conversation_participants` 表表达 user / agent 参与者。

关键字段：

```text
conversation_id
participant_type
participant_id
display_name
role
status
joined_at
left_at
```

## Message

`messages` 表表达可回放的历史消息。

关键字段：

```text
conversation_id
run_id
sender_type
sender_id
sender_name
content
content_format
status
created_at
```

## 多 Agent 消息

- 一个 Run 可以产生多条 Agent message。
- 每条 Agent message 必须有独立 message id。
- 每条 Agent message 应保存 sender_name。
- Artifact 必须能追溯到具体 message。

## 禁止

- 把群聊 Agent 参与者全部塞进一个字符串字段。
- 把大 Artifact 放进 message content。
- 只保存最终合并消息，丢失各 Agent 原始回复。
- 在历史消息中丢失 sender_name。
