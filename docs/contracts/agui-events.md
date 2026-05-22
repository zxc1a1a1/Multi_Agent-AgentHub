# AG-UI Event Contract

## 1. 文档目的

本文档定义 AgentHub 中 Frontend ↔ Gateway 的 AG-UI Event Contract。

AG-UI Event Contract 的目标是固定以下内容：

- `/api/agui/run` 的 SSE 事件流格式。
- Gateway 向 Frontend 输出的实时事件类型。
- 文本流式回复事件。
- Tool Call 事件。
- Error 事件。
- State Update 事件。
- A2A → AG-UI 的协议转换规则。
- Artifact → Frontend Skill 的映射规则。
- MVP v0.1 中 `code-agent → code Artifact → code_preview` 的最小闭环。

一句话：

**AG-UI 只负责 Frontend ↔ Gateway 的实时 Agent/UI 事件，不负责 REST 资源查询，也不负责 A2A Agent 间通信。**

---

## 2. 协议边界

### 2.1 本 Contract 负责

本 Contract 只负责：

```text
Frontend ↔ Gateway
```

具体包括：

- `POST /api/agui/run` 返回的 SSE event stream。
- Frontend AG-UI Client 如何消费事件。
- Gateway 如何输出事件。
- Orchestrator 如何把 A2A stream event 转换为 AG-UI event。
- Artifact 如何转换为 Tool Call。
- Frontend Skill 如何根据 Tool Call 参数渲染产物。

---

### 2.2 本 Contract 不负责

本 Contract 不负责：

- REST API response schema。
- `docs/contracts/openapi.yaml` 的资源 API 字段。
- Conversation / Message / Agent / Artifact 的普通查询接口。
- Gateway ↔ Orchestrator 的内部调用 Contract。
- Orchestrator ↔ Child Agent 的 A2A Task Contract。
- Child Agent 的 AgentCard 完整结构。
- ADK Runtime 内部生命周期。
- Artifact 的完整持久化 schema。
- Frontend Runtime Skills 的完整参数 schema。

对应职责归属：

| 内容 | 所属 Contract |
|---|---|
| REST API / OpenAPI | `platform-api-contract` |
| Gateway ↔ Orchestrator | `gateway-orchestrator-contract` |
| Orchestrator ↔ Child Agent | `a2a-agent-contract` |
| Artifact 结构 | `artifact-contract` |
| Frontend Skill 参数 | `frontend-runtime-skills-contract` |
| ADK Runtime | `adk-runtime-contract` |

---

## 3. MVP v0.1 事件范围

MVP v0.1 的目标是跑通：

```text
用户发消息
→ Gateway
→ Orchestrator
→ code-agent
→ A2A text / artifact
→ AG-UI text / tool call
→ code_preview 代码预览
```

MVP v0.1 必须支持以下事件：

```text
RUN_STARTED
TEXT_MESSAGE_START
TEXT_MESSAGE_CONTENT
TEXT_MESSAGE_END
TOOL_CALL_START
TOOL_CALL_ARGS
TOOL_CALL_END
RUN_FINISHED
RUN_ERROR
```

MVP v0.1 可选支持：

```text
STATE_UPDATE
```

`STATE_UPDATE` 在 MVP 中可以只保留 schema，不强制复杂实现。后续 v0.2 / v0.3 可用于展示意图编排状态、当前活跃 Agent、多 Agent 执行阶段等。

---

## 4. 完整事件范围

当前 Contract 定义以下事件：

| 事件 | 是否 MVP 必须 | 说明 |
|---|---:|---|
| `RUN_STARTED` | 是 | Run 开始 |
| `RUN_FINISHED` | 是 | Run 正常完成 |
| `RUN_ERROR` | 是 | Run 失败 |
| `TEXT_MESSAGE_START` | 是 | Agent 文本消息开始 |
| `TEXT_MESSAGE_CONTENT` | 是 | Agent 文本 chunk |
| `TEXT_MESSAGE_END` | 是 | Agent 文本消息结束 |
| `TOOL_CALL_START` | 是，存在 Artifact 时 | Frontend Skill 调用开始 |
| `TOOL_CALL_ARGS` | 是，存在 Artifact 时 | Frontend Skill 参数片段 |
| `TOOL_CALL_END` | 是，存在 Artifact 时 | Frontend Skill 调用结束 |
| `STATE_UPDATE` | 否 | 运行状态更新 |

