---
name: platform-api-contract
description: Use when changing Frontend-to-Gateway REST APIs, OpenAPI schemas, API response formats, pagination, auth headers, or generated API client contracts.
---

# platform-api-contract

## 1. Skill 目的

本 Skill 用于定义 AgentHub 项目的 REST API / OpenAPI 契约规则。

`platform-api-contract` 的核心职责是固定 Frontend 与 Gateway 之间的持久化资源 API，包括：

- 会话管理
- 消息查询
- Agent 列表与 AgentCard 查询
- 自建 Agent 管理
- Artifact 查询与预览
- 统一响应格式
- 分页格式
- 错误格式
- 鉴权 Header
- 资源 ID 规范
- OpenAPI 维护规则
- 前端 API 类型生成规则
- 后端 Handler 与 OpenAPI 的一致性规则

本 Skill 不负责 AG-UI 实时事件格式，不负责 A2A Task 格式，不负责 Gateway 与 Orchestrator 的内部事件格式。

一句话：  
**凡是 Frontend 通过 REST API 查询或管理持久化资源的接口，都必须由本 Skill 约束，并写入 `docs/contracts/openapi.yaml`。**

---

## 2. 适用场景

当任务涉及以下内容时，必须使用本 Skill：

- 编写或修改 REST API。
- 编写或修改 `docs/contracts/openapi.yaml`。
- 设计 Frontend 与 Gateway 的 HTTP API。
- 设计会话、消息、Agent、Artifact 的查询接口。
- 设计自建 Agent 的创建、更新、删除接口。
- 定义 REST API response schema。
- 定义 REST API error schema。
- 定义分页参数和分页响应。
- 定义鉴权 Header。
- 生成前端 API Client。
- 生成或校验 Go Handler。
- 做 API contract review。
- 判断某个接口应该是 REST API、AG-UI 事件、A2A 任务，还是内部 Gateway-Orchestrator Contract。

---

## 3. Contract 所属边界

本 Skill 只约束：

```text
React Frontend ↔ Gateway Service
```

本 Skill 不约束：

```text
Gateway Service ↔ Orchestrator Service
Orchestrator Service ↔ Child Agents
Gateway Service ↔ Frontend 的 AG-UI 实时事件
Child Agent 的 A2A endpoints
```

对应关系如下：

| 通信方向 | 使用协议 / Contract | 是否由本 Skill 管 |
|---|---|---|
| Frontend → Gateway 查询会话 | REST API / OpenAPI | 是 |
| Frontend → Gateway 查询消息 | REST API / OpenAPI | 是 |
| Frontend → Gateway 查询 Agent | REST API / OpenAPI | 是 |
| Frontend → Gateway 查询 Artifact | REST API / OpenAPI | 是 |
| Frontend ↔ Gateway 实时流式输出 | AG-UI | 否，由 `agui-event-contract` 管 |
| Gateway ↔ Orchestrator 内部 Run | Internal Contract | 否，由 `gateway-orchestrator-contract` 管 |
| Orchestrator ↔ Child Agent | A2A | 否，由 `a2a-agent-contract` 管 |
| Artifact 数据结构 | Artifact Contract | 否，由 `artifact-contract` 管，但 REST API 要引用其 schema |
| Frontend Runtime Skills 参数 | Frontend Runtime Skills Contract | 否，由 `frontend-runtime-skills-contract` 管 |

---

## 4. 核心文件

本 Skill 的唯一事实源文件是：

```text
docs/contracts/openapi.yaml
```

所有 REST API 都必须写进这个文件。

可选辅助文件：

```text
docs/contracts/openapi.md
docs/contracts/api-error-codes.md
docs/contracts/api-review-checklist.md
```

但这些辅助文件不能替代 `openapi.yaml`。

---

## 5. OpenAPI 唯一事实源规则

### 5.1 基本规则

`docs/contracts/openapi.yaml` 是 AgentHub REST API 的唯一事实源。

任何 REST API 的新增、修改、删除，都必须先修改 `docs/contracts/openapi.yaml`，再修改前端和后端代码。

禁止出现：

- 后端已经实现接口，但 OpenAPI 没有定义。
- 前端已经调用字段，但 OpenAPI 没有定义。
- OpenAPI 定义了字段，但后端返回不一致。
- OpenAPI 定义了字段，但前端手写了另一套类型。
- Go Handler 临时添加 response 字段但没有更新 OpenAPI。
- 前端根据 mock 数据随意扩展 response 字段。
- API 错误格式在不同 handler 中不一致。

