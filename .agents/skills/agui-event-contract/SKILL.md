---
name: agui-event-contract
description: Use when changing AG-UI events, SSE streaming behavior, run lifecycle events, text streaming, tool calls, state updates, or frontend event reducers.
---

# agui-event-contract

## 1. Skill 目的

本 Skill 用于定义 AgentHub 项目的 AG-UI 事件契约。

`agui-event-contract` 的核心职责是固定 Frontend 与 Gateway 之间的实时交互事件格式，包括：

- AG-UI Run 的 SSE 事件流。
- 用户发送消息后，Agent 流式回复的事件序列。
- A2A text stream 到 AG-UI text event 的转换规则。
- A2A Artifact 到 AG-UI Tool Call 的转换规则。
- Frontend Runtime Skills 的触发事件格式。
- Tool Call 参数拼接、结束、执行结果回传规则。
- Run 生命周期事件。
- Message 生命周期事件。
- Error / Cancel / State Update 事件边界。
- MVP v0.1 必须实现的最小事件集。
- Post-MVP 扩展事件集。

一句话：

**凡是 Frontend 与 Gateway 之间通过 AG-UI / SSE 传输的实时事件，都必须由本 Skill 约束。**

---

## 2. 适用场景

当任务涉及以下内容时，必须使用本 Skill：

- 设计或修改 AG-UI event schema。
- 编写或修改 `docs/contracts/agui-events.md`。
- 编写或修改 `docs/contracts/agui-events.schema.json`。
- 设计 `POST /api/agui/run` 的 SSE event stream。
- 设计 Frontend AG-UI Client 的事件处理逻辑。
- 设计 Gateway AG-UI Server 的 SSE 输出逻辑。
- 设计 Orchestrator 到 Gateway 的 AG-UI event channel。
- 设计 A2A event 到 AG-UI event 的协议转换。
- 设计 Artifact 到 Frontend Skill Tool Call 的转换。
- 设计 `code_preview` 的 Tool Call 事件链路。
- 处理 `TEXT_MESSAGE_CONTENT` 流式文本事件。
- 处理 `TOOL_CALL_START / TOOL_CALL_ARGS / TOOL_CALL_END`。
- 判断某个字段应该属于 AG-UI event、REST API、A2A Task、Artifact Contract，还是 Frontend Runtime Skills Contract。

---

## 3. Contract 所属边界

本 Skill 只约束：

```text
Frontend ↔ Gateway
AG-UI realtime event stream
```

本 Skill 不约束：

```text
Frontend ↔ Gateway 的普通 REST API response schema
Gateway ↔ Orchestrator 的内部 Run Contract
Orchestrator ↔ Child Agent 的 A2A Task / A2A StreamEvent
Child Agent 的 AgentCard
Artifact 的持久化完整 schema
Frontend Runtime Skills 的完整参数 schema
```

对应关系如下：

| 内容 | 负责 Skill / Contract |
|---|---|
| REST API request / response | `platform-api-contract` |
| AG-UI SSE event | `agui-event-contract` |
| Gateway ↔ Orchestrator 内部接口 | `gateway-orchestrator-contract` |
| A2A Task / A2A AgentCard | `a2a-agent-contract` |
| Artifact 完整格式 | `artifact-contract` |
| Frontend Skill 注册和参数 | `frontend-runtime-skills-contract` |
| 意图编排 ExecutionPlan | `intent-orchestration-contract` |

---

## 4. 核心文件

本 Skill 需要维护的核心 Contract 文件：

```text
docs/contracts/agui-events.md
docs/contracts/agui-events.schema.json
```

建议后续补充的辅助文件：

```text
docs/contracts/agui-run-request.md
docs/contracts/agui-tool-result.md
docs/contracts/agui-event-review-checklist.md
```

但第一阶段最重要的是：

```text
docs/contracts/agui-events.md
docs/contracts/agui-events.schema.json
```

---

## 5. 与 PDR / MVP / UML 的关系

本 Skill 同时约束两个层次：