---

## 5. SSE 传输格式

AG-UI 事件通过 SSE 返回。

### 5.1 基本格式

```text
event: message
data: {"type":"RUN_STARTED","runId":"run-xxx"}
```

每条事件必须以 JSON 对象形式出现在 `data:` 中。

推荐 Gateway 输出格式：

```text
event: message
data: {event JSON}

```

注意：

- 每个事件必须包含 `type`。
- 所有字段必须使用 camelCase。
- 不允许输出 snake_case API 字段。
- 不允许把多个事件拼成一个 JSON 数组输出。
- 不允许把 A2A 原始事件直接输出给 Frontend。
- 不允许把大文件或大产物塞进 `TEXT_MESSAGE_CONTENT`。

---

### 5.2 事件字段关联

事件可通过以下字段关联上下文：

| 字段 | 用途 |
|---|---|
| `runId` | 一次 AG-UI Run |
| `threadId` | 对话 / 会话线程 |
| `messageId` | 一条 Agent 消息 |
| `toolCallId` | 一次 Frontend Skill 调用 |
| `toolName` | Frontend Skill 名称 |

规则：

- `RUN_STARTED` 和 `RUN_FINISHED` 必须包含 `runId`。
- `TEXT_MESSAGE_*` 必须通过 `messageId` 关联。
- `TOOL_CALL_*` 必须通过 `toolCallId` 关联。
- `TOOL_CALL_START` 可以携带 `messageId`，用于绑定到对应 Agent 消息下方。
- `TOOL_CALL_ARGS` 可以分片发送，Frontend 必须按 `toolCallId` 聚合。

---

## 6. 事件生命周期

MVP v0.1 成功链路：

```text
RUN_STARTED
TEXT_MESSAGE_START
TEXT_MESSAGE_CONTENT*
TEXT_MESSAGE_END
TOOL_CALL_START?
TOOL_CALL_ARGS*
TOOL_CALL_END?
RUN_FINISHED
```

含义：

- `TEXT_MESSAGE_CONTENT*` 表示可出现 0 到多次。
- 如果没有 Artifact，可以没有 `TOOL_CALL_*`。
- 如果有 `code` Artifact，必须输出 `code_preview` 的 `TOOL_CALL_*`。
- 如果运行失败，应输出 `RUN_ERROR`。
- `RUN_ERROR` 后不应继续输出普通完成事件。

---

## 7. Run 生命周期事件

### 7.1 RUN_STARTED

表示一次 Run 开始。

示例：

```json
{
  "type": "RUN_STARTED",
  "runId": "run-001",
  "threadId": "conv-001",
  "timestamp": "2026-05-22T10:00:00Z"
}
```

要求：

- `type` 固定为 `RUN_STARTED`。
- `runId` 必填。
- `threadId` 可选。
- `timestamp` 可选，推荐 RFC3339 格式。

---

### 7.2 RUN_FINISHED

表示一次 Run 正常完成。

示例：

```json
{
  "type": "RUN_FINISHED",
  "runId": "run-001",
  "timestamp": "2026-05-22T10:00:30Z"
}
```

要求：

- `type` 固定为 `RUN_FINISHED`。
- `runId` 必填。
- 应在文本结束和 Tool Call 结束后输出。
- 不应在 `TOOL_CALL_END` 之前输出。

---

### 7.3 RUN_ERROR

表示一次 Run 失败。

示例：

```json
{
  "type": "RUN_ERROR",
  "runId": "run-001",
  "code": "A2A_STREAM_ERROR",
  "message": "code-agent stream failed",
  "error": "connection reset by peer"
}
```

要求：

- `type` 固定为 `RUN_ERROR`。
- `runId` 可选，但推荐携带。
- `code` 可选，用于机器可读错误码。
- `message` 可选，用于用户或开发者可读错误说明。
- `error` 可选，用于调试错误。
- 不允许泄漏密钥、Token、LLM API Key、数据库连接串。

