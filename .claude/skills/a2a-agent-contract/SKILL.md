---
name: a2a-agent-contract
description: "用于定义 AgentHub 子 Agent 的 A2A 协议契约，包括 AgentCard 元数据、任务发送接口、流式订阅接口、任务状态、产物输出、错误处理以及前端不可直接访问 A2A 的边界规则。"
---

# a2a-agent-contract

## 1. Skill 目的

本 Skill 用于定义 AgentHub 项目中 **Orchestrator ↔ Child Agent** 的 A2A Agent Contract。

它约束 Orchestrator 如何发现、识别、调用 Child Agent，以及 Child Agent 如何通过 AgentCard、A2A endpoints、Streaming Task、Artifact、错误状态与 ADK Runtime 对外提供能力。

一句话：

**Child Agent 必须以 A2A 兼容的方式暴露能力，Orchestrator 必须只通过 A2A 调用 Child Agent，不能绕过 AgentCard、A2A Task、ADK Runtime 和 Artifact 约定。**

---

## 2. 适用场景

当任务涉及以下内容时，必须使用本 Skill：

- 设计或修改 Child Agent。
- 设计或修改 `code-agent`。
- 设计或修改 `web-agent`、`doc-agent`、`custom-agent`。
- 设计 AgentCard。
- 设计 Agent capabilities。
- 设计 Agent skills。
- 设计 Agent inputModes / outputModes。
- 设计 A2A endpoint。
- 设计 A2A task request / response。
- 设计 A2A streaming event。
- 设计 A2A error。
- 设计 Orchestrator 调用 Child Agent 的 A2A Client。
- 设计 Mock A2A Agent。
- 设计 ADK Runtime 对 A2A 的适配层。
- Review Child Agent 是否符合 A2A Contract。
- Review Orchestrator 是否绕过 A2A 直接调用 Agent 内部逻辑。
- Review Agent Artifact 是否能被 Orchestrator 转换为 AG-UI Tool Call。

---

## 3. Contract 所属边界

本 Skill 只约束：

```text
Orchestrator ↔ Child Agent
```

本 Skill 不约束：

```text
Frontend ↔ Gateway REST API
Frontend ↔ Gateway AG-UI Event Stream
Gateway ↔ Orchestrator Internal Contract
Artifact 最终持久化 Schema
Frontend Runtime Skills 参数 Schema
ADK Runtime 内部实现细节
```

对应关系如下：

| 通信方向 / 内容 | 使用协议 / Contract | 是否由本 Skill 管 |
|---|---|---|
| Frontend ↔ Gateway REST API | OpenAPI | 否，由 `platform-api-contract` 管 |
| Frontend ↔ Gateway AG-UI Event Stream | AG-UI Event Contract | 否，由 `agui-event-contract` 管 |
| Gateway ↔ Orchestrator | Internal Contract | 否，由 `gateway-orchestrator-contract` 管 |
| Orchestrator ↔ Child Agent | A2A | 是 |
| Child Agent 运行时生命周期 | ADK Runtime | 部分引用，详细由 `adk-runtime-contract` 管 |
| Artifact 字段与存储 | Artifact Contract | 否，由 `artifact-contract` 管 |
| Artifact → Frontend Skill 参数 | Frontend Runtime Skills Contract | 否，由 `frontend-runtime-skills-contract` 管 |
| ExecutionPlan / TaskPlan | Intent Orchestration Contract | 否，由 `intent-orchestration-contract` 管 |

---

## 4. 核心文件

本 Skill 落地后应生成或维护：

```text
docs/contracts/a2a-agent-card.md
docs/contracts/a2a-task.md
docs/contracts/a2a-errors.md
```

可选维护：

```text
docs/contracts/a2a-agent-card.schema.json
docs/contracts/a2a-task.schema.json
docs/contracts/a2a-review-checklist.md
```

MVP v0.1 阶段必须至少明确：

```text
code-agent 的 AgentCard
/a2a/tasks/sendSubscribe
A2A streaming event: status / text / artifact
code Artifact 输出约定
A2A error → RUN_ERROR 的映射边界
```

---

## 5. 四份设计文档的优先级解释

本 Skill 必须同时遵守四类文档：