1. **完整目标层**：以 PDR 为准，AG-UI 支持单聊、群聊、多 Agent 协作、状态更新、Tool Call、交互式 Skill 和产物预览。
2. **MVP v0.1 实施层**：以 MVP 文档为准，优先跑通 `code-agent + code_preview` 的最小 AG-UI 事件闭环。

UML 文档中的时序图是 AG-UI 事件顺序、协议转换和前端渲染逻辑的重要参考。

MVP 可以裁剪事件范围，但不能破坏以下方向：

- Frontend 仍然只通过 Gateway 接收 AG-UI 事件。
- Gateway 仍然负责向 Frontend 输出 SSE event stream。
- Orchestrator 仍然负责把 A2A 事件转换成 AG-UI 事件。
- Artifact 仍然不能直接塞进普通文本流。
- Artifact 必须通过 AG-UI Tool Call 触发 Frontend Skill。
- `code` Artifact 必须映射到 `code_preview`。

---

## 6. MVP v0.1 必须实现的事件集

MVP v0.1 只强制实现最小事件链路：

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

这些事件必须支持以下 Demo 闭环：

```text
用户发送消息
→ Gateway 返回 AG-UI SSE
→ RUN_STARTED
→ TEXT_MESSAGE_START
→ 多个 TEXT_MESSAGE_CONTENT
→ TEXT_MESSAGE_END
→ TOOL_CALL_START(code_preview)
→ TOOL_CALL_ARGS({code, language, filename})
→ TOOL_CALL_END
→ RUN_FINISHED
```

MVP v0.1 中，`STATE_UPDATE`、`RUN_CANCELLED`、复杂 ToolResult、交互式 Skill 可以作为 Post-MVP 规划，不作为当前必做项。

---

## 7. Post-MVP 规划事件集

Post-MVP 可以扩展以下事件：

```text
STATE_UPDATE
RUN_CANCELLED
RUN_WARNING
RUN_PROGRESS
TOOL_CALL_RESULT
TOOL_CALL_ERROR
USER_INPUT_REQUIRED
FRONTEND_SKILL_RESULT
AGENT_SWITCHED
EXECUTION_PLAN_CREATED
SUBTASK_STARTED
SUBTASK_FINISHED
```

这些事件用于：

- 群聊多 Agent 协作。
- 多 Agent 并行 / 串行执行。
- 显示当前活跃 Agent。
- 展示 ExecutionPlan。
- 展示编排阶段状态。
- 交互式 Skill，例如 `confirm_action`、`form_input`、`file_upload`。
- 复杂错误恢复与降级。

Post-MVP 事件必须向后兼容 MVP v0.1 的事件集。

---

## 8. AG-UI SSE Wire Format

Gateway 向 Frontend 输出 AG-UI 事件时，MVP v0.1 统一使用 SSE。

SSE 基础格式：

```text
event: message
data: {JSON}

```

示例：

```text
event: message
data: {"type":"TEXT_MESSAGE_CONTENT","messageId":"msg-1","content":"package main"}

```

要求：

- 每个事件必须是一条合法 JSON。
- 每个事件必须包含 `type` 字段。
- `event` 名称 MVP 阶段统一使用 `message`。
- `data` 中不得包含非 JSON 字符串。
- 每次写入 SSE event 后必须 flush。
- Gateway 不得把多个事件合并成一个 JSON array 输出。
- Frontend AG-UI Client 必须支持 SSE 粘包 / 拆包处理。
- `TOOL_CALL_ARGS.content` 允许分块，但 MVP 可以一次性输出完整 JSON 字符串。

---

## 9. 通用事件字段

所有 AG-UI event 都应遵守以下通用字段约定。

| 字段 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| type | string | 是 | 事件类型 |
| runId | string | 视事件而定 | 当前 Run ID |
| threadId | string | 视事件而定 | 当前对话 / thread ID |
| messageId | string | 视事件而定 | 当前消息 ID |
| toolCallId | string | 视事件而定 | Tool Call ID |
| timestamp | string | 建议 | RFC3339 时间 |
| traceId | string | 建议 | 链路追踪 ID |

