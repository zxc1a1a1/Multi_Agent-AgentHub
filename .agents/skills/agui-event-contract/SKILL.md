---
name: agui-event-contract
description: "用于定义 AgentHub Frontend 与 Gateway 之间的 AG-UI/SSE 实时事件流契约，包括 Run 生命周期、文本消息、工具调用、状态更新、多 Agent 消息归属、错误脱敏、事件顺序和前端聚合规则。"
---

# agui-event-contract

## 1. Skill 目的

本 Skill 用于定义 AgentHub 项目中：

```text
Frontend ↔ Gateway
```

之间的 AG-UI/SSE 实时事件流契约。

本 Skill 的核心职责是固定：

- Gateway 通过 SSE 向 Frontend 输出哪些事件。
- 每个事件有哪些字段。
- 事件之间必须遵守什么顺序。
- 前端如何按 `runId`、`messageId`、`toolCallId` 聚合流式数据。
- 多 Agent / 群聊场景下，消息归属如何表达。
- 运行状态、编排状态、降级重试如何表达。
- 错误信息如何稳定、可展示、可脱敏。
- SSE wire format 如何避免粘包、拆包、半包导致的解析错误。

一句话：

**凡是 Frontend 通过 Gateway SSE 收到的实时事件，都必须符合本 Skill。**

---

## 2. 独立性原则

本 Skill 必须独立可读。

阅读本 Skill 不应要求先理解其他 Skill。

本 Skill 只定义最终发给前端的 AG-UI event，不定义后端内部事件来源。

本 Skill 不定义：

```text
普通 REST API response
子 Agent 内部通信协议
Agent Runtime 内部实现
产物持久化格式
前端组件内部实现
Planner 决策算法
LLM Provider 调用方式
数据库 Schema
```

如果某个后端内部事件最终要显示给前端，必须先转换为本文定义的 AG-UI event。

---

## 3. 当前阶段

当前项目阶段：

```text
profile = v1.0-sprint
mvpStatus = completed
```

MVP v0.1 已完成，仅作为历史回归基线。

v1.0 Sprint 当前要求 AG-UI 事件流支持：

- 2+ Agent 的消息展示。
- 单聊与群聊。
- LLM 意图编排状态展示。
- `single` 与 `ordered-parallel` 执行策略。
- 多条 assistant message。
- 多 Agent 消息归属。
- `STATE_UPDATE`。
- `code_preview` / `web_preview` / `markdown_render` 等工具调用传输。
- 降级与重试提示。
- SSE 解析稳定性。

因此，以下内容不再作为 Post-MVP 禁止项：

```text
STATE_UPDATE
多 Agent 消息
群聊消息归属
web_preview Tool Call
markdown_render Tool Call
ordered-parallel 状态展示
fallback / retrying 状态
```

---

## 4. Version Profiles

### 4.1 MVP v0.1 Historical Profile

MVP v0.1 的历史事件集：

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

MVP 历史基线仍必须回归可用。

### 4.2 v1.0 Sprint Profile

v1.0 Sprint 当前必需事件集：

```text
RUN_STARTED
RUN_FINISHED
RUN_ERROR

TEXT_MESSAGE_START
TEXT_MESSAGE_CONTENT
TEXT_MESSAGE_END

TOOL_CALL_START
TOOL_CALL_ARGS
TOOL_CALL_END

STATE_UPDATE
```

v1.0 允许：

```text
一个 Run 产生一条 assistant message
一个 Run 产生多条 assistant message
一个 Run 中出现多个 sender
一个 message 绑定多个 Tool Call
一个 Tool Call 的 args 分多段传输
STATE_UPDATE 在任意普通业务事件之间穿插
```

v1.0 不强制：

```text
RUN_CANCELLED
TOOL_CALL_RESULT
TOOL_CALL_ERROR
USER_INPUT_REQUIRED
FRONTEND_SKILL_RESULT
STATE_SNAPSHOT
STATE_DELTA
CUSTOM_EVENT
真正并发交错输出多个 Agent 的 token
```

