# Gateway-Orchestrator Events Contract

## 1. 文档目的

本文档定义 Gateway ↔ Orchestrator 内部事件输出规则。

这里的“事件”不是新的前端协议，而是：

```text
Orchestrator → Gateway EventSink / EventChan
```

Orchestrator 输出给 Gateway 的事件最终会被 Gateway 包装成 SSE 返回给 Frontend。

因此：

- 内部事件 payload 必须遵守 `agui-event-contract`。
- Gateway 可以负责 SSE 包装。
- Gateway 不应该修改事件语义。
- A2A 原始事件不得直接透传给 Frontend。
- 本文档不重新定义 AG-UI event schema，只定义 Orchestrator 向 Gateway 输出事件的顺序、来源和边界。

---

## 2. 内部事件通道

MVP v0.1 推荐使用：

```go
type EventSink interface {
    Emit(ctx context.Context, event AGUIEvent) error
}
```

或：

```go
type EventChan chan<- AGUIEvent
```

基本规则：

- Orchestrator 只向 EventSink 输出 AG-UI Event。
- Gateway 从 EventSink 读取事件，并写出 SSE。
- EventSink 必须支持 context cancellation。
- EventSink 不得传递 A2A 原始事件。
- EventSink 不得传递内部调试对象。
- EventSink 不得传递不符合 `agui-events.schema.json` 的事件。

---

## 3. MVP v0.1 事件链

MVP v0.1 正常事件链：

```text
RUN_STARTED
TEXT_MESSAGE_START
TEXT_MESSAGE_CONTENT*
TEXT_MESSAGE_END
TOOL_CALL_START?
TOOL_CALL_ARGS?
TOOL_CALL_END?
RUN_FINISHED
```

说明：

- `TEXT_MESSAGE_CONTENT*` 表示可以出现多次。
- 如果没有 Artifact，可以没有 `TOOL_CALL_*`。
- 如果有 `code` Artifact，必须输出 `code_preview` 的 `TOOL_CALL_*`。
- 运行失败时必须输出 `RUN_ERROR`。
- MVP v0.1 不强制复杂 `STATE_UPDATE`，但 Post-MVP 可以扩展。

---

## 4. Orchestrator 输出事件来源

| A2A 输入 / 内部状态 | Orchestrator 输出事件 |
|---|---|
| Run 开始 | `RUN_STARTED` |
| A2A `status: working` | `TEXT_MESSAGE_START` |
| A2A `text` | `TEXT_MESSAGE_CONTENT` |
| A2A `artifact` | 缓存，不立即输出 |
| A2A `status: completed` | `TEXT_MESSAGE_END` → flush artifacts → `TOOL_CALL_*` → `RUN_FINISHED` |
| A2A `status: failed` | `RUN_ERROR` |
| Orchestrator 内部错误 | `RUN_ERROR` |
| context cancelled | `RUN_ERROR` 或停止输出并返回 cancelled |
| timeout | `RUN_ERROR` |

---

## 5. 正常事件顺序

### 5.1 Run 开始

Orchestrator 在开始处理后应首先输出：

```json
{
  "type": "RUN_STARTED",
  "runId": "run-001",
  "threadId": "conv-001"
}
```

规则：

- `runId` 必须来自 `OrchestratorRequest.RunID`。
- `threadId` 应来自 `OrchestratorRequest.ThreadID`。
- `RUN_STARTED` 不代表 Child Agent 已经开始输出，只代表 Orchestrator 已接受并开始处理。

---

### 5.2 文本开始

A2A 返回 `status: working` 后，ProtocolConverter 输出：

```json
{
  "type": "TEXT_MESSAGE_START",
  "messageId": "msg-001",
  "role": "agent",
  "agentName": "code-agent"
}
```

规则：

- `messageId` 必须稳定。
- 后续 `TEXT_MESSAGE_CONTENT` 和 `TEXT_MESSAGE_END` 必须使用同一个 `messageId`。
- MVP v0.1 中 `agentName` 通常是 `code-agent`。

---

### 5.3 文本流式内容

每个 A2A text chunk 转换为：

```json
{
  "type": "TEXT_MESSAGE_CONTENT",
  "messageId": "msg-001",
  "content": "package main"
}
```

规则：

- `content` 只允许传文本 chunk。
- 大代码、大网页、大文件不允许直接塞进 `TEXT_MESSAGE_CONTENT`。
- 代码块可以作为普通回复的一部分流式输出，但正式产物必须通过 Artifact → Tool Call。
- Gateway 不得把多个不同 messageId 的内容混在一起。

