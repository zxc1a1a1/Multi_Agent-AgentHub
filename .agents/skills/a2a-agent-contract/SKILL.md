---
name: a2a-agent-contract
description: "用于定义 AgentHub 中任意 Child Agent 接入 Orchestrator 的通用 A2A 协议契约，包括 AgentCard、Agent Registry、健康检查、A2A Streaming Task、Artifact 输出、错误处理、安全边界与契约测试。适用于 MVP 完成后的 v1.0 及后续 2+ Agent 扩展，不固定约束 code-agent 或 web-agent。"
---

# a2a-agent-contract

## 1. Skill 目的

本 Skill 定义 AgentHub 中任意 Child Agent 接入平台时必须遵守的通用 A2A 契约。

它约束的是：

```text
Orchestrator ↔ Child Agent
```

而不是某一个具体 Agent 的实现。

本 Skill 解决以下问题：

- 一个新的子 Agent 如何声明自己的能力？
- Orchestrator 如何发现和调用它？
- Planner 如何理解它能做什么？
- Registry 如何记录它是否健康？
- 子 Agent 如何流式输出文本、产物和错误？
- 子 Agent 输出的 Artifact 如何进入 AG-UI / Frontend Runtime Skill 链路？
- 多 Agent 场景下，如何避免硬编码 agentName？
- 如何保证 Frontend / Gateway Handler 不绕过 Orchestrator 直接调用子 Agent？

一句话：

**任何 Child Agent 只要满足 AgentCard + Health Check + A2A Streaming Task + Artifact Output + Registry Discovery 规则，就可以接入 AgentHub；本契约不固定具体 Agent 名称。**

---

## 2. 当前开发阶段

当前项目阶段：

```text
profile = v1.0-sprint
a2aContractStyle = generic-child-agent-contract
mvpStatus = completed
```

MVP v0.1 已完成，历史基线仅用于回归测试。

v1.0 Sprint 的目标是从单 Agent 升级为 2+ Child Agents，并为后续更多子 Agent 扩展建立稳定契约。

因此，本 Skill 不应把以下能力视为 Post-MVP 禁止项：

```text
2+ Child Agents
AgentCard Registry
Agent 健康检查
LLM Planner 选择 Agent
single / ordered_parallel 编排
Agent fallback / retry
web_preview
markdown_render
更多 outputModes / Artifact 类型
```

但本 Skill 也不应把 `code-agent` / `web-agent` 写死为唯一允许的 Agent。

---

## 3. 适用场景

当任务涉及以下内容时，必须使用本 Skill：

- 新增任意 Child Agent。
- 修改任意 Child Agent 的 AgentCard。
- 修改任意 Child Agent 的 `/health`。
- 修改任意 Child Agent 的 A2A endpoint。
- 修改 `agents/adk/` 中与 A2A 暴露有关的逻辑。
- 修改 `server/internal/a2a/`。
- 修改 `server/internal/registry/`。
- 修改 Orchestrator 调用 Child Agent 的逻辑。
- 修改 A2A streaming event。
- 修改 A2A Artifact 输出。
- 修改 Agent Registry 发现、缓存、健康检查逻辑。
- 修改 Planner 使用 AgentCard 的方式。
- Review 新增 Agent 是否能接入 AgentHub。
- Review 代码是否硬编码了 `code-agent` / `web-agent` 判断能力。
- Review Frontend / Gateway Handler 是否绕过 Orchestrator 直接调用 Child Agent。

---

## 4. 核心原则

### 4.1 不固定 Agent 名称

本 Skill 不允许把某个 Agent 名称写成平台能力边界。

错误做法：

```text
如果 agentName == code-agent，则说明它会输出 code。
如果 agentName == web-agent，则说明它会输出 webpage。
```

正确做法：

```text
通过 AgentCard.skills、inputModes、outputModes、capabilities 判断 Agent 能力。
```

Agent 名称只用于：

- 唯一标识 Agent。
- Registry 查找 Agent。
- Orchestrator 调用目标 Agent。
- 日志、trace、UI 展示。
- @agent-name 手动路由。

Agent 名称不得用于推断能力。

### 4.2 能力由 AgentCard 声明

每个 Child Agent 必须通过 AgentCard 声明：

- name
- description
- url
- version
- capabilities
- skills
- inputModes
- outputModes

Planner、Registry、Orchestrator 必须基于 AgentCard 理解 Agent。

### 4.3 Orchestrator 是唯一 A2A 调用入口

Frontend 不得直接调用 Child Agent。

Gateway Handler 不得直接调用 Child Agent。

Child Agent 只能由以下模块访问：

