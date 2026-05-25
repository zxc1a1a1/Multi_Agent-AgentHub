---
name: data-persistence-contract
description: "用于定义 AgentHub 的数据持久化事实源契约，包括关系型数据模型、Conversation/Message/Run/Agent/Artifact/ToolCall 关联、群聊与多 Agent 数据、迁移规则、JSON 字段、索引、软删除、数据安全和存储演进策略。"
---

# data-persistence-contract

## 1. Skill 目的

本 Skill 用于规范 AgentHub 项目的数据持久化层。

它定义：

- 哪些数据必须落库。
- 哪些数据只是临时状态。
- Conversation、Message、Run、Agent、Artifact、ToolCall 如何关联。
- 单聊与群聊如何建模。
- 多 Agent 参与同一次 Run 时如何追踪。
- Agent 注册信息与健康状态如何持久化。
- 迁移文件如何编写。
- JSON 字段如何约束。
- 数据安全、脱敏、索引、软删除、归档如何处理。

一句话：

**数据库是 AgentHub 历史对话、运行记录、Agent 状态、产物、工具调用与审计链路的事实源；实时事件、内存状态、缓存和日志都不能替代数据库事实源。**

---

## 2. 独立性原则

本 Skill 是独立的数据持久化契约。

本 Skill 不要求读者先阅读其他 Skill 才能理解数据层规则。

本 Skill 不定义：

- 前端组件实现。
- 实时事件字段。
- 子 Agent 内部运行时 API。
- 外部 Agent 协议细节。
- Planner 的具体算法。
- Docker 服务编排。
- 具体 UI 交互。

本 Skill 只定义这些外部行为在数据层中的**可持久化事实、ID 关联、状态、索引和安全边界**。

---

## 3. 当前阶段识别

当前项目已完成 MVP v0.1，正在进行 v1.0 及后续迭代。

### 3.1 MVP v0.1 Historical Profile

MVP v0.1 已完成，仅作为历史兼容和回归测试基线。

历史基线包括：

- 单用户或 demo 用户。
- 单 Agent 对话。
- 单聊 conversation。
- 基础 message 持久化。
- 基础 run 记录。
- 基础 artifact 记录。
- 基础 tool call 记录。
- MySQL 8 作为事实源。

这些历史基线不得继续作为当前开发禁令。

### 3.2 v1.0 Generic Persistence Profile

v1.0 当前数据层必须支持：

- 2+ Child Agents。
- 单聊与群聊。
- `conversation_type`。
- `conversation_participants`。
- Agent Registry 数据。
- Agent 健康状态。
- 一个 Run 关联多个 Agent 任务。
- 一个 Run 产生多条消息。
- 一个消息产生多个产物。
- 多类型 Artifact。
- 多类型 ToolCall。
- fallback / retry 状态记录。
- `trace_id`、`request_id`、`run_id`、`message_id`、`artifact_id` 等可追踪 ID。

本 Skill 不固定具体 Agent 名称。

### 3.3 Post-v1.0 Planned Profile

后续可扩展：

- 审批与人工确认表。
- 下载授权 token 表。
- Prompt 审计表。
- Artifact 版本历史表。
- 长期归档表。
- 多租户组织表。
- PostgreSQL Profile。
- Object Storage 大对象 Profile。

---

## 4. 数据层事实源原则

必须落库的数据：

- 用户。
- 会话。
- 会话参与者。
- 消息。
- Agent 注册信息。
- Agent 健康状态。
- Run。
- RunStep。
- AgentTask。
- ToolCall。
- Artifact。
- 关键错误状态。
- 可追踪 ID。

可以作为临时状态的数据：

- SSE 当前连接。
- 实时 streaming buffer。
- 前端 loading 状态。
- 临时重试倒计时。
- Registry 内存缓存。
- 健康检查本轮探测上下文。

禁止：