---

### 5.4 文本结束

A2A 返回 `status: completed` 后，先输出：

```json
{
  "type": "TEXT_MESSAGE_END",
  "messageId": "msg-001"
}
```

规则：

- 必须在 flush Artifact 之前输出。
- 表示该 Agent 回复文本完成。
- 前端可在该事件后进行 Markdown 最终渲染。

---

## 6. Artifact flush 与 Tool Call

A2A artifact 不立即输出给 Frontend，必须先缓存。

当 A2A `status: completed` 到达后，执行：

```text
TEXT_MESSAGE_END
→ flush artifactBuffer
→ TOOL_CALL_START
→ TOOL_CALL_ARGS
→ TOOL_CALL_END
→ RUN_FINISHED
```

### 6.1 code Artifact 映射

`code` Artifact 必须映射为：

```text
toolName = code_preview
```

参数：

```json
{
  "code": "string",
  "language": "string",
  "filename": "string"
}
```

### 6.2 Tool Call Start

```json
{
  "type": "TOOL_CALL_START",
  "toolCallId": "tc-001",
  "toolName": "code_preview",
  "messageId": "msg-001"
}
```

规则：

- `toolCallId` 必须唯一。
- `toolName` 必须来自 Frontend 注册的 Skill。
- MVP v0.1 必须支持 `code_preview`。
- 如果前端未声明 `code_preview`，Orchestrator 应跳过该 Tool Call 或输出可诊断错误，不能发送未知 Skill。

### 6.3 Tool Call Args

```json
{
  "type": "TOOL_CALL_ARGS",
  "toolCallId": "tc-001",
  "content": "{\"code\":\"package main\",\"language\":\"go\",\"filename\":\"main.go\"}"
}
```

规则：

- `content` 是 JSON 字符串片段。
- `TOOL_CALL_ARGS` 可以分片发送。
- 前端必须按 `toolCallId` 聚合。
- 聚合后的 JSON 必须符合 Frontend Skill 参数 schema。
- MVP v0.1 中 `code_preview` 至少需要 `code`、`language`、`filename`。

### 6.4 Tool Call End

```json
{
  "type": "TOOL_CALL_END",
  "toolCallId": "tc-001"
}
```

规则：

- 表示该 Tool Call 参数传输结束。
- 前端应在此时解析聚合后的 args 并执行对应 Skill。
- 如果 args 不是合法 JSON，前端应进入错误状态，Gateway/Orchestrator 后续应通过 Contract Test 避免此类问题。

---

## 7. Run Finished

所有文本和 Tool Call 完成后输出：

```json
{
  "type": "RUN_FINISHED",
  "runId": "run-001"
}
```

规则：

- `RUN_FINISHED` 必须出现在本次 run 的最后一个正常事件。
- `RUN_FINISHED` 之后不应再输出 `TEXT_MESSAGE_CONTENT` 或 `TOOL_CALL_ARGS`。
- Gateway 收到 `RUN_FINISHED` 后可以准备关闭 SSE。
- Gateway 可以在此后持久化 `OrchestratorResult`。

---

## 8. Run Error

错误事件：

```json
{
  "type": "RUN_ERROR",
  "runId": "run-001",
  "code": "ORCHESTRATOR_A2A_CONNECT_FAILED",
  "message": "调用 code-agent 失败"
}
```

触发场景：

- `OrchestratorRequest` 非法。
- 找不到目标 Agent。
- A2A 连接失败。
- A2A stream 中断。
- ProtocolConverter 失败。
- context cancelled。
- timeout。
- Orchestrator 内部错误。

规则：

- 错误事件不能泄漏堆栈、token、API key。
- 如果 Orchestrator 已输出 `RUN_ERROR`，Gateway 不应重复输出冲突错误。
- 如果 Orchestrator 返回错误但未输出 `RUN_ERROR`，Gateway 必须补发 `RUN_ERROR`。
- `RUN_ERROR` 后通常不再输出 `RUN_FINISHED`，除非后续 Contract 明确允许带状态的结束事件。

---

## 9. Context cancellation 事件规则

当客户端断开或用户取消：

```text
Gateway cancel context
→ Orchestrator 停止处理
→ A2A Client 停止读取
→ EventSink 停止写入
```

规则：

- context cancelled 后不应继续发送新事件。
- 如果取消发生在已向前端输出部分内容之后，可以输出 `RUN_ERROR` 或直接停止，具体策略必须在实现中保持一致。
- MVP v0.1 可以只支持浏览器断开触发取消，不强制实现完整 cancel endpoint。