- Agent Registry：读取 AgentCard、健康检查。
- Orchestrator / A2A Client：提交 A2A Task、读取流式事件。

### 4.4 Child Agent 不直接输出 AG-UI

Child Agent 输出的是 A2A event，不是 AG-UI event。

Child Agent 不得直接构造：

- `TEXT_MESSAGE_START`
- `TEXT_MESSAGE_CONTENT`
- `TEXT_MESSAGE_END`
- `TOOL_CALL_START`
- `TOOL_CALL_ARGS`
- `TOOL_CALL_END`
- `RUN_ERROR`
- `STATE_UPDATE`

这些事件只能由 Gateway / Orchestrator / ProtocolConverter 生成。

### 4.5 Artifact 映射由契约链决定

Child Agent 可以输出 Artifact，但它不能决定前端如何渲染。

Artifact 能否渲染，取决于：

1. AgentCard.outputModes 是否声明。
2. `artifact-contract` 是否支持该 Artifact type。
3. `frontend-runtime-skills-contract` 是否支持对应 Frontend Skill。
4. ProtocolConverter 是否实现映射。
5. Security Contract 是否允许该渲染方式。

---

## 5. 与其他 Skills 的边界

| 内容 | 由谁负责 | 本 Skill 是否负责 |
|---|---|---:|
| Frontend ↔ Gateway REST API | `platform-api-contract` | 否 |
| Frontend ↔ Gateway AG-UI Event Stream | `agui-event-contract` | 否 |
| Gateway ↔ Orchestrator 内部调用 | `gateway-orchestrator-contract` | 否 |
| Planner / ExecutionPlan / TaskPlan | `intent-orchestration-contract` | 部分引用 |
| Orchestrator ↔ Child Agent A2A | 本 Skill | 是 |
| Child Agent runtime 细节 | `adk-runtime-contract` | 部分引用 |
| Artifact Schema | `artifact-contract` | 部分引用 |
| Artifact → Frontend Skill 参数 | `frontend-runtime-skills-contract` | 部分引用 |
| Agent 运行安全边界 | `security-boundary-contract` | 部分引用 |
| 测试和 Review | `testing-review-contract` | 部分引用 |

---

## 6. 必须维护的文件

本 Skill 的主文件：

```text
.claude/skills/a2a-agent-contract/SKILL.md
```

本 Skill 的配套 reference 文件：

```text
.claude/skills/a2a-agent-contract/references/agent-card-policy.md
.claude/skills/a2a-agent-contract/references/task-endpoints.md
.claude/skills/a2a-agent-contract/references/send-subscribe-streaming.md
.claude/skills/a2a-agent-contract/references/artifact-policy.md
.claude/skills/a2a-agent-contract/references/a2a-error-policy.md
.claude/skills/a2a-agent-contract/references/registry-discovery.md
.claude/skills/a2a-agent-contract/references/child-agent-lifecycle.md
.claude/skills/a2a-agent-contract/references/a2a-review-checklist.md
```

需要同步维护的项目契约文件：

```text
docs/contracts/a2a-agent-card.md
docs/contracts/a2a-task.md
docs/contracts/a2a-errors.md
docs/contracts/a2a-review-checklist.md
```

可能涉及的实现目录：

```text
agents/adk/
agents/*-agent/
server/internal/a2a/
server/internal/registry/
server/internal/orchestrator/
server/internal/model/
```

---

## 7. Version Profiles

### 7.1 MVP v0.1 Historical Profile

MVP v0.1 已完成，只作为回归测试基线。

历史基线：

```text
Child Agent 数量：1
Agent discovery：静态配置
Routing：硬编码 direct routing
Strategy：single
Artifact：code 为主
Frontend Skill：code_preview 为主
```

MVP 历史基线不得继续阻止 v1.0 扩展。

### 7.2 v1.0 Sprint Profile

v1.0 Sprint 当前要求支持：

```text
Child Agent 数量：2+
Agent discovery：Agent Registry
Agent 能力声明：AgentCard
Agent 健康检查：/health
任务提交：/a2a/tasks/sendSubscribe
Planner：基于 AgentCard.skills / outputModes 选择 Agent
Execution Strategy：single / ordered_parallel
失败处理：fallback / retry hint
输出：text streaming + artifact streaming
```

v1.0 Sprint 不固定：

```text
具体 Agent 名称
具体 Agent 数量上限
具体 Agent 技能名称集合
具体 Artifact 类型全集
```

### 7.3 Post-v1.0 Profile

Post-v1.0 可扩展：

