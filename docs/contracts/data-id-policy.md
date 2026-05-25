# 数据 ID 与 Trace 契约

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

## Artifact ID 三层映射

Core Artifact.artifactId、DB artifacts.id、Public API Artifact.id 是**同一个系统 ID 在不同层级的命名**，不是三套不同 ID：

| 层级 | 字段名 | 说明 |
|---|---|---|
| Core Artifact（内部 JSON / schema） | `artifactId` | 事实源字段名，如 `art_01H...` |
| DB（MySQL `artifacts` 表） | `id` 或 `artifact_id` | 存储 Core `artifactId` 值，不另行生成独立的 public id |
| Public API DTO（对外 JSON） | `id` | `artifactId` 的公开投影 |

命名边界：

- **schema 字段**：使用 `artifactId`。
- **DB 字段**：使用 `id` 或 `artifact_id`，二者必须映射到同一个 `artifactId` 值。
- **API DTO 字段**：使用 `id`，其值为 `artifactId` 的公开投影。
- 不强制修改数据库字段名，不新增迁移代码。
- 不得为 Public API 另行生成与 `artifactId` 不同的 public id。

## 其他资源 ID 规则

- `conversationId` 是 AgentHub 内部主字段。AG-UI `threadId` 和 A2A `metadata.threadId` 是协议别名，不得视为独立会话 ID。
- `runId` 在各层统一使用同一个值，不另行映射。

## 规则

- 每条 message 必须有稳定 id。
- 每次 run 必须有稳定 id。
- 多 Agent 场景下每个 AgentTask 必须能关联 run。
- Artifact 必须能关联 message、run、agent。
- ToolCall 必须能关联 message 或 artifact。
- 错误必须能通过 request_id 或 trace_id 排查。

## 不允许

- 只在日志里保存 trace 信息。
- 前端临时 ID 覆盖数据库 ID。
- 同一 Run 内多个 Agent message 复用同一个 message id。
- 两个不同 Agent 生成同名产物时发生覆盖。
- 为 Public API 另行生成与 artifactId 不同的 public id。
- 在文档中暗示 artifactId、DB id、API id 是三套不同 ID。

## 推荐字段类型

- ID 可使用 UUID、ULID 或项目统一 ID 方案。
- 数据库存储可用 `CHAR(36)` 或项目统一类型。
- 对外 JSON 使用 camelCase，如 `runId`。
- 数据库使用 snake_case，如 `run_id`。