---

## 8. Text Message 事件

### 8.1 TEXT_MESSAGE_START

表示 Agent 文本消息开始。

示例：

```json
{
  "type": "TEXT_MESSAGE_START",
  "messageId": "msg-001",
  "role": "agent",
  "agentName": "code-agent"
}
```

要求：

- `type` 固定为 `TEXT_MESSAGE_START`。
- `messageId` 必填。
- `role` 可选，MVP 推荐为 `agent`。
- `agentName` 可选，MVP 推荐为 `code-agent`。
- Frontend 收到后应创建空的 Agent 消息气泡。

---

### 8.2 TEXT_MESSAGE_CONTENT

表示 Agent 文本 chunk。

示例：

```json
{
  "type": "TEXT_MESSAGE_CONTENT",
  "messageId": "msg-001",
  "content": "package main\n"
}
```

要求：

- `type` 固定为 `TEXT_MESSAGE_CONTENT`。
- `messageId` 必填。
- `content` 必填。
- `content` 只能是文本 chunk。
- 不允许把 Artifact、文件、代码预览参数、大型 JSON 塞进 `content`。
- 前端应按 `messageId` 追加到当前流式消息。

---

### 8.3 TEXT_MESSAGE_END

表示 Agent 文本消息结束。

示例：

```json
{
  "type": "TEXT_MESSAGE_END",
  "messageId": "msg-001"
}
```

要求：

- `type` 固定为 `TEXT_MESSAGE_END`。
- `messageId` 必填。
- Frontend 收到后应将流式消息标记为完成。
- Markdown 解析建议在结束后执行，避免每个 chunk 都解析导致闪烁或性能问题。

---

## 9. Tool Call 事件

Tool Call 用于触发 Frontend Skill。

MVP v0.1 中，最重要的 Tool Call 是：

```text
code Artifact → code_preview
```

Tool Call 必须按顺序输出：

```text
TOOL_CALL_START
TOOL_CALL_ARGS*
TOOL_CALL_END
```

---

### 9.1 TOOL_CALL_START

表示一次 Frontend Skill 调用开始。

示例：

```json
{
  "type": "TOOL_CALL_START",
  "toolCallId": "tc-001",
  "toolName": "code_preview",
  "messageId": "msg-001"
}
```

要求：

- `type` 固定为 `TOOL_CALL_START`。
- `toolCallId` 必填。
- `toolName` 必填。
- `messageId` 可选，推荐携带。
- `toolName` 必须是 Frontend 已注册的 Skill。
- MVP v0.1 中 `code` Artifact 必须映射为 `code_preview`。

---

### 9.2 TOOL_CALL_ARGS

表示 Tool Call 参数片段。

示例：

```json
{
  "type": "TOOL_CALL_ARGS",
  "toolCallId": "tc-001",
  "content": "{\"code\":\"package main\\n\",\"language\":\"go\",\"filename\":\"main.go\"}"
}
```

要求：

- `type` 固定为 `TOOL_CALL_ARGS`。
- `toolCallId` 必填。
- `content` 必填。
- `content` 是 JSON 字符串片段。
- 允许分片发送。
- Frontend 必须按 `toolCallId` 聚合所有 `content`，在 `TOOL_CALL_END` 时再解析完整 JSON。
- 不允许在 `TEXT_MESSAGE_CONTENT` 中传 Tool 参数。

---

### 9.3 TOOL_CALL_END

表示 Tool Call 参数传输结束。

示例：

```json
{
  "type": "TOOL_CALL_END",
  "toolCallId": "tc-001"
}
```

要求：

- `type` 固定为 `TOOL_CALL_END`。
- `toolCallId` 必填。
- Frontend 收到后应解析聚合后的 args JSON。
- 如果 `toolName = code_preview`，args 必须符合 `CodePreviewArgs`。

---

## 10. State Update 事件

### 10.1 STATE_UPDATE

表示运行过程中的状态更新。

MVP v0.1 可以保留 schema，不强制复杂实现。

示例：