### 4.3 Post-v1.0 Profile

Post-v1.0 可以扩展：

```text
RUN_CANCELLED
RUN_WARNING
RUN_PROGRESS
TOOL_CALL_RESULT
TOOL_CALL_ERROR
USER_INPUT_REQUIRED
FRONTEND_SKILL_RESULT
STATE_SNAPSHOT
STATE_DELTA
CUSTOM_EVENT
```

这些事件必须向后兼容 v1.0 事件模型。

---

## 5. Contract 文件

使用本 Skill 时，必须维护：

```text
docs/contracts/agui-events.md
docs/contracts/agui-events.schema.json
docs/contracts/agui-event-review-checklist.md
```

根据需要维护：

```text
frontend/src/agui/client.ts
frontend/src/agui/events.ts
frontend/src/stores/messageStore.ts
server/internal/model/agui.go
server/internal/handler/agui.go
server/internal/orchestrator/converter.go
server/internal/orchestrator/orchestrator.go
```

注意：

- 本 Skill 不要求在同一任务中修改所有代码文件。
- 但任何事件字段变更，都必须先更新 `docs/contracts/agui-events.md` 和 `docs/contracts/agui-events.schema.json`。

---

## 6. SSE Wire Format

Gateway 向 Frontend 输出 AG-UI 事件时，统一使用 SSE。

推荐 wire format：

```text
event: message
data: {"type":"TEXT_MESSAGE_CONTENT","runId":"run-1","messageId":"msg-1","delta":"hello"}

```

硬性规则：

1. 一个 SSE event block 只包含一个 AG-UI JSON event。
2. SSE event block 必须以空行结束。
3. Frontend 必须按 `\n\n` 拆分 event block。
4. Frontend 不得假设一次 `reader.read()` 就是一条完整事件。
5. Frontend 必须处理粘包、拆包、半包。
6. Frontend 必须保留最后一个 incomplete buffer。
7. Frontend 必须安全跳过 malformed JSON。
8. Gateway 不得把多个业务事件合并成 JSON array。
9. Gateway 每写出一个 SSE event 后必须 flush。
10. `data:` 中必须是合法 JSON 字符串。
11. 不得在 SSE 注释行中承载业务状态。
12. 默认 `event` 名称使用 `message`。

兼容规则：

```text
v1.0 事件 JSON 字段优先使用 delta/state。
旧实现中的 content 仍可兼容读取。
前端聚合时应使用 event.delta ?? event.content。
```

---

## 7. 通用事件字段

所有 AG-UI event 都应遵守以下字段约定。

| 字段 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `type` | string | 是 | 事件类型，大写下划线 |
| `runId` | string | 视事件而定 | 当前运行 ID |
| `threadId` | string | 视事件而定 | 当前对话 ID |
| `messageId` | string | 视事件而定 | 当前消息 ID |
| `toolCallId` | string | 视事件而定 | 当前 Tool Call ID |
| `timestamp` | string | 建议 | RFC3339 时间 |
| `traceId` | string | 建议 | 链路追踪 ID |
| `sender` | object | 视事件而定 | 消息发送者 |
| `delta` | string | 视事件而定 | 流式增量文本或参数片段 |
| `state` | object | 视事件而定 | 状态更新对象 |
| `error` | object | 视事件而定 | 稳定错误对象 |

命名规则：

- JSON 字段统一使用 camelCase。
- `type` 使用大写下划线，例如 `TEXT_MESSAGE_CONTENT`。
- ID 字段使用字符串。
- 时间字段使用 RFC3339 / ISO 8601。
- 不允许在 AG-UI event 中混用 snake_case。

---

## 8. Run 生命周期事件

### 8.1 RUN_STARTED

表示 Gateway 已接受本次 Run，并开始通过 SSE 输出事件。

最小事件：

```json
{
  "type": "RUN_STARTED",
  "runId": "run-001",
  "threadId": "conv-001"
}
```

规则：