```text
Agent 动态注册
Agent marketplace
用户自定义 Agent
Agent 权限模型
A2A 鉴权
A2A task cancel
A2A task history
多模态输入
大文件 Artifact 存储
跨 Agent 结果聚合总结
真正并发事件交错处理
```

---

## 8. AgentCard Contract

AgentCard 是 Child Agent 的能力声明。

AgentCard 用于：

- Registry 发现 Agent。
- Registry 缓存 Agent 能力。
- Registry 判断 Agent 是否健康。
- Planner 理解 Agent 能力。
- Orchestrator 校验目标 Agent。
- 前端通过 Gateway 展示 Agent 摘要。

AgentCard 不是：

- system prompt。
- 私有配置文件。
- API key 容器。
- 前端组件 schema。
- A2A Task 本身。
- OpenAPI REST response 的替代品。

### 8.1 最小 AgentCard

```json
{
  "name": "agent-name",
  "description": "说明该 Agent 能做什么",
  "url": "http://agent-service:8081",
  "version": "0.1.0",
  "capabilities": {
    "streaming": true,
    "artifacts": true,
    "tools": false,
    "cancellable": false
  },
  "skills": [
    {
      "id": "skill_id",
      "name": "技能名称",
      "description": "该技能能完成什么任务",
      "inputTypes": ["text"],
      "outputTypes": ["text", "code"]
    }
  ],
  "inputModes": ["text"],
  "outputModes": ["text", "code"]
}
```

### 8.2 字段规则

| 字段 | 必填 | 规则 |
|---|---:|---|
| `name` | 是 | 全局稳定，不能随版本随意变化 |
| `description` | 是 | 给 Planner 和 UI 使用，必须简洁准确 |
| `url` | 是 | A2A Server 地址，不得包含 token |
| `version` | 是 | 用于兼容性和排查 |
| `capabilities` | 是 | 声明是否 streaming、artifacts、tools、cancellable |
| `skills` | 是 | 能力列表，Planner 主要依据 |
| `inputModes` | 是 | 支持的输入模式 |
| `outputModes` | 是 | 支持的输出模式 |

规则：

- `skills[].id` 必须稳定。
- `skills[].id` 是 Agent 对外声明的**能力 ID 事实源**。`TaskPlan.capabilityIds` 必须引用此 ID，`CapabilitySummary.id` 是它的公开摘要投影。
- `skills[].outputTypes` 必须与 `outputModes` 兼容。
- `outputModes` 必须能映射到 Artifact / Frontend Runtime Skill。
- AgentCard 不得暴露 API key、token、完整 system prompt、内部密钥。
- 新增 Child Agent 前必须先定义 AgentCard。

---

## 9. inputModes / outputModes

### 9.1 outputModes 的定位

`outputModes` 是 Agent 声明可产出的**语义类别**，不是平台归一化后的 `artifact.type`，也不是前端 `toolName`。

```text
AgentCard.outputModes  = Agent 声明可产出的语义类别（Agent 视角）
Artifact.type          = 平台归一化后的产物类型（平台视角）
Runtime toolName       = 前端执行/渲染能力名（前端视角）
```

三者必须通过映射表转换，不得直接等同：

- `outputModes` ≠ `artifact.type`（除非映射表显式声明兼容）
- `outputModes` ≠ Runtime `toolName`
- `artifact.type` ≠ Runtime `toolName`

### 9.2 v1.0 最小支持

```text
inputModes:
- text

outputModes:
- text
- code
- webpage
- document
```

注意：

- 不是所有 Agent 都必须支持所有 outputModes。
- 每个 Agent 只能声明自己真实支持的 outputModes。
- Planner / Orchestrator 不能假设某个 Agent 一定支持某种输出。

### 9.3 三层映射链

`outputMode` 经过平台归一化后确定 `artifact.type`，再通过 `previewType` 映射到前端 `toolName`：

```text
outputMode → artifact.type → previewType → toolName
```

推荐映射：

| outputMode | artifact.type | previewType | toolName | 说明 |
|---|---|---|---|---|
| `text` | 无 Artifact | — | `markdown_render` / StreamingText | 纯文本流，不产生 Artifact |
| `code` | `code` | `code_preview` | `code_preview` | 代码类产物 |
| `webpage` | `webpage` | `web_preview` | `web_preview` | 网页类产物 |
| `document` | `document` 或 `markdown` | `document_preview` 或 `markdown_render` | `document_preview` 或 `markdown_render` | 文档类产物，归一化时根据 metadata.format 确定 artifact.type |

规则：

