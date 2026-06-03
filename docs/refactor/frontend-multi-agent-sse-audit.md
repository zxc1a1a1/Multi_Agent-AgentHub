# Frontend Multi-Agent SSE 审计报告

## 1. 审计范围与目的

本审计文档是 AgentHub v1.0 Productization Stage Step 1-D 的产物。

**前置条件：**
- Step 1-A 已通过：`current-architecture-state.md` 与 `legacy-boundary.md` 已完成
- Step 1-B 已通过：`readme.md` 已更新
- Step 1-C 已通过：new-arch-smoke workflow 已收口为正式质量门禁

**审计目的：** 只审计 frontend 是否正确承接当前 Gateway SSE 中的 multi-agent 输出，不修改业务代码。

**禁止事项：** 本步不修改任何业务代码（frontend/、services/、agents/、server/、pkg/、docker-compose、CI、smoke），只产出审计文档。

---

## 2. 当前前端 SSE 链路图

### 2.1 完整事件流（含事件类型转换）

```text
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Orchestrator (services/orchestrator)              │
│  handler_run_stream.go → OrderedParallelExecutor / SingleExecutor           │
│                                                                             │
│  内部事件类型 (ExecutionEvent.Type):                                        │
│    run_started, message_start, message_delta, message_end,                  │
│    run_finished, run_error                                                  │
│                                                                             │
│  转为 OrchestratorStreamEvent (SSE):                                        │
│    event: run_started    data: {"type":"run_started","runId":"...",        │
│                                "sender":{"type":"agent","name":"..."}}      │
│    event: message_start  data: {"type":"message_start","messageId":"...",  │
│                                "sender":{"type":"agent","name":"web-agent"}}│
│    event: message_delta  data: {"type":"message_delta","messageId":"...",  │
│                                "sender":{...},"delta":"text..."}            │
│    event: message_end    data: {"type":"message_end","messageId":"...",    │
│                                "sender":{...}}                              │
│    event: run_finished   data: {"type":"run_finished","runId":"...",       │
│                                "state":{"status":"completed"}}              │
│    event: run_error      data: {"type":"run_error","error":{...}}          │
└───────────────────┬─────────────────────────────────────────────────────────┘
                    │ HTTP/SSE (internal)
                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                orchestratorclient.go (services/gateway)                     │
│  parseSSEStream(): OrchestratorStreamEvent → adk.Event                      │
│                                                                             │
│  映射:                                                                       │
│   message_start → adk.Event{Author, Actions:{StateDelta:{messageId,...}}}   │
│   message_delta → adk.Event{Author, Content:{TextPart}, Partial:true}       │
│   message_end   → adk.Event{Author, Content:{TextPart}, Final:true}         │
│   run_started   → adk.Event{Author, Actions:{StateDelta}}                   │
│   run_finished  → adk.Event{Author, Actions:{StateDelta}, Final:true}       │
│   run_error     → yield error (not adk.Event)                              │
│                                                                             │
│  ⚠ messageId 丢失: message_delta/message_end 的 messageId 不传入 adk.Event │
│  ⚠ sender 降级: sender{type,name} → Author string (agent name only)        │
│  ⚠ runId 丢失: runId 不传入 adk.Event                                     │
└───────────────────┬─────────────────────────────────────────────────────────┘
                    │ adk.Event
                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                agui.Translator (pkg/runtime/agui/translator.go)             │
│  Translate(): adk.Event → []agui.Event                                     │
│                                                                             │
│  映射:                                                                       │
│   Content.TextPart → Event{Type:"message"|"message.delta", Text, Author}    │
│   Content.ToolCallPart → Event{Type:"tool.call", ToolCall}                  │
│   Actions.StateDelta → Event{Type:"state.delta", StateDelta}               │
│   Actions.ArtifactDelta → Event{Type:"artifact.delta", Artifact}           │
│   Final+Content → Event{Type:"message.end"}                                │
│                                                                             │
│  ⚠ 非标准事件类型: "message" 而非 TEXT_MESSAGE_CONTENT                       │
│  ⚠ 非标准事件类型: "message.delta" 而非 TEXT_MESSAGE_CONTENT                │
│  ⚠ 非标准事件类型: "state.delta" 而非 STATE_UPDATE                          │
│  ⚠ 非标准事件类型: "message.end" 而非 TEXT_MESSAGE_END                      │
│  ⚠ 非标准事件类型: "tool.call" 而非 TOOL_CALL_START/ARGS/END               │
│  ⚠ 非标准事件类型: "artifact.delta" 而非 TOOL_CALL_*                        │
│  ⚠ 无 RUN_STARTED / RUN_FINISHED / RUN_ERROR 生成                           │
│  ⚠ 使用 "author" 字段而非 "sender" 对象                                     │
│  ⚠ 使用 "text" 字段而非 "delta" 字段                                        │
│  ⚠ 使用 "id" 字段而非 "messageId" 字段                                      │
│  ⚠ 无 runId / threadId / timestamp / traceId 字段                           │
└───────────────────┬─────────────────────────────────────────────────────────┘
                    │ agui.Event
                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                Gateway SSE Writer (services/gateway/sse/sse.go)             │
│  WriteEvent(): agui.Event → SSE wire format                                 │
│                                                                             │
│  wire format:                                                               │
│    event: <event.Type>          ← 非标准类型名                               │
│    data: <JSON(agui.Event)>     ← 非标准字段名                               │
│                                                                             │
│  示例:                                                                       │
│    event: message.delta                                                     │
│    data: {"type":"message.delta","author":"web-agent","text":"hello"}       │
│                                                                             │
│  WriteError(): 仅用于 stream error                                          │
│    event: error                                                             │
│    data: {"type":"error","text":"assistant run failed","stateDelta":{...}}  │
└───────────────────┬─────────────────────────────────────────────────────────┘
                    │ HTTP/SSE (public)
                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                Frontend SSE Client (frontend/src/agui/client.ts)            │
│  runAgent(): POST /api/chat → ReadableStream → parseSSEBlock               │
│                                                                             │
│  ✅ 按 \n\n 拆分 event block                                                │
│  ✅ 处理粘包 (tail buffer)                                                  │
│  ✅ 处理拆包 (跨 chunk 拼接)                                                │
│  ✅ 安全跳过 malformed JSON                                                 │
│  ✅ 识别 event: / data: 行                                                  │
│  ✅ event.type 回退到 SSE event name                                        │
└───────────────────┬─────────────────────────────────────────────────────────┘
                    │ AGUIEvent (TypeScript)
                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                Frontend messageStore (frontend/src/stores/messageStore.ts)  │
│  switch(event.type):                                                        │
│                                                                             │
│  已处理:                                                                     │
│    TEXT_MESSAGE_START     ✅ → 重置 agentMsgId, agentContent                │
│    TEXT_MESSAGE_CONTENT   ✅ → 追加 delta                                   │
│    message / message.delta ✅ → 追加 delta                                  │
│    TEXT_MESSAGE_END       ✅ → finishStreamingMessage                       │
│    message.end            ✅ → finishStreamingMessage                       │
│    TOOL_CALL_START        ✅ → 初始化 toolCallArgs                          │
│    TOOL_CALL_ARGS         ✅ → 追加 args delta                              │
│    TOOL_CALL_END          ✅ → 解析并执行 tool                              │
│    tool.call              ✅ → 解析旧格式 tool call                         │
│    artifact.delta         ✅ → 追加 code/webpage artifact                   │
│    RUN_FINISHED           ✅ → stop streaming                               │
│    RUN_ERROR / error      ✅ → fail message                                 │
│                                                                             │
│  未处理 (被忽略):                                                            │
│    state.delta            ❌ → switch 无匹配 case, 事件被丢弃               │
│    STATE_UPDATE           ❌ → switch 无匹配 case                            │
│    RUN_STARTED            ❌ → switch 无匹配 case                            │
│    tool.result            ❌ → switch 无匹配 case                            │
│    thinking               ❌ → Gateway sse.go 主动过滤                       │
└──────────────────┬──────────────────────────────────────────────────────────┘
                   │ Message[]
                   ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                Frontend Components (frontend/src/components/)               │
│  MessageBubble → StreamingText | CodePreview | WebPreview                   │
│                                                                             │
│  ✅ MessageBubble 按 senderType/senderName 渲染气泡                          │
│  ✅ StreamingText 流式显示纯文本, 结束后 Markdown 渲染                        │
│  ✅ CodePreview 语法高亮, 复制, 不执行代码                                   │
│  ✅ WebPreview Safe Mode (<pre><code> 文本化 HTML, 无 iframe)               │
│  ⚠ 不同 sender 的消息可能被合并为一个气泡 (见 P0 缺口)                       │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 关键事件转换表

| Orchestrator SSE (snake_case) | orchestratorclient → adk.Event | agui.Translator → agui.Event | Gateway SSE event: 行 | Frontend 匹配 case |
|---|---|---|---|---|
| `run_started` | `{Actions:{StateDelta}, Author}` | `{Type:"state.delta", StateDelta}` | `event: state.delta` | ❌ 未处理 |
| `message_start` | `{Actions:{StateDelta:{messageId}}, Author}` | `{Type:"state.delta", StateDelta}` | `event: state.delta` | ❌ 未处理 |
| `message_delta` | `{Content:{TextPart}, Author, Partial}` | `{Type:"message.delta", Text, Author}` | `event: message.delta` | ✅ `message.delta` |
| `message_end` | `{Content:{TextPart}, Author, Final}` | `{Type:"message.end", Text, Author}` | `event: message.end` | ✅ `message.end` |
| `run_finished` | `{Actions:{StateDelta}, Author, Final}` | `{Type:"state.delta", StateDelta, Final}` | `event: state.delta` | ❌ 未处理 |
| `run_error` | yield error (not adk.Event) | N/A (Gateway 调用 WriteError) | `event: error` | ✅ `error` |
| — | — | `{Type:"tool.call", ToolCall}` | `event: tool.call` | ✅ `tool.call` |
| — | — | `{Type:"artifact.delta", Artifact}` | `event: artifact.delta` | ✅ `artifact.delta` |

---

## 3. 关键文件清单

### 3.1 前端文件

| 文件 | 职责 | 审计重点 |
|---|---|---|
| `frontend/src/agui/client.ts` | SSE 客户端, 请求发送, 流解析 | SSE 粘包/拆包, JSON 解析, 错误处理 |
| `frontend/src/agui/events.ts` | sendMessage hook | — |
| `frontend/src/agui/skills.ts` | Runtime Capability 注册表 | 仅含 `code_preview` |
| `frontend/src/stores/messageStore.ts` | 事件分发, 消息聚合, 状态管理 | 多 Agent 消息聚合核心逻辑 |
| `frontend/src/stores/agentStore.ts` | Agent 列表, 选项管理 | AgentName 硬编码 |
| `frontend/src/stores/conversationStore.ts` | Conversation CRUD | — |
| `frontend/src/services/api.ts` | REST API 调用 | 仅调 `/api/*` |
| `frontend/src/types/index.ts` | TypeScript 类型定义 | AGUIEvent, Message 等 |
| `frontend/src/lib/agents.ts` | Agent 名称/选项工具函数 | `isSupportedAgentName` 硬编码 |
| `frontend/src/components/MessageBubble.tsx` | 消息气泡渲染 | senderName 展示 |
| `frontend/src/components/StreamingText.tsx` | 流式文本 + Markdown 渲染 | react-markdown |
| `frontend/src/components/CodePreview.tsx` | 代码预览组件 | 语法高亮, 不执行 |
| `frontend/src/components/WebPreview.tsx` | Web 预览 Safe Mode | HTML 文本化, 无 iframe |
| `frontend/src/components/ChatWindow.tsx` | 聊天窗口主组件 | Agent 选择器 |
| `frontend/src/components/ErrorBoundary.tsx` | 错误边界 | — |

### 3.2 Gateway 文件

| 文件 | 职责 | 审计重点 |
|---|---|---|
| `services/gateway/httpapi/server.go` | Gateway HTTP API, `/api/chat` handler | 请求 schema, SSE 流处理 |
| `services/gateway/sse/sse.go` | SSE writer | 事件序列化, flush |
| `services/gateway/orchestratorclient/client.go` | Orchestrator SSE 消费, adk.Event 映射 | messageId/runId/sender 丢失 |
| `services/gateway/runservice/routing.go` | 静态路由 (fallback 模式) | — |
| `services/gateway/cmd/gateway/main.go` | Gateway 启动, 配置加载 | agent endpoints 硬编码 |

### 3.3 Orchestrator 文件

| 文件 | 职责 | 审计重点 |
|---|---|---|
| `services/orchestrator/httpapi/handler_run_stream.go` | Orchestrator SSE handler, 事件序列化 | OrchestratorStreamEvent schema |
| `services/orchestrator/executor/ordered_parallel_executor.go` | ordered_parallel 执行器 | sender/agentName 注入, summary 生成 |
| `services/orchestrator/executor/single_executor.go` | single 执行器 | sender 注入 |

### 3.4 共享基础设施

| 文件 | 职责 | 审计重点 |
|---|---|---|
| `pkg/runtime/agui/events.go` | agui.Event DTO | 非标准字段 (author/text 而非 sender/delta) |
| `pkg/runtime/agui/translator.go` | adk.Event → agui.Event 转换 | 非标准事件类型名 |
| `pkg/runtime/agui/filter.go` | 文本安全过滤 | secret/token/stack 过滤 |

---

## 4. 当前支持能力

### 4.1 前端 API 调用路径

| 审计项 | 状态 | 说明 |
|---|---|---|
| 前端仅调 Gateway `/api/**` | ✅ 通过 | `api.ts` 所有路由使用 `/api` base |
| 无直连 Orchestrator | ✅ 通过 | 无 Orchestrator URL 引用 |
| 无直连 Agent | ✅ 通过 | 无 Agent URL 引用 |
| `/api/chat` 使用 Gateway schema | ✅ 通过 | `conversationId + message + optional agentName` |

### 4.2 SSE 接收与解析

| 审计项 | 状态 | 说明 |
|---|---|---|
| SSE parser 位置 | ✅ 通过 | `client.ts` splitSSEBlocks + parseSSEBlock |
| 按 `\n\n` 拆分 block | ✅ 通过 | `splitSSEBlocks` |
| 处理粘包 (tail buffer) | ✅ 通过 | buffer rest 保留 |
| 处理拆包 (跨 chunk) | ✅ 通过 | stream:true decode + buffer append |
| 识别 `event:` / `data:` 行 | ✅ 通过 | `parseSSEBlock` |
| 安全跳过 malformed JSON | ✅ 通过 | try/catch 跳过 |
| `delta` 字段兼容 | ✅ 通过 | `event.delta ?? event.content ?? event.text` |
| `author` 字段识别 | ✅ 通过 | `resolveEventAgentName` 检查 `event.author` |
| `senderName` 字段识别 | ✅ 通过 | `resolveEventSenderName` 检查 `event.senderName` |
| `messageId` 识别 | ✅ 通过 | `ensureAgentMessage` 使用 `event.messageId \|\| event.id` |
| `runId` 字段 | ⚠ 存在但未用 | AGUIEvent 有 `runId` 但 messageStore 未使用 |

### 4.3 Runtime Preview / Artifact

| 审计项 | 状态 | 说明 |
|---|---|---|
| CodePreview 组件 | ✅ 通过 | 语法高亮, 复制, 不执行 |
| WebPreview Safe Mode | ✅ 通过 | HTML 文本化渲染 (`<pre><code>`), 无 iframe |
| Markdown 渲染 | ✅ 通过 | react-markdown + remarkGfm, streaming 期间纯文本 |
| `code_preview` tool 处理 | ✅ 通过 | `handleToolPayload` → `appendCodePreview` |
| `web_preview` tool 处理 | ✅ 通过 | `handleToolPayload` → `appendWebPreview` |
| `artifact.delta` 处理 | ✅ 部分 | code/webpage 类型, 不支持 markdown/image 等 |
| `frontendSkills` 注册表 | ⚠ 不完整 | 仅 `['code_preview']`, 缺 `web_preview` `markdown_render` |

### 4.4 错误展示

| 审计项 | 状态 | 说明 |
|---|---|---|
| `RUN_ERROR` / `error` 处理 | ✅ 通过 | 标记 failed, 停止 streaming |
| 错误消息脱敏 | ✅ 通过 | Gateway WriteError 使用 filter, 仅显示 "assistant run failed" |
| 内部信息不泄露 | ✅ 通过 | filter.go 过滤 secret/token/stack/path |
| 用户可理解失败提示 | ✅ 通过 | "Failed to generate response" |

### 4.5 测试现状

| 审计项 | 状态 | 说明 |
|---|---|---|
| `npm test` (vitest) | ✅ 存在 | client.test.ts, messageStore.test.ts, CodePreview.test.tsx, WebPreview.test.tsx |
| `npm run build` | ✅ 存在 | `tsc -b && vite build` |
| e2e (Playwright) | ✅ 存在 | `mvp-happy-path.spec.ts` |
| SSE 解析测试 | ✅ 存在 | client.test.ts: 粘包/拆包/malformed JSON/空 block |
| Web preview safety 测试 | ✅ 存在 | WebPreview.test.tsx: 确认 Safe Mode |
| run_error 测试 | ✅ 存在 | e2e mocks.ts buildErrorSSE, mvp-happy-path "error handling" |
| CodePreview 测试 | ✅ 存在 | CodePreview.test.tsx: 6 个测试用例 |
| Multi-agent SSE 测试 | ❌ 不存在 | 无混合 Agent 场景的 SSE 测试 |
| STATE_UPDATE 测试 | ❌ 不存在 | 无 state.delta / STATE_UPDATE 处理测试 |

---

## 5. 缺口列表

### 5.1 P0 缺口（导致 mixed ordered_parallel 在 UI 上不可用）

#### P0-1: 多 Agent 消息被合并为单一气泡

**严重程度:** P0

**根因:** Orchestrator 输出的 `message_start` 事件（携带 `messageId` 和 `sender`）被 orchestratorclient 转换为 `adk.Event{Actions:{StateDelta:{messageId:...}}}`，再被 agui.Translator 转为 `Event{Type:"state.delta"}`。前端 messageStore 不处理 `state.delta` 类型（switch 中无匹配 case），因此 `message_start` 信息被完全丢弃。

同时，`message_delta` 和 `message_end` 事件的 `messageId` 在 orchestratorclient 转换时丢失（不传入 adk.Event），导致前端 `ensureAgentMessage` 无法通过 `messageId` 变化来区分不同 Agent 的消息。

**影响:**
- single code 场景: 通过 E2E mock 可用，但真实 Gateway 输出的事件类型不同
- single web 场景: 同上
- mixed ordered_parallel 场景: web-agent、code-agent、orchestrator summary 的全部内容被合并到一个 assistant 气泡中
- 不同 sender 无法区分，显示错误的 sender 名称

**涉及文件:**
- `services/gateway/orchestratorclient/client.go` — messageId 未传入 adk.Event (line 220-231)
- `pkg/runtime/agui/translator.go` — 不生成 TEXT_MESSAGE_START (line 41-105)
- `pkg/runtime/agui/events.go` — 无 messageId/runId/sender 字段 (line 4-16)
- `frontend/src/stores/messageStore.ts` — 不处理 state.delta (line 500-595)

#### P0-2: Gateway 不产生 AG-UI 标准事件类型

**严重程度:** P0

**根因:** agui.Translator 产生的是非标准事件类型名（`message`, `message.delta`, `message.end`, `state.delta`, `tool.call`, `artifact.delta`），而非 AG-UI 契约规定的 UPPER_SNAKE_CASE 事件类型（`TEXT_MESSAGE_START`, `TEXT_MESSAGE_CONTENT`, `TEXT_MESSAGE_END`, `STATE_UPDATE`, `TOOL_CALL_START/ARGS/END`, `RUN_STARTED`, `RUN_FINISHED`, `RUN_ERROR`）。

前端 messageStore 同时兼容了两种格式，但 E2E mock 使用的是标准 AG-UI 事件类型，因此 E2E 通过不代表真实运行时可用。

**影响:**
- E2E 测试与真实运行时行为不一致
- `RUN_STARTED` / `RUN_FINISHED` 事件不被 Gateway 产生
- `STATE_UPDATE` 事件不被 Gateway 产生
- `TEXT_MESSAGE_START` 事件不被 Gateway 产生
- 烟雾测试通过但前端实际不可用

**涉及文件:**
- `pkg/runtime/agui/translator.go` — 事件类型命名
- `pkg/runtime/agui/events.go` — DTO 字段名
- `services/gateway/sse/sse.go` — SSE 输出

#### P0-3: messageId / runId 在 translator 链路中丢失

**严重程度:** P0

**根因:** 
1. Orchestrator → orchestratorclient: `messageId` 仅在 `message_start` 的 StateDelta 中保留，`message_delta`/`message_end` 的 `messageId` 不传入 adk.Event
2. orchestratorclient → agui.Translator: adk.Event 不携带 messageId/runId 字段
3. agui.Event 结构体无 messageId/runId/threadId 字段

**影响:**
- 前端无法按 messageId 聚合多 Agent 消息
- 前端无法按 runId 管理运行生命周期
- 无法关联 message 和 tool call

**涉及文件:**
- `services/gateway/orchestratorclient/client.go` — line 206-248, messageId/runId 丢失
- `pkg/runtime/agui/events.go` — 无 messageId/runId 字段
- `pkg/runtime/agui/translator.go` — 无 messageId/runId 透传逻辑

#### P0-4: sender 对象丢失，降级为 author 字符串

**严重程度:** P0

**根因:** Orchestrator 输出的 `sender: {type: "agent", name: "web-agent"}` 被 orchestratorclient 降级为 `currentAuthor` 字符串（仅保留 name）。agui.Event 使用 `Author` 字符串字段而非 `sender` 对象。前端 `AGUIEvent` 类型有 `author` 和 `senderName` 字段，但无 `sender` 对象。

**影响:**
- 无法区分 sender.type（agent vs orchestrator vs user）
- 前端只能通过 agentName 回退推断 sender 显示名
- 未来多类型 sender（如 system, tool）无法区分

**涉及文件:**
- `services/gateway/orchestratorclient/client.go` — line 188-190, sender 降级
- `pkg/runtime/agui/events.go` — Author 而非 sender
- `frontend/src/types/index.ts` — 无 sender 对象字段

### 5.2 P1 缺口（影响多 Agent 可读性、调试性或体验）

#### P1-1: state.delta / STATE_UPDATE 事件被前端忽略

**严重程度:** P1

**根因:** Gateway 产出 `state.delta` 事件（携带 planId, phase, messageId, status 等信息），但前端 messageStore 的 switch 不匹配 `state.delta` 或 `STATE_UPDATE`，事件被静默丢弃。

**影响:**
- 用户看不到编排进度（planning/dispatching/agent_streaming）
- planId 无法在前端展示或调试
- message_start 信息（messageId, status）被丢弃
- run_started/run_finished 状态无法显示

**涉及文件:**
- `frontend/src/stores/messageStore.ts` — 缺少 state.delta/STATE_UPDATE case

#### P1-2: frontendSkills 注册表不完整

**严重程度:** P1

**根因:** `frontend/src/agui/skills.ts` 仅注册 `['code_preview']`，缺少 `web_preview` 和 `markdown_render`。

**影响:**
- `web_preview` 和 `markdown_render` 能力不在注册表中
- 未来校验 toolName 是否已注册时会失败
- 违反 `frontend-runtime-skills-contract` 的 Runtime Capability Registry 要求

**涉及文件:**
- `frontend/src/agui/skills.ts`

#### P1-3: 前端通过 agentName 决定渲染逻辑

**严重程度:** P1

**根因:** `messageStore.ts` 的 `appendWebPreviewFromMessageContent` 函数显式检查 `currentAgentName !== 'web-agent'`，这是通过 agentName 决定组件行为的明确违规。

`lib/agents.ts` 的 `isSupportedAgentName` 硬编码了 `'code-agent' | 'web-agent'`。

**影响:**
- 违反 `frontend-runtime-skills-contract` 的核心原则："禁止通过 agentName 决定组件或能力"
- 新增 Agent 时前端需要修改代码
- 违反了 productization-stage-guide 的 section 8.5 禁止项

**涉及文件:**
- `frontend/src/stores/messageStore.ts` — line 393-398
- `frontend/src/lib/agents.ts` — line 112-114

#### P1-4: Agent 名称硬编码

**严重程度:** P1

**根因:** `lib/agents.ts` 的 `AGENT_OPTIONS` 和 `knownDisplayNameMap` 硬编码了 `'code-agent'` 和 `'web-agent'`。`isSupportedAgentName` 类型守卫硬编码了这两个名称。Gateway `main.go` 的 `loadRuntimeConfigFromEnv` 硬编码了 agent endpoints。

**影响:**
- 新增 Agent 需要修改前端代码
- 前端不认识 `orchestrator` 作为 sender（摘要消息的 sender）
- `orchestrator` 显示名被 humanize 为 "Orchestrator" 而非预定义显示名

**涉及文件:**
- `frontend/src/lib/agents.ts`
- `services/gateway/cmd/gateway/main.go`

#### P1-5: 无 RUN_STARTED / RUN_FINISHED 事件输出

**严重程度:** P1

**根因:** agui.Translator 不产生 `RUN_STARTED` / `RUN_FINISHED` 事件。Orchestrator 的 `run_started` / `run_finished` 被转换为 `state.delta`。

**影响:**
- 前端无法通过 `RUN_STARTED` 初始化运行状态显示
- 前端无法通过 `RUN_FINISHED` 精确结束 streaming（当前依赖 ReadableStream 的 onComplete 回调）
- 违反 AG-UI 协议的 Run 生命周期事件要求
- `RUN_FINISHED` case 在 messageStore 中存在但永不触发

**涉及文件:**
- `pkg/runtime/agui/translator.go`
- `services/gateway/orchestratorclient/client.go`

#### P1-6: Orchestrator 状态/调试信息不可见

**严重程度:** P1

**根因:** `run_started` 携带 `state: {phase, planId, validated}` 但前端不展示。taskId 不在 AGUIEvent 类型中。无任何调试入口可以查看 planId / taskId。

**影响:**
- 开发者无法在前端调试编排计划
- 用户看不到编排进度
- 违反 `observability-debugging-contract` 的前端可观测性要求

#### P1-7: Smoke 脚本通过但前端不可用的假阳性

**严重程度:** P1

**根因:** Smoke 脚本检查 Gateway SSE 的原始文本（搜索 `"author":"web-agent"` 等模式），这些检查能通过。但前端实际渲染取决于 messageStore 的事件处理逻辑，而 messageStore 无法正确处理当前 Gateway 输出的事件类型。

**影响:**
- Smoke 通过给人"前端可用"的假象
- 实际前端无法正确显示 mixed ordered_parallel 的多 Agent 输出

### 5.3 P2 缺口（后续 Artifact / Runtime Skill / 持久化增强项）

#### P2-1: 缺少结构化 Artifact 事件处理

**严重程度:** P2

**说明:** 当前 `artifact.delta` 仅处理 `code` 和 `webpage`/`html` 类型。缺失:
- `markdown` artifact 类型
- `image` / `image_ref` artifact 类型
- `file` / `file_summary` artifact 类型
- `vision_analysis` artifact 类型
- `artifact.type` → `previewType` → `toolName` → component 的完整映射链
- artifact `contentRef` 处理（引用外部存储而非内联内容）

#### P2-2: 缺少 Tool Call 结果事件处理

**严重程度:** P2

**说明:** agui.Translator 产生 `tool.result` 事件，但前端 messageStore 不处理此类型。后续如果需要展示 Tool Call 的执行结果（成功/失败），需要添加此处理。

#### P2-3: 缺少 Markdown Render 组件注册

**严重程度:** P2

**说明:** `frontendSkills` 不包含 `markdown_render`。虽然 StreamingText 使用 react-markdown 渲染最终内容，但没有作为显式 Runtime Capability 注册。后续 Milestone 5 (Artifact 与 Runtime Preview) 需要此能力。

#### P2-4: 缺少持久化相关字段处理

**严重程度:** P2

**说明:** 当前前端不处理 `threadId`, `timestamp`, `traceId` 等持久化相关字段。后续 Milestone 4 (Conversation/Run/Message 持久化) 需要这些字段来保证消息可恢复和可审计。

#### P2-5: E2E 测试不覆盖真实 Gateway 事件格式

**严重程度:** P2

**说明:** E2E mock 使用标准 AG-UI 事件类型（TEXT_MESSAGE_START, TEXT_MESSAGE_CONTENT 等），而真实 Gateway 输出不同的事件类型。需要有一致性测试确保 E2E mock 与真实 Gateway 输出对齐。

---

## 6. 风险等级汇总

### P0: 导致 mixed ordered_parallel 在 UI 上不可用或安全风险

| 编号 | 缺口 | 影响 |
|------|------|------|
| P0-1 | 多 Agent 消息合并为单一气泡 | mixed 场景完全不可用 |
| P0-2 | Gateway 不产��� AG-UI 标准事件 | E2E 假阳性，真实运行时行为与预期不符 |
| P0-3 | messageId/runId 链路丢失 | 无法按 messageId/runId 聚合和管理消息 |
| P0-4 | sender 对象丢失 | 无法正确区分不同 sender |

### P1: 影响多 Agent 可读性、调试性或体验

| 编号 | 缺口 | 影响 |
|------|------|------|
| P1-1 | state.delta/STATE_UPDATE 被忽略 | 编排进度不可见 |
| P1-2 | frontendSkills 注册表不完整 | web_preview 未注册 |
| P1-3 | 通过 agentName 决定渲染 | 违反契约, 新增 Agent 需改前端 |
| P1-4 | Agent 名称硬编码 | 扩展性差 |
| P1-5 | 无 RUN_STARTED/RUN_FINISHED | 生命周期事件缺失 |
| P1-6 | Orchestrator 状态不可见 | 不可调试 |
| P1-7 | Smoke 假阳性 | 误判前端可用 |

### P2: 后续 Artifact / Runtime Skill / 持久化增强项

| 编号 | 缺口 | 影响 |
|------|------|------|
| P2-1 | 缺结构化 Artifact 事件处理 | 完整 Artifact pipeline 未建立 |
| P2-2 | 缺 Tool Call 结果处理 | tool.result 被忽略 |
| P2-3 | 缺 markdown_render 注册 | Runtime Capability 注册表不完整 |
| P2-4 | 缺持久化字段处理 | 后续持久化 Milestone 的前置条件 |
| P2-5 | E2E 不覆盖真实 Gateway 格式 | E2E 与运行时不一致 |

---

## 7. 下一步：是否需要改代码

### 结论：需要。P0 缺口必须修复后才能进入下一 Milestone。

### 建议拆分为以下小任务：

#### Task 1: 修复 agui.Event DTO 字段对齐 AG-UI 契约 (P0-2, P0-3, P0-4)
- **修改文件:** `pkg/runtime/agui/events.go`
- **内容:**
  - 新增 `messageId`, `runId`, `threadId`, `timestamp`, `traceId` 字段
  - 将 `text` 改为 `delta`，保留 `content` 兼容
  - 将 `author` 改为 `sender` 对象 `{type, name, displayName}`
  - `id` 保留作为 toolCallId 使用
- **风险:** 低（向后兼容旧字段）

#### Task 2: 修复 agui.Translator 产生 AG-UI 标准事件类型 (P0-2, P0-5)
- **修改文件:** `pkg/runtime/agui/translator.go`
- **内容:**
  - `message`/`message.delta` → `TEXT_MESSAGE_CONTENT`
  - `message.end` → `TEXT_MESSAGE_END`
  - `state.delta` → `STATE_UPDATE`
  - `tool.call` → `TOOL_CALL_START/ARGS/END`（拆分）
  - `artifact.delta` → `TOOL_CALL_START/ARGS/END`（按 artifact.type 映射 toolName）
  - 新增 `RUN_STARTED` 生成
  - 新增 `RUN_FINISHED` 生成
  - 新增 `TEXT_MESSAGE_START` 生成（从 message_start 的 StateDelta 信息）
- **风险:** 中（影响所有现有事件流，需要全部测试验证）

#### Task 3: 修复 orchestratorclient 完整传递 messageId/runId/sender (P0-1, P0-3, P0-4)
- **修改文件:** `services/gateway/orchestratorclient/client.go`
- **内容:**
  - 在 adk.Event 中通过 metadata 或扩展字段传递 messageId
  - 传递 runId
  - 传递完整的 sender 对象（而非仅 name）
- **风险:** 低（adk.Event 字段扩展）

#### Task 4: 前端 messageStore 支持多 Agent 消息分气泡 (P0-1)
- **修改文件:** `frontend/src/stores/messageStore.ts`
- **内容:**
  - 处理 `STATE_UPDATE` / `state.delta` 事件
  - 在收到 `TEXT_MESSAGE_START` (或从 STATE_UPDATE 中解析 messageId 变化的 message_start) 时创建新消息气泡
  - 确保不同 sender 的消息不被合并
  - 展示 sender 信息
- **风险:** 中（核心聚合逻辑变更）

#### Task 5: 前端支持 STATE_UPDATE 展示编排进度 (P1-1, P1-6)
- **修改文件:** `frontend/src/stores/messageStore.ts`, 新增状态组件
- **内容:**
  - 添加 `STATE_UPDATE` case 处理
  - 解析 `phase`, `planId`, `activeAgent` 等信息
  - 渲染编排进度条或状态提示
- **风险:** 低（纯增量，不影响现有逻辑）

#### Task 6: 修复 frontendSkills 注册表 (P1-2)
- **修改文件:** `frontend/src/agui/skills.ts`
- **内容:**
  - 添加 `web_preview`
  - 添加 `markdown_render`
- **风险:** 极低

#### Task 7: 移除 agentName 硬编码逻辑 (P1-3, P1-4)
- **修改文件:** `frontend/src/stores/messageStore.ts`, `frontend/src/lib/agents.ts`
- **内容:**
  - 移除 `appendWebPreviewFromMessageContent` 中的 `currentAgentName !== 'web-agent'` 检查
  - 移除 `isSupportedAgentName` 类型守卫
  - 将 AGENT_OPTIONS 改为从 `/api/agents` 动态加载
  - 使用 toolName/artifact.type 而非 agentName 决定渲染
- **风险:** 中（需确保不破坏现有 CodePreview/WebPreview 功能）

#### Task 8: 对齐 E2E mock 与真实 Gateway 输出 (P2-5)
- **修改文件:** `frontend/e2e/mocks.ts`
- **内容:**
  - 确保 E2E mock 输出事件类型与修复后的 Gateway 一致
  - 添加 mixed ordered_parallel E2E 测试
- **风险:** 低

#### Task 9: 前端补全多 Agent SSE 测试
- **新增文件:** `frontend/src/stores/messageStore.multi-agent.test.ts`
- **内容:**
  - 测试 mixed ordered_parallel 场景的消息聚合
  - 测试不同 sender 创建不同气泡
  - 测试 STATE_UPDATE 处理
- **风险:** 低

---

## 8. 禁止项自查

| 禁止项 | 状态 | 说明 |
|---|---|---|
| 不修改 frontend 代码 | ✅ 遵守 | 本步仅审计 |
| 不修改 Gateway / Orchestrator 代码 | ✅ 遵守 | 本步仅审计 |
| 不修改 smoke | ✅ 遵守 | 本步仅审计 |
| 不修改 CI | ✅ 遵守 | 本步仅审计 |
| 不新增测试 | ✅ 遵守 | 本步仅审计 |
| 只产出审计文档 | ✅ 遵守 | 本文件 |

---

## 9. 修改文件清单

本步仅新增 1 个文件：

- `docs/refactor/frontend-multi-agent-sse-audit.md`（本文件）

未修改任何其他文件。

---

## 10. 审计文档结构摘要

1. **审计范围与目的** — Step 1-D 定位与禁止项
2. **当前前端 SSE 链路图** — 完整事件流（含事件类型转换表）
3. **关键文件清单** — 前端/Gateway/Orchestrator 关键文件
4. **当前支持能力** — 5 个维度的现状评估
5. **缺口列表** — 4 个 P0, 7 个 P1, 5 个 P2
6. **风险等级汇总** — P0/P1/P2 缺口摘要表
7. **下一步建议** — 9 个子任务拆分
8. **禁止项自查**
9. **审计命令**

---

## 11. P0/P1/P2 缺口摘要

### P0 (4 项) — 必须修复才能进入下一 Milestone
1. **多 Agent 消息合并为单一气泡** — state.delta 被忽略, messageId 丢失
2. **Gateway 不产生 AG-UI 标准事件** — 非标准事件类型名
3. **messageId/runId 链路丢失** — translator 层不传递
4. **sender 对象丢失** — 降级为 author 字符串

### P1 (7 项) — 影响体验、可读性、调试性
1. state.delta 事件被忽略
2. frontendSkills 注册表不完整
3. 通过 agentName 决定渲染逻辑
4. Agent 名称硬编码
5. 无 RUN_STARTED/RUN_FINISHED
6. Orchestrator 状态不可见
7. Smoke 假阳性

### P2 (5 项) — 后续 Milestone 增强
1. 缺结构化 Artifact 处理
2. 缺 Tool Call 结果处理
3. 缺 markdown_render 注册
4. 缺持久化字段处理
5. E2E 不覆盖真实 Gateway 格式

---

## 12. 是否建议进入前端修复步骤

**是。** 当前 P0 缺口表明 mixed ordered_parallel 场景在真实运行时前端不可用。必须先完成 Task 1-4（修复 agui.Event DTO、agui.Translator、orchestratorclient、messageStore）才能进入 Milestone 4 (持久化)。

建议完成 Task 1-7 后再进入下一 Milestone。

---

## 13. 审核命令

```bash
echo "## changed files"
git status --short

echo "## changed file names"
git diff --name-only

echo "## should only add frontend audit doc"
git diff --name-only | grep -vE "^docs/refactor/frontend-multi-agent-sse-audit\.md$" && exit 1 || true

echo "## audit doc key refs"
grep -n "SSE\|sender\|author\|code-agent\|web-agent\|orchestrator summary\|run_error\|Artifact\|WebPreview\|P0\|P1\|P2" docs/refactor/frontend-multi-agent-sse-audit.md | head -200
```

---

## 14. Commit 建议

```text
docs: add frontend multi-agent SSE audit report (Step 1-D)

P0 gaps found: multi-agent messages merged into single bubble,
messageId/runId/sender lost in translator chain, Gateway outputs
non-standard AG-UI event types. Frontend requires code changes
before Milestone 4 (Persistence).

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
```

---

## 15. 版本

- **审计日期:** 2026-06-03
- **审计范围:** AgentHub v1.0 Productization Stage Step 1-D
- **审计方法:** 纯静态代码审计，不修改代码，不运行测试
- **前置步骤:** Step 1-A, 1-B, 1-C 已通过