- 每个 Run 最多发送一次 `RUN_STARTED`。
- `RUN_STARTED` 必须早于任何 `TEXT_MESSAGE_*`、`TOOL_CALL_*`、`STATE_UPDATE`。
- 如果鉴权失败或请求格式错误，不应进入 SSE 流，而应返回 HTTP 错误。
- 如果 SSE 已建立后发生业务错误，应发送 `RUN_ERROR`。

### 8.2 RUN_FINISHED

表示本次 Run 正常结束。

最小事件：

```json
{
  "type": "RUN_FINISHED",
  "runId": "run-001"
}
```

规则：

- 每个正常完成的 Run 必须发送一次 `RUN_FINISHED`。
- `RUN_FINISHED` 必须晚于本 Run 的所有 `TEXT_MESSAGE_*` 和 `TOOL_CALL_*`。
- Gateway 发送 `RUN_FINISHED` 后不得再发送新的业务事件。
- Frontend 收到后必须关闭当前 Run 的 streaming/loading 状态。

### 8.3 RUN_ERROR

表示 Run 已经开始，但中途发生不可恢复错误。

最小事件：

```json
{
  "type": "RUN_ERROR",
  "runId": "run-001",
  "error": {
    "code": "AGUI_RUN_FAILED",
    "message": "运行失败，请稍后重试",
    "retryable": true
  }
}
```

规则：

- `RUN_ERROR` 只用于 SSE 已建立后的运行时错误。
- `RUN_ERROR` 后不应继续发送普通业务事件。
- Frontend 收到后必须停止 loading。
- Frontend 应将当前 streaming message 标记为 failed。
- `error.message` 必须可直接展示给用户。
- `error.code` 必须稳定，便于测试和排障。
- 错误信息必须脱敏。

---

## 9. Text Message 事件

Text Message 事件用于创建和流式更新 assistant 消息。

### 9.1 TEXT_MESSAGE_START

表示开始创建一条 assistant 消息。

最小事件：

```json
{
  "type": "TEXT_MESSAGE_START",
  "runId": "run-001",
  "threadId": "conv-001",
  "messageId": "msg-001",
  "role": "assistant",
  "sender": {
    "type": "agent",
    "name": "agent-name",
    "displayName": "Agent Display Name"
  }
}
```

规则：

- 每条 assistant 消息必须先有 `TEXT_MESSAGE_START`。
- `messageId` 必须在后续 `TEXT_MESSAGE_CONTENT` 和 `TEXT_MESSAGE_END` 中保持一致。
- 单 Agent 对话中，一个 Run 通常只有一条 assistant message。
- 多 Agent / 群聊中，一个 Run 可以产生多条 assistant message。
- 不同 assistant message 不得复用 `messageId`。
- `sender.name` 不得写死为某个具体 Agent 名称。
- `sender.displayName` 可以省略，前端应回退到 `sender.name`。

### 9.2 TEXT_MESSAGE_CONTENT

表示一条 assistant 消息的流式文本片段。

推荐事件：

```json
{
  "type": "TEXT_MESSAGE_CONTENT",
  "runId": "run-001",
  "messageId": "msg-001",
  "delta": "这是一个流式文本片段"
}
```

兼容事件：

```json
{
  "type": "TEXT_MESSAGE_CONTENT",
  "runId": "run-001",
  "messageId": "msg-001",
  "content": "这是旧字段兼容文本片段"
}
```

规则：

- 新实现优先使用 `delta`。
- 前端必须兼容 `event.delta ?? event.content`。
- `delta` / `content` 只能承载文本片段。
- 不得把大型代码文件、网页、图片、二进制、压缩包塞入 `TEXT_MESSAGE_CONTENT`。
- Markdown 文本可以通过 `TEXT_MESSAGE_CONTENT` 流式输出。
- 前端 streaming 阶段可以按纯文本或轻量 Markdown 渲染，结束后再做完整渲染。

### 9.3 TEXT_MESSAGE_END

表示一条 assistant 消息结束。

最小事件：

```json
{
  "type": "TEXT_MESSAGE_END",
  "runId": "run-001",
  "messageId": "msg-001"
}
```

规则：