---

### 5.2 修改顺序

REST API 的正确修改顺序：

```text
修改 docs/contracts/openapi.yaml
  → 生成 / 更新前端 API 类型
  → 修改 Go Handler / Service / Model
  → 修改前端调用
  → 编写或更新 Contract Test
  → Review
```

禁止先写前后端实现再补 OpenAPI。

---

## 6. REST API 与其他协议的职责分离

### 6.1 REST API 负责什么

REST API 负责持久化资源的查询和管理，包括：

- conversation
- message
- agent
- custom agent
- artifact
- artifact preview
- artifact download
- 用户可见的资源元数据

REST API 是“资源查询 / 资源管理”接口，不是实时事件流接口。

---

### 6.2 REST API 不负责什么

REST API 不负责：

- Agent 文本流式输出。
- AG-UI event stream。
- `TEXT_MESSAGE_CONTENT`。
- `TOOL_CALL_START` / `TOOL_CALL_ARGS` / `TOOL_CALL_END`。
- A2A Task 创建。
- A2A Task Streaming。
- Orchestrator 内部事件。
- Child Agent 的 AgentCard 生成。
- ExecutionPlan 生成。
- 多 Agent 调度。
- Frontend Skill 执行。

这些内容必须分别交给：

```text
AG-UI                 → agui-event-contract
Gateway-Orchestrator  → gateway-orchestrator-contract
A2A                   → a2a-agent-contract
Artifact              → artifact-contract
Frontend Runtime Skills       → frontend-runtime-skills-contract
Intent Orchestration  → intent-orchestration-contract
```

---

## 7. 必须覆盖的 REST API

`docs/contracts/openapi.yaml` 第一版必须覆盖以下接口。

### 7.1 Conversation API

```text
GET    /api/conversations
POST   /api/conversations
GET    /api/conversations/{id}
PATCH  /api/conversations/{id}
DELETE /api/conversations/{id}
GET    /api/conversations/{id}/messages
```

用途：

- 查询对话列表。
- 创建对话。
- 查询对话详情。
- 更新对话标题、置顶、归档等状态。
- 删除或归档对话。
- 分页查询消息。

---

### 7.2 Agent API

```text
GET    /api/agents
GET    /api/agents/{name}/card
```

用途：

- 获取所有可用 Agent 列表。
- 获取指定 Agent 的完整 AgentCard。

注意：

- AgentCard 的完整结构由 `a2a-agent-contract` 定义。
- REST API 中可以引用 AgentCard schema，但不能在本 Skill 中重新发明字段。
- `GET /api/agents` 可以返回 AgentCard 摘要。
- `GET /api/agents/{name}/card` 返回完整 AgentCard。

---

### 7.3 Custom Agent API

```text
POST   /api/agents/custom
PATCH  /api/agents/custom/{id}
DELETE /api/agents/custom/{id}
```

用途：

- 创建自定义 Agent。
- 修改自定义 Agent。
- 删除自定义 Agent。

注意：

- 自定义 Agent 最终也必须兼容 A2A。
- 自定义 Agent 也必须能生成或暴露 AgentCard。
- 自定义 Agent 的运行时规范由 `adk-runtime-contract` 约束。
- 自定义 Agent 的能力字段不能绕过 AgentCard 规范。

---

### 7.4 Artifact API

```text
GET    /api/artifacts/{id}
GET    /api/artifacts/{id}/preview
```

用途：

- 查询 Artifact 元数据。
- 查询 Artifact 预览数据。
- 获取下载地址或预览地址。

注意：

- Artifact schema 由 `artifact-contract` 定义。
- REST API 可以返回 Artifact 数据，但不能定义与 `artifact-contract` 冲突的字段。
- 大文件必须通过 `file_url` 或预签名 URL 返回，不允许直接塞进普通 JSON 大字段。
- 前端展示组件由 `frontend-runtime-skills-contract` 决定。

---

## MVP v0.1 API 实施范围

`platform-api-contract` 同时维护两个层次：

1. **完整 API Contract**：`docs/contracts/openapi.yaml` 可以保留 PDR 规划的完整 Frontend ↔ Gateway REST API。
2. **MVP v0.1 必须实现 API**：当前实现阶段只要求完成 MVP Demo 必需的最小接口。

MVP v0.1 必须实现的 REST / HTTP 接口：