1. **PDR**：定义完整目标架构，Child Agents 基于 ADK Runtime，暴露 AgentCard，并通过 A2A 被 Orchestrator 调用。
2. **MVP 文档**：定义 v0.1 最小实施范围，只要求 `code-agent` 跑通，支持流式回复和代码产物。
3. **UML 文档**：定义 Orchestrator 通过 A2A Client 调用 Child Agent，Child Agent 输出 A2A stream event，再由 ProtocolConverter 转为 AG-UI Event。
4. **Skills 设计规范**：定义 `a2a-agent-contract` 是必须自建的项目级 Skill，不能依赖泛社区 Skill。

解释原则：

```text
PDR 决定长期方向。
MVP 决定当前范围。
UML 决定关键流程。
Skills 设计规范决定 AI 开发约束。
```

---

## 6. MVP v0.1 实施范围

MVP v0.1 只强制实现一个 Child Agent：

```text
code-agent
```

MVP v0.1 必须支持：

- `code-agent` 暴露 AgentCard。
- `code-agent` 提供 A2A Server。
- `code-agent` 支持 `/a2a/tasks/sendSubscribe`。
- `code-agent` 使用 ADK Runtime 约定。
- `code-agent` 能接收用户消息和历史上下文。
- `code-agent` 能流式输出文本。
- `code-agent` 能在检测到代码块后生成 `code` Artifact。
- `code` Artifact 的 metadata 至少包含 `language`。
- `code` Artifact 的 title 可作为文件名，如 `main.go`。
- `code` Artifact 最终可被 Orchestrator 转换为 `code_preview` Tool Call。
- A2A `status/text/artifact/completed/failed` 能被 ProtocolConverter 识别。

MVP v0.1 暂不强制实现：

- `web-agent`。
- `doc-agent`。
- `custom-agent`。
- Agent 动态注册中心。
- Agent 自动发现。
- Agent 健康检查面板。
- 多 Agent 并行 / 串行协作。
- A2A 非流式完整能力。
- 复杂 Artifact 类型。
- 复杂错误降级。
- AgentCard 市场展示。
- 自建 Agent 发布流程。

MVP v0.1 可以使用配置文件写死 Agent：

```text
name: code-agent
url: http://code-agent:8081
```

但不能因此取消 AgentCard 或 A2A endpoint 的 Contract 要求。

---

## 7. Post-MVP 完整目标

Post-MVP 可以扩展：

- `web-agent`
- `doc-agent`
- `custom-agent`
- Agent Registry
- AgentCard 动态拉取
- Agent 健康检查
- Agent capability 匹配
- Agent skill 路由
- 多 Agent ExecutionPlan
- 并行 / 串行 A2A Task
- fallback / retry
- Agent outputModes → Frontend Runtime Skills 自动映射
- Artifact 多类型输出
- A2A task cancel
- A2A task get
- A2A task history
- A2A authentication
- A2A service-to-service trace

扩展时必须保持：

- Orchestrator 仍然只通过 A2A 调 Child Agent。
- Child Agent 仍然必须暴露 AgentCard。
- AgentCard schema 向后兼容。
- A2A Task schema 向后兼容。
- Artifact 输出仍然必须能被 Artifact Contract / Frontend Runtime Skills Contract 接收。
- 不得让 Frontend 直接调用 Child Agent。
- 不得让 Gateway 直接调用 Child Agent。

---

## 8. AgentCard 职责

AgentCard 是 Child Agent 的能力声明。

它用于：

- 告诉 Orchestrator 该 Agent 是谁。
- 告诉 Orchestrator 该 Agent 的 URL。
- 告诉 Orchestrator 该 Agent 支持哪些 inputModes。
- 告诉 Orchestrator 该 Agent 可能输出哪些 outputModes。
- 告诉 Orchestrator 该 Agent 有哪些 skills。
- 支持 Post-MVP 的 Agent 选择、路由、能力过滤和前端展示。

AgentCard 不是：

- 运行时消息。
- A2A Task。
- 前端 UI schema。
- OpenAPI REST response 的替代品。
- Agent 内部 prompt 的完整暴露。
- 敏感配置公开文件。

---

## 9. AgentCard 必需字段

MVP v0.1 推荐最小 AgentCard：

```json
{
  "name": "code-agent",
  "description": "负责生成、解释和审查代码的 Agent",
  "url": "http://code-agent:8081",
  "version": "0.1.0",
  "capabilities": {
    "streaming": true,
    "artifacts": true
  },
  "skills": [
    {
      "id": "code_generate",
      "name": "代码生成",
      "description": "根据用户需求生成代码",
      "outputTypes": ["code", "text"]
    }
  ],
  "inputModes": ["text"],
  "outputModes": ["text", "code"]
}
```

字段说明：