- 每条 `TEXT_MESSAGE_START` 必须对应一个 `TEXT_MESSAGE_END`，除非 Run 中途 `RUN_ERROR`。
- `TEXT_MESSAGE_END` 必须晚于该消息所有 `TEXT_MESSAGE_CONTENT`。
- 与该消息相关的 Tool Call 通常应在 `TEXT_MESSAGE_END` 后发送。
- Frontend 收到后应将消息状态从 streaming 转为 sent。

---

## 10. Tool Call 事件

Tool Call 用于让 Gateway 通知前端渲染某类产物或交互控件。

本 Skill 只定义 Tool Call 的传输事件，不定义每个 `toolName` 的完整参数 Schema。

v1.0 常见 `toolName` 示例：

```text
code_preview
web_preview
markdown_render
```

这些只是示例，不是本协议唯一允许的工具名。

### 10.1 TOOL_CALL_START

表示开始一次前端工具调用。

最小事件：

```json
{
  "type": "TOOL_CALL_START",
  "runId": "run-001",
  "messageId": "msg-001",
  "toolCallId": "tc-001",
  "toolName": "web_preview"
}
```

规则：

- 每次 Tool Call 必须先发送 `TOOL_CALL_START`。
- `toolCallId` 必须在后续 `TOOL_CALL_ARGS` 和 `TOOL_CALL_END` 中保持一致。
- `messageId` 用于把 Tool Call 归属到某条 assistant message。
- `toolName` 必须是字符串。
- 未知 `toolName` 不得导致前端崩溃。
- Gateway 不应发送明显未被前端支持的 Tool Call；若发送，前端必须安全降级。

### 10.2 TOOL_CALL_ARGS

表示 Tool Call 的参数片段。

推荐事件：

```json
{
  "type": "TOOL_CALL_ARGS",
  "runId": "run-001",
  "toolCallId": "tc-001",
  "delta": "{\"html\":\"<html>...</html>\"}"
}
```

兼容事件：

```json
{
  "type": "TOOL_CALL_ARGS",
  "runId": "run-001",
  "toolCallId": "tc-001",
  "content": "{\"html\":\"<html>...</html>\"}"
}
```

规则：

- 新实现优先使用 `delta`。
- 前端必须兼容 `event.delta ?? event.content`。
- `delta` / `content` 是字符串。
- 前端必须按 `toolCallId` 顺序拼接所有参数片段。
- `TOOL_CALL_ARGS` 可以一次性发送完整 JSON 字符串。
- `TOOL_CALL_ARGS` 也可以分块发送 JSON 字符串片段。
- 拼接完成后的字符串必须是合法 JSON。
- 大文件不应直接放入 args；应使用可控引用或后续文件机制。
- `TOOL_CALL_ARGS` 不得携带无法解析的随意文本。

### 10.3 TOOL_CALL_END

表示 Tool Call 参数发送完成，前端可以解析并执行。

最小事件：

```json
{
  "type": "TOOL_CALL_END",
  "runId": "run-001",
  "toolCallId": "tc-001"
}
```

规则：

- Frontend 收到 `TOOL_CALL_END` 后，才能解析累计 args。
- 如果 args JSON 解析失败，Frontend 应显示 fallback 或记录 warning，不得崩溃。
- `TOOL_CALL_END` 后，同一 `toolCallId` 不应再收到 `TOOL_CALL_ARGS`。
- 如果 Tool Call 归属于某条消息，前端应把渲染结果挂到该消息下。

---

## 11. STATE_UPDATE 事件

`STATE_UPDATE` 用于向前端展示运行过程状态。

它可以用于：

- 已接受请求。
- 正在编排。
- 已分派给一个或多个 Agent。
- 当前活跃 Agent。
- 当前策略。
- 正在重试。
- 正在切换 fallback。
- 编排完成。
- 子任务状态展示。

最小事件：

```json
{
  "type": "STATE_UPDATE",
  "runId": "run-001",
  "threadId": "conv-001",
  "state": {
    "phase": "planning",
    "message": "正在分析任务并选择合适的 Agent"
  }
}
```

