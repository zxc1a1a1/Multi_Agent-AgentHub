---
name: platform-api-contract
description: "用于定义 AgentHub Frontend 与 Gateway Service 之间的平台资源 HTTP API 契约，包括 OpenAPI 唯一事实源、公开 API 与内部 API 隔离、统一响应格式、错误码、鉴权、分页、Conversation/Message/Agent/Artifact/Run API、前端类型生成和 Gateway Handler 一致性规则。本 Skill 不绑定具体 Agent。"
---

# platform-api-contract

## 1. Skill 目的

本 Skill 定义 AgentHub 的 **Frontend ↔ Gateway Service Platform API** 契约。

Platform API 是前端访问平台资源的公开 HTTP API，包括会话、消息、Agent 摘要、运行记录、Artifact 元数据、用户可见状态和错误信息。

本 Skill 的目标是：

- 让 `docs/contracts/openapi.yaml` 成为 Frontend ↔ Gateway API 的唯一事实源。
- 阻止前端直接访问 Orchestrator Service、Child Agent Service 或任何内部服务 API。
- 保证 Gateway Handler、前端 API Client、Mock、测试和文档都以 OpenAPI 为准。
- 支持 v1.0 及后续扩展：2+ Agent、群聊、Agent Registry 摘要、健康状态展示、多消息、多 Artifact、Run 查询与取消。
- 不固定任何具体 Agent 名称。

## 2. 独立性原则

本 Skill 必须独立可读。阅读者不需要先理解其他 Skill，才能知道 Platform API 应该如何设计。

允许在边界章节说明本 Skill 不负责的内容，但不得复制其他 Skill 的完整规则。

## 3. 当前阶段识别

### v1 Generic Platform API Profile

当前 Platform API 必须面向：

- Gateway Service 与 Orchestrator Service 分进程。
- Frontend 只访问 Gateway Service。
- Orchestrator Service 的内部 API 不暴露给 Frontend。
- 2+ Agent 候选。
- 单聊与群聊。
- 通用 Agent 摘要与能力摘要。
- Run 创建、查询、取消。
- 多条消息与多 Agent 消息归属。
- Artifact 元数据、预览入口、内容引用。
- 统一响应 envelope。
- 统一错误码与安全错误信息。
- OpenAPI 3.1 事实源。

### MVP v0.1 Historical Profile

MVP v0.1 已完成，仅作为历史兼容和回归测试基线。

历史基线包括最小会话、消息、Agent 列表、Run 入口、固定 token 或 env token、`code / data / message` 响应 envelope。

这些历史规则不得继续作为当前开发禁令。任何“post-MVP 才能做”的旧描述必须改成 v1 可扩展能力或 planned lifecycle。

## 4. 本 Skill 负责什么

本 Skill 负责 REST / HTTP API 边界、OpenAPI 唯一事实源、公开 API 与内部 API 隔离、资源 URL、request / response / error schema、鉴权 header、trace header、分页、排序、搜索、Conversation / Message / Agent / Artifact / Run 平台资源 API、Frontend API Client 生成规则、Gateway Handler 一致性、Mock-first 与 contract test。

## 5. 本 Skill 不负责什么

本 Skill 不负责 Gateway ↔ Orchestrator 的内部 API、Orchestrator 编排计划、Child Agent 协议、LLM Provider 调用、实时事件流完整 schema、Artifact 内部完整内容 schema、前端 Runtime Capability 组件绑定、数据库 DDL、Docker Compose、Go / TypeScript 业务实现代码。

## 6. Public Platform API 边界

Platform API 只暴露在 Gateway Service 上。

Frontend 只能访问：

```text
/api/**
```

Frontend 不得直接访问：

```text
/internal/**
/orchestrator/**
/a2a/**
/.well-known/agent.json
/a2a/tasks/**
```

Orchestrator Service 的内部 endpoint 不得写入 `docs/contracts/openapi.yaml`。

## 7. Gateway 与 Orchestrator 分进程硬规则

Gateway Service 与 Orchestrator Service 必须分进程。对 Platform API 来说，Gateway 是唯一公开 HTTP API 入口，Orchestrator 是 Gateway 的内部下游服务。

Frontend 不知道 Orchestrator 的真实地址，不持有 service-to-service token，`ORCHESTRATOR_URL`、内部鉴权 token、内部 stream endpoint 不得出现在公开 API response 中。

## 8. OpenAPI 唯一事实源

`docs/contracts/openapi.yaml` 是 Frontend ↔ Gateway Platform API 的唯一事实源。

规则：

- 新增、修改、删除公开 API，必须先更新 OpenAPI。
- OpenAPI 必须使用 `openapi: 3.1.0`。
- 每个公开 operation 必须有稳定 `operationId`。
- request / response / error schema 必须在 `components.schemas` 中定义或复用。
- 鉴权必须在 `components.securitySchemes` 中声明。
- 前端类型必须由 OpenAPI 生成，不得手写 response 类型。
- Gateway Handler 返回字段不得超出 OpenAPI。
- Mock 数据不得超出 OpenAPI。
- 内部服务对象不得直接作为公开 response。