命名规则：

- JSON 字段统一使用 camelCase。
- 事件 `type` 使用大写下划线格式，例如 `TEXT_MESSAGE_CONTENT`。
- ID 字段使用字符串。
- 时间字段使用 RFC3339 / ISO 8601。
- 不允许在 AG-UI event 中混用 snake_case。

---

## 10. Run 生命周期事件

### 10.1 RUN_STARTED

表示 Gateway 已接受本次 Run，并开始处理。

最小字段：

```json
{
  "type": "RUN_STARTED",
  "runId": "run-uuid",
  "threadId": "conversation-uuid"
}
```

规则：

- 每个 Run 必须最多发送一次 `RUN_STARTED`。
- `RUN_STARTED` 必须在任何文本消息事件之前发送。
- 如果鉴权失败或请求格式错误，不应进入 SSE 流，而应由 REST / HTTP 错误返回处理。
- 如果 Run 已启动后发生业务错误，应通过 `RUN_ERROR` 返回。

---

### 10.2 RUN_FINISHED

表示本次 Run 正常结束。

最小字段：

```json
{
  "type": "RUN_FINISHED",
  "runId": "run-uuid"
}
```

规则：

- 每个正常完成的 Run 必须发送一次 `RUN_FINISHED`。
- `RUN_FINISHED` 必须在所有 `TEXT_MESSAGE_*` 和 `TOOL_CALL_*` 事件之后发送。
- Frontend 收到 `RUN_FINISHED` 后可以关闭 SSE 读取状态。
- Gateway 在发送 `RUN_FINISHED` 后不应再发送新的业务事件。

---

### 10.3 RUN_ERROR

表示 Run 已经开始，但中途发生错误。

最小字段：

```json
{
  "type": "RUN_ERROR",
  "runId": "run-uuid",
  "error": {
    "code": "AGUI_RUN_FAILED",
    "message": "Code-Agent 调用失败"
  }
}
```

规则：

- `RUN_ERROR` 用于 SSE 已建立后的运行时错误。
- 错误结构必须稳定。
- 不允许在 `message` 中泄漏 API Key、数据库连接串、内部堆栈、LLM 原始密钥。
- Frontend 收到后应停止 loading，并将当前 streaming message 标记为 failed。

---

## 11. Text Message 生命周期事件

### 11.1 TEXT_MESSAGE_START

表示开始创建一条 Agent 回复消息。

最小字段：

```json
{
  "type": "TEXT_MESSAGE_START",
  "messageId": "msg-uuid",
  "runId": "run-uuid",
  "threadId": "conversation-uuid",
  "sender": {
    "type": "agent",
    "name": "code-agent"
  }
}
```

规则：

- 每条 Agent 文本回复必须先有 `TEXT_MESSAGE_START`。
- `messageId` 必须在该消息的后续 `TEXT_MESSAGE_CONTENT` 和 `TEXT_MESSAGE_END` 中保持一致。
- MVP v0.1 中，通常一个 Run 只有一条 Agent 回复消息。
- Post-MVP 群聊中，一个 Run 可以产生多条 Agent 消息，每条消息必须有独立 `messageId`。

---

### 11.2 TEXT_MESSAGE_CONTENT

表示 Agent 回复文本的一个流式片段。

最小字段：

```json
{
  "type": "TEXT_MESSAGE_CONTENT",
  "messageId": "msg-uuid",
  "content": "package main"
}
```

规则：

- `content` 只能承载文本片段。
- `content` 可以是任意长度的 token chunk，但不应过大。
- 大型代码文件、网页、图片、二进制、压缩包不得塞进 `TEXT_MESSAGE_CONTENT`。
- 如果 Agent 生成代码块，可以先以文本流方式显示说明，但最终代码产物必须通过 Artifact → Tool Call 触发 `code_preview`。
- Frontend 在 streaming 阶段可以先按纯文本渲染，结束后再做 Markdown 解析。

---

### 11.3 TEXT_MESSAGE_END

表示一条 Agent 文本回复结束。

最小字段：