```json
{
  "type": "STATE_UPDATE",
  "state": {
    "phase": "orchestrating",
    "activeAgent": "code-agent",
    "message": "正在调用 code-agent"
  }
}
```

要求：

- `type` 固定为 `STATE_UPDATE`。
- `state` 必填。
- `state` 必须是 JSON object。
- 不得包含敏感信息。
- 后续多 Agent 场景可以用于展示编排计划、当前 Agent、阶段进度等。

---

## 11. Error 事件

AG-UI 错误统一使用 `RUN_ERROR`。

常见场景：

| 场景 | 建议 code |
|---|---|
| AG-UI request 参数错误 | `AGUI_BAD_REQUEST` |
| 鉴权失败 | `AGUI_UNAUTHORIZED` |
| Orchestrator 调用失败 | `ORCHESTRATOR_ERROR` |
| A2A 连接失败 | `A2A_CONNECT_ERROR` |
| A2A 流式响应失败 | `A2A_STREAM_ERROR` |
| Child Agent 失败 | `AGENT_ERROR` |
| LLM 调用失败 | `LLM_ERROR` |
| Tool Call 参数构造失败 | `TOOL_ARGS_ERROR` |
| 未知错误 | `UNKNOWN_ERROR` |

规则：

- 发生错误时必须输出 `RUN_ERROR`，除非 HTTP 层在 SSE 建立前已经失败。
- SSE 已经建立后，不应再只依赖 HTTP status 表达错误。
- `RUN_ERROR` 后不应继续输出 `RUN_FINISHED`。
- 错误事件不得泄漏敏感配置。

---

## 12. A2A → AG-UI 映射规则

MVP v0.1 的核心协议转换：

```text
A2A status:working
→ AG-UI TEXT_MESSAGE_START

A2A text
→ AG-UI TEXT_MESSAGE_CONTENT

A2A artifact
→ 缓存 Artifact，不立即输出

A2A status:completed
→ AG-UI TEXT_MESSAGE_END
→ flush artifactBuffer
→ TOOL_CALL_START / TOOL_CALL_ARGS / TOOL_CALL_END
→ RUN_FINISHED

A2A status:failed
→ RUN_ERROR
```

映射表：

| A2A 输入事件 | AG-UI 输出事件 | 说明 |
|---|---|---|
| `status: working` | `TEXT_MESSAGE_START` | 创建 Agent 消息 |
| `text` | `TEXT_MESSAGE_CONTENT` | 追加文本 chunk |
| `artifact` | 不立即输出 | 存入 `artifactBuffer` |
| `status: completed` | `TEXT_MESSAGE_END` + Tool Call + `RUN_FINISHED` | 文本结束后批量处理产物 |
| `status: failed` | `RUN_ERROR` | 失败终止 |

硬性要求：

- A2A 原始事件不得直接暴露给 Frontend。
- Artifact 必须缓存，不能在收到时直接塞入文本流。
- `RUN_FINISHED` 必须在 Tool Call 完成后输出。
- 如果 A2A 没有输出 `status: working`，Orchestrator 可以在收到首个 text 前补发 `TEXT_MESSAGE_START`，但必须保证事件顺序一致。

---

## 13. Artifact → Frontend Skill 映射规则

Artifact 必须映射为 Frontend Skill 的 Tool Call。

推荐映射：

| Artifact type | Frontend Skill |
|---|---|
| `code` | `code_preview` |
| `webpage` | `web_preview` |
| `file` | `file_download` |
| `image` | `image_preview` |
| `document` | `markdown_render` |

MVP v0.1 只强制实现：

```text
code → code_preview
```

规则：

- 未注册的 Skill 不应触发 Tool Call。
- 未注册 Skill 的 Artifact 可以被忽略、记录日志，或在后续 Artifact API 中展示。
- Tool Call 参数必须由 Artifact 构造。
- 大型 Artifact 应通过 Artifact Contract / fileUrl 处理，不应放入文本 chunk。

---

## 14. code Artifact → code_preview 参数规范

MVP v0.1 中，`code` Artifact 必须转换为 `code_preview` Tool Call。

### 14.1 输入 Artifact