```text
GET  /api/conversations
POST /api/conversations
GET  /api/conversations/{id}/messages
GET  /api/agents
POST /api/agui/run
```

MVP v0.1 暂不强制实现，但可以保留在 OpenAPI 规划中的接口：

```text
GET    /api/conversations/{id}
PATCH  /api/conversations/{id}
DELETE /api/conversations/{id}
GET    /api/agents/{name}/card
POST   /api/agents/custom
PATCH  /api/agents/custom/{id}
DELETE /api/agents/custom/{id}
GET    /api/artifacts/{id}
GET    /api/artifacts/{id}/preview
POST   /api/agui/run/{runId}/cancel
POST   /api/agui/run/{runId}/tool-result
```

MVP v0.1 API 实施要求：

- 即使只实现最小接口，也必须遵守 `docs/contracts/openapi.yaml`。
- 即使只实现最小接口，也必须使用统一响应格式 `code / data / message`。
- 错误响应必须保持 `code / data: null / message`。
- 列表接口必须保持分页格式 `list / total / page / pageSize`，如果 MVP 暂不做真实分页，也必须保持 schema 兼容。
- 鉴权仍然使用 `Authorization: Bearer <token>`；MVP 中 token 可以是固定 Token 或环境变量 Token。
- JSON 字段仍然使用 camelCase。
- `POST /api/agui/run` 只登记 HTTP 形态，事件语义由 `agui-event-contract` 定义。
- 不得把 AG-UI 事件 schema 展开成普通 REST response。
- 不得把 A2A endpoint 暴露成 Frontend REST API。
- 不得把 Gateway ↔ Orchestrator 内部接口暴露成 Frontend REST API。
- Frontend 仍然不能手写 API response 类型。
- Backend 仍然不能返回 OpenAPI 未定义字段。

MVP v0.1 中，`openapi.yaml` 可以使用 `x-agenthub-stage` 或描述文本标记接口阶段，例如：

```yaml
x-agenthub-stage: mvp-required
```

或：

```yaml
x-agenthub-stage: post-mvp-planned
```

如果后续选择使用该扩展字段，必须保持所有接口标记一致。

---

## 8. AG-UI Endpoint 的边界说明

PDR 中还包含以下 AG-UI 相关 endpoint：

```text
POST /api/agui/run
POST /api/agui/run/{runId}/cancel
POST /api/agui/run/{runId}/tool-result
```

这些 endpoint 虽然路径上属于 Gateway 的 HTTP endpoint，但它们的核心语义属于 AG-UI 实时交互，而不是普通 REST 资源 API。

因此：

- `platform-api-contract` 可以在 OpenAPI 中登记这些 endpoint 的基础 HTTP 形态。
- 但它们的事件流、Tool Call、Tool Result 语义必须由 `agui-event-contract` 定义。
- 不允许在本 Skill 中重新定义 AG-UI event schema。
- 不允许把 AG-UI 事件当作普通 REST response 处理。
- `POST /api/agui/run` 的响应如果是 SSE stream，必须在 OpenAPI 中明确标注为流式响应或引用 AG-UI contract。

---

## 9. 统一响应格式

### 9.1 成功响应

AgentHub REST API 默认采用 PDR 中定义的统一响应格式：

```json
{
  "code": 0,
  "data": {},
  "message": "success"
}
```

字段含义：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| code | integer | 是 | 业务状态码。成功固定为 `0` |
| data | any | 是 | 业务数据。不同接口使用不同 schema |
| message | string | 是 | 成功时默认 `"success"` |

---

### 9.2 错误响应

错误响应统一使用：

```json
{
  "code": 400001,
  "data": null,
  "message": "错误描述"
}
```

字段含义：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| code | integer | 是 | 业务错误码。不能使用 `0` |
| data | null | 是 | 错误时固定为 `null` |
| message | string | 是 | 用户或开发者可读的错误说明 |

---

### 9.3 分页响应

分页响应统一使用：

```json
{
  "code": 0,
  "data": {
    "list": [],
    "total": 100,
    "page": 1,
    "pageSize": 20
  },
  "message": "success"
}
```

分页字段含义：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| list | array | 是 | 当前页数据 |
| total | integer | 是 | 总记录数 |
| page | integer | 是 | 当前页，从 `1` 开始 |
| pageSize | integer | 是 | 每页条数 |

默认规则：