```json
{
  "type": "TEXT_MESSAGE_END",
  "messageId": "msg-uuid"
}
```

规则：

- 每条 `TEXT_MESSAGE_START` 必须对应一个 `TEXT_MESSAGE_END`，除非 Run 中途 `RUN_ERROR`。
- `TEXT_MESSAGE_END` 必须在该消息所有 `TEXT_MESSAGE_CONTENT` 后发送。
- Artifact 转换出的 Tool Call 应在 `TEXT_MESSAGE_END` 之后发送。
- Frontend 收到后应把 streaming message 转入正式消息列表，并标记为 sent。

---

## 12. Tool Call 生命周期事件

AG-UI Tool Call 用于触发 Frontend Runtime Skills。

MVP v0.1 中，最重要的 Tool Call 是：

```text
code_preview
```

### 12.1 TOOL_CALL_START

表示开始一次前端 Skill 调用。

最小字段：

```json
{
  "type": "TOOL_CALL_START",
  "runId": "run-uuid",
  "toolCallId": "tc-uuid",
  "toolName": "code_preview"
}
```

规则：

- 每次 Tool Call 必须先发送 `TOOL_CALL_START`。
- `toolCallId` 必须在后续 `TOOL_CALL_ARGS` 和 `TOOL_CALL_END` 中保持一致。
- `toolName` 必须是 Frontend 已注册的 Skill 名称。
- MVP v0.1 必须支持 `code_preview`。
- 如果前端没有注册该 Skill，Orchestrator / Converter 不应发送该 Tool Call，或者必须发送可处理的错误事件。

---

### 12.2 TOOL_CALL_ARGS

表示 Tool Call 的参数片段。

最小字段：

```json
{
  "type": "TOOL_CALL_ARGS",
  "toolCallId": "tc-uuid",
  "content": "{\"code\":\"package main\",\"language\":\"go\",\"filename\":\"main.go\"}"
}
```

规则：

- `content` 是字符串，内容应为 JSON 片段或完整 JSON 字符串。
- Frontend 必须按 `toolCallId` 累积所有 `TOOL_CALL_ARGS.content`。
- MVP v0.1 可以一次性发送完整 JSON 字符串。
- Post-MVP 可以分块发送大参数，但最终拼接后必须是合法 JSON。
- `TOOL_CALL_ARGS` 不应携带无法解析的随意文本。
- 大文件不应直接放入 args，应使用 `fileUrl` 或 Artifact 引用。

---

### 12.3 TOOL_CALL_END

表示 Tool Call 参数发送完成，可以执行对应 Frontend Skill。

最小字段：

```json
{
  "type": "TOOL_CALL_END",
  "toolCallId": "tc-uuid"
}
```

规则：

- Frontend 收到 `TOOL_CALL_END` 后，才能解析累计 args 并执行 Skill。
- 如果 args JSON 解析失败，Frontend 应记录错误并显示 fallback，而不是崩溃。
- MVP v0.1 中，`TOOL_CALL_END` 后前端应渲染 `<CodePreview />`。

---

## 13. `code_preview` Tool Call 参数

MVP v0.1 的 `code_preview` 参数必须至少包含：

```json
{
  "code": "package main\n...",
  "language": "go",
  "filename": "main.go"
}
```

字段说明：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| code | string | 是 | 代码内容 |
| language | string | 是 | 代码语言，例如 `go`、`tsx`、`python` |
| filename | string | 建议 | 文件名，例如 `main.go` |

规则：

- `code_preview` 参数 schema 的完整定义后续由 `frontend-runtime-skills-contract` 维护。
- 本 Skill 只规定 AG-UI Tool Call 如何传输参数。
- `code` 可以来源于 A2A Artifact 的 `content`。
- `language` 可以来源于 A2A Artifact 的 `metadata.language`。
- `filename` 可以来源于 A2A Artifact 的 `title`。

---

## 14. A2A → AG-UI 事件映射规则

Orchestrator / ProtocolConverter 必须按照稳定规则把 A2A stream event 转成 AG-UI event。

MVP v0.1 映射表：