- 只把历史消息存在前端状态。
- 只把产物存在实时事件流。
- 只把 Agent 健康状态存在日志里。
- 只把 Run 执行链路存在内存里。
- 只靠日志排查用户历史问题。

---

## 5. Storage Profiles

### 5.1 Current Profile: MySQL 8

v1.0 当前使用 MySQL 8 作为关系型事实源。

要求：

- 所有核心实体必须有稳定主键。
- 常用查询字段必须建索引。
- 时间字段统一使用 `created_at`、`updated_at`、`deleted_at`。
- 对外 API 字段可以是 camelCase，但数据库字段使用 snake_case。
- JSON 字段必须有结构说明。
- 破坏性 schema 变更必须走 migration。

### 5.2 Planned Relational Profile: PostgreSQL

PostgreSQL 是长期可选演进方向，不影响当前 MySQL 落地。

如果后续迁移到 PostgreSQL：

- JSON 字段可演进为 JSONB。
- 常用 JSON 查询路径必须补索引。
- 不得把经常 join / filter / order 的字段藏在 JSONB。

### 5.3 Cache Profile: Redis

Redis 只能用于：

- 缓存。
- 限流。
- 临时锁。
- 短期任务状态。
- 非事实源队列状态。

Redis 不得成为：

- 消息唯一事实源。
- Artifact 唯一事实源。
- Run 唯一事实源。
- Agent Registry 唯一事实源。

### 5.4 Large Object Profile: Object Storage

Object Storage 用于大型内容：

- 大文件。
- 图片。
- zip。
- 大型 HTML / 文档。
- 长日志。
- 可下载产物。

关系型数据库保存 `content_ref`、元数据、权限和关联关系，不直接保存大型二进制内容。

---

## 6. 核心实体总览

v1.0 数据层建议包含：

| 实体 | 作用 |
|---|---|
| `users` | 用户或 demo 用户 |
| `conversations` | 单聊 / 群聊会话 |
| `conversation_participants` | 会话参与者，支持 user / agent |
| `messages` | 用户、Agent、系统消息 |
| `agents` | Agent 注册信息与能力摘要 |
| `agent_health_checks` | Agent 健康检查历史或快照 |
| `runs` | 一次用户请求触发的运行 |
| `run_steps` | Run 内部阶段与步骤 |
| `agent_tasks` | 发给某个 Agent 的子任务 |
| `tool_calls` | 前端可消费的工具调用记录 |
| `artifacts` | Agent 生成的产物事实源 |

可选或规划实体：

| 实体 | 作用 |
|---|---|
| `approvals` | 高风险操作人工确认 |
| `prompt_audits` | Prompt 审计摘要 |
| `artifact_versions` | 产物版本历史 |
| `download_tokens` | 下载授权与过期控制 |

---

## 7. v1.0 Required Tables

### 7.1 users

最小字段：

```text
id
name
email
status
created_at
updated_at
deleted_at
```

规则：

- MVP 可以只有 demo 用户。
- v1.0 不得把用户身份硬编码到业务逻辑。
- 密码、token、API key 不得明文保存。

### 7.2 conversations

最小字段：

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

规则：

- `conversation_type = single | group`。
- `primary_agent_name` 只表示默认 Agent 或展示用途，不代表群聊唯一 Agent。
- 群聊参与者必须通过 `conversation_participants` 表表达。
- 删除默认软删除。

### 7.3 conversation_participants

最小字段：

```text
id
conversation_id
participant_type
participant_id
display_name
role
status
joined_at
left_at
created_at
updated_at
```

规则：

- `participant_type = user | agent`。
- `role = owner | member | agent`。
- 群聊至少应有一个 user 和一个或多个 agent participant。
- Agent 名称不应写死。

### 7.4 messages

最小字段：

```text
id
conversation_id
run_id
sender_type
sender_id
sender_name
content
content_format
status
created_at
updated_at
deleted_at
```

规则：