```text
page 默认值：1
pageSize 默认值：20
pageSize 最大值：100
```

---

### 9.4 是否允许改统一响应格式

第一版默认使用 PDR 中的 `code / data / message` 结构。

如果团队决定改成更标准的 HTTP Problem Details、`ErrorResponse` 或其他形式，必须同时更新：

```text
PDR
docs/contracts/openapi.yaml
Frontend API Client
Go Handler
Contract Test
相关文档
```

禁止只改实现，不改 Contract。

---

## 10. HTTP 状态码与业务 code 的关系

HTTP 状态码用于表达 HTTP 层是否成功。

业务 `code` 用于表达业务层状态。

建议规则：

| 场景 | HTTP Status | body.code |
|---|---:|---:|
| 成功 | 200 / 201 / 204 | 0 |
| 参数错误 | 400 | 400001 |
| 未登录 | 401 | 401001 |
| 无权限 | 403 | 403001 |
| 资源不存在 | 404 | 404001 |
| 冲突 | 409 | 409001 |
| 限流 | 429 | 429001 |
| 内部错误 | 500 | 500001 |
| 下游服务错误 | 502 | 502001 |
| 服务超时 | 504 | 504001 |

注意：

- 不允许所有错误都返回 HTTP 200。
- 不允许错误时 `code = 0`。
- 不允许同一类错误在不同接口返回不同结构。
- 具体错误码可以后续在 `docs/contracts/api-error-codes.md` 中扩展。

---

## 11. 鉴权 Header 规范

所有需要登录的 REST API 必须使用统一鉴权 Header：

```text
Authorization: Bearer <token>
```

规则：

- Gateway 负责校验 JWT。
- Frontend 不得把 token 放在 query string。
- 后端不得接受多个不一致的鉴权字段。
- 未登录返回 HTTP 401。
- 无权限返回 HTTP 403。
- 自建 Agent、会话、Artifact 等用户资源必须做权限校验。
- Gateway-Orchestrator 内部鉴权不由本 Skill 详细定义，应由 `gateway-orchestrator-contract` 定义。

---

## 12. Trace Header 规范

Frontend 请求 Gateway 时建议携带或由 Gateway 生成：

```text
X-Request-Id
X-Trace-Id
```

规则：

- `X-Request-Id` 标识单次 HTTP 请求。
- `X-Trace-Id` 标识跨服务链路。
- Gateway 必须向 Orchestrator 透传 `traceId`。
- 日志必须能根据 `traceId` 串起 Frontend、Gateway、Orchestrator、Child Agent 链路。
- 如果 Frontend 未传，Gateway 应生成。
- 不允许在日志中输出敏感 token。

---

## 13. 资源 ID 规范

### 13.1 通用规则

所有核心资源 ID 推荐使用 UUID 字符串。

包括：

```text
user.id
conversation.id
message.id
agent.id
artifact.id
runId
threadId
taskId
toolCallId
```

### 13.2 路径参数命名

路径参数必须统一使用：

```text
{id}
{name}
{runId}
```

示例：

```text
/api/conversations/{id}
/api/agents/{name}/card
/api/artifacts/{id}
```

不要混用：

```text
conversationId
conversation_id
cid
agentName
agent_name
```

除非某接口确实需要表达多个资源 ID，此时应在 OpenAPI 中明确命名。

---

## 14. 时间字段规范

所有 API 返回的时间字段必须使用 ISO 8601 / RFC3339 字符串。

示例：

```json
{
  "createdAt": "2026-05-21T10:30:00Z",
  "updatedAt": "2026-05-21T10:31:00Z"
}
```

字段命名使用 camelCase：

```text
createdAt
updatedAt
deletedAt
lastMessageAt
lastCheckAt
```

后端数据库可以使用 snake_case，但 API response 必须保持 camelCase。

---

## 15. 字段命名规范

REST API JSON 字段统一使用 camelCase。

推荐：

```json
{
  "conversationId": "uuid",
  "senderType": "agent",
  "aguiRunId": "run-xxx",
  "a2aTaskId": "task-xxx",
  "pageSize": 20
}
```

禁止在 API JSON 中混用 snake_case：

```json
{
  "conversation_id": "uuid",
  "sender_type": "agent",
  "page_size": 20
}
```

例外：

- A2A 官方规范中已有 snake_case 字段时，由 `a2a-agent-contract` 决定。
- 数据库字段可以使用 snake_case。
- Go struct 内部字段命名不受 API JSON 影响，但 json tag 必须符合 API 规范。

