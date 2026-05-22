# A2A Task Contract

## 1. 文档目的

本文档定义 AgentHub 中 Orchestrator ↔ Child Agent 的 A2A Task 契约。

它约束：

- Orchestrator 如何创建 A2A Task。
- Child Agent 如何通过 streaming event 返回状态、文本和 Artifact。
- A2A `status/text/artifact` 如何被 ProtocolConverter 转换为 AG-UI Event。
- MVP v0.1 中 `code-agent`、`sendSubscribe`、`code` Artifact 的最小可行链路。

---

## 2. 协议边界

A2A Task 属于：

```text
Orchestrator ↔ Child Agent
```

不属于：

```text
Frontend ↔ Gateway REST API
Frontend ↔ Gateway AG-UI Event Stream
Gateway ↔ Orchestrator Internal Contract
```

规则：

- Frontend 不允许直接调用 A2A endpoint。
- Gateway handler 不允许直接调用 A2A endpoint。
- Orchestrator / A2A Client 是唯一允许调用 Child Agent A2A endpoint 的边界。
- A2A endpoint 不属于 Frontend REST API，不得写入 `docs/contracts/openapi.yaml`。
- A2A event 不得直接暴露给 Frontend。
- A2A event 必须先经过 ProtocolConverter。

---

## 3. MVP v0.1 范围

MVP v0.1 必须支持：

```text
POST /a2a/tasks/sendSubscribe
```

MVP v0.1 的目标 Agent：

```text
code-agent
```

MVP v0.1 必须支持的 stream event：

```text
status: working
text
artifact
status: completed
status: failed
```

MVP v0.1 必须支持的 Artifact：

```text
type = code
```

MVP v0.1 暂不强制：

- `POST /a2a/tasks/send`
- `GET /a2a/tasks/{id}`
- `POST /a2a/tasks/{id}/cancel`
- 多 Agent Task。
- 非文本输入。
- 复杂 Task history。
- requiresInput。
- 复杂 retry / fallback。

---

## 4. A2A Endpoint 范围

MVP 必须：

```text
GET  /.well-known/agent.json
POST /a2a/tasks/sendSubscribe
```

Post-MVP 可扩展：

```text
POST /a2a/tasks/send
GET  /a2a/tasks/{id}
POST /a2a/tasks/{id}/cancel
```

规则：

- `sendSubscribe` 用于 streaming task。
- `send` 可用于非流式任务，MVP 不强制。
- `cancel` 可用于任务取消，MVP 不强制。
- endpoint 只允许 Orchestrator / A2A Client 调用。
- endpoint 不得暴露给 Frontend。

---

## 5. sendSubscribe Request

Endpoint：

```text
POST /a2a/tasks/sendSubscribe
```

推荐 request：

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

字段说明：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `id` | string | 是 | A2A task ID |
| `messages` | array | 是 | 当前任务消息和必要历史 |
| `metadata` | object | 否 | 链路上下文 |

`metadata` 推荐字段：

| 字段 | 说明 |
|---|---|
| `runId` | AG-UI Run ID |
| `threadId` | conversation / thread ID |
| `traceId` | 跨服务链路追踪 ID |

规则：

- `id` 必须稳定。
- `messages` 必须至少包含用户消息。
- 历史上下文由 Orchestrator 构造。
- Child Agent 不直接查询 Gateway 会话数据库。
- `metadata` 不得携带用户 token、API key、完整敏感 prompt。
- 字段命名使用 camelCase。

---

## 6. A2A Message

推荐结构：

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

- MVP 中 `content` 可以是字符串。
- Post-MVP 可以扩展为结构化 parts。
- system prompt 可由 ADK Runtime / Agent 配置注入，不应由 Frontend 任意传入。
- Orchestrator 应控制传入 Agent 的上下文长度。

---

## 7. Streaming Response 格式

MVP v0.1 推荐 SSE 或等价 streaming response。

示例：

```text
data: {"type":"status","status":"working"}

data: {"type":"text","content":"好的，我来帮你写..."}

data: {"type":"artifact","artifact":{"type":"code","title":"main.go","content":"package main","metadata":{"language":"go"}}}

data: {"type":"status","status":"completed"}
```

规则：

