# ID 与 Trace 规则

## 必须支持的 ID

```text
request_id
trace_id
conversation_id
message_id
run_id
step_id
agent_task_id
external_task_id
tool_call_id
artifact_id
agent_name
user_id
```

## 规则

- 每条 message 必须有稳定 id。
- 每次 run 必须有稳定 id。
- 多 Agent 场景下每个 AgentTask 必须能关联 run。
- Artifact 必须能关联 message、run、agent。DB `artifacts.id` 存储 Core `artifactId` 值。Core `artifactId` = DB `id` = Public API `id` 是同一个系统 ID 在不同层级的命名，不另行生成独立的 public id。
- `conversationId` 是 AgentHub 内部主字段。AG-UI `threadId` 和 A2A `metadata.threadId` 是其协议别名，不得视为独立会话 ID。
- ToolCall 必须能关联 message 或 artifact。
- 错误必须能通过 request_id 或 trace_id 排查。

## 不允许

- 只在日志里保存 trace 信息。
- 前端临时 ID 覆盖数据库 ID。
- 同一 Run 内多个 Agent message 复用同一个 message id。
- 两个不同 Agent 生成同名产物时发生覆盖。

## 推荐字段类型

- ID 可使用 UUID、ULID 或项目统一 ID 方案。
- 数据库存储可用 `CHAR(36)` 或项目统一类型。
- 对外 JSON 使用 camelCase，如 `runId`。
- 数据库使用 snake_case，如 `run_id`。