- `outputMode` 是粗粒度语义类别，同一个 `outputMode` 可能对应多个 `artifact.type`（如 `outputMode=document` → `artifact.type=document` 或 `markdown`）。
- `artifact.type` 由 ArtifactRegistry 在归一化时根据 ArtifactDraft、mimeType、metadata、outputMode 等信息确定。
- `toolName` 由 `preview.previewType` 决定，不由 `outputMode` 直接决定。
- 不得通过 `outputMode` 直接选择前端组件。
- 不得通过 `agentName` 推断 outputMode 或选择 toolName。

新增 outputMode 时，必须同步：

- `artifact-contract`
- `frontend-runtime-skills-contract`
- ProtocolConverter
- Contract tests

---

## 10. Required Endpoints

每个 Child Agent 至少必须暴露：

```text
GET /.well-known/agent.json
GET /health
POST /a2a/tasks/sendSubscribe
```

v1.0 不强制：

```text
POST /a2a/tasks/send
GET /a2a/tasks/{id}
POST /a2a/tasks/{id}/cancel
GET /a2a/tasks/{id}/history
```

规则：

- `/.well-known/agent.json` 返回 AgentCard。
- `/health` 返回健康状态。
- `/a2a/tasks/sendSubscribe` 用于提交任务并读取流式结果。
- A2A endpoint 不写入 Frontend OpenAPI。
- Frontend 不直接调用这些 endpoint。
- Gateway Handler 不直接调用这些 endpoint。

---

## 11. Health Check

Endpoint：

```text
GET /health
```

最小响应：

```json
{
  "status": "ok",
  "agent": "agent-name",
  "version": "0.1.0"
}
```

`/health.status` 是 A2A 原始探针状态，进入 Registry 后必须归一化为 `Agent.health`：

```text
/health.status = ok       → Agent.health = healthy
/health.status = degraded → Agent.health = degraded
timeout / non-2xx / invalid response → Agent.health = unhealthy
未探测                        → Agent.health = unknown
```

`Agent.status`（生命周期/启用状态）与 `Agent.health`（健康状态）分离：

| 字段 | 含义 | 枚举 |
|---|---|---|
| `Agent.status` | 生命周期/启用状态 | `enabled` / `disabled` / `experimental` / `deprecated` |
| `Agent.health` | 当前健康状态 | `healthy` / `degraded` / `unhealthy` / `unknown` |

规则：

- `/health` 不应触发 LLM 请求。
- `/health` 不应执行耗时工具。
- `/health` 不应泄漏密钥。
- Registry 必须周期性检查 `/health` 并归一化为 `Agent.health`。
- `disabled` 属于 `Agent.status`，不属于 `Agent.health`。
- unhealthy / degraded Agent 不应进入 Planner 的候选列表。
- fallback 不应选择 unhealthy Agent。

---

## 12. A2A Task Contract

v1.0 核心 endpoint：

```text
POST /a2a/tasks/sendSubscribe
```

推荐请求：

```json
{
  "id": "task-001",
  "messages": [
    {
      "role": "user",
      "content": "请生成一个登录页面"
    }
  ],
  "metadata": {
    "runId": "run-001",
    "threadId": "conversation-001",
    "traceId": "trace-001",
    "agentName": "target-agent"
  }
}
```

规则：

- `id` 是 A2A task id。
- `messages` 必须包含当前用户输入。
- `messages` 可以包含结构化历史上下文。
- `metadata.runId` 对应 AG-UI run。
- `metadata.threadId` 是 `conversationId` 的 A2A 协议别名。内部统一使用 `conversationId`，不得将 threadId 视为独立的第二套会话 ID。
- `metadata.traceId` 用于跨服务追踪。
- `metadata.agentName` 应与目标 AgentCard.name 一致。
- metadata 不得携带 API key、Authorization token、用户私密 token。

---

## 13. A2A Streaming Event Contract

v1.0 必须支持以下事件语义：

```text
status: working
text
artifact
status: completed
status: failed
```

### 13.1 working

```json
{
  "type": "status",
  "status": "working"
}
```

### 13.2 text

```json
{
  "type": "text",
  "content": "这里是流式文本片段"
}
```

### 13.3 artifact

A2A streaming 中的 `event.artifact` 是 **ArtifactDraft**，不是标准 Core Artifact。

```json
{
  "type": "artifact",
  "artifact": {
    "type": "code",
    "title": "main.go",
    "content": "package main\n\nfunc main() {}",
    "metadata": {
      "language": "go"
    }
  }
}
```