- 每条 event 必须是合法 JSON。
- event 必须包含 `type`。
- MVP 支持 `status`、`text`、`artifact`。
- 不得输出未定义事件。
- streaming 结束前必须输出 `completed` 或 `failed`，除非连接异常中断。

---

## 8. Streaming Event 类型

### 8.1 status: working

```json
{
  "type": "status",
  "status": "working"
}
```

含义：

```text
Agent 已开始处理任务。
```

转换：

```text
A2A status:working
→ AG-UI TEXT_MESSAGE_START
```

---

### 8.2 text

```json
{
  "type": "text",
  "content": "package main"
}
```

含义：

```text
Agent 输出一段文本 chunk。
```

转换：

```text
A2A text
→ AG-UI TEXT_MESSAGE_CONTENT
```

规则：

- `content` 只表示文本 chunk。
- 大型代码产物应最终作为 Artifact 输出。
- 可以流式输出 Markdown 文本。
- 不能把结构化 Artifact 伪装成 text。

---

### 8.3 artifact

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

含义：

```text
Agent 输出非文本产物。
```

转换：

```text
A2A artifact
→ Orchestrator 缓存
→ completed 后 flush
→ AG-UI TOOL_CALL_*
```

规则：

- Artifact 不得直接透传给 Frontend。
- Artifact 不得直接塞进 AG-UI `TEXT_MESSAGE_CONTENT`。
- MVP v0.1 只强制 `code` Artifact。
- 完整 Artifact schema 由 `artifact-contract` 定义。

---

### 8.4 status: completed

```json
{
  "type": "status",
  "status": "completed"
}
```

含义：

```text
Agent 已完成任务。
```

转换：

```text
A2A status:completed
→ AG-UI TEXT_MESSAGE_END
→ flush artifactBuffer
→ TOOL_CALL_START / TOOL_CALL_ARGS / TOOL_CALL_END
→ RUN_FINISHED
```

规则：

- `completed` 后不应继续输出 `text` 或 `artifact`。
- 如果存在 Artifact，必须在 completed 后由 Orchestrator flush。
- 如果无 Artifact，可以直接进入 `RUN_FINISHED`。

---

### 8.5 status: failed

```json
{
  "type": "status",
  "status": "failed",
  "error": "LLM API timeout"
}
```

或：

```json
{
  "type": "status",
  "status": "failed",
  "error": {
    "code": "A2A_LLM_ERROR",
    "message": "LLM API timeout",
    "retryable": false
  }
}
```

含义：

```text
Agent 执行失败。
```

转换：

```text
A2A status:failed
→ AG-UI RUN_ERROR
```

规则：

- `failed` 后不应继续输出正常事件。
- 错误不能泄漏 token、API key、内部堆栈。
- MVP v0.1 可使用字符串错误，Post-MVP 推荐结构化错误。

---

## 9. Task 生命周期

MVP v0.1 推荐生命周期：

```text
submitted
working
completed
failed
```

Post-MVP 可扩展：

```text
queued
cancelled
requiresInput
```

合法流转：

```text
submitted → working → completed
submitted → working → failed
submitted → cancelled
```

规则：

- `completed` 是成功终态。
- `failed` 是失败终态。
- `cancelled` 是取消终态，MVP 不强制。
- 终态后不应继续输出正常内容。

---

## 10. code Artifact

MVP v0.1 只强制支持：

```text
type = code
```

推荐结构：

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

字段说明：

| 字段 | 必填 | 说明 |
|---|---:|---|
| `type` | 是 | 固定为 `code` |
| `title` | 是 | 推荐作为文件名 |
| `content` | 是 | 代码文本 |
| `metadata.language` | 是 | 代码语言 |

规则：

- `title` 可作为 `code_preview.filename`。
- `content` 可作为 `code_preview.code`。
- `metadata.language` 可作为 `code_preview.language`。
- 大代码应作为 Artifact，不应只塞进 text chunk。
- Agent 不直接决定前端组件，只声明 Artifact type 和内容。

---

## 11. code Artifact → code_preview

MVP v0.1 映射：

```text
A2A Artifact:
{
  type: "code",
  title: "main.go",
  content: "...",
  metadata: { language: "go" }
}

→ Orchestrator / ProtocolConverter

AG-UI Tool Call:
{
  toolName: "code_preview",
  args: {
    code: artifact.content,
    language: artifact.metadata.language,
    filename: artifact.title
  }
}
```