| 字段 | 必填 | 说明 |
|---|---:|---|
| `name` | 是 | Agent 唯一名称，MVP 为 `code-agent` |
| `description` | 是 | Agent 简要说明 |
| `url` | 是 | A2A Server 地址 |
| `version` | 是 | Agent 版本 |
| `capabilities` | 是 | 能力声明 |
| `skills` | 是 | Agent 具备的技能 |
| `inputModes` | 是 | 支持的输入类型 |
| `outputModes` | 是 | 可能输出的产物类型 |

规则：

- `name` 必须稳定。
- `url` 不应暴露内部敏感 token。
- `version` 必须可用于排查兼容性。
- `skills[].id` 必须稳定。
- `outputModes` 必须能映射到 Artifact / Frontend Skill。
- MVP v0.1 的 `code-agent` 必须包含 `code` outputMode。
- AgentCard 不应暴露完整 system prompt、API key、内部服务 token。

---

## 10. Agent capabilities

推荐 capabilities：

```json
{
  "streaming": true,
  "artifacts": true,
  "tools": false,
  "cancellable": false
}
```

字段说明：

| 字段 | 说明 |
|---|---|
| `streaming` | 是否支持流式输出 |
| `artifacts` | 是否可能输出 Artifact |
| `tools` | 是否支持调用外部工具 |
| `cancellable` | 是否支持取消 task |

MVP v0.1：

```text
streaming = true
artifacts = true
tools = false 或暂不声明
cancellable = false 或暂不强制
```

Post-MVP 可扩展 tools、cancel、multiModal 等能力。

---

## 11. Agent skills

AgentCard 中的 skills 用于表达 Agent 能力。

示例：

```json
{
  "id": "code_generate",
  "name": "代码生成",
  "description": "根据用户需求生成代码",
  "outputTypes": ["code", "text"]
}
```

规则：

- `id` 必须稳定。
- `name` 可用于展示。
- `description` 用于 Orchestrator / 用户理解。
- `outputTypes` 必须是系统认可的产物类型。
- MVP v0.1 中 `code-agent` 至少包含代码生成能力。
- Post-MVP 可加入 `code_review`、`code_explain`、`test_generate` 等能力。
- `skills` 不能伪造 Agent 实际不支持的能力。

---

## 12. inputModes / outputModes

推荐 inputModes：

```text
text
file
image
url
```

MVP v0.1 只强制：

```text
text
```

推荐 outputModes：

```text
text
code
webpage
file
image
document
diff
terminal
chart
```

MVP v0.1 只强制：

```text
text
code
```

映射关系建议：

| outputMode | Artifact type | Frontend Skill |
|---|---|---|
| `code` | `code` | `code_preview` |
| `webpage` | `webpage` | `web_preview` |
| `file` | `file` | `file_download` |
| `image` | `image` | `image_preview` |
| `document` | `document` | `markdown_render` |
| `diff` | `diff` | `diff_preview` |
| `terminal` | `terminal` | `terminal_output` |
| `chart` | `chart` | `chart_render` |

本 Skill 只规定 AgentCard 中声明 outputModes。  
具体 Artifact schema 由 `artifact-contract` 定义。  
具体 Frontend Skill 参数由 `frontend-runtime-skills-contract` 定义。

---

## 13. A2A endpoint 范围

MVP v0.1 必须支持：

```text
GET  /.well-known/agent.json
POST /a2a/tasks/sendSubscribe
```

Post-MVP 可扩展：

```text
POST /a2a/tasks/send
POST /a2a/tasks/sendSubscribe
GET  /a2a/tasks/{id}
POST /a2a/tasks/{id}/cancel
```

规则：

- Orchestrator 是唯一允许调用这些 endpoint 的系统角色。
- Frontend 不允许调用 A2A endpoint。
- Gateway handler 不允许直接调用 A2A endpoint。
- A2A endpoint 不属于 Frontend REST API，不得写进 `docs/contracts/openapi.yaml`。
- A2A endpoint 的详细 contract 应写入 `docs/contracts/a2a-task.md`。
- AgentCard contract 应写入 `docs/contracts/a2a-agent-card.md`。
- 错误 contract 应写入 `docs/contracts/a2a-errors.md`。

---

## 14. AgentCard endpoint

AgentCard endpoint：

```text
GET /.well-known/agent.json
```

响应必须返回 AgentCard。

规则：