示例：

```json
{
  "type": "code",
  "title": "main.go",
  "content": "package main\n",
  "metadata": {
    "language": "go"
  }
}
```

### 14.2 Tool Call

`TOOL_CALL_START`：

```json
{
  "type": "TOOL_CALL_START",
  "toolCallId": "tc-001",
  "toolName": "code_preview",
  "messageId": "msg-001"
}
```

`TOOL_CALL_ARGS` 的完整参数：

```json
{
  "code": "package main\n",
  "language": "go",
  "filename": "main.go"
}
```

`TOOL_CALL_ARGS` 事件中 `content` 应为上述 JSON 的字符串形式。

`TOOL_CALL_END`：

```json
{
  "type": "TOOL_CALL_END",
  "toolCallId": "tc-001"
}
```

字段要求：

| 字段 | 类型 | 必填 | 来源 |
|---|---|---:|---|
| `code` | string | 是 | `artifact.content` |
| `language` | string | 是 | `artifact.metadata.language` |
| `filename` | string | 是 | `artifact.title` |

兜底规则：

- 如果 `language` 缺失，可用 `text` 或 `plain` 兜底。
- 如果 `filename` 缺失，可用 `snippet.txt` 兜底。
- 如果 `code` 为空，不应触发 `code_preview`。

---

## 15. 事件顺序约束

### 15.1 正常文本 + 代码产物

```text
RUN_STARTED
TEXT_MESSAGE_START
TEXT_MESSAGE_CONTENT
TEXT_MESSAGE_CONTENT
TEXT_MESSAGE_END
TOOL_CALL_START
TOOL_CALL_ARGS
TOOL_CALL_END
RUN_FINISHED
```

### 15.2 正常文本，无产物

```text
RUN_STARTED
TEXT_MESSAGE_START
TEXT_MESSAGE_CONTENT
TEXT_MESSAGE_END
RUN_FINISHED
```

### 15.3 失败

```text
RUN_STARTED
TEXT_MESSAGE_START?
TEXT_MESSAGE_CONTENT*
RUN_ERROR
```

约束：

- `TEXT_MESSAGE_CONTENT` 不应出现在对应 `TEXT_MESSAGE_START` 之前。
- `TEXT_MESSAGE_END` 不应出现在对应 `TEXT_MESSAGE_START` 之前。
- `TOOL_CALL_ARGS` 不应出现在对应 `TOOL_CALL_START` 之前。
- `TOOL_CALL_END` 不应出现在对应 `TOOL_CALL_START` 之前。
- `RUN_FINISHED` 不应出现在未完成的 Tool Call 之前。
- `RUN_ERROR` 后不应再输出 `RUN_FINISHED`。

---

## 16. 前端处理规则

Frontend AG-UI Client 必须：

- 通过 SSE 读取 `data:` 行。
- 解析 JSON event。
- 根据 `event.type` 分发。
- 使用 `messageId` 聚合文本消息。
- 使用 `toolCallId` 聚合 Tool Call args。
- 在 `TEXT_MESSAGE_START` 时创建空 Agent 消息。
- 在 `TEXT_MESSAGE_CONTENT` 时追加文本。
- 在 `TEXT_MESSAGE_END` 时结束流式状态。
- 在 `TOOL_CALL_END` 时解析完整 args JSON。
- 在 `toolName = code_preview` 时渲染代码预览。
- 在 `RUN_FINISHED` 时关闭 loading。
- 在 `RUN_ERROR` 时展示错误状态。

禁止：

- 把 A2A 原始事件当 AG-UI event 处理。
- 在 `TOOL_CALL_ARGS` 分片未结束时解析 JSON。
- 把 `TEXT_MESSAGE_CONTENT` 当 Artifact 容器。
- 手写与 schema 冲突的事件类型。

---

## 17. Gateway 输出规则

Gateway 必须：

- 接收 `/api/agui/run`。
- 建立 SSE 响应。
- 设置 `Content-Type: text/event-stream`。
- 禁用缓存。
- 将 Orchestrator 产生的 AG-UI event 序列化为 JSON。
- 每个事件单独 flush。
- 不输出 A2A 原始事件。
- 不在 Gateway Handler 中写复杂协议转换逻辑。
- 不在 SSE 文本 chunk 中塞大产物。