| A2A 输入事件 | AG-UI 输出事件 |
|---|---|
| `status: working` | `TEXT_MESSAGE_START` |
| `text: content` | `TEXT_MESSAGE_CONTENT` |
| `artifact: {...}` | 缓存，不立即输出 |
| `status: completed` | `TEXT_MESSAGE_END` + `TOOL_CALL_*` + `RUN_FINISHED` |
| `status: failed` | `RUN_ERROR` |

规则：

- `artifact` 事件必须先缓存。
- Artifact 不应在文本流中直接输出为大 JSON。
- 当收到 `status: completed` 后，先发送 `TEXT_MESSAGE_END`。
- 然后把缓存的 Artifact 转成一个或多个 Tool Call。
- 最后发送 `RUN_FINISHED`。
- 如果 A2A stream 失败，应发送 `RUN_ERROR`。

---

## 15. Artifact → Tool Call 映射规则

Artifact 到 Frontend Skill 的映射由 Orchestrator / ProtocolConverter 执行。

完整规划映射：

| Artifact type | Frontend Skill |
|---|---|
| code | code_preview |
| webpage | web_preview |
| diff | diff_preview |
| file | file_download |
| image | image_preview |
| deploy | deploy_status |
| document | markdown_render |
| terminal | terminal_output |
| chart | chart_render |

MVP v0.1 只强制实现：

```text
code → code_preview
```

规则：

- 如果 Frontend 未声明或未注册某个 Skill，Converter 不应盲目发送该 Tool Call。
- MVP v0.1 的 `tools` 中应包含 `code_preview`。
- 未注册的 Artifact type 可以暂时跳过，但必须记录日志。
- Post-MVP 需要统一补充 fallback 事件或错误事件。

---

## 16. AG-UI RunRequest 边界

`POST /api/agui/run` 的 HTTP endpoint 基础形态由 `platform-api-contract` 登记。

本 Skill 只规定它进入 SSE 后的事件语义。

MVP v0.1 的 RunRequest 至少应包含：

```json
{
  "threadId": "conversation-uuid",
  "runId": "run-uuid",
  "agentName": "code-agent",
  "messages": [],
  "tools": [
    {
      "name": "code_preview"
    }
  ]
}
```

规则：

- `threadId` 用于关联 conversation。
- `runId` 用于关联本次运行。
- `agentName` 在 MVP 中用于直接路由到 `code-agent`。
- `messages` 是本次上下文消息。
- `tools` 是前端可执行 Skill 列表。
- Frontend 不应声明未实现的 Skill。
- Gateway 不应把 RunRequest 当普通 REST response 处理，它应建立 SSE 流。

---

## 17. Tool Result 边界

PDR 完整目标中，Frontend Skill 执行后可以通过 ToolResult 回传结果。

例如：

```text
POST /api/agui/run/{runId}/tool-result
```

MVP v0.1 中：

- `code_preview` 可以不阻塞后端流程。
- 前端可以本地渲染完成，不强制回传 ToolResult。
- ToolResult endpoint 可以保留在 OpenAPI planned 中，不作为 MVP 必须实现。

Post-MVP 中：

- `confirm_action`、`form_input`、`file_upload` 等交互式 Skill 必须定义 ToolResult。
- ToolResult schema 由 `frontend-runtime-skills-contract` 和 `agui-event-contract` 协作约束。

---

## 18. STATE_UPDATE 边界

`STATE_UPDATE` 用于展示编排阶段、当前活跃 Agent、ExecutionPlan、进度等状态。

PDR 完整目标中，`STATE_UPDATE` 很重要，例如：

- `phase: orchestrating`
- `activeAgent: web-agent`
- `activeAgent: code-agent`
- `plan: 已拆解为 2 个子任务`
- `summary: 已完成 2 个子任务`

MVP v0.1 中：

- `STATE_UPDATE` 可以暂不强制实现。
- 如果实现，只能作为展示状态，不得替代 `TEXT_MESSAGE_*` 和 `TOOL_CALL_*` 主链路。
- Frontend 不能依赖 `STATE_UPDATE` 才能完成文本渲染。

