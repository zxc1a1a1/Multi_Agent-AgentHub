# PostgreSQL Schema Contract for Post-MVP

## 1. 文档目的

本文档描述 AgentHub 长期 PostgreSQL 数据模型方向。

PDR 长期目标中：

```text
PostgreSQL = 事实源
Redis = 缓存 / 临时状态 / 队列
Object Storage = 大 Artifact 内容
```

MVP v0.1 使用 MySQL 8 不改变长期方向。

## 2. PostgreSQL 适用范围

Post-MVP 使用 PostgreSQL 管理：

- 用户。
- 会话。
- 消息。
- Agent。
- Run。
- Run Step。
- A2A Task。
- Tool Call。
- Approval。
- Artifact metadata。
- Agent health check。
- JSONB metadata。

## 3. JSONB 使用

适合 JSONB 的字段：

```text
messages.content
agents.agent_card
artifacts.metadata
tool_calls.args
runs.context
run_steps.input_summary
run_steps.output_summary
```

规则：

- JSONB 字段必须有 schema 说明。
- 不允许无约束塞任意数据。
- 不允许保存 API key、token、完整 system prompt。
- JSONB 变更必须同步 Contract。

## 4. 长期核心表

```text
users
conversations
conversation_participants
messages
agents
runs
run_steps
a2a_tasks
tool_calls
approvals
artifacts
agent_health_checks
```

## 5. 与 MySQL MVP 的关系

- MySQL MVP 表应尽量保持字段语义可迁移。
- API JSON 字段保持 camelCase。
- DB 列名可使用 snake_case。
- ID 关系保持稳定。
- 迁移到 PostgreSQL 时不得破坏 OpenAPI、AG-UI、A2A、Artifact Contract。