---

## 16. Conversation API Contract 要求

### 16.1 ConversationSummary

`GET /api/conversations` 返回列表项至少包含：

```text
id
title
type
isPinned
isArchived
lastMessage
unreadCount
createdAt
updatedAt
```

`type` 只能是：

```text
single
group
```

### 16.2 CreateConversationRequest

`POST /api/conversations` 请求至少支持：

```text
title
type
participants
initialAgentName
```

注意：

- 单聊可以指定一个 Agent。
- 群聊可以指定多个 Agents。
- 如果没有指定 Agent，可由后续对话通过 Orchestrator 编排。
- 具体 request schema 必须写入 OpenAPI。

### 16.3 UpdateConversationRequest

`PATCH /api/conversations/{id}` 可更新：

```text
title
isPinned
isArchived
```

不允许通过该接口直接修改历史消息。

---

## 17. Message API Contract 要求

### 17.1 MessageSummary

`GET /api/conversations/{id}/messages` 返回消息列表，消息至少包含：

```text
id
conversationId
senderId
senderType
senderAgent
content
status
mentions
aguiRunId
a2aTaskId
createdAt
```

`senderType` 只能是：

```text
user
agent
system
```

`status` 只能是：

```text
sending
streaming
sent
failed
```

### 17.2 Message content

`content` 应作为结构化 JSON，而不是单纯字符串。

最小结构建议：

```json
{
  "type": "text",
  "text": "消息内容"
}
```

可扩展为：

```text
text
markdown
artifact_ref
system
```

注意：

- 大型代码、网页、文件不应直接放在 message.content 中。
- 大产物必须使用 Artifact。
- 消息中可以引用 artifactId。

---

## 18. Agent API Contract 要求

### 18.1 AgentSummary

`GET /api/agents` 返回 Agent 摘要列表，至少包含：

```text
id
name
type
avatar
description
status
skills
inputModes
outputModes
createdAt
updatedAt
```

`type` 只能是：

```text
builtin
custom
```

`status` 建议使用：

```text
healthy
unhealthy
unknown
```

### 18.2 AgentCard

`GET /api/agents/{name}/card` 返回完整 AgentCard。

AgentCard 的结构必须与 `a2a-agent-contract` 保持一致。

本 Skill 只要求：

- OpenAPI 引用 AgentCard schema。
- 不允许 REST API 返回与 A2A AgentCard 冲突的字段。
- 不允许前端根据 AgentSummary 自行推断未声明能力。

---

## 19. Custom Agent API Contract 要求

### 19.1 CreateCustomAgentRequest

`POST /api/agents/custom` 至少应包含：

```text
name
description
systemPrompt
llmModel
skills
tools
inputModes
outputModes
```

规则：

- `name` 必须唯一。
- `name` 必须可用于 AgentCard。
- `skills` 必须能映射到 AgentCard.skills。
- `outputModes` 必须是系统允许的类型。
- 自建 Agent 不能声明系统不支持的 outputModes。

### 19.2 UpdateCustomAgentRequest

`PATCH /api/agents/custom/{id}` 可更新：

```text
description
systemPrompt
llmModel
skills
tools
inputModes
outputModes
isActive
```

不建议允许修改 `name`。如果允许，必须处理 AgentCard、历史会话、路由引用的一致性问题。

### 19.3 DeleteCustomAgent

`DELETE /api/agents/custom/{id}` 第一版建议采用软删除或禁用：

```text
isActive = false
```

不建议物理删除，避免历史 conversation / message 引用断裂。

---

## 20. Artifact API Contract 要求

### 20.1 ArtifactResponse

`GET /api/artifacts/{id}` 返回 Artifact 元数据和可用内容引用，至少包含：

```text
id
type
title
content
fileUrl
metadata
version
messageId
conversationId
createdAt
updatedAt
```

注意：

- `content` 只适合小型文本类 Artifact。
- 大文件必须使用 `fileUrl`。
- `metadata` 结构由 `artifact-contract` 细化。
- `type` 必须属于合法 Artifact type。

### 20.2 ArtifactPreviewResponse

`GET /api/artifacts/{id}/preview` 返回前端预览所需数据。

规则：

- code preview 可以返回 code、language、filename。
- web preview 可以返回 html、css、js、title。
- file preview 可以返回 fileUrl、filename、mimeType、size。
- image preview 可以返回 url、alt、width、height。
- 具体字段必须与 `frontend-runtime-skills-contract` 保持一致。