---

## 19. 事件顺序规则

MVP v0.1 的正常事件顺序必须符合：

```text
RUN_STARTED
TEXT_MESSAGE_START
TEXT_MESSAGE_CONTENT...
TEXT_MESSAGE_END
TOOL_CALL_START
TOOL_CALL_ARGS...
TOOL_CALL_END
RUN_FINISHED
```

如果没有 Artifact，可以没有 Tool Call：

```text
RUN_STARTED
TEXT_MESSAGE_START
TEXT_MESSAGE_CONTENT...
TEXT_MESSAGE_END
RUN_FINISHED
```

如果运行失败：

```text
RUN_STARTED
TEXT_MESSAGE_START?
TEXT_MESSAGE_CONTENT?...
RUN_ERROR
```

规则：

- `TEXT_MESSAGE_CONTENT` 不得早于 `TEXT_MESSAGE_START`。
- `TEXT_MESSAGE_END` 不得早于所有该消息的 `TEXT_MESSAGE_CONTENT`。
- `TOOL_CALL_ARGS` 不得早于对应 `TOOL_CALL_START`。
- `TOOL_CALL_END` 不得早于对应 `TOOL_CALL_ARGS` 完成。
- `RUN_FINISHED` 不得早于所有消息和 Tool Call 结束。
- `RUN_ERROR` 后不应继续发送普通业务事件。

---

## 20. Frontend 事件处理要求

Frontend AG-UI Client 必须遵守：

- 支持 SSE streaming。
- 支持跨 chunk 的 JSON 行解析。
- 支持按 `messageId` 聚合文本消息。
- 支持按 `toolCallId` 聚合 Tool Call 参数。
- streaming 阶段优先按纯文本渲染，结束后再 Markdown 解析。
- `TOOL_CALL_END` 后再执行对应 Frontend Skill。
- 未知事件类型不能导致前端崩溃，应记录 warning。
- 未知 toolName 不能导致前端崩溃，应显示 fallback 或忽略。
- `RUN_ERROR` 必须停止 loading。
- `RUN_FINISHED` 必须清理当前 Run 状态。

---

## 21. Gateway / Orchestrator 事件输出要求

Gateway / Orchestrator 必须遵守：

- Gateway 负责 SSE HTTP response。
- Orchestrator 或 Converter 负责生成 AG-UI event。
- Gateway Handler 不应直接硬编码复杂 A2A → AG-UI 映射。
- Converter 应集中处理 A2A event 到 AG-UI event 的转换。
- 每个 SSE event 写出后必须 flush。
- 事件输出必须可观测，至少包含 runId / traceId 日志。
- 错误时发送 `RUN_ERROR`，不要让 SSE 静默断开。
- 发送给前端的 event 不得包含敏感配置、LLM API Key、数据库连接信息。

---

## 22. 安全规则

AG-UI event 必须遵守：

- 不在 event 中输出 API Key。
- 不在 event 中输出 Authorization token。
- 不在 event 中输出数据库连接字符串。
- 不在 event 中输出完整内部堆栈。
- Tool Call 参数必须防止 XSS。
- `web_preview` 等 Post-MVP HTML 类 Skill 必须经过安全沙箱策略。
- `code_preview` 默认只展示代码，不执行代码。
- `TOOL_CALL_ARGS` 中的 fileUrl 必须做权限控制。
- Frontend 对未知事件和未知工具必须安全降级。

---

## 23. 与其他 Skills 的协作

### 23.1 与 `project-architecture`

`project-architecture` 定义服务边界。  
本 Skill 在该边界内定义 Frontend ↔ Gateway 的实时事件。

即使 MVP 中 Orchestrator 嵌入 Gateway 进程，也不得把 AG-UI 事件逻辑散落在 handler 中。

---

### 23.2 与 `platform-api-contract`

`platform-api-contract` 只登记 `POST /api/agui/run` 的 HTTP 形态。  
本 Skill 定义该 endpoint 建立 SSE 后输出哪些事件、事件字段是什么、事件顺序是什么。