ArtifactDraft 只需包含 Child Agent 能提供的字段（`type`、`title`、`content` 或 `contentRefDraft`、`metadata`）。`artifactId`、`mimeType`、`source.*`、`links.*`、`preview.*`、`version`、`status`、`createdAt` 等平台字段由 Orchestrator / ArtifactRegistry 在归一化时生成或补齐。

### 13.4 completed

```json
{
  "type": "status",
  "status": "completed"
}
```

### 13.5 failed

```json
{
  "type": "status",
  "status": "failed",
  "error": {
    "code": "A2A_AGENT_ERROR",
    "message": "Agent 执行失败",
    "retryable": true
  }
}
```

规则：

- Agent 开始处理任务后应输出 `working`。
- Agent 可以输出多个 `text` chunk。
- Agent 可以输出 0 到多个 `artifact`。
- Agent 最后必须输出 `completed` 或 `failed`。
- `completed` 后不应继续输出内容。
- `failed` 后不应继续输出正常结果。
- A2A event 不直接暴露给 Frontend。
- ProtocolConverter 负责把 A2A event 转换为 AG-UI event。

---

## 14. Artifact Output Contract

Child Agent 通过 A2A streaming 输出的是 **ArtifactDraft**，不是标准 Core Artifact。

ArtifactDraft 是 Child Agent 能自主提供的最小产物描述。Orchestrator / ArtifactRegistry 收到 ArtifactDraft 后，负责归一化为标准 Core Artifact（生成 `artifactId`、补齐 `mimeType`、`source.*`、`links.*`、`preview.*`、`version`、`status`、`createdAt` 等平台字段）。

任何 Child Agent 输出 ArtifactDraft 前必须满足：

1. AgentCard.outputModes 声明该输出类型。
2. Artifact type 被 `artifact-contract` 支持。
3. ProtocolConverter 有映射规则。
4. 前端已注册对应 Runtime Skill。
5. 安全边界允许展示。

### 14.1 ArtifactDraft 最小字段

```json
{
  "type": "code",
  "title": "main.go",
  "content": "package main\n\nfunc main() {}",
  "metadata": {
    "language": "go"
  }
}
```

或使用 `contentRefDraft`：

```json
{
  "type": "webpage",
  "title": "landing-page.html",
  "contentRefDraft": "https://internal-agent/storage/temp-001/index.html",
  "metadata": {
    "language": "html"
  }
}
```

ArtifactDraft 字段：

| 字段 | 必填 | 说明 |
|---|---:|---|
| `type` | 是 | Artifact 类型，如 `code`、`webpage`、`document` |
| `title` | 是 | 产物标题或文件名 |
| `content` | 二选一 | 小型 inline 内容 |
| `contentRefDraft` | 二选一 | 大型内容的内部引用（非 Core contentRef 对象） |
| `metadata` | 推荐 | 类型相关元数据，如 `language`、`mimeType` 提示 |

### 14.2 由 Orchestrator / ArtifactRegistry 生成的字段

以下字段 **不在 ArtifactDraft 中**，由平台归一化时生成或补齐：

| 字段 | 生成者 | 说明 |
|---|---|---|
| `artifactId` | ArtifactRegistry | 全局唯一产物 ID |
| `mimeType` | ArtifactRegistry | 根据 type + metadata 推断或默认 |
| `source.agentName` | Orchestrator | 已知调用目标 |
| `source.taskId` | Orchestrator | 已知 A2A task id |
| `links.conversationId` | Orchestrator | 已知当前会话 |
| `links.messageId` | Orchestrator / Gateway | 消息创建后回填 |
| `links.runId` | Orchestrator | 已知当前 Run |
| `preview.previewType` | ArtifactRegistry | 根据 type 查 Registry |
| `version` | ArtifactRegistry | 默认 1 |
| `status` | ArtifactRegistry | 初始 `pending`，归一化后 `ready` |
| `createdAt` | ArtifactRegistry | 归一化时间 |
| `updatedAt` | ArtifactRegistry | 归一化/更新时间 |

### 14.3 code ArtifactDraft

```json
{
  "type": "code",
  "title": "main.go",
  "content": "package main\n\nfunc main() {}",
  "metadata": {
    "language": "go"
  }
}
```

归一化后映射：

```text
ArtifactDraft type = code → Core Artifact → Frontend Skill = code_preview
```

### 14.4 webpage ArtifactDraft

```json
{
  "type": "webpage",
  "title": "index.html",
  "content": "<!DOCTYPE html><html>...</html>",
  "metadata": {
    "language": "html",
    "css": "body { margin: 0; }",
    "js": "console.log('ready')"
  }
}
```

归一化后映射：

```text
ArtifactDraft type = webpage → Core Artifact → Frontend Skill = web_preview
```