- `sender_type = user | agent | system`。
- `sender_name` 用于 UI 展示和历史回放。
- Agent 消息建议保存 `sender_id` 或 `sender_name` 对应 Agent 名称。
- `content_format = text | markdown | json`。
- 大 Artifact 不得塞进 `messages.content`。
- 一个 Run 可以产生多条 Agent message。

### 7.5 agents

最小字段：

```text
id
name
display_name
description
url
version
status
agent_card
skills
input_modes
output_modes
last_check_at
last_error_code
created_at
updated_at
deleted_at
```

规则：

- `name` 必须唯一。
- `status = healthy | unhealthy | unknown | disabled`。
- `agent_card` 可保存能力声明摘要。
- `skills`、`input_modes`、`output_modes` 可以使用 JSON，但必须有结构说明。
- 不得保存 Agent 内部 secret。

### 7.6 agent_health_checks

最小字段：

```text
id
agent_name
agent_url
status
latency_ms
error_code
error_message
checked_at
created_at
```

规则：

- 健康检查失败必须可追踪。
- `error_message` 必须脱敏。
- 可只保留最近一段时间的历史。
- 如果不建独立历史表，`agents` 表必须至少保存当前健康状态。

### 7.7 runs

最小字段：

```text
id
conversation_id
user_id
status
strategy
intent_summary
trace_id
request_id
started_at
finished_at
error_code
error_message
created_at
updated_at
```

规则：

- `strategy = single | parallel | sequential`。
- `intent_summary` 只能保存脱敏摘要。
- Run 不应绑定唯一 Agent。
- 一个 Run 可以关联多个 message、step、agent task、artifact、tool call。

### 7.8 run_steps

最小字段：

```text
id
run_id
step_type
step_order
agent_name
status
input_summary
output_summary
error_code
error_message
started_at
finished_at
created_at
updated_at
```

规则：

- `step_type = planning | dispatch | agent_task | tool_call | artifact | retry | fallback`。
- `input_summary` 与 `output_summary` 只能保存摘要。
- 不保存完整敏感 prompt。

### 7.9 agent_tasks

最小字段：

```text
id
run_id
step_id
agent_name
status
task_ref
input_summary
output_summary
error_code
error_message
started_at
finished_at
created_at
updated_at
```

规则：

- `task_ref` 可保存外部任务 ID。
- `agent_name` 不固定具体 Agent。
- 同一个 Run 可以有多个 AgentTask。
- v1.0 兼容期可以继续使用旧表名，但契约语义应是通用 AgentTask。

### 7.10 tool_calls

最小字段：

```text
id
run_id
message_id
artifact_id
tool_name
args
status
error_code
error_message
created_at
updated_at
```

规则：

- `tool_name` 不固定具体预览工具。
- `args` 是 JSON，但必须有结构说明。
- 大内容应引用 Artifact，不应直接塞进 `args`。
- ToolCall 必须能关联 message 或 artifact。

### 7.11 artifacts

最小字段：

```text
id
conversation_id
message_id
run_id
agent_name
type
title
mime_type
content
content_ref
metadata
version
status
created_at
updated_at
deleted_at
```

规则：

- `type` 不固定为单一类型。
- v1.0 至少能持久化 `code`、`webpage`、`markdown` 类产物。
- 小内容可以 `content` inline。
- 大内容必须使用 `content_ref`。
- `metadata` 必须有结构说明。
- Artifact 不得只存在于实时事件流。

---

## 8. 字段命名规则

数据库字段使用 snake_case：

```text
conversation_id
message_id
run_id
agent_name
created_at
updated_at
```

对外 JSON 使用 camelCase：

```text
conversationId
messageId
runId
agentName
createdAt
updatedAt
```

规则：

- 数据库字段名不得直接泄漏到 API 作为唯一命名标准。
- API 层必须负责字段转换。
- 不得在同一表中混用 `createdAt` 和 `created_at`。
- 布尔字段使用明确语义，如 `is_archived`、`is_pinned`。