规则：

- 映射由 Orchestrator / ProtocolConverter 完成。
- Child Agent 不直接输出 AG-UI Tool Call。
- Child Agent 不直接调用 Frontend Skill。
- `code_preview` 参数最终由 `frontend-runtime-skills-contract` 细化。

---

## 12. Orchestrator 调用规则

Orchestrator 调用 A2A 时必须：

- 通过 Agent 配置或 Agent Registry 找到 Agent URL。
- 读取或验证 AgentCard。
- 确认 Agent 支持目标 `outputModes`。
- 通过 A2A Client 调用 `/a2a/tasks/sendSubscribe`。
- 将历史上下文转换为 A2A messages。
- 传递 `runId`、`threadId`、`traceId`。
- 读取 A2A stream event。
- 将 A2A event 交给 ProtocolConverter。
- 不直接把 A2A event 传给 Frontend。
- 不让 Gateway handler 参与 A2A 解析。

MVP v0.1 可以直接调用 `code-agent`，但仍应保留 A2A Client 边界。

---

## 13. Mock-first 规则

Mock A2A Agent 必须：

- 暴露合法 AgentCard。
- 暴露 `/a2a/tasks/sendSubscribe`。
- 输出合法 A2A stream event。
- 能模拟 `status: working`。
- 能模拟多个 `text`。
- 能模拟 `code` Artifact。
- 能模拟 `status: completed`。
- 能模拟 `status: failed`。
- 不输出未定义事件。
- 不直接输出 AG-UI Event。
- 不绕过 Orchestrator。

推荐 mock 输出：

```text
status: working
text: "下面是 Go HTTP Server 示例："
text: "```go\npackage main..."
artifact: {type:"code", title:"main.go", content:"...", metadata:{language:"go"}}
status: completed
```

---

## 14. Contract Test Checklist

- [ ] `/a2a/tasks/sendSubscribe` 是否存在？
- [ ] request 是否包含 task id？
- [ ] request 是否包含 messages？
- [ ] metadata 是否包含 `runId`？
- [ ] metadata 是否包含 `threadId`？
- [ ] metadata 是否包含 `traceId`？
- [ ] response 是否是 streaming？
- [ ] 是否输出 `status: working`？
- [ ] 是否输出 `text`？
- [ ] 是否能输出 `artifact`？
- [ ] 是否输出 `status: completed` 或 `status: failed`？
- [ ] completed 后是否不再输出 text / artifact？
- [ ] failed 后是否不再输出正常事件？
- [ ] `code` Artifact 是否包含 `title`？
- [ ] `code` Artifact 是否包含 `content`？
- [ ] `code` Artifact 是否包含 `metadata.language`？
- [ ] A2A event 是否没有直接暴露给 Frontend？
- [ ] Gateway handler 是否没有直接调用 A2A endpoint？
- [ ] Orchestrator 是否通过 A2A Client 调用？


## 15. v1.1 对齐补充

### 15.1 MVP 最小 endpoint 与完整 endpoint

MVP 最小必需：

```text
GET  /.well-known/agent.json
POST /a2a/tasks/sendSubscribe
```

Post-MVP 完整范围：

```text
GET    /.well-known/agent.json
POST   /a2a/tasks/send
POST   /a2a/tasks/sendSubscribe
GET    /a2a/tasks/:id
DELETE /a2a/tasks/:id/cancel
GET    /health
```

### 15.2 cancel 兼容

v1.1 推荐 `DELETE /a2a/tasks/:id/cancel`；`POST /a2a/tasks/{id}/cancel` 可作为兼容路径保留。

### 15.3 /health 说明

`/health` 用于 Agent 健康检查与探活；MVP 可返回最小 healthy 状态，Post-MVP 持久化由后续 `data-persistence-contract` 细化。

### 15.4 安全补充

A2A metadata 不得携带用户 token；A2A 错误不得泄漏 token、API key、stack trace、内部地址、完整 system prompt。

### 15.5 细化边界

`code_preview` 参数与 ToolResult 由后续 `frontend-runtime-skills-contract` 细化。