## 9. API Lifecycle 标记

公开 API 可以使用：

```yaml
x-agenthub-lifecycle: implemented | planned | deprecated
```

`implemented` 必须有实现、前端类型和 contract test；`planned` 不得被前端默认调用；`deprecated` 只用于兼容。公开 OpenAPI 中不得出现可被前端调用的 `internal-only` endpoint。

## 10. REST Resource Policy

推荐资源化路径：

```text
/api/conversations
/api/conversations/{conversationId}
/api/conversations/{conversationId}/messages
/api/agents
/api/agents/{agentName}
/api/runs
/api/runs/{runId}
/api/artifacts/{artifactId}
```

路径使用复数资源名，不在路径中写死具体 Agent 名称，不用路径表达内部协议。

## 11. Response Envelope

成功响应：

```json
{"code": 0, "data": {}, "message": "success"}
```

错误响应：

```json
{"code": 400001, "data": null, "message": "错误描述"}
```

成功时 `code` 必须为 `0`；错误时 `code` 不得为 `0`；错误信息必须安全脱敏；不允许所有错误都返回 HTTP 200。

## 12. Error Response Policy

Platform API 错误必须同时使用合理 HTTP status 与稳定业务错误码。

错误域包括 `AUTH_*`、`VALIDATION_*`、`CONVERSATION_*`、`MESSAGE_*`、`AGENT_*`、`RUN_*`、`ARTIFACT_*`、`GATEWAY_*`、`INTERNAL_*`。

Orchestrator Service 不可用时，Gateway 应向 Frontend 返回 502 或 504。用户取消 Run 不应伪装成 500。

## 13. Auth Header Policy

公开 API 默认使用：

```text
Authorization: Bearer <user-token>
```

token 不得放在 query string；所有用户资源必须做对象级授权；用户 token 不得被当作 Gateway ↔ Orchestrator 的 service token。

## 14. Trace Header Policy

Frontend 可以传入：

```text
X-Request-Id
X-Trace-Id
```

Gateway 必须接受或生成 `requestId` / `traceId`，在 response header 中返回 `X-Request-Id`，并在日志中记录这些字段。trace header 不能作为鉴权凭证。

## 15. Resource ID / Time / Naming Policy

JSON 字段使用 `camelCase`；时间字段使用 ISO 8601 字符串；ID 字段使用稳定字符串；response 中不得出现 Go / SQL / ORM 内部字段名。

## 16. Pagination / Sorting / Search

列表接口必须有分页策略。默认参数为 `page` / `pageSize`，可扩展 `cursor` / `limit`。搜索参数统一为 `keyword`，排序统一为 `sortBy` / `sortOrder`。

## 17. Conversation API Policy

Conversation API 必须支持 `single | group`。`participants` 表示用户、Agent 或系统参与者。兼容期允许 `initialAgentName` 作为 `participants[0]` 的 legacy alias，但新增逻辑应优先使用 `participants`。

## 18. Message API Policy

Message API 必须支持多 Agent 消息归属。推荐字段包括 `id`、`conversationId`、`runId`、`senderType`、`senderId`、`senderName`、`content`、`status`、`artifactRefs`、`toolCallRefs`。

大型 Artifact 不放入 `message.content`，同一个 run 可以产生多条 assistant / agent message。

## 19. Agent API Policy

Agent API 只返回前端展示和选择所需的摘要信息。推荐字段包括 `id`、`name`、`displayName`、`description`、`status`、`health`、`capabilities`、`inputModes`、`outputModes`、`tags`、`version`、`updatedAt`。

`status` 与 `health` 分离：
- `status`：生命周期/启用状态（`enabled` / `disabled` / `experimental` / `deprecated`）。
- `health`：当前健康状态（`healthy` / `degraded` / `unhealthy` / `unknown`）。
- `disabled` 属于 `status`，不属于 `health`。

`CapabilitySummary.id` 是 `AgentCard.skills[].id` 的公开摘要投影，非独立 ID。`capabilityIds` 的事实源是 `AgentCard.skills[].id`。

前端不得通过 Agent 名称推断能力，能力必须来自摘要字段。不得将 `toolName`、`artifact.type` 或 `outputMode` 作为 capabilityId。

## 20. Artifact API Policy

Artifact API 只返回平台资源形态，是对 Core Artifact 的公开投影。

### 字段映射

Public API DTO 字段与 Core Artifact 字段的关系：

| Public API DTO | Core Artifact（事实源） | 说明 |
|---|---|---|
| `id` | `artifactId` | Core `artifactId` = DB `artifacts.id` = Public API `id`，是同一个系统 ID 在不同层级的命名。API `id` 是 `artifactId` 的公开投影，非独立 ID。 |
| `type` | `type` | 直接透传。 |
| `title` | `title` | 直接透传。 |
| `mimeType` | `mimeType` | 直接透传。 |
| `summary` | `summary` | 直接透传。 |
| `contentRef`（string） | `contentRef`（object） | API 中是公开授权访问入口 URL，非 Core contentRef 对象。 |
| `contentUrl` | — | **推荐**使用此字段代替 API contentRef 字符串，避免混名。 |
| `previewType` | `preview.previewType` | API 扁平投影。 |
| `status` | `status` | 直接对齐 Core 枚举：`pending`/`normalizing`/`ready`/`failed`/`superseded`/`deleted`。 |
| `version` | `version` | integer，与 Core 一致。 |
| `messageId` | `links.messageId` | API 扁平投影。 |
| `conversationId` | `links.conversationId` | API 扁平投影。 |
| `runId` | `links.runId` | API 扁平投影。 |