不得在 REST response schema 中展开 AG-UI event stream。

---

### 23.3 与 `gateway-orchestrator-contract`

Gateway ↔ Orchestrator 的内部调用结构由 `gateway-orchestrator-contract` 定义。

本 Skill 可以规定 Orchestrator 最终输出 AG-UI event，但不定义内部 Process 函数签名或内部 HTTP contract。

---

### 23.4 与 `a2a-agent-contract`

A2A event 原始格式由 `a2a-agent-contract` 定义。

本 Skill 只规定 A2A event 如何转换为 AG-UI event。

---

### 23.5 与 `frontend-runtime-skills-contract`

Frontend Runtime Skills 的注册表、参数 schema、组件映射和执行结果由 `frontend-runtime-skills-contract` 定义。

本 Skill 只规定 Tool Call 事件如何传输和触发这些 Skill。

---

### 23.6 与 `artifact-contract`

Artifact 的完整结构、持久化、预览、下载由 `artifact-contract` 定义。

本 Skill 只规定 Artifact 如何通过 AG-UI Tool Call 到达前端。

---

## 24. 必须维护的文件

使用本 Skill 时，必须维护：

```text
docs/contracts/agui-events.md
docs/contracts/agui-events.schema.json
```

根据需要维护：

```text
docs/contracts/agui-event-review-checklist.md
frontend/src/agui/client.ts
frontend/src/agui/events.ts
frontend/src/agui/skills.ts
frontend/src/stores/messageStore.ts
server/internal/handler/agui.go
server/internal/orchestrator/converter.go
server/internal/orchestrator/orchestrator.go
```

MVP v0.1 阶段，重点落地：

```text
frontend/src/agui/client.ts
frontend/src/agui/events.ts
frontend/src/agui/skills.ts
server/internal/handler/agui.go
server/internal/orchestrator/converter.go
```

---

## 25. 硬性规则

Coding Agent 在处理 AG-UI 事件时必须遵守：

1. AG-UI event schema 必须由 `docs/contracts/agui-events.md` 和 `docs/contracts/agui-events.schema.json` 固定。
2. `POST /api/agui/run` 的 REST HTTP 形态由 OpenAPI 登记，但 SSE 事件语义由本 Skill 定义。
3. MVP v0.1 必须实现 `RUN_STARTED / TEXT_MESSAGE_START / TEXT_MESSAGE_CONTENT / TEXT_MESSAGE_END / TOOL_CALL_START / TOOL_CALL_ARGS / TOOL_CALL_END / RUN_FINISHED / RUN_ERROR`。
4. `TEXT_MESSAGE_CONTENT` 只能传文本片段，不能塞大产物。
5. Artifact 必须通过 Tool Call 触发 Frontend Skill。
6. `code` Artifact 必须映射为 `code_preview`。
7. `TOOL_CALL_ARGS.content` 必须最终拼接为合法 JSON。
8. `TOOL_CALL_END` 后前端才能执行 Skill。
9. `RUN_FINISHED` 必须在所有文本和 Tool Call 事件之后发送。
10. `RUN_ERROR` 后不得继续发送普通业务事件。
11. Gateway Handler 不得直接堆叠复杂协议转换逻辑。
12. A2A → AG-UI 转换必须集中在 Orchestrator / ProtocolConverter。
13. 前端必须能处理 SSE 粘包 / 拆包。
14. 前端必须能处理未知事件和未知 toolName。
15. 事件字段统一使用 camelCase。
16. 事件类型统一使用大写下划线格式。
17. 不得把 A2A Task schema 当成 AG-UI event schema。
18. 不得把 REST API response schema 当成 AG-UI event schema。
19. 不得在 AG-UI event 中泄漏 token、API Key、数据库连接串、内部堆栈。
20. 必须遵守 `Contract first / Mock first / Real integration later / Review always`。

---

## 26. 输出要求

当用户要求生成或修改 AG-UI 相关内容时，Coding Agent 必须输出：