- 必须可由 Orchestrator 或 Agent Registry 拉取。
- MVP v0.1 可通过配置文件静态注册，但 endpoint 仍应保留。
- Post-MVP Agent Registry 可周期性拉取并健康检查。
- AgentCard response 不应包含密钥。
- 如果 AgentCard 无效，Orchestrator 不应调用该 Agent。
- 如果 AgentCard 的 `outputModes` 不包含 `code`，MVP 不应将其作为 `code-agent`。

---

## 15. sendSubscribe endpoint

MVP v0.1 核心 endpoint：

```text
POST /a2a/tasks/sendSubscribe
```

用途：

- Orchestrator 创建一个 A2A Task。
- Child Agent 开始处理消息。
- Child Agent 通过 SSE / streaming response 输出 `status`、`text`、`artifact`、`completed`、`failed` 等事件。

请求逻辑结构：

```json
{
  "id": "task-001",
  "messages": [
    {
      "role": "user",
      "content": "帮我写一个 Go HTTP 服务器"
    }
  ],
  "metadata": {
    "runId": "run-001",
    "threadId": "conv-001",
    "traceId": "trace-001"
  }
}
```

规则：

- `id` 必须稳定，用作 taskId。
- `messages` 必须包含用户输入和必要历史上下文。
- `metadata.runId` 应来自 AG-UI Run。
- `metadata.threadId` 应来自会话。
- `metadata.traceId` 应贯穿 Gateway、Orchestrator、Child Agent。
- 不应把 Authorization token、API key、完整敏感 prompt 放入 metadata。
- 请求字段对外 JSON 使用 camelCase。

---

## 16. A2A message 结构

推荐消息结构：

```json
{
  "role": "user",
  "content": "帮我写一个 Go HTTP 服务器"
}
```

推荐 role：

```text
user
agent
system
tool
```

MVP v0.1 至少支持：

```text
user
agent
system
```

规则：

- `content` 在 MVP 中可以是字符串。
- Post-MVP 可以扩展为结构化 parts。
- 历史消息应由 Orchestrator 构造后传给 Child Agent。
- Child Agent 不直接查询 Gateway 会话数据库。
- system prompt 可由 ADK Runtime / Agent 配置注入，不应由 Frontend 任意传入。

---

## 17. A2A streaming event 类型

MVP v0.1 必须支持：

```text
status
text
artifact
```

推荐事件形态：

### status: working

```json
{
  "type": "status",
  "status": "working"
}
```

### text

```json
{
  "type": "text",
  "content": "package main"
}
```

### artifact

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

### status: completed

```json
{
  "type": "status",
  "status": "completed"
}
```

### status: failed

```json
{
  "type": "status",
  "status": "failed",
  "error": "LLM API timeout"
}
```

规则：

- `status: working` 应被 Orchestrator 转换为 AG-UI `TEXT_MESSAGE_START`。
- `text` 应被转换为 `TEXT_MESSAGE_CONTENT`。
- `artifact` 必须先由 Orchestrator 缓存，不能立即透传给 Frontend。
- `status: completed` 触发 `TEXT_MESSAGE_END`、flush artifacts、`TOOL_CALL_*`、`RUN_FINISHED`。
- `status: failed` 触发 `RUN_ERROR`。
- A2A event 不得直接暴露给 Frontend。
- A2A event 不得由 Gateway handler 直接解析。

---

## 18. A2A task 生命周期

MVP v0.1 推荐生命周期：

```text
submitted
working
completed
failed
```

Post-MVP 可扩展：

```text
cancelled
queued
requiresInput
```

生命周期规则：

```text
submitted → working → completed
submitted → working → failed
submitted → cancelled
```

MVP v0.1 中：

- Orchestrator 发送 task 后，Agent 输出 `working`。
- Agent 流式输出多个 `text`。
- Agent 可以输出 0 到多个 `artifact`。
- Agent 最后输出 `completed` 或 `failed`。
- `completed` 后不应继续输出 `text` 或 `artifact`。
- `failed` 后不应继续输出正常事件。

---

## 19. Artifact 输出规则

A2A Artifact 是 Child Agent 输出的非文本产物。

MVP v0.1 只强制支持：

```text
type = code
```

`code` Artifact 推荐结构：

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

规则：

- `type` 必须是 `code`。
- `title` 推荐作为文件名。
- `content` 是代码文本。
- `metadata.language` 必须存在。
- 大代码可以作为 Artifact，不应塞进 `TEXT_MESSAGE_CONTENT`。
- Orchestrator 将 `code` Artifact 映射为 `code_preview`。
- 完整 Artifact schema 由 `artifact-contract` 定义。
- A2A Agent 不直接决定前端组件，只声明 Artifact type 和内容。