### 14.5 document / markdown ArtifactDraft

```json
{
  "type": "document",
  "title": "report.md",
  "content": "# 报告标题\n\n正文...",
  "metadata": {
    "format": "markdown"
  }
}
```

归一化后映射：

```text
ArtifactDraft type = document / markdown → Core Artifact → Frontend Skill = markdown_render
```

规则：

- ArtifactDraft 不应伪装成普通 text chunk。
- 大型结构化产物应使用 ArtifactDraft。
- Child Agent 不得在 ArtifactDraft 中提供 `artifactId`、`version`、`links.*`、`source.*`、`preview.*`、`status`、`createdAt` 等平台字段。
- Child Agent 不得直接输出 `code_preview` / `web_preview` / `markdown_render` Tool Call。
- Tool Call 只能由 ProtocolConverter 生成。
- `contentRefDraft` 是 Child Agent 内部引用，不是 Core Artifact 的 `contentRef` 对象。Orchestrator 负责将 `contentRefDraft` 转换为平台可授权的 `contentRef`。

---

## 15. Agent Registry Contract

Agent Registry 管理所有 Child Agents。

Registry 必须：

- 从配置读取多个 Agent URL。
- 拉取每个 Agent 的 AgentCard。
- 调用每个 Agent 的 `/health` 并归一化为 `Agent.health`。
- 缓存 AgentCard。
- 缓存 healthy 状态。
- 向 Planner 提供 healthy agents。
- 向 Gateway API 提供 Agent 摘要。
- 支持后续新增 Agent 时不修改 Orchestrator 核心流程。

推荐配置形式：

```yaml
agents:
  - name: code-like-agent
    url: http://code-agent:8081
  - name: web-like-agent
    url: http://web-agent:8082
  - name: document-like-agent
    url: http://doc-agent:8083
```

注意：上面的名称只是示例，不是硬约束。

规则：

- Registry 不属于 Frontend。
- Frontend 只能通过 Gateway 查询 Agent 摘要。
- Registry 中的 Agent 必须有合法 AgentCard。
- AgentCard 无效时，该 Agent 不应参与编排。
- unhealthy Agent 不应参与编排。
- fallback 只能选择 healthy Agent。

---

## 16. Orchestrator 调用规则

Orchestrator 调用 Child Agent 时必须：

1. 从 Registry 获取 Agent。
2. 校验 Agent 是否 healthy。
3. 读取 AgentCard。
4. 校验目标 skill 是否被 AgentCard.skills 支持。
5. 校验任务期望输出是否与 AgentCard.outputModes 兼容。
6. 通过 A2A Client 调用 `/a2a/tasks/sendSubscribe`。
7. 传递结构化 messages。
8. 传递 runId、conversationId（A2A 侧称 metadata.threadId）、traceId、agentName。
9. 读取 A2A streaming event。
10. 将 A2A artifact event 中的 ArtifactDraft 交给 ArtifactRegistry 归一化为 Core Artifact。
11. 交给 ProtocolConverter 转换为 AG-UI event。
12. flush Artifact 为 AG-UI `TOOL_CALL_*`。
13. 失败时根据 retryable 和 Registry 状态 fallback。

Orchestrator 不得：

- 根据硬编码 agentName 判断能力。
- 直接调用某个 Agent 的内部函数。
- 让 Gateway Handler 解析 A2A event。
- 把 A2A event 原样给 Frontend。

---

## 17. ordered_parallel 规则

v1.0 正式枚举值为 `ordered_parallel`。

```text
ExecutionPlan.strategy = ordered_parallel
legacy parallel / ordered-parallel 仅作为兼容输入别名，进入 Orchestrator 前归一化为 ordered_parallel。
实际执行可以按顺序调用多个 Agent
前端感知为多个 Agent 参与
避免 SSE 事件交错导致渲染混乱
```

规则：

- Planner 可以返回多个 TaskPlan。
- 每个 TaskPlan 指向一个 Agent。
- Orchestrator 可以顺序执行这些 task。
- 每次切换 Agent 时，应输出 AG-UI `STATE_UPDATE`。
- 每个 Agent 的输出应能在前端显示 agentName。

---

## 18. 错误模型与 fallback

推荐 A2A error：

```json
{
  "type": "status",
  "status": "failed",
  "error": {
    "code": "A2A_AGENT_ERROR",
    "message": "Agent 执行失败",
    "retryable": true
  }
}
```

推荐错误码：