---

## 9. ID 与 Trace 关联规则

必须贯穿的 ID：

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

规则：

- 一个 Run 可以关联多个 message。
- 一个 Run 可以关联多个 step。
- 一个 Run 可以关联多个 agent task。
- 一个 message 可以关联多个 artifact。
- 一个 artifact 必须能追溯到 conversation、message、run、agent。
- 错误必须能关联到 request_id 或 trace_id。
- 不允许只靠日志定位核心链路。

---

## 10. JSON 字段规则

允许 JSON 字段：

```text
agents.agent_card
agents.skills
agents.input_modes
agents.output_modes
runs.plan_json
tool_calls.args
artifacts.metadata
messages.content_json
```

规则：

- JSON 字段必须有结构说明。
- 经常查询、过滤、排序、关联的字段不得只藏在 JSON。
- JSON 字段不得保存 secret。
- JSON 字段不得保存大型文件内容。
- JSON 字段不得变成任意结构垃圾桶。
- 重要 JSON 结构变更必须写 migration 或兼容说明。

---

## 11. Index 与查询规则

建议索引：

```text
conversations(user_id, updated_at)
conversations(conversation_type)
conversation_participants(conversation_id)
conversation_participants(participant_type, participant_id)
messages(conversation_id, created_at)
messages(run_id)
agents(name)
agents(status)
agent_health_checks(agent_name, checked_at)
runs(conversation_id, created_at)
runs(trace_id)
run_steps(run_id, step_order)
agent_tasks(run_id)
agent_tasks(agent_name, created_at)
tool_calls(message_id)
tool_calls(artifact_id)
artifacts(message_id)
artifacts(run_id)
artifacts(agent_name, created_at)
```

规则：

- 聊天历史查询必须走 `conversation_id + created_at`。
- Agent 列表查询必须能按 `status` 过滤。
- Run 排查必须能通过 `trace_id` 定位。
- 群聊成员查询必须走 `conversation_participants.conversation_id`。
- 常用查询必须有索引计划。

---

## 12. Migration 规则

所有 schema 变更必须通过 migration。

规则：

- migration 文件必须有递增版本号。
- 每次 schema 变更必须有 up migration。
- down migration 可以没有，但必须标注 irreversible。
- 破坏性变更必须使用 expand / migrate / contract。
- 字段重命名不能一步完成：先加新字段，双写，回填，切读，再删旧字段。
- 删除列、改类型、改 nullability 必须说明风险。
- 大表回填必须分批。
- migration 后必须更新数据模型文档。
- 禁止手工改库不留记录。

推荐阶段：

```text
expand: 增加兼容字段 / 表 / 索引
migrate: 回填数据、双写、切换读路径
contract: 删除旧字段、清理兼容逻辑
```

---

## 13. 软删除、归档与保留

规则：

- 用户可见历史默认软删除。
- `deleted_at` 存在表示软删除。
- 归档使用 `is_archived`，不要等同删除。
- 删除 conversation 不应立即硬删 messages / artifacts。
- Artifact 删除应避免破坏历史消息引用。
- 健康检查历史可设置保留窗口。
- 大型日志与临时调试数据应有保留策略。

---

## 14. 数据安全与脱敏

数据库不得保存：

- 明文 LLM API key。
- 明文用户 token。
- 明文服务间 token。
- 数据库连接串。
- 完整敏感 system prompt。
- 未脱敏 LLM 原始请求 / 响应。
- 私有文件绝对路径。
- 永久公开下载 URL。
- 内网服务拓扑。
- 未脱敏用户隐私。

规则：

- Secret 应来自环境变量或 secret manager。
- token 只能保存 hash、摘要、过期时间、撤销状态等必要信息。
- error_message 必须面向用户和日志场景分别脱敏。
- summary 字段只能保存摘要，不保存完整敏感输入。
- content_ref 不能是永久公开 URL。