兼容事件：

```json
{
  "type": "STATE_UPDATE",
  "runId": "run-001",
  "content": "{\"phase\":\"planning\",\"message\":\"正在分析任务\"}"
}
```

规则：

- 新实现优先使用 `state` object。
- 兼容期前端可以读取 `event.state`，也可以尝试 `JSON.parse(event.content)`。
- `STATE_UPDATE` 只能表示过程状态。
- `STATE_UPDATE` 不得替代 `TEXT_MESSAGE_*`。
- `STATE_UPDATE` 不得替代 `TOOL_CALL_*`。
- `STATE_UPDATE` 不得作为最终消息内容持久化。
- 前端可以把 `STATE_UPDATE` 渲染为状态条、系统提示或编排可视化。

### 11.1 phase 枚举

v1.0 推荐 `phase`：

```text
accepted
planning
dispatching
agent_streaming
tool_calling
retrying
finished
failed
```

### 11.2 dispatching 示例

```json
{
  "type": "STATE_UPDATE",
  "runId": "run-001",
  "state": {
    "phase": "dispatching",
    "strategy": "parallel",
    "assignedAgents": ["agent-a", "agent-b"],
    "activeAgent": "agent-a",
    "message": "已分派给 2 个 Agent"
  }
}
```

### 11.3 retrying 示例

```json
{
  "type": "STATE_UPDATE",
  "runId": "run-001",
  "state": {
    "phase": "retrying",
    "failedAgent": "agent-a",
    "fallbackAgent": "agent-b",
    "message": "Agent 执行失败，正在切换备用 Agent"
  }
}
```

规则：

- `assignedAgents`、`activeAgent`、`failedAgent`、`fallbackAgent` 是普通字符串，不限定具体 Agent 名称。
- 前端不得依赖某些硬编码 Agent 名称才能渲染。
- `message` 必须适合展示给用户。
- 内部错误详情不得放入 `state.message`。

---

## 12. Multi-Agent / Group Conversation 事件规则

v1.0 支持一个 Run 中出现多个 Agent 消息。

规则：

1. 一个 Run 可以产生多条 assistant message。
2. 每条 assistant message 必须有独立 `messageId`。
3. 每条 assistant message 可以有独立 `sender`。
4. `sender.type` 推荐为 `agent`。
5. `sender.name` 是 Agent 名称，但协议不固定任何具体名称。
6. Tool Call 必须通过 `messageId` 归属到某条 message。
7. `STATE_UPDATE.activeAgent` 可以提示下一条消息来自哪个 Agent。
8. `STATE_UPDATE` 不能替代 `TEXT_MESSAGE_START.sender`。
9. 前端应以 `messageId` 为聚合主键，而不是以 Agent 名称为主键。

多 Agent 正常顺序示例：

```text
RUN_STARTED
STATE_UPDATE(planning)
STATE_UPDATE(dispatching, strategy=parallel, assignedAgents=[...])

TEXT_MESSAGE_START(messageId=msg-a, sender=agent-a)
TEXT_MESSAGE_CONTENT(messageId=msg-a, delta=...)
TEXT_MESSAGE_END(messageId=msg-a)
TOOL_CALL_START(messageId=msg-a, toolCallId=tc-a)
TOOL_CALL_ARGS(toolCallId=tc-a, delta=...)
TOOL_CALL_END(toolCallId=tc-a)

STATE_UPDATE(activeAgent=agent-b)

TEXT_MESSAGE_START(messageId=msg-b, sender=agent-b)
TEXT_MESSAGE_CONTENT(messageId=msg-b, delta=...)
TEXT_MESSAGE_END(messageId=msg-b)

RUN_FINISHED
```

---

## 13. ordered-parallel 事件规则

v1.0 允许 `ordered-parallel`。

含义：

```text
执行策略可以表达为 parallel。
UI 可以展示多个 Agent 参与。
Gateway 输出事件时仍按 message 粒度顺序发送。
不要求多个 Agent 的 token 交错输出。
```

硬性规则：

