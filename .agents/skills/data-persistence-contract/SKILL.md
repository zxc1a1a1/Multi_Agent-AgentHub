---
name: data-persistence-contract
description: "用于定义 AgentHub 的数据持久化契约，包括 MVP 阶段 MySQL 表结构、后续 PostgreSQL 演进、conversation/message/run/artifact 等核心数据关系、迁移规则和数据安全边界。"
---

# data-persistence-contract

## 1. Skill 目的

本 Skill 用于定义 AgentHub 项目的数据持久化边界、数据模型、数据库选择、Redis 使用、对象存储策略、migration 规则和跨协议 ID 关联规则。

一句话：

**数据层必须支撑 Conversation、Message、Run、A2A Task、Tool Call、Artifact、Agent、用户和追踪链路，且不能让 Redis、临时内存或 AG-UI 事件流成为事实源。**

---

## 2. 适用场景

当任务涉及以下内容时，必须使用本 Skill：

- 设计数据库表。
- 设计 MySQL / PostgreSQL schema。
- 设计 Redis 用法。
- 设计 Object Storage 用法。
- 设计 Artifact 持久化。
- 设计 message / conversation / run / task / tool call 的关联。
- 编写 migration。
- 编写 seed 数据。
- 编写 store / repository。
- 修改 REST API 涉及的持久化字段。
- 保存 AG-UI Run 结果。
- 保存 A2A Task 结果。
- 保存 Child Agent Artifact。
- 设计数据清理、软删除、归档。
- Review 数据模型是否破坏 Contract。

---

## 3. 数据层边界

数据层负责：

- 用户和鉴权相关数据。
- Conversation。
- Conversation participants。
- Message。
- Agent 注册信息。
- Run。
- Run step。
- A2A task。
- Tool call。
- Artifact。
- Approval。
- Agent health check。
- Prompt / LLM 调用记录的安全摘要。
- traceId / requestId / runId 等可观测字段。

数据层不负责：

- AG-UI 实时事件传输。
- A2A stream 直接转发。
- 前端组件状态。
- LLM prompt 原文无限制持久化。
- API key 明文保存。
- 大文件内容直接塞入普通消息表。
- Redis 作为事实源。

---

## 4. PDR 与 MVP 的数据层解释

PDR 完整目标：

```text
PostgreSQL = 事实源
Redis = 缓存 / 临时状态 / 轻量队列 / rate limit
Object Storage = 大 Artifact 内容
```

MVP v0.1 当前实现：

```text
MySQL 8 = 当前事实源
暂不强制 Redis
暂不强制 Object Storage
Artifact 可以先存 JSON / TEXT 字段或文件路径
```

解释规则：

```text
MVP 可以用 MySQL 8 快速落地。
长期 Contract 仍保留 PostgreSQL + Redis + Object Storage 演进方向。
不得让 Redis 或内存成为消息历史 / Artifact 的唯一存储。
```

---

## 5. 核心文件

本 Skill 落地后应生成或维护：

```text
docs/contracts/data-model.md
docs/contracts/mysql-schema.md
docs/contracts/postgres-schema.md
docs/contracts/redis-usage.md
docs/contracts/object-storage-policy.md
docs/contracts/migration-policy.md
```

MVP v0.1 最小可先落地：

```text
docs/contracts/data-model.md
docs/contracts/mysql-schema.md
docs/contracts/migration-policy.md
```

Post-MVP 再补：

```text
docs/contracts/postgres-schema.md
docs/contracts/redis-usage.md
docs/contracts/object-storage-policy.md
```

---

## 6. MVP v0.1 最小数据模型

MVP v0.1 至少需要：

```text
users
conversations
messages
agents
artifacts
runs
a2a_tasks
tool_calls
```

最小链路：

```text
conversation.id
→ message.conversationId
→ run.conversationId
→ run.id
→ a2a_task.runId
→ artifact.runId
→ artifact.messageId
→ tool_call.artifactId
```