```text
A2A_INVALID_REQUEST
A2A_UNAUTHORIZED
A2A_AGENT_NOT_READY
A2A_TASK_NOT_FOUND
A2A_TASK_CANCELLED
A2A_LLM_ERROR
A2A_TOOL_ERROR
A2A_ARTIFACT_ERROR
A2A_STREAM_INTERRUPTED
A2A_INTERNAL
```

规则：

- A2A `failed` 不一定立即变成 AG-UI `RUN_ERROR`。
- 如果 `retryable = true`，Orchestrator 可以 fallback。
- fallback 开始时应输出 AG-UI `STATE_UPDATE`。
- 所有候选 Agent 都失败后，才输出 AG-UI `RUN_ERROR`。
- 错误不得泄漏 API key、token、内部堆栈、完整 system prompt。

---

## 19. 安全规则

必须遵守：

- Frontend 不得直接调用 Child Agent。
- Gateway Handler 不得直接调用 Child Agent。
- AgentCard 不得泄漏密钥。
- `/health` 不得泄漏敏感配置。
- A2A metadata 不得携带用户 token。
- 日志不得打印 LLM API key。
- 日志不得打印完整敏感 system prompt。
- Artifact 内容展示必须走 Artifact Contract 和 Frontend Runtime Skills Contract。
- Agent 不得信任 Frontend 任意传来的 tool / skill 名称。
- HTML / webpage 类 Artifact 必须经过前端 sandbox 策略。

---

## 20. Mock-first 规则

新增 Agent 时可以先实现 Mock A2A Agent。

Mock Agent 必须：

- 暴露合法 AgentCard。
- 暴露 `/health`。
- 暴露 `/a2a/tasks/sendSubscribe`。
- 输出合法 A2A streaming event。
- 能模拟 `working`。
- 能模拟多个 `text` chunk。
- 能模拟至少一个 Artifact。
- 能模拟 `completed`。
- 能模拟 `failed`。
- 不直接输出 AG-UI event。

---

## 21. Contract Test 规则

### 21.1 所有 Child Agent 必须通过

- AgentCard endpoint 存在。
- AgentCard JSON 合法。
- AgentCard.name 存在且稳定。
- AgentCard.version 存在。
- AgentCard.skills 非空。
- AgentCard.inputModes 非空。
- AgentCard.outputModes 非空。
- `/health` 存在。
- `/health` 不触发 LLM。
- `/a2a/tasks/sendSubscribe` 存在。
- streaming lifecycle 合法。
- completed / failed 后不再输出正常内容。
- 不泄漏 secret。

### 21.2 按 outputMode 动态测试

如果 AgentCard.outputModes 包含 `code`：

- 必须能输出 `code` Artifact。
- `code` Artifact 必须包含 `title`、`content`、`metadata.language`。
- `code` Artifact 必须能映射到 `code_preview`。

如果 AgentCard.outputModes 包含 `webpage`：

- 必须能输出 `webpage` Artifact。
- `webpage` Artifact 必须包含 `title`、`content`。
- `webpage` Artifact 必须能映射到 `web_preview`。

如果 AgentCard.outputModes 包含 `document`：

- 必须能输出 `document` 或 `markdown` Artifact。
- Artifact 必须能映射到 `markdown_render`。

---

## 22. Example Agent Profiles

以下只是示例，不是硬约束。

### 22.1 code-like agent

适用于：

- 代码生成
- 代码解释
- 代码审查
- 测试生成
- 重构建议

推荐：

```text
inputModes = text
outputModes = text, code
skills = code_generate, code_explain, code_review, test_generate
```

### 22.2 web-like agent

适用于：

- HTML/CSS/JS 页面生成
- UI 组件生成
- 响应式布局
- Web 页面预览

推荐：

```text
inputModes = text
outputModes = text, code, webpage
skills = web_generation, ui_design, responsive_layout
```

### 22.3 document-like agent

适用于：

- Markdown 文档
- 总结报告
- PRD / 方案文档
- 技术说明

推荐：

```text
inputModes = text
outputModes = text, document
skills = document_generate, markdown_write, summarize
```

### 22.4 search-like agent

适用于：

- 搜索
- 信息检索
- 资料汇总
- 引用整理

推荐：

```text
inputModes = text
outputModes = text, document
skills = search, summarize, cite_sources
```

---

## 23. Review Checklist

Review A2A / Child Agent 相关改动时必须检查：

### AgentCard

- 是否暴露 `/.well-known/agent.json`？
- 是否包含 name / description / url / version？
- 是否包含 capabilities？
- 是否包含 skills？
- 是否包含 inputModes / outputModes？
- skills.outputTypes 是否与 outputModes 兼容？
- 是否没有泄漏 secret？