- 不同 message 的 `TEXT_MESSAGE_CONTENT` 可以不交错。
- 如果实现交错输出，必须保证每个 content 都带正确 `messageId`。
- v1.0 推荐先使用顺序输出，降低前端聚合复杂度。
- `strategy = parallel` 不等于 SSE token 必须并发交错。

---

## 14. Error 事件规则

错误对象必须稳定。

推荐结构：

```json
{
  "code": "AGUI_AGENT_FAILED",
  "message": "Agent 执行失败，请稍后重试",
  "retryable": true,
  "details": {
    "phase": "agent_streaming"
  }
}
```

推荐错误码：

```text
AGUI_BAD_REQUEST
AGUI_UNAUTHORIZED
AGUI_STREAM_INTERRUPTED
AGUI_PLANNING_FAILED
AGUI_AGENT_UNAVAILABLE
AGUI_AGENT_FAILED
AGUI_TOOL_ARGS_INVALID
AGUI_TOOL_UNSUPPORTED
AGUI_INTERNAL
```

脱敏规则：

`RUN_ERROR.error.message` 和 `RUN_ERROR.error.details` 不得包含：

```text
API key
Authorization token
数据库连接串
内部堆栈
完整 system prompt
LLM provider 原始敏感错误
内网服务拓扑
用户不可理解的 panic 信息
```

降级规则：

- 如果可以 fallback，应优先发送 `STATE_UPDATE(phase=retrying)`。
- 只有不可恢复时才发送 `RUN_ERROR`。
- `RUN_ERROR` 代表本 Run 失败。
- `RUN_ERROR` 后不应继续发送普通业务事件。

---

## 15. Frontend 聚合规则

Frontend AG-UI Client 必须：

1. 按 SSE event block 解析事件。
2. 对 malformed JSON 安全跳过或记录 warning。
3. 按 `runId` 管理运行状态。
4. 按 `messageId` 聚合文本消息。
5. 按 `toolCallId` 聚合 Tool Call 参数。
6. 对 `TEXT_MESSAGE_CONTENT` 使用 `event.delta ?? event.content`。
7. 对 `TOOL_CALL_ARGS` 使用 `event.delta ?? event.content`。
8. 在 `TOOL_CALL_END` 后解析累计 args。
9. 未知事件类型不得导致崩溃。
10. 未知 `toolName` 不得导致崩溃。
11. 收到 `RUN_FINISHED` 后停止 loading。
12. 收到 `RUN_ERROR` 后停止 loading，并标记失败。
13. 支持 per-conversation streaming 状态。
14. 支持多 Agent 消息独立渲染。
15. 支持 `STATE_UPDATE` 作为非消息状态显示。

---

## 16. Gateway 输出规则

Gateway 必须：

1. 使用 SSE 输出实时事件。
2. 每个 AG-UI event 输出一个 SSE block。
3. 每个 SSE block 输出合法 JSON。
4. 每次写出后 flush。
5. 保证事件字段符合 `docs/contracts/agui-events.schema.json`。
6. 保证每个 Run 有明确结束：`RUN_FINISHED` 或 `RUN_ERROR`。
7. 不在错误中泄漏敏感信息。
8. 不把大型产物塞进 `TEXT_MESSAGE_CONTENT`。
9. 不输出前端无法区分归属的多 Agent 消息。
10. 多 Agent 消息必须带独立 `messageId`。
11. 需要展示过程状态时，使用 `STATE_UPDATE`。
12. 不依赖具体 Agent 名称生成事件结构。

---

## 17. 事件顺序规则

### 17.1 无 Tool Call 的正常流程

```text
RUN_STARTED
STATE_UPDATE*
TEXT_MESSAGE_START
TEXT_MESSAGE_CONTENT*
TEXT_MESSAGE_END
STATE_UPDATE*
RUN_FINISHED
```

### 17.2 有 Tool Call 的正常流程

```text
RUN_STARTED
STATE_UPDATE*
TEXT_MESSAGE_START
TEXT_MESSAGE_CONTENT*
TEXT_MESSAGE_END
TOOL_CALL_START
TOOL_CALL_ARGS*
TOOL_CALL_END
RUN_FINISHED
```