必须能支撑：

- 对话列表。
- 创建对话。
- 查询对话消息。
- 保存用户消息。
- 保存 Agent 回复。
- 保存 Run 状态。
- 保存 A2A taskId。
- 保存 code Artifact。
- 保存 code_preview Tool Call 记录。
- 根据 traceId 排查一次端到端请求。

---

## 7. 推荐核心表

### 7.1 users

MVP 可使用固定用户，但表设计保留：

```text
id
username
displayName
avatarUrl
createdAt
updatedAt
```

MVP 可用：

```text
userId = "demo-user"
```

### 7.2 conversations

```text
id
userId
title
type
agentName
isPinned
isArchived
createdAt
updatedAt
deletedAt
```

规则：

- MVP `type` 可固定为 `single`。
- MVP `agentName` 可固定为 `code-agent`。
- Post-MVP 可支持 group。
- 删除建议软删除。

### 7.3 messages

```text
id
conversationId
runId
senderId
senderType
senderAgent
content
status
createdAt
updatedAt
```

规则：

- `senderType`：`user` / `agent` / `system`
- `status`：`sending` / `streaming` / `sent` / `failed`
- `content` 可存结构化 JSON。
- 大 Artifact 不得塞进 `content`。
- message 可通过 `runId` 关联一次 AG-UI Run。

### 7.4 agents

```text
id
name
type
url
description
status
agentCard
createdAt
updatedAt
lastCheckAt
```

规则：

- MVP 至少注册 `code-agent`。
- `agentCard` 可用 JSON 存储。
- `status`：`healthy` / `unhealthy` / `unknown`
- Post-MVP 支持 Agent Registry / health check。

### 7.5 runs

```text
id
conversationId
userId
agentName
status
traceId
requestId
startedAt
finishedAt
createdAt
updatedAt
```

规则：

- `id` 对应 AG-UI `runId`。
- `status`：`running` / `completed` / `failed` / `cancelled`
- 每次 `/api/agui/run` 应生成或传入 runId。
- 错误时必须记录失败状态。

### 7.6 a2a_tasks

```text
id
runId
agentName
status
traceId
startedAt
finishedAt
errorCode
errorMessage
createdAt
updatedAt
```

规则：

- `id` 对应 A2A taskId。
- `runId` 关联 AG-UI Run。
- 不保存敏感 prompt。
- 错误信息不得保存 API key / token / 堆栈。

### 7.7 tool_calls

```text
id
runId
messageId
artifactId
toolName
args
status
createdAt
updatedAt
```

规则：

- `toolName` MVP 至少支持 `code_preview`。
- `args` 必须符合 Frontend Runtime Skill schema。
- `artifactId` 关联 Artifact。
- 不保存过大内容时应引用 Artifact。

### 7.8 artifacts

```text
id
conversationId
messageId
runId
type
title
content
fileUrl
metadata
version
createdAt
updatedAt
```

规则：

- MVP 至少支持 `type = code`。
- `metadata.language` 对 code Artifact 必须存在。
- 大内容 Post-MVP 应进入 Object Storage。
- Artifact 必须能映射到 Frontend Runtime Skill。
- Artifact 不得只存在于 AG-UI 事件流中。

---

## PDR / Post-MVP 长期表规划

除 MVP v0.1 最小数据模型外，`data-persistence-contract` 必须保留 PDR 和 v1.1 Skills 设计规范中的长期数据模型规划。

以下表属于 **Post-MVP planned**，MVP v0.1 暂不强制实现，但不能从长期数据设计中删除。

### CONVERSATION_PARTICIPANT

用途：

```text
记录会话参与者，用于群聊、多 Agent 协作、用户与 Agent 混合会话。
```

建议字段：

```text
id
conversationId
participantType
participantId
role
joinedAt
leftAt
createdAt
updatedAt
```

字段说明：

