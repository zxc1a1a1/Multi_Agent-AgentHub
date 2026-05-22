# Data Model Contract

## 1. 文档目的

本文档定义 AgentHub 的数据模型边界和核心实体关系。

数据层必须支持：

```text
Conversation
Message
Run
A2A Task
Tool Call
Artifact
Agent
User
Trace
```

数据层不能让 Redis、内存、AG-UI 事件流成为事实源。

## 2. PDR 与 MVP 解释

PDR 长期目标：

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

MVP 可以用 MySQL 8 快速落地，但长期数据模型保留 PostgreSQL + Redis + Object Storage 方向。

## 3. MVP 最小核心表

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

最小关联链路：

```text
conversation.id
→ message.conversationId
→ run.conversationId
→ run.id
→ a2aTask.runId
→ artifact.runId
→ artifact.messageId
→ toolCall.artifactId
```

## 4. users

用途：记录用户信息。MVP 可使用固定用户。

字段：

```text
id
username
displayName
avatarUrl
createdAt
updatedAt
```

MVP 可固定：

```text
userId = demo-user
```

## 5. conversations

字段：

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
- 删除建议软删除。
- Post-MVP 支持 group。

## 6. messages

字段：

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
- 大 Artifact 不得塞进 `content`。
- message 可通过 `runId` 关联一次 AG-UI Run。

## 7. agents

字段：

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
- Post-MVP 支持 Agent Registry 和健康检查。

## 8. runs

字段：

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

## 9. a2a_tasks

字段：

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
- 错误信息不得保存 API key / token / 堆栈。

## 10. tool_calls

字段：

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

- MVP 至少支持 `toolName = code_preview`。
- `args` 必须符合 Frontend Runtime Skill schema。
- `artifactId` 关联 Artifact。

## 11. artifacts

字段：

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
- code Artifact 必须有 `metadata.language`。
- 大内容 Post-MVP 进入 Object Storage。
- Artifact 不得只存在于 AG-UI 事件流中。

## 12. Post-MVP planned 表

以下表属于长期规划，MVP v0.1 暂不强制实现，但不能从长期模型删除。

### CONVERSATION_PARTICIPANT

用于群聊、多 Agent 协作、用户与 Agent 混合会话。

字段：

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

### RUN_STEP

用于记录 Run 内部步骤，便于调试 Orchestrator、A2A、ProtocolConverter、Tool Call 和 Artifact flush。

字段：

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

`inputSummary` / `outputSummary` 必须脱敏。

### APPROVAL

用于记录 `confirm_action` 和高危操作审批。

字段：

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

MVP v0.1 不实现高危操作时可暂不建表。

### AGENT_HEALTH_CHECK

用于 Agent Registry、Agent 状态展示、fallback 和调试。

字段：

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

MVP v0.1 静态注册 `code-agent` 时可暂不建表。

## 13. 跨协议 ID

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
- 所有跨协议 ID 必须可追溯。