### outputMode / artifact.type / toolName 边界

- `outputModes` 是否只声明语义类别，没有被当作 `artifact.type`？
- `outputModes` 是否没有被直接当作 `toolName`？
- 是否没有通过 `outputMode` 直接选择前端组件？
- 是否所有转换都通过映射表表达？
- 是否没有通过 `agentName` 推断 outputMode 或选择 toolName？

### capabilityId 边界

- `skills[].id` 是否是能力 ID 事实源？
- `TaskPlan.capabilityIds` 是否引用 `AgentCard.skills[].id`？
- 是否没有把 `toolName` 当作 capabilityId？
- 是否没有把 `artifact.type` 当作 capabilityId？
- 是否没有把 `outputMode` 当作 capabilityId？

### Health

- 是否暴露 `/health`？
- `/health` 是否轻量？
- `/health` 是否不触发 LLM？
- `/health.status` 是否归一化为 `Agent.health`（非直接等同）？
- `Agent.status` 与 `Agent.health` 是否分离？
- `disabled` 是否没有出现在 health 值中？
- unhealthy Agent 是否不会进入 Planner？

### A2A

- 是否暴露 `/a2a/tasks/sendSubscribe`？
- 是否支持 streaming？
- 是否输出 working / text / artifact / completed / failed？
- failed 是否包含 error code？

### Artifact

- Artifact type 是否被 AgentCard.outputModes 声明？
- Artifact type 是否被 `artifact-contract` 支持？
- Artifact 是否能映射到 Frontend Runtime Skill？
- Child Agent 是否没有直接输出 AG-UI Tool Call？
- Child Agent 输出的 event.artifact 是否是 ArtifactDraft（非标准 Core Artifact）？
- Child Agent 是否没有在 ArtifactDraft 中提供 `artifactId`、`version`、`links.*`、`source.*` 等平台字段？

### Orchestrator

- 是否通过 Registry 获取 Agent？
- 是否根据 AgentCard 判断能力？
- 是否没有硬编码 agentName 判断能力？
- 是否通过 A2A Client 调用？
- 是否没有让 Gateway Handler 直接调用 Child Agent？
- 是否没有让 Frontend 直接调用 Child Agent？

---

## 24. 硬性规则

1. Child Agent 必须暴露 AgentCard。
2. Child Agent 必须暴露 `/health`。
3. Child Agent 必须暴露 `/a2a/tasks/sendSubscribe`。
4. v1.0 必须支持 2+ Child Agents。
5. 本 Skill 不固定具体 Agent 名称。
6. Agent 能力必须由 AgentCard 声明。
7. Planner / Orchestrator 不得依赖 agentName 判断能力。
8. Orchestrator 必须通过 Registry + A2A Client 调用 Child Agent。
9. Frontend 不得直接调用 Child Agent。
10. Gateway Handler 不得直接调用 Child Agent。
11. Child Agent 不得直接输出 AG-UI event。
12. Child Agent 不得直接输出 Frontend Tool Call。
13. Artifact 映射必须经过 ProtocolConverter。
14. 新增 outputMode 必须同步 Artifact / Frontend Runtime Skills Contract。
15. A2A error 必须可 fallback 或映射为 AG-UI RUN_ERROR。
16. AgentCard、health、metadata、logs 不得泄漏 secret。
17. Mock Agent 也必须遵守本契约。
18. Child Agent 输出的 A2A event.artifact 是 ArtifactDraft，不得包含 `artifactId`、`version`、`links.*`、`source.*`、`preview.*`、`status`、`createdAt` 等平台字段。
19. 标准 Core Artifact 只能由 Orchestrator / ArtifactRegistry 归一化生成。

---

## 25. 输出要求

当用户要求设计或 Review A2A / Child Agent 时，必须输出：

1. 当前属于 MVP 历史基线、v1.0 Sprint 还是 Post-v1.0。
2. AgentCard 设计。
3. capabilities。
4. skills。
5. inputModes / outputModes。
6. required endpoints。
7. health check。
8. sendSubscribe request。
9. streaming event。
10. artifact 输出。
11. registry 发现规则。
12. orchestrator 调用规则。
13. 错误与 fallback。
14. 安全边界。
15. contract tests。
16. review checklist。

除非用户明确要求，不要直接生成业务实现代码。

---

## References

- `references/agent-card-policy.md`
- `references/task-endpoints.md`
- `references/send-subscribe-streaming.md`
- `references/artifact-policy.md`
- `references/a2a-error-policy.md`
- `references/registry-discovery.md`
- `references/child-agent-lifecycle.md`
- `references/a2a-review-checklist.md`