- `conversationId` 关联 `conversations.id`。
- `participantType` 可为 `user / agent`。
- `participantId` 指向用户 ID 或 Agent name / Agent ID。
- `role` 可为 `owner / member / agent`。
- MVP v0.1 单聊可暂不建该表。
- Post-MVP 群聊和多 Agent 协作必须使用该表或等价模型。

### RUN_STEP

用途：

```text
记录一次 Run 内部的关键步骤，用于调试 Orchestrator、A2A 调用、ProtocolConverter、Tool Call 和失败定位。
```

建议字段：

```text
id
runId
stepType
agentName
a2aTaskId
toolCallId
status
inputSummary
outputSummary
errorCode
errorMessage
startedAt
finishedAt
createdAt
updatedAt
```

字段说明：

- `runId` 关联 `runs.id`。
- `stepType` 可为 `orchestration / a2a_task / converter / tool_call / artifact_flush`。
- `a2aTaskId` 可关联 `a2a_tasks.id`。
- `toolCallId` 可关联 `tool_calls.id`。
- `inputSummary / outputSummary` 只能保存脱敏摘要，不得保存 API key、token、完整敏感 prompt。
- MVP v0.1 可先不落库，但 Post-MVP 调试和观测应补充。
- `converter`、`tool_call`、`artifact_flush` 的细节由后续对应 Skill 细化。

### APPROVAL

用途：

```text
记录 confirm_action 和高危操作审批结果。
```

建议字段：

```text
id
runId
toolCallId
userId
actionType
status
requestPayload
decisionPayload
requestedAt
decidedAt
createdAt
updatedAt
```

字段说明：

- `runId` 关联 `runs.id`。
- `toolCallId` 关联 `tool_calls.id`。
- `actionType` 可为 `deploy / run_command / file_overwrite / external_request`。
- `status` 可为 `pending / approved / rejected / expired`。
- `requestPayload` 和 `decisionPayload` 必须脱敏。
- MVP v0.1 不实现高危操作时可以暂不建表。
- Post-MVP 引入 confirm_action、部署、命令执行、文件覆盖前必须补齐该表或等价模型。
- `confirm_action` 的 Frontend Runtime Skill 参数和 ToolResult 由后续对应 Skill 细化。
- 高危操作安全策略由 `security-boundary-contract` 细化。

### AGENT_HEALTH_CHECK

用途：

```text
记录 Agent 健康检查结果，用于 Agent Registry、Agent 状态展示、fallback 和调试。
```

建议字段：

```text
id
agentName
agentUrl
status
latencyMs
errorCode
errorMessage
checkedAt
createdAt
```

字段说明：

- `agentName` 对应 `agents.name`。
- `status` 可为 `healthy / unhealthy / unknown`。
- `latencyMs` 记录健康检查延迟。
- `errorMessage` 必须脱敏。
- MVP v0.1 配置文件静态注册 `code-agent` 时可以暂不建该表。
- Post-MVP 引入 Agent Registry、动态发现、fallback / retry 前应补充该表或等价模型。
- Agent Registry 和路由策略由后续对应 Skill 细化。

## 8. 字段命名规则

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

Go struct 示例：

```go
type Message struct {
    ID             string    `json:"id" db:"id"`
    ConversationID string   `json:"conversationId" db:"conversation_id"`
    CreatedAt      time.Time `json:"createdAt" db:"created_at"`
}
```

---

## 9. ID 关联规则

必须贯穿：

```text
requestId
traceId
conversationId
messageId
runId
a2aTaskId
toolCallId
artifactId
agentName
userId
```

规则：

- `traceId` 贯穿 Frontend、Gateway、Orchestrator、A2A、Child Agent。
- `runId` 关联一次 AG-UI run。
- `a2aTaskId` 关联一次 Child Agent task。
- `artifactId` 关联产物。
- `messageId` 关联消息。
- 所有跨协议 ID 必须可追溯。
- 不允许只靠日志定位核心链路。