注意：

- `ArtifactPreviewResponse` 不应绕过 Frontend Skill contract。
- 预览数据不应包含敏感 token。
- 私有文件下载应使用短期有效 URL。

---

## 21. Query 参数规范

### 21.1 分页参数

所有列表接口统一使用：

```text
page
pageSize
```

示例：

```text
GET /api/conversations?page=1&pageSize=20
```

默认：

```text
page=1
pageSize=20
```

限制：

```text
page >= 1
1 <= pageSize <= 100
```

---

### 21.2 排序参数

如需排序，统一使用：

```text
sortBy
sortOrder
```

`sortOrder` 只能是：

```text
asc
desc
```

示例：

```text
GET /api/conversations?sortBy=updatedAt&sortOrder=desc
```

---

### 21.3 搜索参数

如需搜索，统一使用：

```text
keyword
```

示例：

```text
GET /api/conversations?keyword=登录页面
```

---

## 22. OpenAPI 编写要求

### 22.1 必须包含的顶层信息

`openapi.yaml` 必须包含：

```yaml
openapi: 3.1.0
info:
  title: AgentHub Platform API
  version: 0.1.0
servers:
  - url: http://localhost:8080
```

### 22.2 必须包含的 components

至少包含：

```text
ApiResponse
ErrorResponse
PageResponse
PageData
ConversationSummary
ConversationDetail
MessageSummary
AgentSummary
AgentCard
CustomAgent
Artifact
ArtifactPreview
```

### 22.3 必须包含的 securitySchemes

```yaml
components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
```

需要登录的接口必须声明：

```yaml
security:
  - bearerAuth: []
```

### 22.4 operationId 规范

每个 API 必须有稳定的 `operationId`。

示例：

```text
listConversations
createConversation
getConversation
updateConversation
deleteConversation
listMessages
listAgents
getAgentCard
createCustomAgent
updateCustomAgent
deleteCustomAgent
getArtifact
previewArtifact
```

operationId 用于生成前端 client，不能随意改名。

---

## 23. 前端 API Client 规则

Frontend 不能手写 REST API response 类型。

正确方式：

```text
docs/contracts/openapi.yaml
  → 生成 TypeScript API Client / Types
  → Frontend 调用生成代码
```

禁止：

- 在前端手写 `Conversation`、`Message`、`Agent` 的 API response 类型后长期维护。
- 前端根据 mock response 自行增加字段。
- 前端调用未进入 OpenAPI 的接口。
- 前端绕过 API Client 直接散落 `fetch('/api/...')`。
- 前端忽略统一错误格式。

允许：

- 前端定义 UI ViewModel。
- 前端把 API response 转换为 UI state。
- 前端定义组件内部 props 类型。

但 API response 类型必须来自 OpenAPI。

---

## 24. 后端 Go Handler 规则

Gateway 的 Go Handler 必须遵守：

- Handler 入参必须能对应 OpenAPI request schema。
- Handler 出参必须能对应 OpenAPI response schema。
- Handler 不允许临时返回未定义字段。
- Handler 不允许对同一错误返回不同结构。
- Handler 应保持薄层，不写复杂业务编排。
- 会话、消息、Agent、Artifact 逻辑应进入 service 层。
- 所有接口必须统一通过 response helper 返回。
- 所有接口必须有 requestId / traceId 日志。

示例禁止行为：

```go
c.JSON(200, gin.H{"ok": true})
```

应使用统一结构：

```json
{
  "code": 0,
  "data": {},
  "message": "success"
}
```

---

## 25. Mock-first 规则

在真实实现前，可以先基于 OpenAPI 建 mock。

第一阶段推荐：

```text
openapi.yaml
  → mock server
  → Frontend 接 mock API
  → Gateway 实现最小 mock handler
  → Contract test
  → 替换为真实 DB
```

Mock 数据必须遵守 OpenAPI。

禁止 mock 返回 OpenAPI 中不存在的字段。

---

## 26. Contract Test 规则

每个 REST API 至少应有 contract 层检查：

- response 是否符合 OpenAPI。
- error response 是否符合 ErrorResponse。
- 分页 response 是否符合 PageResponse。
- 必填字段是否存在。
- enum 是否合法。
- 时间字段是否是 RFC3339。
- 未鉴权接口是否按规则返回 401。
- 不存在资源是否返回 404。
- pageSize 超限是否返回 400。