---

## 10. Timeout 事件规则

当 Orchestrator 或 A2A 调用超时：

```text
timeout
→ RUN_ERROR
→ OrchestratorResult.Status = failed
```

推荐错误码：

```text
ORCHESTRATOR_TIMEOUT
```

规则：

- timeout 信息可以展示给用户。
- 不能泄漏内部服务地址、堆栈、密钥。
- Post-MVP 可以通过 `STATE_UPDATE` 表示 retrying / fallback。

---

## 11. Post-MVP STATE_UPDATE

Post-MVP 可以扩展：

```json
{
  "type": "STATE_UPDATE",
  "state": {
    "phase": "orchestrating",
    "activeAgent": "code-agent",
    "message": "正在分析任务"
  }
}
```

规则：

- MVP v0.1 不强制复杂实现。
- `STATE_UPDATE` 仍然属于 AG-UI Event。
- 不能把 Orchestrator 内部私有对象直接塞进 `state`。
- 多 Agent 编排状态应由 `intent-orchestration-contract` 进一步定义。

---

## 12. Gateway 写出 SSE 规则

Gateway 收到 Orchestrator 事件后写出 SSE：

```text
event: message
data: {"type":"RUN_STARTED","runId":"run-001"}
```

规则：

- Gateway 只负责 SSE 包装和 flush。
- Gateway 不改变 event payload 语义。
- Gateway 不把 A2A event 写给前端。
- Gateway 不输出未进入 `agui-event-contract` 的新 event type。
- Gateway 必须处理客户端断开。
- Gateway 必须避免 response buffering。
- Gateway 应在每个事件后 flush。

---

## 13. 禁止事项

禁止：

- Orchestrator 直接写 SSE。
- Gateway 修改 Orchestrator 输出事件语义。
- A2A event 直接透传给 Frontend。
- `artifact` 直接作为 `TEXT_MESSAGE_CONTENT` 发给前端。
- `TOOL_CALL_ARGS` 不带 `toolCallId`。
- `TOOL_CALL_START` 和 `TOOL_CALL_END` 的 `toolCallId` 不一致。
- `RUN_FINISHED` 后继续输出内容事件。
- 后端新增事件类型但不更新 `agui-events.md` 和 `agui-events.schema.json`。
- 把 Gateway-Orchestrator 内部错误对象直接传给 Frontend。
- 把 `/internal/*` 的内部事件流暴露给 Frontend。

---

## 14. Review Checklist

- [ ] Orchestrator 是否只通过 EventSink / EventChan 输出事件？
- [ ] 输出事件是否符合 `agui-event-contract`？
- [ ] Gateway 是否只负责 SSE 包装？
- [ ] Gateway 是否没有修改事件语义？
- [ ] A2A 原始事件是否没有暴露给 Frontend？
- [ ] `status: working` 是否转换为 `TEXT_MESSAGE_START`？
- [ ] `text` 是否转换为 `TEXT_MESSAGE_CONTENT`？
- [ ] `artifact` 是否先缓存？
- [ ] `completed` 是否触发 `TEXT_MESSAGE_END`、flush artifacts、`TOOL_CALL_*`、`RUN_FINISHED`？
- [ ] `failed` 是否触发 `RUN_ERROR`？
- [ ] `code` Artifact 是否映射为 `code_preview`？
- [ ] `code_preview` 参数是否包含 `code`、`language`、`filename`？
- [ ] `TEXT_MESSAGE_CONTENT` 是否只传文本 chunk？
- [ ] `TOOL_CALL_ARGS` 是否可以按 `toolCallId` 聚合？
- [ ] `RUN_ERROR` 是否不会泄漏敏感信息？
- [ ] context cancelled 后是否停止输出事件？
- [ ] timeout 是否映射为 `RUN_ERROR`？


## 15. v1.1 OrchestratorEvent 长期模型补充

### 15.1 长期事件模型

Post-MVP / v1.1 推荐 Orchestrator 输出内部 `OrchestratorEvent`，由 Gateway 统一映射为 AG-UI Event：

```text
run.started
run.finished
run.failed
message.started
message.delta
message.ended
agent.active_changed
plan.created
artifact.produced
frontend_tool.requested
approval.required
```

### 15.2 与 MVP 简化兼容

- MVP v0.1 可直接输出 AG-UI-compatible event。
- 长期标准推荐改为 `OrchestratorEvent -> Gateway 映射 -> AG-UI Event`。
- 两种模式可按阶段共存，但不能把 MVP 简化写死为长期唯一架构。