### 规则

- Gateway Handler 输出 Public API DTO 时负责将 Core Artifact 投影为扁平 API 结构。
- API 可以返回 `previewType`，但不定义前端组件。
- 大内容必须通过 `contentRef`（string URL）或 `contentUrl` 提供下载入口，且 URL 必须短期有效。
- 推荐 Public API 使用 `contentUrl` 字段提供公开访问地址，避免与 Core `contentRef` 对象混名。
- `previewType` 是预览意图，不等于前端组件名。

## 21. Run API Policy

v1 推荐资源化路径：

```text
POST /api/runs
GET  /api/runs/{runId}
POST /api/runs/{runId}/cancel
```

兼容路径：

```text
POST /api/agui/run
POST /api/agui/run/{runId}/cancel
```

新增 API 优先使用 `/api/runs`。Run response 只保留平台资源摘要，不展开 Orchestrator 内部计划。

### 21.1 Run.status 枚举

Run.status 是粗粒度生命周期状态，只使用 5 个值：

```text
accepted
running
completed
failed
cancelled
```

不得将内部细粒度阶段（如 `planning`、`dispatching`、`retrying`、`fallback`）写入 Run.status。

### 21.2 Run.phase 与 STATE_UPDATE.state.phase

- `Run.status`：粗粒度生命周期状态，持久化到 `runs.status`，通过 Public API 返回。
- `Run.phase`：可选细粒度当前阶段（内部字段），可通过 API 返回用于展示细节，但不等同于 status。
- `STATE_UPDATE.state.phase`：前端 SSE 事件中的阶段提示，用于 UI 展示当前进度，不持久化。
- `run_steps.step_type`：持久化详细步骤类型，用于审计和排障。

需要展示运行细节时使用 `Run.phase` 或 `STATE_UPDATE.state.phase`，不应扩展 Run.status 枚举。

## 22. Frontend API Client 规则

Frontend API 类型必须由 OpenAPI 生成；不得手写 response interface；不得直接 `fetch` OpenAPI 未定义路径；不得访问 `/internal/**` 或 Orchestrator 真实地址。

## 23. Gateway Handler 规则

Gateway Handler 是 Platform API 的实现层，不是编排层。它可以处理 HTTP、鉴权、权限、校验、统一 response 和日志，但不得直接选择 Agent、调用 Child Agent、调用 LLM Provider、生成 OrchestrationPlan 或返回 OpenAPI 未定义字段。

## 24. Mock-first 规则

新 API 可先写 OpenAPI + Mock，再接真实 Handler。Mock response 必须与 schema 完全一致，不得包含内部 URL、token、密钥或真实用户数据。

## 25. Contract Test 规则

每个 `implemented` API 至少需要验证 HTTP method/path、request schema、success response、error response、鉴权失败、对象级权限失败、Gateway Handler 不返回未定义字段。

## 26. 安全规则

所有生产公开 API 必须走 HTTPS 终止后的可信链路。用户 token 不得出现在 URL、日志、错误信息或 Artifact。内部 service token 不得暴露给 Frontend。文件下载和 preview URL 必须有时效和权限控制。

## 27. Review Checklist

提交 Platform API 变更前必须检查：是否只定义 Frontend ↔ Gateway API；是否没有暴露 `/internal/**`；是否没有暴露 Orchestrator 或 Child Agent endpoint；是否没有写死具体 Agent 名称；OpenAPI 是否更新；operationId 是否稳定；schema 是否完整；前端类型是否由 OpenAPI 生成；Gateway Handler 是否没有编排逻辑；错误是否脱敏；Run.status 是否只使用 5 值枚举；细粒度阶段是否未写入 Run.status；Artifact `id` 是否是 `artifactId` 的公开投影（非独立 ID）；`CapabilitySummary.id` 是否是 `AgentCard.skills[].id` 的公开摘要投影（非独立 ID）；是否没有将 `toolName`、`artifact.type`、`outputMode` 当作 capabilityId；`AgentSummary.status` 与 `AgentSummary.health` 是否分离；`disabled` 是否没有出现在 health 字段中；是否没有暴露内部 healthcheck 细节。

## 28. 完成定义

一次 Platform API 变更只有同时满足以下条件才算完成：OpenAPI 已更新，相关 contract 文档已更新，前端类型生成方式明确，Gateway Handler 行为与 OpenAPI 一致，success/error/auth/permission 场景有 contract test，无内部服务 API 泄漏，无具体 Agent 名称硬编码，无敏感信息泄漏。