### 17.3 多消息流程

```text
RUN_STARTED
STATE_UPDATE*

TEXT_MESSAGE_START(msg-1)
TEXT_MESSAGE_CONTENT*(msg-1)
TEXT_MESSAGE_END(msg-1)
TOOL_CALL_*(msg-1)?

STATE_UPDATE*

TEXT_MESSAGE_START(msg-2)
TEXT_MESSAGE_CONTENT*(msg-2)
TEXT_MESSAGE_END(msg-2)
TOOL_CALL_*(msg-2)?

RUN_FINISHED
```

### 17.4 失败流程

```text
RUN_STARTED
STATE_UPDATE*
TEXT_MESSAGE_START?
TEXT_MESSAGE_CONTENT*
STATE_UPDATE(phase=retrying)?
RUN_ERROR
```

硬性规则：

- `TEXT_MESSAGE_CONTENT` 不得早于对应 `TEXT_MESSAGE_START`。
- `TEXT_MESSAGE_END` 不得早于对应所有 `TEXT_MESSAGE_CONTENT`。
- `TOOL_CALL_ARGS` 不得早于对应 `TOOL_CALL_START`。
- `TOOL_CALL_END` 不得早于对应 `TOOL_CALL_ARGS` 完成。
- `RUN_FINISHED` 不得早于所有消息和 Tool Call 结束。
- `RUN_ERROR` 后不应继续发送普通业务事件。
- `STATE_UPDATE` 可以穿插，但不得破坏主事件顺序。

---

## 18. 兼容规则

为了从 MVP 平滑升级到 v1.0：

```text
TEXT_MESSAGE_CONTENT.content 兼容读取，但新实现优先 delta。
TOOL_CALL_ARGS.content 兼容读取，但新实现优先 delta。
STATE_UPDATE.content 兼容读取，但新实现优先 state object。
单 Agent 单消息流程必须继续可用。
code_preview Tool Call 必须继续可用。
```

前端兼容建议：

```ts
const textDelta = event.delta ?? event.content ?? ''
const argsDelta = event.delta ?? event.content ?? ''
const state = event.state ?? safeParseJSON(event.content)
```

---

## 19. 安全规则

AG-UI event 必须遵守：

- 不在 event 中输出 API Key。
- 不在 event 中输出 Authorization token。
- 不在 event 中输出数据库连接字符串。
- 不在 event 中输出完整内部堆栈。
- 不在 event 中输出完整 system prompt。
- 不在 event 中输出 provider 原始敏感错误。
- Tool Call 参数必须是可解析 JSON。
- HTML 类 Tool Call 的渲染必须由前端沙箱保护。
- 未知事件和未知工具必须安全降级。
- `TEXT_MESSAGE_CONTENT` 默认只渲染为文本或 Markdown，不执行代码。
- `code_preview` 默认只展示代码，不执行代码。

---

## 20. Contract Test 规则

至少应测试：

### 20.1 SSE 解析

- 能按 `\n\n` 拆分 event block。
- 能处理一个 chunk 内多个 event。
- 能处理一个 event 被拆成多个 chunk。
- 能保留 incomplete buffer。
- 能跳过 malformed JSON。

### 20.2 Run 生命周期

- 正常流程有 `RUN_STARTED` 和 `RUN_FINISHED`。
- 错误流程有 `RUN_ERROR`。
- `RUN_ERROR` 后 loading 停止。
- `RUN_FINISHED` 后 loading 停止。

### 20.3 Text Message

- `TEXT_MESSAGE_START / CONTENT / END` 成对。
- `messageId` 一致。
- 多 Agent 消息 `messageId` 独立。
- `sender.name` 可展示但不写死。

### 20.4 Tool Call

- `TOOL_CALL_START / ARGS / END` 成对。
- `toolCallId` 一致。
- args 可以分片聚合。
- args 结束后可解析 JSON。
- 未知 `toolName` 不崩溃。

### 20.5 State

