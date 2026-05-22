# Migration Policy

## 1. 文档目的

本文档定义 AgentHub 数据库 migration 规则。

所有 schema 变更必须有版本化 migration，不允许手工改库后不留记录。

## 2. 基本规则

- migration 文件必须版本化。
- migration 必须可重复执行或可明确失败。
- 不允许直接在生产库执行未 review SQL。
- migration 必须包含 up/down 或明确不可逆说明。
- 修改表字段必须同步数据模型文档。
- 修改 API 相关字段必须同步 OpenAPI。
- 修改 Artifact 相关字段必须同步 Artifact Contract。
- 修改安全相关字段必须同步 Security Contract。

## 3. 命名建议

```text
000001_init_schema.sql
000002_add_runs.sql
000003_add_artifacts.sql
000004_add_tool_calls.sql
000005_add_post_mvp_tracking_tables.sql
```

## 4. MVP v0.1

MVP v0.1 可以先只包含核心表：

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

Post-MVP planned 表可以后续 migration：

```text
conversation_participants
run_steps
approvals
agent_health_checks
```

## 5. Review Checklist

- [ ] migration 是否版本化？
- [ ] 是否说明 up/down？
- [ ] 是否同步 data-model.md？
- [ ] 是否同步 OpenAPI？
- [ ] 是否同步 Artifact Contract？
- [ ] 是否没有明文密钥字段？
- [ ] 是否没有把大文件直接塞进普通业务表？
- [ ] 是否保留跨协议 ID？