Contract Test 可以后续由 `testing-review-contract` 细化。

---

## 27. 安全规则

REST API 必须遵守：

- 所有用户资源必须做权限校验。
- token 只允许通过 Authorization Header 传递。
- 不允许在 query string 中传 token。
- 不允许在错误 message 中泄漏内部密钥、数据库连接、LLM API key。
- Artifact 预览必须避免 XSS。
- 文件下载链接应使用短期有效 URL。
- 自建 Agent 的 systemPrompt 不应泄漏给无权限用户。
- AgentCard 中不应暴露内部敏感配置。
- 日志不得打印 Authorization Header。
- CORS 必须由 Gateway 统一配置。

---

## 28. 与 PDR 的一致性要求

本 Skill 必须遵守 PDR 中的以下决策：

- Gateway 是对外 API 入口。
- Frontend 只连接 Gateway。
- REST API 用于会话、消息、Agent、Artifact 等持久化资源。
- AG-UI 用于前端实时交互。
- A2A 用于 Orchestrator 与 Child Agents。
- 默认统一响应格式为 `code / data / message`。
- 分页响应使用 `list / total / page / pageSize`。
- API Key / JWT 等敏感信息必须安全处理。
- Docker Compose 第一版以 Gateway 暴露 API 服务。

如需改变这些决策，必须同步更新：

```text
PDR
docs/contracts/openapi.yaml
相关 Skill
前端 API client
Go Handler
测试
```

---

## 29. 与其他 Skills 的协作

### 29.1 与 project-architecture

`project-architecture` 规定服务边界。  
本 Skill 在该边界内细化 REST API。

如果发现 API 设计导致 Frontend 直连 Orchestrator 或 Child Agent，必须拒绝。

---

### 29.2 与 agui-event-contract

`agui-event-contract` 规定实时事件。

本 Skill 不定义：

```text
RUN_STARTED
TEXT_MESSAGE_CONTENT
TOOL_CALL_START
STATE_UPDATE
```

只可在 OpenAPI 中引用 AG-UI endpoint 的 HTTP 形态。

---

### 29.3 与 gateway-orchestrator-contract

Gateway 内部调用 Orchestrator 的接口不属于普通 Frontend REST API。  
不要把 `/internal/runs` 当作前端 API 暴露。

---

### 29.4 与 a2a-agent-contract

A2A endpoints 不属于 Gateway REST API。  
不要把以下接口放到 Frontend Platform API 中：

```text
/.well-known/agent.json
/a2a/tasks/send
/a2a/tasks/sendSubscribe
/a2a/tasks/{id}
/a2a/tasks/{id}/cancel
```

这些由 Child Agent 暴露，由 Orchestrator 调用。

---

### 29.5 与 artifact-contract

本 Skill 只定义 Artifact API 如何查询。  
Artifact 的内部字段、type、metadata 由 `artifact-contract` 详细定义。

---

### 29.6 与 frontend-runtime-skills-contract

Artifact preview 返回的数据必须能映射到 Frontend Runtime Skills。  
具体 Skill 参数 schema 由 `frontend-runtime-skills-contract` 定义。

---

## 30. 硬性规则

Coding Agent 在处理 REST API 时必须遵守：

1. `docs/contracts/openapi.yaml` 是 REST API 唯一事实源。
2. 所有 Frontend ↔ Gateway REST API 必须进入 OpenAPI。
3. Frontend 不能手写 API response 类型。
4. Backend 不能随意修改 response schema。
5. API 默认使用 PDR 中的 `code / data / message` 统一响应格式。
6. 分页默认使用 `list / total / page / pageSize`。
7. 错误响应必须统一为 `code / data: null / message`。
8. 鉴权 Header 必须使用 `Authorization: Bearer <token>`。
9. JSON 字段必须使用 camelCase。
10. 时间字段必须使用 RFC3339 / ISO 8601。
11. 资源 ID 推荐使用 UUID 字符串。
12. 列表接口必须统一使用 `page / pageSize`。
13. REST API 不负责 AG-UI 事件流。
14. REST API 不负责 A2A Task。
15. REST API 不负责 Orchestrator 内部 Run 协议。
16. Gateway Handler 必须遵守 OpenAPI。
17. Mock 数据必须遵守 OpenAPI。
18. Contract Test 必须校验 OpenAPI。
19. 任何 API 字段变更必须先改 Contract。
20. 如果统一响应格式发生改变，必须同步更新 PDR、OpenAPI、前端 API Client、Go Handler 和测试。
21. 必须遵守 `Contract first / Mock first / Real integration later / Review always`。
22. MVP v0.1 只强制实现最小 API 集，但完整 API Contract 可以保留在 `openapi.yaml` 中作为后续规划。
23. MVP v0.1 未实现的接口必须标记为 planned / post-MVP，避免 Codex 误判为当前必须完成。
24. MVP v0.1 的固定 Token 鉴权不能改变 API Contract 中 `Authorization: Bearer <token>` 的形式。