---

## 10. Redis 使用规则

Post-MVP Redis 可用于：

- 在线状态。
- 轻量缓存。
- rate limit。
- session 临时状态。
- run 临时状态。
- SSE fanout 辅助。
- 轻量队列。
- 分布式锁。

Redis 不允许作为：

- 消息历史唯一存储。
- Artifact 唯一存储。
- 用户数据唯一存储。
- AgentCard 唯一存储。
- Run 结果唯一存储。

MVP v0.1 暂不使用 Redis 是允许的。

---

## 11. Object Storage 规则

Post-MVP Object Storage 用于：

- 大代码包。
- HTML / CSS / JS 压缩包。
- 图片。
- 文档。
- zip。
- 日志文件。
- 大型 Artifact。
- 部署产物。

规则：

- 数据库只保存 `fileUrl` / object key / metadata。
- 下载 URL 应短期有效。
- 私有 Artifact 必须鉴权。
- 不允许公开暴露敏感文件。
- 删除 Artifact 时必须处理对象存储清理。
- MVP 可暂不使用对象存储，但 Contract 必须保留演进方向。

---

## 12. Migration 规则

所有 schema 变更必须通过 migration。

规则：

- migration 文件必须版本化。
- migration 必须可重复执行或可明确失败。
- 不允许手工改库不留记录。
- 不允许直接在生产库执行未 review SQL。
- migration 必须包含 up/down 或明确不可逆说明。
- 修改表字段必须同步更新数据模型文档。
- 修改 API 相关字段必须同步 OpenAPI。
- 修改 Artifact 相关字段必须同步 Artifact Contract。

推荐命名：

```text
000001_init_schema.sql
000002_add_runs.sql
000003_add_artifacts.sql
```

---

## 13. JSON / JSONB 字段规则

长期 PostgreSQL 可使用 JSONB。  
MVP MySQL 可使用 JSON 类型。

适用字段：

```text
messages.content
agents.agent_card
artifacts.metadata
tool_calls.args
runs.context
```

规则：

- 每个 JSON 字段必须有 schema 说明。
- 不允许无约束地塞任意数据。
- 不允许把 API key / token 塞进 JSON 字段。
- 不允许把大文件内容长期塞进 JSON 字段。
- JSON 字段变更必须更新 Contract。

---

## 14. MVP v0.1 数据落地规则

MVP v0.1 必须优先支持：

- 对话列表。
- 创建对话。
- 消息保存。
- 历史消息查询。
- `code-agent` 注册。
- `/api/agui/run` 对应 run 记录。
- A2A taskId 记录。
- code Artifact 记录。
- code_preview tool call 记录。

MVP v0.1 可暂不支持：

- 多用户注册登录。
- 群聊参与者完整模型。
- Redis。
- Object Storage。
- Artifact 版本历史。
- 高级搜索。
- 复杂消息分页。
- Agent 健康检查历史。
- approval 记录。
- deploy 记录。

---

## 15. 安全规则

数据层不得保存：

- 明文 LLM API key。
- 明文用户 token。
- 明文服务间 token。
- 完整敏感 system prompt。
- 未脱敏生产日志。
- 用户隐私文件明文路径。

必须：

- 对 API key 做环境变量 / secret 管理。
- 对敏感字段做脱敏日志。
- 对私有 Artifact 做权限校验。
- 对软删除数据做访问过滤。
- 对 object storage URL 做过期控制。
- 对 file upload 做类型和大小限制。

---

## 16. Contract Test 规则

数据层至少应测试：

- migration 是否可运行。
- 核心表是否存在。
- 必填字段是否存在。
- 索引是否覆盖常用查询。
- message 能关联 conversation。
- run 能关联 conversation / message。
- a2a_task 能关联 run。
- artifact 能关联 run / message。
- tool_call 能关联 artifact。
- JSON 字段是否符合 schema。
- 删除 / 归档是否不破坏历史引用。
- MVP 查询接口能返回 OpenAPI 要求字段。