Post-MVP 可扩展：

```text
webpage
file
image
document
diff
terminal
chart
```

---

## 20. code-agent MVP Contract

MVP v0.1 的 `code-agent` 必须满足：

### 20.1 AgentCard

```text
name = code-agent
inputModes 包含 text
outputModes 包含 text, code
capabilities.streaming = true
capabilities.artifacts = true
skills 至少包含 code_generate
```

### 20.2 A2A Server

必须暴露：

```text
GET  /.well-known/agent.json
POST /a2a/tasks/sendSubscribe
```

### 20.3 流式输出

必须输出：

```text
status: working
text chunk*
artifact? 
status: completed
```

失败时输出：

```text
status: failed
error
```

### 20.4 代码产物

如果回复中生成代码块，应输出 `code` Artifact。

Artifact 必须包含：

```text
type = code
title
content
metadata.language
```

### 20.5 ADK Runtime

`code-agent` 应通过 ADK Runtime 的统一能力输出：

```text
ctx.StreamText(...)
ctx.AddArtifact(...)
```

ADK Runtime 的内部接口由 `adk-runtime-contract` 进一步细化。

---

## 21. Orchestrator 调用规则

Orchestrator 调用 Child Agent 时必须：

- 根据配置或 Agent Registry 找到 Agent URL。
- 读取或验证 AgentCard。
- 根据 AgentCard 判断该 Agent 是否支持目标 outputModes。
- 通过 A2A Client 调用 `/a2a/tasks/sendSubscribe`。
- 将 Gateway 传入的 history 转换为 A2A messages。
- 传递 `runId`、`threadId`、`traceId`。
- 读取 A2A stream event。
- 将 A2A event 交给 ProtocolConverter。
- 不直接把 A2A event 传给 Frontend。
- 不直接修改 Frontend UI state。
- 不让 Gateway handler 参与 A2A 解析。

MVP v0.1 可以跳过复杂 Agent 选择，直接调用 `code-agent`。  
但仍应保留 AgentCard 和 A2A Client 边界。

---

## 22. Agent Registry 规则

MVP v0.1 可以使用配置文件静态注册：

```yaml
agents:
  - name: code-agent
    url: http://code-agent:8081
```

Post-MVP 可扩展 Agent Registry：

- 拉取 AgentCard。
- 健康检查。
- 缓存 Agent 能力。
- 按 skills / outputModes 查询 Agent。
- 支持自建 Agent 注册。
- 支持 Agent 版本管理。
- 支持 fallback / retry 选择。

规则：

- Agent Registry 不属于 Frontend。
- Frontend 可以通过 Gateway 查询 Agent 摘要，但不能直接读取 A2A endpoint。
- Agent Registry 中的 Agent 必须有合法 AgentCard。
- 无 AgentCard 的 Agent 不应参与编排。

---

## 23. 错误模型

A2A error 推荐结构：

```json
{
  "type": "status",
  "status": "failed",
  "error": {
    "code": "A2A_AGENT_ERROR",
    "message": "Agent 执行失败",
    "retryable": false
  }
}
```

MVP v0.1 也允许简单字符串错误：