推荐响应头：

```text
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
X-Accel-Buffering: no
```

---

## 18. Orchestrator 转换规则

Orchestrator / ProtocolConverter 必须：

- 接收 A2A stream event。
- 维护当前 `messageId`。
- 维护 `artifactBuffer`。
- 将 A2A `text` 转为 `TEXT_MESSAGE_CONTENT`。
- 将 A2A `artifact` 缓存。
- 在 A2A `completed` 时输出 `TEXT_MESSAGE_END`。
- 在 `TEXT_MESSAGE_END` 后 flush artifacts。
- 将 `code` Artifact 转为 `code_preview` Tool Call。
- 最后输出 `RUN_FINISHED`。
- 失败时输出 `RUN_ERROR`。
- 不把 A2A 原始结构暴露给 Frontend。

---

## 19. 禁止事项

禁止：

- 把 AG-UI 事件写成普通 REST response schema。
- 把 A2A endpoint 暴露给 Frontend。
- 把 Gateway ↔ Orchestrator 内部协议写进 AG-UI Event Contract。
- 把大型代码、文件、网页、图片塞进 `TEXT_MESSAGE_CONTENT`。
- 把 Artifact 直接拼成 Markdown 文本当作最终预览。
- `TOOL_CALL_ARGS` 不带 `toolCallId`。
- `TEXT_MESSAGE_CONTENT` 不带 `messageId`。
- `RUN_FINISHED` 早于 Tool Call 完成。
- `RUN_ERROR` 后继续输出 `RUN_FINISHED`。
- 事件字段混用 snake_case。
- 前端在 `TOOL_CALL_END` 前解析未完整的 args JSON。
- Gateway Handler 直接承担全部协议转换职责。

---

## 20. Review Checklist

- [ ] 是否覆盖 MVP 必须事件链？
- [ ] 是否定义了完整事件范围？
- [ ] 是否明确 AG-UI 只负责 Frontend ↔ Gateway 实时事件？
- [ ] 是否没有把 AG-UI 事件写成 REST response？
- [ ] 是否没有把 A2A event 暴露给 Frontend？
- [ ] 是否没有把 Gateway-Orchestrator 内部协议混入 AG-UI？
- [ ] 是否没有把大产物塞进 `TEXT_MESSAGE_CONTENT`？
- [ ] 是否 `TEXT_MESSAGE_CONTENT` 只传文本 chunk？
- [ ] 是否 `code` Artifact 映射到了 `code_preview`？
- [ ] 是否 `code_preview` args 包含 `code / language / filename`？
- [ ] 是否 `TOOL_CALL_ARGS` 可以按 `toolCallId` 聚合？
- [ ] 是否所有事件字段使用 camelCase？
- [ ] 是否所有事件都能被 `agui-events.schema.json` 校验？
- [ ] 是否 `RUN_ERROR` 能覆盖失败场景？
- [ ] 是否 `RUN_FINISHED` 在 Tool Call 完成后输出？
- [ ] 是否符合 MVP 文档中的 `/api/agui/run` SSE 事件流？
- [ ] 是否符合 UML 中的 A2A → AG-UI 协议转换时序？


## 21. v1.1 对齐补充

### 21.1 通用映射与 MVP 映射兼容

- v1.1 通用运行级映射：`A2A Task.status.started -> RUN_STARTED`。
- MVP v0.1 消息级映射：`A2A status:working -> TEXT_MESSAGE_START`。
- 两种映射可以共存，不能互相替代。

### 21.2 RUN_ERROR 脱敏

`RUN_ERROR` 不得泄漏 stack trace、token、API key、内部服务地址、数据库连接串、完整 system prompt。

### 21.3 Artifact / Tool Call 细化边界

- 大 Artifact 不得进入 `TEXT_MESSAGE_CONTENT`。
- Artifact 结构由后续 `artifact-contract` 细化。
- Tool Call 参数与 ToolResult 由后续 `frontend-runtime-skills-contract` 细化。