---

## 17. 与其他 Skills 的协作

- REST API 字段必须能由数据模型支撑。
- AG-UI event 是实时过程，不是持久化事实源。
- Gateway 负责保存用户消息和 OrchestratorResult。
- A2A taskId 和 Agent 输出 Artifact 必须可持久化。
- Artifact 具体字段、类型、metadata 由 Artifact Contract 细化。
- 敏感数据、API key、文件权限、下载 URL 由 Security Contract 细化。

---

## 18. 硬性规则

1. MVP v0.1 使用 MySQL 8 是允许的。
2. 长期目标保留 PostgreSQL + Redis + Object Storage。
3. 数据库是持久化事实源。
4. Redis 不能作为消息历史或 Artifact 唯一存储。
5. Object Storage 保存大 Artifact。
6. 所有 migration 必须版本化。
7. API JSON 使用 camelCase，数据库列名可用 snake_case。
8. 所有跨协议 ID 必须可关联。
9. `runId`、`traceId`、`messageId`、`artifactId` 必须可追踪。
10. 大 Artifact 不得塞进 message.content。
11. Artifact 不能只存在于 AG-UI 事件流中。
12. JSON 字段必须有 schema 说明。
13. 敏感信息不得明文入库。
14. 修改持久化字段必须同步 Contract。
15. 必须遵守 `Contract first / Mock first / Real integration later / Review always`。

---

## 19. 必须维护的文件

本 Skill 本体：

```text
skills/data-persistence-contract/SKILL.md
```

正式 Contract：

```text
docs/contracts/data-model.md
docs/contracts/mysql-schema.md
docs/contracts/postgres-schema.md
docs/contracts/redis-usage.md
docs/contracts/object-storage-policy.md
docs/contracts/migration-policy.md
```

MVP 最小可先维护：

```text
docs/contracts/data-model.md
docs/contracts/mysql-schema.md
docs/contracts/migration-policy.md
```

---

## 20. Review Checklist

- [ ] 是否明确 MVP MySQL 8？
- [ ] 是否保留长期 PostgreSQL / Redis / Object Storage？
- [ ] 是否定义 conversations？
- [ ] 是否定义 messages？
- [ ] 是否定义 agents？
- [ ] 是否定义 runs？
- [ ] 是否定义 a2a_tasks？
- [ ] 是否定义 tool_calls？
- [ ] 是否定义 artifacts？
- [ ] 是否定义跨协议 ID？
- [ ] 是否说明 Redis 不能作为事实源？
- [ ] 是否说明 Object Storage 保存大 Artifact？
- [ ] 是否有 migration policy？
- [ ] JSON 字段是否有 schema？
- [ ] 是否没有保存明文 API key / token？
- [ ] 是否与 OpenAPI / Artifact / A2A Contract 一致？

- [ ] 是否明确 `CONVERSATION_PARTICIPANT` 属于 Post-MVP 群聊 / 多 Agent 参与者模型？
- [ ] 是否明确 `RUN_STEP` 属于 Post-MVP Run 内部步骤追踪模型？
- [ ] 是否明确 `APPROVAL` 属于 Post-MVP confirm_action / 高危操作审批模型？
- [ ] 是否明确 `AGENT_HEALTH_CHECK` 属于 Post-MVP Agent Registry / 健康检查模型？
- [ ] 是否明确这些长期表 MVP v0.1 暂不强制实现，但不能从长期规划中删除？
- [ ] 是否在涉及尚未完成的 Skill 时只写“由后续对应 Skill 细化”，没有强制要求当前文件已存在？


## References

- `references/mysql-mvp-schema.md`
- `references/postgres-post-mvp-schema.md`
- `references/migration-policy.md`
- `references/persistence-id-policy.md`
- `references/artifact-persistence-policy.md`
- `references/data-review-checklist.md`