- `STATE_UPDATE` 支持 `state` object。
- 兼容 `content` JSON 字符串。
- `phase` 合法。
- `retrying` 可展示。
- `activeAgent` 不要求固定名称。

### 20.6 安全

- 错误脱敏。
- Tool args malformed 不崩溃。
- 未知事件不崩溃。
- HTML 参数不直接注入父页面 DOM。

---

## 21. Review Checklist

接受任何 AG-UI 事件变更前，必须检查：

- 是否更新 `docs/contracts/agui-events.md`？
- 是否更新 `docs/contracts/agui-events.schema.json`？
- 是否更新 `docs/contracts/agui-event-review-checklist.md`？
- 是否保持 `SKILL.md` 独立可读？
- 是否没有把其他内部协议细节写进本 Skill？
- 是否定义了事件字段？
- 是否定义了事件顺序？
- 是否定义了错误脱敏？
- 是否定义了前端聚合规则？
- 是否兼容 MVP 历史事件字段？
- 是否支持 `STATE_UPDATE`？
- 是否支持多 Agent 消息归属？
- 是否没有写死具体 Agent 名称？
- 是否没有把大型产物塞进 `TEXT_MESSAGE_CONTENT`？
- 是否没有要求真正并发交错 token？
- 是否有 SSE 粘包 / 拆包测试？

---

## 22. 硬性规则

Coding Agent 在处理 AG-UI 事件时必须遵守：

1. AG-UI event schema 必须由 `docs/contracts/agui-events.md` 和 `docs/contracts/agui-events.schema.json` 固定。
2. 本 Skill 只定义 Frontend ↔ Gateway 的 SSE 实时事件流。
3. 本 Skill 不得依赖其他 Skill 才能读懂。
4. v1.0 必须支持 `STATE_UPDATE`。
5. v1.0 必须支持多 Agent 消息归属。
6. v1.0 必须支持一个 Run 多条 assistant message。
7. v1.0 必须支持 `code_preview`、`web_preview`、`markdown_render` 等 Tool Call 名称的传输。
8. 本 Skill 不定义具体工具参数 Schema，只定义 Tool Call 传输机制。
9. `TEXT_MESSAGE_CONTENT` 新实现优先使用 `delta`。
10. `TOOL_CALL_ARGS` 新实现优先使用 `delta`。
11. `STATE_UPDATE` 新实现优先使用 `state` object。
12. 前端必须兼容旧字段 `content`。
13. SSE 必须按 event block 解析，不得按单行 JSON 解析。
14. `RUN_ERROR` 必须脱敏。
15. 未知事件不得导致前端崩溃。
16. 未知工具不得导致前端崩溃。
17. 多 Agent 场景不得复用 `messageId`。
18. Tool Call 必须通过 `messageId` 归属到消息。
19. `RUN_FINISHED` 或 `RUN_ERROR` 必须结束当前 Run。
20. 不得硬编码具体 Agent 名称作为事件协议规则。

---

## 23. 完成定义

本 Skill 视为完成，当且仅当：

```text
.claude/skills/agui-event-contract/SKILL.md
docs/contracts/agui-events.md
docs/contracts/agui-events.schema.json
docs/contracts/agui-event-review-checklist.md
```

已经明确：

- Skill 独立可读。
- Frontend ↔ Gateway SSE 边界清晰。
- v1.0 事件集完整。
- `STATE_UPDATE` 是当前必需事件。
- 多 Agent / 群聊消息归属规则明确。
- `ordered-parallel` 事件顺序明确。
- Tool Call 传输规则明确。
- `delta` 与 `content` 兼容规则明确。
- SSE wire format 明确。
- 前端聚合规则明确。
- 错误脱敏规则明确。
- Review Checklist 明确。

---

## References

- `references/sse-wire-format.md`
- `references/event-lifecycle.md`
- `references/text-message-events.md`
- `references/tool-call-events.md`
- `references/state-update-events.md`
- `references/multi-agent-events.md`
- `references/error-events.md`
- `references/frontend-aggregation.md`
- `references/agui-review-checklist.md`