---

## 15. Contract Test 规则

数据层至少应验证：

- Migration 可以从空库执行成功。
- 核心表存在。
- 核心索引存在。
- 可以创建单聊 conversation。
- 可以创建群聊 conversation。
- 可以添加 user participant。
- 可以添加 agent participant。
- 可以保存 user message。
- 可以保存 agent message 且包含 sender_name。
- 可以保存 run。
- 可以保存 run_steps。
- 可以保存 agent_tasks。
- 可以保存 tool_calls。
- 可以保存 artifacts。
- 可以通过 conversation 查询完整历史。
- 可以通过 run_id 查询运行链路。
- 可以通过 trace_id 定位问题。
- JSON 字段结构有效。
- soft delete 不破坏历史关联。

---

## 16. 禁止事项

禁止：

- 把 MVP 历史基线当成当前开发禁令。
- 把 Agent 名称写死在 schema 里。
- 只支持单一 Agent。
- 只支持单聊。
- 只支持单一 Artifact 类型。
- 不写 migration 直接改 init.sql。
- 用 JSON 字段逃避建模。
- 把大文件塞进数据库 content。
- 把 secret 存入数据库。
- 把内部栈和原始 prompt 存入用户可见错误字段。
- 只靠前端状态或日志恢复历史。

---

## 17. Review Checklist

### 阶段

- 是否没有把 MVP 历史基线当成当前禁令？
- 是否支持 2+ Agent？
- 是否支持单聊和群聊？
- 是否不固定具体 Agent 名称？

### 表结构

- 是否有 conversations？
- 是否有 conversation_participants？
- 是否有 messages？
- 是否有 agents？
- 是否有 agent_health_checks 或等价字段？
- 是否有 runs？
- 是否有 run_steps？
- 是否有 agent_tasks 或兼容表？
- 是否有 tool_calls？
- 是否有 artifacts？

### 关联

- message 是否关联 conversation？
- run 是否关联 conversation？
- run 是否能关联多 message？
- agent_task 是否关联 run？
- artifact 是否关联 message / run / agent？
- tool_call 是否关联 message / artifact？
- 多 Agent 消息是否能追溯 agentName？

### Migration

- 是否有版本化 migration？
- 是否使用 expand / migrate / contract 处理破坏性变更？
- 是否避免手工改库不留记录？
- 是否更新数据模型文档？

### JSON

- JSON 字段是否有结构说明？
- 常用查询字段是否没有藏在 JSON？
- JSON 是否没有保存 secret？

### 安全

- 是否没有明文 API key / token？
- 是否没有完整 system prompt？
- 是否没有未脱敏 LLM 请求响应？
- 是否没有永久公开文件 URL？

---

## 18. 完成定义

本 Skill 视为完成，当且仅当：

- 数据层事实源原则明确。
- MVP v0.1 已降级为历史基线。
- v1.0 支持 2+ Agent 的数据模型。
- v1.0 支持单聊与群聊。
- Agent Registry 与 health 状态有持久化策略。
- Run、RunStep、AgentTask、ToolCall、Artifact 关联明确。
- JSON 字段规则明确。
- Migration 规则明确。
- MySQL 当前 Profile 明确。
- Redis / Object Storage / PostgreSQL 的边界明确。
- 数据安全与脱敏规则明确。
- Review Checklist 明确。

---

## References

- `references/data-model-overview.md`
- `references/mysql-current-schema.md`
- `references/migration-policy.md`
- `references/id-and-trace-policy.md`
- `references/conversation-message-policy.md`
- `references/run-and-task-policy.md`
- `references/agent-registry-persistence.md`
- `references/artifact-persistence-policy.md`
- `references/json-field-policy.md`
- `references/redis-object-storage-policy.md`
- `references/data-security-policy.md`
- `references/data-review-checklist.md`