---

## 31. 必须维护的文件

使用本 Skill 时，至少需要维护：

```text
docs/contracts/openapi.yaml
```

根据需要维护：

```text
docs/contracts/openapi.md
docs/contracts/api-error-codes.md
docs/contracts/api-review-checklist.md
frontend/src/services/api.ts
frontend/src/types/generated/
gateway/internal/handler/
gateway/internal/service/
gateway/internal/model/
gateway/internal/middleware/auth.go
```

如果 API 变更影响架构边界，还需要同步检查：

```text
docs/architecture/service-boundaries.md
skills/project-architecture/SKILL.md
```

---

## 32. 输出要求

当用户要求生成或修改 REST API 时，Coding Agent 必须输出：

1. 该 API 属于哪个资源。
2. 是否属于 Frontend ↔ Gateway REST API。
3. 是否需要写入 `docs/contracts/openapi.yaml`。
4. request schema。
5. response schema。
6. error response。
7. 分页规则。
8. 鉴权要求。
9. 是否影响前端 API Client。
10. 是否需要 Contract Test。
11. 是否与 AG-UI / A2A / Gateway-Orchestrator Contract 有边界冲突。

除非用户明确要求，否则不要直接生成业务实现代码。

---

## 33. Review Checklist

在接受任何 REST API 设计或实现前，必须检查：

### OpenAPI 检查

- 是否写入 `docs/contracts/openapi.yaml`？
- 是否有稳定 operationId？
- 是否有 request schema？
- 是否有 response schema？
- 是否有 error response？
- 是否声明 security？
- 是否声明 path parameters？
- 是否声明 query parameters？
- 是否复用 components schema？

### 响应格式检查

- 成功响应是否为 `code / data / message`？
- 错误响应是否为 `code / data: null / message`？
- 分页响应是否为 `list / total / page / pageSize`？
- HTTP status 和业务 code 是否合理？
- 是否避免所有错误都返回 HTTP 200？

### Frontend 检查

- 前端是否从 OpenAPI 生成类型？
- 是否没有手写 API response 类型？
- 是否没有调用未定义接口？
- 是否没有绕过 Gateway？
- 是否没有把 REST API 当作 AG-UI event stream？

### Gateway 检查

- Handler 返回是否符合 OpenAPI？
- Handler 是否没有意图编排逻辑？
- Handler 是否没有直接调 Child Agent？
- Handler 是否统一错误格式？
- Handler 是否做了鉴权和权限校验？
- Handler 是否有 traceId / requestId 日志？

### 协议边界检查

- AG-UI 事件是否没有写进 REST response schema？
- A2A endpoints 是否没有暴露为 Frontend API？
- `/internal/runs` 是否没有暴露给 Frontend？
- Artifact schema 是否与 `artifact-contract` 一致？
- AgentCard 是否与 `a2a-agent-contract` 一致？

### 安全检查

- token 是否只通过 Authorization Header？
- 是否没有在 query string 传 token？
- 是否没有在错误信息中泄漏敏感信息？
- Artifact preview 是否考虑 XSS？
- 文件 URL 是否有权限控制？
- 自建 Agent 配置是否有权限隔离？


## 34. v1.1 对齐补充

### 34.1 与 data-persistence-contract 的联动

当 REST API 字段涉及持久化模型变更时，必须同步 `data-persistence-contract` 以及 `docs/contracts/data-model.md` 等数据模型文档。

### 34.2 与 security-boundary-contract 的联动

REST API 的鉴权、资源权限、错误脱敏、token 处理和敏感信息边界由 `security-boundary-contract` 细化。



## References

- `references/rest-api-policy.md`
- `references/auth-header-policy.md`
- `references/error-response-policy.md`
- `references/frontend-api-client-policy.md`
- `references/api-review-checklist.md`