```json
{
  "type": "status",
  "status": "failed",
  "error": "LLM API timeout"
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

错误映射规则：

- A2A failed → OrchestratorError 或 AG-UI `RUN_ERROR`。
- 错误不能泄漏 API key、token、内部堆栈。
- `retryable = true` 可供 Post-MVP fallback / retry 使用。
- MVP v0.1 不强制自动 retry。
- A2A 连接失败由 Orchestrator 转成 `ORCHESTRATOR_A2A_CONNECT_FAILED` 或 AG-UI `RUN_ERROR`。

---

## 24. 安全规则

A2A Agent Contract 必须遵守：

- Frontend 不得直接调用 A2A endpoint。
- Gateway handler 不得直接调用 A2A endpoint。
- 只有 Orchestrator / A2A Client 可以调用 Child Agent。
- AgentCard 不得泄漏 API key。
- A2A metadata 不得携带用户 token。
- Agent 日志不得打印 LLM API key。
- Agent 日志不得打印完整敏感 system prompt。
- Artifact 内容在展示前必须经过 Artifact / Frontend Runtime Skills Contract 处理。
- Agent 不能信任 Frontend 任意传来的 tool / skill 名称。
- Post-MVP A2A 调用应增加服务间鉴权。
- 自建 Agent 必须做权限隔离和能力校验。

---

## 25. Trace 与观测性

A2A 调用必须支持追踪字段：

```text
traceId
runId
threadId
taskId
agentName
```

规则：

- Gateway 生成或透传 `traceId`。
- Orchestrator 将 `traceId` 传入 A2A metadata。
- Child Agent 日志必须包含 `traceId` 和 `taskId`。
- A2A 错误必须能关联回 `runId`。
- 不得在日志中打印 Authorization token 或 LLM API key。
- Post-MVP 可在 AgentCard 中声明 observability 能力。

---

## 26. Mock-first 规则

在真实 LLM / Agent 完成前，可以使用 Mock A2A Agent。

Mock A2A Agent 必须：

- 暴露合法 AgentCard。
- 暴露 `/a2a/tasks/sendSubscribe`。
- 输出合法 A2A stream event。
- 能模拟 `status: working`。
- 能模拟多段 `text`。
- 能模拟 `code` Artifact。
- 能模拟 `status: completed`。
- 能模拟 `status: failed`。
- 不输出未定义事件。
- 不绕过 Orchestrator。
- 不直接输出 AG-UI Event。

推荐 mock 流程：

```text
status: working
text: "下面是 Go HTTP Server 示例："
text: "```go\npackage main..."
artifact: {type:"code", title:"main.go", content:"...", metadata:{language:"go"}}
status: completed
```

---

## 27. Contract Test 规则

A2A Agent Contract 至少应验证：

### AgentCard

- `/.well-known/agent.json` 是否存在。
- `name` 是否稳定。
- `url` 是否存在。
- `version` 是否存在。
- `capabilities.streaming` 是否正确。
- `inputModes` 是否包含 `text`。
- `outputModes` 是否包含 `code`。
- `skills` 是否包含代码生成能力。
- AgentCard 是否不泄漏敏感信息。

### sendSubscribe

- `/a2a/tasks/sendSubscribe` 是否存在。
- request 是否包含 task id。
- request 是否包含 messages。
- request 是否透传 `runId`、`threadId`、`traceId`。
- response 是否是 streaming。
- 是否输出 `status: working`。
- 是否输出 `text`。
- 是否输出 `status: completed` 或 `status: failed`。
- completed 后是否不再输出内容。

### Artifact

- 是否能输出 `code` Artifact。
- `code` Artifact 是否包含 `title`。
- `code` Artifact 是否包含 `content`。
- `code` Artifact 是否包含 `metadata.language`。
- Artifact 是否没有直接伪装成 text chunk。

### 边界

- Frontend 是否不能调用 A2A endpoint。
- Gateway handler 是否没有直接调用 A2A endpoint。
- Orchestrator 是否通过 A2A Client 调用。
- A2A event 是否没有直接暴露给 Frontend。
- A2A event 是否能被 ProtocolConverter 处理。

---

## 28. 与其他 Skills 的协作

### 28.1 与 project-architecture

`project-architecture` 定义服务边界。  
本 Skill 细化 Orchestrator 与 Child Agent 的 A2A 边界。

如果发现 Frontend 或 Gateway handler 直接调用 A2A endpoint，必须拒绝。

---

### 28.2 与 gateway-orchestrator-contract

`gateway-orchestrator-contract` 规定 Gateway 如何调用 Orchestrator。  
本 Skill 规定 Orchestrator 如何调用 Child Agent。

A2A Client 属于 Orchestrator 边界。

---

### 28.3 与 agui-event-contract

A2A event 不直接到前端。  
必须由 ProtocolConverter 转为 AG-UI event。

A2A `text` → AG-UI `TEXT_MESSAGE_CONTENT`。  
A2A `artifact` → 缓存 → AG-UI `TOOL_CALL_*`。

---

### 28.4 与 artifact-contract

A2A Artifact 的最终 schema 由 `artifact-contract` 细化。  
本 Skill 只规定 Child Agent 可以输出 Artifact，且 MVP 必须支持 `code` Artifact。

---

### 28.5 与 frontend-runtime-skills-contract

Artifact 最终映射到 Frontend Skill。  
MVP 中 `code` Artifact 必须映射到 `code_preview`。

具体 `code_preview` 参数 schema 由 `frontend-runtime-skills-contract` 定义。

---

### 28.6 与 adk-runtime-contract

ADK Runtime 规定 Child Agent 内部如何处理 task、stream、artifact、LLM。

本 Skill 只要求 Child Agent 对外符合 A2A Contract。  
ADK Runtime 内部接口由 `adk-runtime-contract` 进一步定义。

---

### 28.7 与 intent-orchestration-contract

Post-MVP 中 Orchestrator 根据 ExecutionPlan 选择 Agent。  
ExecutionPlan 与 Agent 选择策略由 `intent-orchestration-contract` 定义。  
本 Skill 只规定被选中的 Agent 必须可通过 A2A 调用。

---

## 29. 硬性规则

Coding Agent 在处理 A2A / Child Agent 相关任务时必须遵守：

1. Child Agent 必须暴露 AgentCard。
2. Child Agent 必须通过 A2A endpoint 被调用。
3. Orchestrator 是唯一允许调用 A2A endpoint 的系统角色。
4. Frontend 不允许直接调用 A2A endpoint。
5. Gateway handler 不允许直接调用 A2A endpoint。
6. A2A endpoint 不属于 Frontend REST API，不得写入 `openapi.yaml`。
7. MVP v0.1 必须支持 `code-agent`。
8. MVP v0.1 必须支持 `/a2a/tasks/sendSubscribe`。
9. MVP v0.1 必须支持 A2A `status/text/artifact` stream event。
10. MVP v0.1 必须支持 `code` Artifact。
11. `code` Artifact 必须能映射到 `code_preview`。
12. AgentCard 的 `outputModes` 必须真实反映 Agent 能力。
13. AgentCard 不得泄漏 API key、token、完整 system prompt。
14. A2A event 不得直接暴露给 Frontend。
15. A2A Artifact 不得直接塞进 `TEXT_MESSAGE_CONTENT`。
16. A2A error 必须能映射为 `RUN_ERROR`。
17. Mock Agent 也必须遵守 A2A Contract。
18. Post-MVP 扩展新 Agent 前必须先定义 AgentCard。
19. Post-MVP 扩展新 Artifact type 前必须同步 Artifact / Frontend Runtime Skills Contract。
20. 必须遵守 `Contract first / Mock first / Real integration later / Review always`。

---

## 30. 必须维护的文件

使用本 Skill 时，至少需要维护：

```text
skills/a2a-agent-contract/SKILL.md
docs/contracts/a2a-agent-card.md
docs/contracts/a2a-task.md
docs/contracts/a2a-errors.md
```

根据需要维护：

```text
docs/contracts/a2a-agent-card.schema.json
docs/contracts/a2a-task.schema.json
docs/contracts/a2a-review-checklist.md
agents/adk/
agents/code-agent/
server/internal/a2a/
server/internal/orchestrator/
```

MVP v0.1 阶段不要求马上生成业务代码。  
如果用户只要求 Contract，则不要创建 Go 实现。

---

## 31. 输出要求

当用户要求设计 A2A / Child Agent Contract 时，Coding Agent 必须输出：

1. 当前属于 MVP `code-agent` 范围还是 Post-MVP 多 Agent 范围。
2. AgentCard 字段。
3. Agent capabilities。
4. Agent skills。
5. inputModes / outputModes。
6. A2A endpoint。
7. `sendSubscribe` request。
8. A2A streaming event。
9. A2A task lifecycle。
10. Artifact 输出规则。
11. code-agent MVP Contract。
12. Orchestrator 调用规则。
13. 错误模型。
14. 安全规则。
15. traceId / runId / taskId。
16. Mock-first 规则。
17. Contract Test。
18. Review Checklist。

除非用户明确要求，不要直接生成 Child Agent / A2A Server 业务实现代码。

---

## 32. Review Checklist

在接受任何 A2A Agent 设计或实现前，必须检查：

### AgentCard

- 是否暴露 `/.well-known/agent.json`？
- 是否包含 `name`？
- 是否包含 `url`？
- 是否包含 `version`？
- 是否包含 `capabilities`？
- 是否包含 `skills`？
- 是否包含 `inputModes`？
- 是否包含 `outputModes`？
- `code-agent` 是否包含 `code` outputMode？
- AgentCard 是否没有泄漏敏感信息？

### A2A endpoint

- 是否暴露 `/a2a/tasks/sendSubscribe`？
- 是否没有把 A2A endpoint 写入 Frontend OpenAPI？
- 是否没有让 Frontend 直接调用？
- 是否没有让 Gateway handler 直接调用？
- 是否由 Orchestrator / A2A Client 调用？

### Streaming event

- 是否输出 `status: working`？
- 是否输出 `text` chunk？
- 是否能输出 `artifact`？
- 是否输出 `status: completed`？
- 失败时是否输出 `status: failed`？
- completed / failed 后是否停止正常输出？
- A2A event 是否能被 ProtocolConverter 处理？

### Artifact

- MVP 是否支持 `code` Artifact？
- `code` Artifact 是否包含 `title`？
- `code` Artifact 是否包含 `content`？
- `code` Artifact 是否包含 `metadata.language`？
- Artifact 是否没有直接伪装成 text chunk？
- `code` Artifact 是否能映射到 `code_preview`？

### Orchestrator 边界

- Orchestrator 是否通过 A2A Client 调用？
- Orchestrator 是否没有绕过 AgentCard？
- Orchestrator 是否没有直接依赖 Agent 内部实现？
- Orchestrator 是否把 A2A event 交给 ProtocolConverter？
- A2A error 是否能转成 `RUN_ERROR`？

### MVP / Post-MVP

- MVP 是否只强制 `code-agent`？
- MVP 是否不要求 `web-agent` / `doc-agent`？
- MVP 是否不要求复杂 Agent Registry？
- Post-MVP 扩展是否保留 AgentCard / A2A 兼容性？

---

## 33. 完成定义

本 Skill 视为完成，当且仅当：

```text
skills/a2a-agent-contract/SKILL.md
```

已经明确：

- A2A 属于 Orchestrator ↔ Child Agent 边界。
- Child Agent 必须暴露 AgentCard。
- Child Agent 必须提供 A2A endpoint。
- MVP v0.1 只强制 `code-agent`。
- MVP v0.1 必须支持 `sendSubscribe`。
- MVP v0.1 必须支持 `status/text/artifact` 流式事件。
- MVP v0.1 必须支持 `code` Artifact。
- `code` Artifact 必须能映射到 `code_preview`。
- A2A error 必须能映射到 `RUN_ERROR`。
- AgentCard / A2A Task / A2A Errors 的正式文档清单。
- 硬性规则。
- Review Checklist。

正式落地时还应生成：

```text
docs/contracts/a2a-agent-card.md
docs/contracts/a2a-task.md
docs/contracts/a2a-errors.md
```


## 34. v1.1 对齐补充

### 34.1 MVP 最小必需保持不变

MVP v0.1 最小必需仍为：

```text
GET  /.well-known/agent.json
POST /a2a/tasks/sendSubscribe
```

### 34.2 v1.1 / Post-MVP 完整 endpoint 范围

```text
GET    /.well-known/agent.json
POST   /a2a/tasks/send
POST   /a2a/tasks/sendSubscribe
GET    /a2a/tasks/:id
DELETE /a2a/tasks/:id/cancel
GET    /health
```

### 34.3 兼容说明

- v1.1 推荐取消路径：`DELETE /a2a/tasks/:id/cancel`。
- 历史 `POST /a2a/tasks/{id}/cancel` 可作为兼容路径保留，新实现优先 DELETE。
- MVP / PDR 使用 `/.well-known/agent.json`；Post-MVP 可兼容 `/.well-known/agent-card.json`；两者语义必须一致。

### 34.4 /health 说明

`/health` 用于 Agent 健康检查、Agent Registry、fallback、调试与部署探活。MVP v0.1 可先返回最小 healthy 状态；Post-MVP 可由后续 `data-persistence-contract` 细化 `AGENT_HEALTH_CHECK` 持久化。

### 34.5 A2A 安全补充

- A2A metadata 不得携带用户 token。
- A2A 错误不得泄漏 token、API key、stack trace、内部地址、完整 system prompt。
- A2A endpoint 不得暴露给 Frontend。
- Gateway handler 不得直接调用 A2A endpoint。

### 34.6 跨 Skill 引用补充

- A2A 安全边界由后续 `security-boundary-contract` 细化。
- A2A taskId、Agent health check 持久化由后续 `data-persistence-contract` 细化。
- Agent 路由、fallback、registry 策略由后续 `intent-orchestration-contract` 细化。
- Artifact 输出结构由后续 `artifact-contract` 细化。
- Frontend Runtime Skill 参数与 ToolResult 由后续 `frontend-runtime-skills-contract` 细化。



## References

- `references/agent-card-policy.md`
- `references/task-endpoints.md`
- `references/send-subscribe-streaming.md`
- `references/artifact-policy.md`
- `references/a2a-error-policy.md`
- `references/a2a-review-checklist.md`