1. 该事件属于 MVP 必须事件还是 Post-MVP 规划事件。
2. 事件名称。
3. 事件触发时机。
4. 事件字段 schema。
5. 事件顺序约束。
6. Frontend 如何处理。
7. Gateway / Orchestrator 如何产生。
8. 是否涉及 A2A → AG-UI 转换。
9. 是否涉及 Artifact → Tool Call。
10. 是否影响 Frontend Runtime Skills。
11. 是否需要更新 `docs/contracts/agui-events.md`。
12. 是否需要更新 `docs/contracts/agui-events.schema.json`。
13. 是否和 REST / A2A / Artifact / Frontend Runtime Skills 边界冲突。

除非用户明确要求，否则不要直接生成业务实现代码。

---

## 27. Review Checklist

在接受任何 AG-UI 事件设计或实现前，必须检查：

### Contract 检查

- 是否更新了 `docs/contracts/agui-events.md`？
- 是否更新了 `docs/contracts/agui-events.schema.json`？
- 是否明确事件属于 MVP required 还是 Post-MVP planned？
- 是否有事件字段 schema？
- 是否有事件顺序说明？
- 是否有错误处理说明？

### 事件顺序检查

- 是否先 `RUN_STARTED`？
- 是否先 `TEXT_MESSAGE_START` 再 `TEXT_MESSAGE_CONTENT`？
- 是否 `TEXT_MESSAGE_END` 后再 Tool Call？
- 是否 `TOOL_CALL_START → TOOL_CALL_ARGS → TOOL_CALL_END` 顺序正确？
- 是否最后 `RUN_FINISHED`？
- 失败时是否 `RUN_ERROR`？

### MVP 检查

- 是否优先支持 `code-agent + code_preview`？
- 是否没有强制实现 Post-MVP 事件？
- 是否没有提前要求 `web_preview`、`form_input`、`confirm_action`？
- 是否没有把群聊状态事件作为 MVP 必需项？

### Frontend 检查

- 是否支持 SSE 粘包 / 拆包？
- 是否按 `messageId` 聚合文本？
- 是否按 `toolCallId` 聚合 Tool Call args？
- 是否 `TOOL_CALL_END` 后才执行 Skill？
- 未知事件是否不会导致崩溃？
- 未知 toolName 是否不会导致崩溃？

### Gateway / Orchestrator 检查

- Gateway 是否只负责 SSE 输出？
- ProtocolConverter 是否负责 A2A → AG-UI 转换？
- Artifact 是否先缓存再 Tool Call？
- 是否没有把大产物塞入 `TEXT_MESSAGE_CONTENT`？
- 是否有 runId / traceId 日志？

### 边界检查

- 是否没有把 REST response schema 写进 AG-UI event？
- 是否没有把 A2A endpoint 暴露给 Frontend？
- 是否没有把 Artifact 完整 schema 重复定义在本 Contract？
- 是否没有把 Frontend Skill 参数完整定义抢先写死？

### 安全检查

- 是否没有输出 token / API Key？
- 是否没有输出数据库连接串？
- 是否没有输出内部堆栈？
- Tool Call 参数是否有 XSS 风险说明？
- `code_preview` 是否只展示代码、不执行代码？


## 28. v1.1 对齐补充

### 28.1 v1.1 通用映射与 MVP 映射兼容

- v1.1 通用运行级映射中，`A2A Task.status.started` 可以映射为 `RUN_STARTED`。
- MVP v0.1 消息级流式链路中，`A2A status:working` 可以映射为 `TEXT_MESSAGE_START`。
- 二者属于不同粒度，可以共存，不能互相替代。

### 28.2 RUN_ERROR 脱敏规则

`RUN_ERROR` 不得泄漏 stack trace、token、API key、内部服务地址、数据库连接串、完整 system prompt；详细错误只允许进入脱敏日志。

### 28.3 Artifact / Tool Call 边界

- 大 Artifact 不得塞进 `TEXT_MESSAGE_CONTENT`。
- Artifact 结构由后续 `artifact-contract` 细化。
- Tool Call 参数与 ToolResult 由后续 `frontend-runtime-skills-contract` 细化。


