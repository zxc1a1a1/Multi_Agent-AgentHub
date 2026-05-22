# MySQL Schema Contract for MVP v0.1

## 1. 文档目的

本文档描述 MVP v0.1 使用 MySQL 8 时的数据模型规则。

注意：本文档不是 SQL migration，不生成可执行 SQL。实际 SQL 由后续实现阶段生成。

## 2. MVP 数据库定位

MVP v0.1：

```text
MySQL 8 = 当前事实源
Redis = 暂不强制
Object Storage = 暂不强制
```

长期方向仍保留 PostgreSQL + Redis + Object Storage。

## 3. 字段命名

数据库列名推荐 snake_case：

```text
conversation_id
created_at
updated_at
run_id
trace_id
```

API / JSON 字段必须 camelCase：

```text
conversationId
createdAt
updatedAt
runId
traceId
```

## 4. MVP 表清单

MVP v0.1 需要：

```text
users
conversations
messages
agents
runs
a2a_tasks
tool_calls
artifacts
```

Post-MVP planned：

```text
conversation_participants
run_steps
approvals
agent_health_checks
```

## 5. JSON 字段

MySQL 可使用 JSON 类型保存：

```text
messages.content
agents.agent_card
artifacts.metadata
tool_calls.args
runs.context
```

规则：

- 每个 JSON 字段必须有 Contract 说明。
- 不允许保存 API key、token、完整敏感 prompt。
- 不允许把大文件长期塞进 JSON 字段。

## 6. 索引建议

MVP 常用查询需要考虑：

```text
conversations.user_id
messages.conversation_id
messages.run_id
runs.conversation_id
runs.trace_id
a2a_tasks.run_id
artifacts.message_id
artifacts.run_id
tool_calls.run_id
tool_calls.artifact_id
```

## 7. 删除策略

- Conversation 删除建议软删除。
- Message 原则上保留历史，不做物理删除。
- Artifact 删除需要同时处理内容字段或后续对象存储对象。
- Post-MVP 应补充归档策略。

## 8. MVP 不强制项

MVP v0.1 暂不强制：

- 群聊参与者完整表。
- Run step 详细追踪表。
- Approval 表。
- Agent health check 历史表。
- Redis 缓存。
- Object Storage。
- 全量多用户权限模型。

这些必须保留在长期规划中。
