# AgentHub 仓库调研与联调修复计划

> **调研日期**: 2026-06-06
> **调研分支**: `g`
> **调研范围**: 只读，不修改、不重构、不提交

---

## 1. Skills 读取结果

### 已读取的 13 个 Skills

| # | Skill | 核心约束 |
|---|-------|----------|
| 1 | project-architecture | 新架构 5 服务拓扑是唯一开发目标；`server/` 和 `agents/` 为遗留代码，禁止修改 |
| 2 | frontend-runtime-skills-contract | 前端只通过 Gateway 访问后端，禁止直连 Orchestrator 或 Agent |
| 3 | gateway-orchestrator-contract | Gateway↔Orchestrator 目标协议是 gRPC streaming，HTTP/SSE 是临时兼容 |
| 4 | ai-collaboration-workflow | 契约优先开发，先改契约再实现 |
| 5 | a2a-agent-contract | AgentCard、A2A task 传输协议，Orchestrator dispatcher 规范 |
| 6 | llm-provider-contract | LLM provider 只能在 `pkg/runtime/model` 和 `pkg/runtime/registry`，禁止放 `pkg/adk` |
| 7 | data-persistence-contract | MySQL 是目标持久化，SQLite 仅用于 demo/profile；Gateway 负责持久化 |
| 8 | docker-compose-delivery | 6 服务拓扑：frontend/gateway/orchestrator/code-agent/web-agent/mysql |
| 9 | testing-review-contract | 接口合规测试和冒烟测试是阻塞项 |
| 10 | security-boundary-contract | 禁止 secret 泄露、禁止浏览器直连 Agent、AgentCard 不含 secret |
| 11 | observability-debugging-contract | 跨 Gateway/Orchestrator/Agent 的追踪和日志规范 |
| 12 | code-style-and-conventions | Go 新代码在 `pkg/*` 和 `services/*`，非 `server/` 或 `agents/` |
| 13 | commit-security-review | 提交前检查 diff 中的 secrets、AgentCard 安全 |

### 技能对本次调研的关键约束

1. **项目架构识别**: 新架构是唯一开发目标；`pkg/adk` + `pkg/runtime` + `services/gateway` + `services/orchestrator` + `services/agents/*` + `frontend`
2. **前端运行时约定**: 前端只调用 Gateway REST API + SSE，不直连其他服务
3. **Agent/A2A/Orchestrator 边界**: Orchestrator 是唯一编排层（Planner→Validator→Executor→Dispatcher→Registry），Agent 是 A2A 子代理
4. **LLM provider 配置**: 只能在 `pkg/runtime/model` 和 `pkg/runtime/registry`，不在 `pkg/adk`
5. **数据持久化**: MySQL 目标，SQLite demo-only；Gateway 负责 conversation/message 持久化
6. **Docker/Compose**: 6 服务拓扑，目标 gRPC streaming，MySQL 目标
7. **测试安全**: 冒烟测试阻塞项，错误信息需要 sanitize
8. **本轮禁止修改**: 所有 `server/`、`agents/` 遗留路径、`.env` 文件、任何业务代码

---

## 2. 调研结论摘要

### 仓库类型
**Monorepo**（Go workspace + React frontend），含 5 个活跃服务 + 2 个遗留服务目录。

### 架构状态
- 新架构 5 服务已基本搭建完成，但 **Agent 均为 mock 实现**，不调用真实 LLM
- Docker 构建效率低：**4 个 Go Dockerfile 全部 COPY 整个 monorepo**
- 前端 SSE 流式传输基础链路已打通，但 **UI 体验不完整**（无自动编排、Web Preview 仅显示源码、无会话标题自动生成）
- Orchestrator 编排逻辑已实现（RulePlanner + LLMPlanner），但 **编排元数据直接暴露给前端**

### 联调问题严重程度
- **P0（阻塞演示）**: 无自动编排入口、Web Preview 不可用、无 streaming 感知、会话标题全部 "New Conversation"
- **P1（编排正确性）**: 编排过程污染用户输出、Web Preview 被 Code Agent 触发
- **P2（工程化）**: Docker 缓存失效、GOPROXY 配置、长上下文截断

---

## 3. 仓库结构

### 一级目录

```
Multi_Agent-AgentHub/
├── frontend/                 # React 前端 (新架构)
├── services/                 # 新架构后端服务
│   ├── gateway/              # Gateway - 公共 API、SSE、auth、CORS、持久化
│   ├── orchestrator/         # Orchestrator - 规划/路由/执行/分发
│   └── agents/
│       ├── code-agent/       # Code Agent A2A 子代理
│       └── web-agent/        # Web Agent A2A 子代理
├── pkg/                      # 共享库
│   ├── adk/                  # ADK 引擎 (A2A client/server, agent model, tools, runner)
│   └── runtime/              # Runtime 框架 (AG-UI, config, launcher, model providers)
├── server/                   # [遗留] 旧 Gateway + 嵌入式 Orchestrator
├── agents/                   # [遗留] 21 个旧 Agent 实现
├── docs/                     # 契约文档、设计规范
│   └── contracts/            # 37 个契约文件
├── .claude/skills/           # 18 个 Claude Skills
├── docker-compose.new-arch.yml  # 新架构 Compose（5 服务）
├── docker-compose.yml        # 遗留 Compose
├── go.work                   # Go workspace
├── Makefile
├── CLAUDE.md
└── AGENTS.md
```

### 模块依赖关系

```
Frontend (React, :3000)
  → Gateway (Go, :8080) — REST API + SSE
    → Orchestrator (Go, :8090) — HTTP/SSE (目标 gRPC)
      → RulePlanner / LLMPlanner
      → Validator
      → Executor (Single / OrderedParallel)
        → A2A Dispatcher
          → code-agent (Go, :8081) — A2A JSON-RPC
          → web-agent  (Go, :8082) — A2A JSON-RPC
```

### Go Workspace 模块 (`go.work`)

```
go 1.26.0
use (
    ./agents                        # 遗留
    ./server                        # 遗留
    ./pkg/adk
    ./pkg/runtime
    ./services/gateway
    ./services/agents/code-agent
    ./services/agents/web-agent
    ./services/orchestrator
)
```

---

## 4. 前端模块说明

### 4.1 技术栈

- **框架**: React 18 + TypeScript
- **构建**: Vite 6
- **状态管理**: Zustand 4
- **Markdown 渲染**: react-markdown + remark-gfm
- **代码高亮**: highlight.js
- **CSS**: Tailwind CSS 3
- **测试**: Vitest + Playwright

### 4.2 启动命令

```bash
cd frontend && npm run dev    # :5173，/api 代理到 localhost:8080
```

### 4.3 关键文件路径

| 模块 | 文件路径 |
|------|----------|
| 入口 | `frontend/src/main.tsx` |
| App 根组件 | `frontend/src/App.tsx` |
| 聊天布局 | `frontend/src/components/ChatLayout.tsx` |
| 聊天窗口 | `frontend/src/components/ChatWindow.tsx` |
| 消息输入 | `frontend/src/components/MessageInput.tsx` |
| 消息气泡 | `frontend/src/components/MessageBubble.tsx` |
| 流式文本渲染 | `frontend/src/components/StreamingText.tsx` |
| Code 预览 | `frontend/src/components/CodePreview.tsx` |
| Web 预览 | `frontend/src/components/WebPreview.tsx` |
| 编排卡片 | `frontend/src/components/OrchestrationCard.tsx` |
| 会话列表 | `frontend/src/components/ConversationList.tsx` |
| Agent 头像 | `frontend/src/components/AgentAvatar.tsx` |
| API 层 | `frontend/src/services/api.ts` |
| SSE 客户端 | `frontend/src/agui/client.ts` |
| SSE 事件 Hook | `frontend/src/agui/events.ts` |
| 消息 Store | `frontend/src/stores/messageStore.ts` |
| 会话 Store | `frontend/src/stores/conversationStore.ts` |
| Agent Store | `frontend/src/stores/agentStore.ts` |
| Agent 配置 | `frontend/src/lib/agents.ts` |
| 类型定义 | `frontend/src/types/index.ts` |

### 4.4 当前功能状态

| 功能 | 状态 | 说明 |
|------|------|------|
| SSE streaming | 已实现 | `fetch` + `ReadableStream` 在 `agui/client.ts:29` |
| Stop/Abort | 已实现 | `AbortController` 在 `messageStore.ts:767` |
| Loading 状态 | 部分 | StreamingText 有 "Thinking..." 动画；输入框在 streaming 时 disabled |
| Code block 复制 | **已实现** | `CodePreview.tsx:28` 有 `handleCopy` + Clipboard API |
| Agent 选择 | 仅强制选择 | `ChatWindow.tsx:89` 的 `<select>` 只有 code-agent 和 web-agent，无 "自动" 选项 |
| Web Preview | **不可用** | 仅以 `<pre><code>` 显示原始 HTML，无 sandbox iframe 渲染 |
| 会话标题 | **全部 "New Conversation"** | `api.ts:44` hardcode 默认值，ConversationList 显示 `conv.title \|\| 'New Conversation'` |

### 4.5 前端 API 端点

| 端点 | 方法 | 请求体 | 响应 |
|------|------|--------|------|
| `/api/conversations` | GET | - | `Conversation[]` |
| `/api/conversations` | POST | `{userId, agentName}` | `Conversation` |
| `/api/conversations/:id/messages` | GET | - | `Message[]` |
| `/api/agents` | GET | - | `Agent[]` |
| `/api/chat` | POST | `{conversationId, message, agentName?}` | SSE stream |

### 4.6 SSE 事件类型（前端处理）

前端 `messageStore.ts` 处理的事件类型：
- `TEXT_MESSAGE_START` — 创建 agent 消息气泡
- `TEXT_MESSAGE_CONTENT` / `message` / `message.delta` — 追加文本 delta
- `TEXT_MESSAGE_END` / `message.end` — 完成流式，设置 status='sent'
- `RUN_STARTED` — 记录 run 开始
- `RUN_FINISHED` — 设置 streaming=false
- `RUN_ERROR` / `error` — 错误处理 + 敏感信息过滤
- `STATE_UPDATE` / `state.delta` — 提取编排信息（intent, reasoning, strategy 等）
- `TOOL_CALL_START` / `TOOL_CALL_ARGS` / `TOOL_CALL_END` / `tool.call` — 工具调用
- `artifact.delta` — artifact 更新（code / webpage）

---

## 5. 后端模块说明

### 5.1 技术栈

- **语言**: Go 1.26
- **HTTP**: 标准库 `net/http` (ServeMux)
- **SSE**: 自定义实现 (`services/gateway/sse/sse.go`)
- **A2A**: 自定义 JSON-RPC (`pkg/adk/a2a/`)
- **持久化**: MemoryStore (默认) / SQLite (可选)
- **LLM**: 当前均为 mock 实现

### 5.2 启动命令

```bash
# 单独启动
go run ./services/gateway/cmd/gateway          # Gateway :8080
go run ./services/orchestrator/cmd/orchestrator # Orchestrator :8090
go run ./services/agents/code-agent/cmd/code-agent  # Code Agent :8081
go run ./services/agents/web-agent/cmd/web-agent    # Web Agent :8082

# Docker 全栈
docker compose -f docker-compose.new-arch.yml up --build
```

### 5.3 关键文件路径

#### Gateway

| 模块 | 文件路径 |
|------|----------|
| 入口 | `services/gateway/cmd/gateway/main.go` |
| Gateway 核心 | `services/gateway/gateway.go` |
| HTTP API 路由 | `services/gateway/httpapi/server.go` |
| SSE Writer | `services/gateway/sse/sse.go` |
| Orchestrator 客户端 | `services/gateway/orchestratorclient/client.go` |
| MemoryStore | `services/gateway/store/store.go` |
| SQLite 持久化 | `services/gateway/internal/persistence/sqlite/` |
| 持久化写入器 | `services/gateway/httpapi/persistence_writer.go` |

#### Orchestrator

| 模块 | 文件路径 |
|------|----------|
| 入口 | `services/orchestrator/cmd/orchestrator/main.go` |
| HTTP API | `services/orchestrator/httpapi/server.go` |
| Run Stream Handler | `services/orchestrator/httpapi/handler_run_stream.go` |
| RulePlanner | `services/orchestrator/planner/rule_planner.go` |
| LLMPlanner | `services/orchestrator/planner/llm_planner.go` |
| Planner 接口 | `services/orchestrator/planner/planner.go` |
| Plan 类型 | `services/orchestrator/plan/types.go` |
| A2A Dispatcher | `services/orchestrator/dispatcher/a2a_dispatcher.go` |
| SingleExecutor | `services/orchestrator/executor/single_executor.go` |
| OrderedParallelExecutor | `services/orchestrator/executor/ordered_parallel_executor.go` |
| Agent Registry | `services/orchestrator/registry/static_registry.go` |
| Validator | `services/orchestrator/validator/validator.go` |

#### Agents

| 模块 | Code Agent | Web Agent |
|------|------------|-----------|
| 入口 | `services/agents/code-agent/cmd/code-agent/main.go` | `services/agents/web-agent/cmd/web-agent/main.go` |
| Agent 核心 | `services/agents/code-agent/agent.go` | `services/agents/web-agent/agent.go` |
| HTTP Server | `services/agents/code-agent/server.go` | `services/agents/web-agent/server.go` |
| Tools | `services/agents/code-agent/tools.go` | `services/agents/web-agent/tools.go` |
| Prompt | `services/agents/code-agent/prompt.go` | `services/agents/web-agent/prompt.go` |

### 5.4 聊天接口全链路

```
POST /api/chat (Gateway httpapi/server.go:245)
  → 验证 conversationId + message
  → store.AppendMessage (保存 user message)
  → sse.SetHeaders (Content-Type: text/event-stream)
  → runner.Run(ctx, conversationID, adk.Content)
    → OrchestratorRunService.Run (orchestratorclient/client.go:73)
      → POST /internal/orchestrator/runs/stream (Orchestrator)
        → handler_run_stream.go:61 handleRunStream
          → Planner.Plan (RulePlanner 或 LLMPlanner)
          → Validator.Validate
          → emitEvent("run_started", planState)
          → Executor.Execute (SingleExecutor 或 OrderedParallelExecutor)
            → A2ADispatcher.Dispatch (JSON-RPC to agent)
            → emitEvent("message_delta", agent response)
          → emitEvent("run_finished")
  → translator.Translate (AG-UI event mapping)
  → writer.WriteEvent (SSE write + flush)
  → store.AppendMessage (保存 assistant message)
```

### 5.5 会话标题生成

**当前状态**: 无自动标题生成。
- `MemoryStore.CreateConversation` 只保存 `UserID` 和 `AgentName`，不生成标题
- SQLite `Conversation` 模型有 `Title` 字段，但创建时为空
- 前端 `api.ts:44` 默认为 `'New Conversation'`
- 后端完全没有标题生成逻辑

### 5.6 消息长度/上下文限制

- `MemoryStore`: 无任何长度限制（内存中）
- SQLite `Message.Content`: `TEXT` 类型，无显式长度限制
- `Store.Message.Text`: `string` 类型，无截断
- 前端 `ChatWindow`: 无消息数量限制
- SSE streaming: nginx `proxy_read_timeout 3600s`，无 body 大小限制
- **结论**: 没有显式的消息长度限制代码；长对话截断可能发生在 LLM 上下文窗口层（但当前 agent 是 mock），或前端渲染性能问题

---

## 6. Agent 与编排模块说明

### 6.1 当前 Agent

| Agent | 实现类型 | 输出模式 | 工具 |
|-------|----------|----------|------|
| code-agent | MOCK（确定性） | text, code, artifact_ref | generate_code_snippet, explain_code_snippet |
| web-agent | MOCK（确定性） | text, webpage, html, artifact_ref | generate_html_snippet, summarize_ui_request |

**注意**: 两个 Agent 当前都是 mock 实现，不调用任何真实 LLM。`agent.go` 中的 `Generate` 方法返回硬编码的响应。

### 6.2 Agent 职责边界

- **code-agent**: 代码生成和解释（`services/agents/code-agent/agent.go:66`）
- **web-agent**: 生成安全 HTML UI 预览（`services/agents/web-agent/agent.go:75`）
- **Orchestrator**: 规划→验证→执行→分发，是唯一的编排层
- **Gateway**: 不做编排，不做 LLM 调用，不做直接 Agent 调用

### 6.3 Agent 注册机制

1. Gateway 侧: `main.go:178-191` 硬编码 `AgentEndpoint` 列表（code-agent + web-agent）
2. Orchestrator 侧: `main.go:64-81` 硬编码 `StaticAgentRegistry`
3. 前端侧: `agents.ts:19-32` 硬编码 `AGENT_OPTIONS`

三处需要同步维护，没有统一的注册中心。

### 6.4 Orchestrator Agent 选择逻辑

`RulePlanner.determineAgent()` (`rule_planner.go:116-157`):
1. 优先使用请求中的 `agentName`（显式选择）
2. 其次使用 `selectedAgentNames[0]`
3. 关键词匹配：web 关键词 → web-agent，code 关键词 → code-agent
4. 默认 fallback → code-agent

**重要**: 当 `agentName` 为空时，RulePlanner 走关键词匹配逻辑。但前端强制传了 `agentName`（默认 `code-agent`），所以关键词匹配实际上被绕过。

### 6.5 编排过程暴露问题

`handler_run_stream.go:163-185` 的 `run_started` 事件包含大量内部元数据：
- `planState["reasoning"]` — planner 推理过程
- `planState["intent"]` — 意图摘要
- `planState["strategy"]` — 编排策略
- `planState["plannerSource"]` — 使用哪个 planner

这些通过 `STATE_UPDATE` → `OrchestrationCard` 组件渲染给用户。**编排推理过程可能过于技术化，且污染用户界面**。

### 6.6 Web Preview 与 Code Agent 的边界问题

当前 `messageStore.ts:454-468` 的 `appendWebPreviewFromMessageContent()` 函数在流式完成后检查 agent 内容是否包含 HTML，如果包含则提取为 Web Preview。**这导致 Code Agent 输出中的 HTML 也会被误识别为 Web Preview**。

---

## 7. Docker / Compose 说明

### 7.1 服务拓扑 (`docker-compose.new-arch.yml`)

| 服务 | Dockerfile | 端口 | 依赖 |
|------|-----------|------|------|
| code-agent-new | `services/agents/code-agent/Dockerfile` | 8081:8080 | - |
| web-agent-new | `services/agents/web-agent/Dockerfile` | 8082:8080 | - |
| orchestrator-new | `services/orchestrator/Dockerfile` | 8090:8080 | code-agent-new, web-agent-new |
| gateway-new | `services/gateway/Dockerfile` | 8080:8080 | orchestrator-new |
| frontend-new | `frontend/Dockerfile` | 3000:3000 | gateway-new |

### 7.2 Dockerfile COPY 分析

**4 个 Go Dockerfile 全部 COPY 整个 monorepo**：

```dockerfile
# 所有 Go Dockerfile 都包含（以 gateway 为例）：
COPY go.work go.work.sum ./
COPY agents ./agents           # 遗留，不需要
COPY server ./server           # 遗留，不需要
COPY pkg ./pkg                 # 共享库，全部需要
COPY services/gateway ./services/gateway
COPY services/orchestrator ./services/orchestrator   # 不需要
COPY services/agents/code-agent ./services/agents/code-agent  # 不需要
COPY services/agents/web-agent ./services/agents/web-agent    # 不需要
```

**问题**:
- 每个服务 COPY 了不需要的目录
- 修改任意 `services/gateway` 文件 → 所有 4 个服务构建缓存失效
- 修改 `agents/` 或 `server/`（遗留代码）→ 所有新架构服务缓存失效

**frontend Dockerfile 是最佳实践**：只 COPY `package.json` + `package-lock.json`，然后 `npm ci`，最后 COPY 源码。

### 7.3 GOPROXY 配置

所有 Dockerfile 中均**未设置 GOPROXY**。需要在 `go build` 前设置 `ENV GOPROXY=https://proxy.golang.org,direct` 或在 `RUN` 命令中设置。

### 7.4 环境变量

Compose 中已配置的环境变量：
- Gateway: `GATEWAY_ADDR`, `ORCHESTRATOR_URL`, `ORCHESTRATOR_INTERNAL_TOKEN`, `GATEWAY_ENABLE_AUTH`, `GATEWAY_ALLOWED_ORIGINS`, `AGENTHUB_GATEWAY_STORE`, `AGENTHUB_SQLITE_PATH`
- Orchestrator: `ORCHESTRATOR_ADDR`, `INTERNAL_SERVICE_TOKEN`, `CODE_AGENT_URL`, `WEB_AGENT_URL`
- Agent: `*_ADDR`, `*_PUBLIC_URL`

**缺失**: LLM API Key 变量（`ANTHROPIC_API_KEY` / `OPENAI_API_KEY` / `ORCHESTRATOR_LLM_API_KEY`）未在 compose 中配置，LLM planner 模式下无法工作。

### 7.5 Docker 优化建议（本轮不修改）

1. **分层 COPY**: 每个 Go Dockerfile 只 COPY 自己需要的目录 + 共享 pkg
2. **go.sum 先 COPY**: `COPY go.work go.work.sum ./` 然后 `COPY pkg/adk/go.* pkg/adk/` 等
3. **GOPROXY 缓存**: 设置 `GOPROXY` 并使用 BuildKit 缓存挂载
4. **builder 阶段分离**: 每个服务独立 builder，共享 base image

---

## 8. 端到端调用链路

```
用户输入 "写一个登录页面"
  │
  ▼
[1] 前端 Chat UI
    文件: frontend/src/components/ChatWindow.tsx
    函数: ChatWindow.handleSend() → sendMessage(conversationId, content, { agentName })
  │
  ▼
[2] 前端发送请求
    文件: frontend/src/stores/messageStore.ts
    函数: sendMessage() → 构建 AGUIChatRequest → runAgent()
  │
  ▼
[3] 前端 SSE Client
    文件: frontend/src/agui/client.ts
    函数: runAgent() → fetch POST /api/chat, 读取 ReadableStream, SSE 解析
    事件: TEXT_MESSAGE_START, TEXT_MESSAGE_CONTENT, STATE_UPDATE, TOOL_CALL_*, RUN_FINISHED
  │
  ▼
[4] Gateway HTTP API
    文件: services/gateway/httpapi/server.go
    函数: handleChat() → 验证 conversationId + message → store.AppendMessage(user msg)
  │
  ▼
[5] Gateway SSE Setup
    文件: services/gateway/sse/sse.go
    函数: SetHeaders(), NewWriter()
  │
  ▼
[6] Gateway → Orchestrator Client
    文件: services/gateway/orchestratorclient/client.go
    函数: OrchestratorRunService.Run() → HTTP POST /internal/orchestrator/runs/stream
    请求: {conversationId, messages: [{role:"user", text:"..."}], agentName:"code-agent", planningMode:"auto"}
  │
  ▼
[7] Orchestrator Run Stream Handler
    文件: services/orchestrator/httpapi/handler_run_stream.go
    函数: handleRunStream() → 解析请求 → Planner.Plan() → Validator.Validate() → emitEvent(run_started) → Executor.Execute()
  │
  ▼
[8] Planner
    文件: services/orchestrator/planner/rule_planner.go (默认)
    函数: RulePlanner.Plan() → 关键词匹配 → 生成 OrchestrationPlan
    结果: StrategySingle(code-agent) 或 StrategyOrderedParallel(code+web)
  │
  ▼
[9] Executor
    文件: services/orchestrator/executor/single_executor.go (单 Agent)
    函数: SingleExecutor.Execute() → registry.Get(agentName) → emitEvent(message_start) → dispatcher.Dispatch() → emitEvent(message_delta) → emitEvent(message_end) → emitEvent(run_finished)
  │
  ▼
[10] A2A Dispatcher
     文件: services/orchestrator/dispatcher/a2a_dispatcher.go
     函数: A2ADispatcher.Dispatch() → client.SendJSONRPC(agentURL, RunRequest)
  │
  ▼
[11] Code Agent / Web Agent
     文件: services/agents/code-agent/agent.go (MOCK)
     函数: CodeAgent.Generate() → buildMockResponse() → 返回确定性文本
     或
     文件: services/agents/web-agent/agent.go (MOCK)
     函数: WebAgent.Generate() → buildMockResponse() → renderSafeHTMLSnippet() + 返回 HTML
  │
  ▼
[12] 响应流回 Gateway
     SSE 事件: run_started → message_start → message_delta → message_end → run_finished
  │
  ▼
[13] Gateway AG-UI Translate + SSE Write
     文件: services/gateway/httpapi/server.go handler → translator.Translate(event) → writer.WriteEvent()
     文件: services/gateway/sse/sse.go → WriteEvent() → fmt.Fprintf("event: %s\ndata: %s\n\n")
  │
  ▼
[14] 前端 SSE 事件处理
     文件: frontend/src/stores/messageStore.ts
     事件处理:
       TEXT_MESSAGE_START → ensureAgentMessage() → 创建 streaming bubble
       TEXT_MESSAGE_CONTENT → updateAgentMessage() → 追加 content + 更新 UI
       STATE_UPDATE → OrchestrationCard 显示编排信息
       TOOL_CALL_END → handleToolPayload() → appendCodePreview / appendWebPreview
       RUN_FINISHED → finishStreamingMessage() → status='sent'
  │
  ▼
[15] 前端消息渲染
     文件: frontend/src/components/MessageBubble.tsx → StreamingText.tsx → react-markdown
     文件: frontend/src/components/CodePreview.tsx → highlight.js + Copy 按钮
     文件: frontend/src/components/WebPreview.tsx → <pre><code> 显示原始 HTML
  │
  ▼
[16] 会话上下文保存
     Gateway: store.AppendMessage(assistant text) → MemoryStore / SQLite
     文件: services/gateway/httpapi/server.go → 保存到 store
     文件: services/gateway/httpapi/persistence_writer.go → SQLite 镜像写入
```

---

## 9. 联调问题逐项定位

| 编号 | 联调问题 | 现象 | 可能根因 | 涉及文件 | 修复难度 | 建议优先级 | 是否建议立即修 |
|------|----------|------|----------|----------|----------|------------|----------------|
| 1 | Dockerfile COPY 整个 monorepo | 构建缓存频繁失效，每次 `--build` 都重编 | 4 个 Go Dockerfile 都 COPY 了 agents/、server/、以及不相关的 services/ 子目录 | `services/*/Dockerfile` x4 | 中 | P2 | 否（阶段3） |
| 2 | GOPROXY / Go 模块代理 | 国内网络拉取 Go 模块慢/失败 | Dockerfile 中未设置 `GOPROXY` | `services/*/Dockerfile` x4 | 低 | P2 | 否（阶段3） |
| 3 | mock 测试注册模型包 | 测试需要 import 额外包 | 未知，需要运行测试确认 | `services/**/*_test.go` | 低 | P2 | 否（需确认） |
| 4 | 只能强制选择 Agent | 无 "自动编排" 入口，用户必须手动选 | 前端 `ChatWindow.tsx` 下拉框只有 2 个选项；无 "auto" 选项；`agents.ts` 无 auto 配置 | `frontend/src/components/ChatWindow.tsx:89-101`, `frontend/src/lib/agents.ts:19-32` | 低 | **P0** | **是** |
| 5 | Code block 没有复制按钮 | 用户报告无法复制代码 | **实际已实现** — `CodePreview.tsx:28` 有 `handleCopy` + Clipboard API + fallback | `frontend/src/components/CodePreview.tsx` | - | - | 已修复，可能需验证渲染条件 |
| 6 | 没有 streaming / loading | 发送后无等待反馈 | **Streaming 基础链路已实现**。可能问题：1) mock agent 响应太快无感知 2) "Thinking" 动画仅在 content 为空时显示 3) 无打字机效果 | `frontend/src/components/StreamingText.tsx:10-17`, `frontend/src/stores/messageStore.ts` | 低 | **P0** | **是**（增强体验） |
| 7 | Web Preview 没有真正截取 HTML | 只显示 `<pre><code>` 原始 HTML | `WebPreview.tsx` 只用 `<pre><code>` 渲染，没有 sandbox iframe。Web Agent 返回的 HTML 没有被当作可渲染预览 | `frontend/src/components/WebPreview.tsx:17-22` | 中 | **P0** | **是** |
| 8 | 会话标题全部 "New Conversation" | 所有会话显示同样标题 | 后端无标题生成逻辑；MemoryStore 不存标题；SQLite Conversation 有 `title` 字段但为空；前端默认值 hardcode | `frontend/src/services/api.ts:44`, `services/gateway/store/store.go`, `services/gateway/httpapi/server.go:175` | 中 | **P0** | **是** |
| 9 | 编排输出过于抽象或暴露内部过程 | 用户可见技术性编排细节 | `handler_run_stream.go:163-185` 在 `run_started` 的 state 中发送 reasoning/intent/strategy/plannerSource；前端 `OrchestrationCard` 展示全部 | `services/orchestrator/httpapi/handler_run_stream.go:163-185`, `frontend/src/components/OrchestrationCard.tsx` | 低 | P1 | 否（阶段2） |
| 10 | Web Preview 被 Code Agent 调用 | Code Agent 输出含 HTML 时被误识别 | `messageStore.ts:454-468` 的 `appendWebPreviewFromMessageContent()` 检查 agent 内容是否含 HTML 标签 | `frontend/src/stores/messageStore.ts:454-468` | 低 | P1 | 否（阶段2） |
| 11 | 对话过长被截断 | 长对话某处停止 | 无显式长度限制代码。可能原因：1) nginx `proxy_read_timeout 3600s` 2) SSE 连接中断 3) 前端渲染大数据量性能 | `frontend/nginx.conf`, `services/gateway/sse/sse.go` | 中 | P2 | 否（需复现确认） |
| 12 | 上下文保存/读取确认 | 需要确认链路完整性 | MemoryStore 内存中正常；SQLite 持久化通过 `PersistenceWriter` 镜像写入 | `services/gateway/store/store.go`, `services/gateway/internal/persistence/sqlite/store.go` | 低 | P2 | 否（已有基础） |

---

## 10. 修复优先级

### 阶段 1：比赛演示必修（P0）

| 优先级 | 问题 | 目标 |
|--------|------|------|
| 1 | 无自动编排入口 | 添加 "Auto / Smart" 选项，agentName 为空时由 Orchestrator 自动选择 |
| 2 | Loading/Streaming 体验 | 增强 streaming 感知：打字机效果、明确的 loading 指示器 |
| 3 | Web Preview 不可用 | 使用 sandbox iframe 渲染 HTML；只在 web-agent 输出时触发 |
| 4 | 会话标题自动生成 | 首次用户消息后自动生成标题（前端提取首条消息前 N 字 + 后端存储） |
| 5 | 验证 Copy 按钮 | 确认 CodePreview 在所有场景下正常渲染 |

### 阶段 2：编排正确性（P1）

| 优先级 | 问题 | 目标 |
|--------|------|------|
| 1 | Web Preview 与 Code Agent 边界 | 只在明确来自 web-agent 时创建 Web Preview |
| 2 | 编排过程不污染输出 | 将 STATE_UPDATE 中的技术细节折叠/隐藏，用户只看到简洁摘要 |
| 3 | 多 Agent 汇总格式 | OrderedParallel 模式下汇总两个 Agent 的输出 |

### 阶段 3：工程化优化（P2）

| 优先级 | 问题 | 目标 |
|--------|------|------|
| 1 | Docker 缓存优化 | 分层 COPY，每个服务只 COPY 必要目录 |
| 2 | GOPROXY 配置 | 统一设置国内可用的 GOPROXY |
| 3 | 长上下文截断定位 | 添加日志/监控确认截断位置 |
| 4 | 测试补齐 | 确认 mock 测试模型包问题已修复 |

---

## 11. 第一轮建议修改文件清单

### 前端（4 个文件）

| 文件 | 修改内容 | 难度 |
|------|----------|------|
| `frontend/src/lib/agents.ts` | 添加 "auto" AgentOption；`DEFAULT_AGENT_NAME` 改为 "auto"（或空字符串）；更新 `buildAgentOptionsFromSummary` | 低 |
| `frontend/src/components/ChatWindow.tsx` | Agent 下拉添加 "Auto (Smart)" 选项；当选择 auto 时不传 agentName | 低 |
| `frontend/src/components/WebPreview.tsx` | 重构为 sandbox iframe 渲染（`srcdoc` 属性） | 中 |
| `frontend/src/stores/messageStore.ts` | `appendWebPreviewFromMessageContent()` 增加 agentName 检查；增强 streaming 状态展示 | 低 |

### 后端（2 个文件）

| 文件 | 修改内容 | 难度 |
|------|----------|------|
| `services/gateway/httpapi/server.go` | 会话创建时自动生成标题（取 agentName + 时间戳）；或支持前端传入 title | 低 |
| `services/gateway/store/store.go` | MemoryStore Conversation 增加 Title 字段；CreateConversation 自动生成默认标题 | 低 |

### 可能的额外文件

| 文件 | 修改内容 | 难度 |
|------|----------|------|
| `services/agents/web-agent/server.go` | 如果需要真正的 Web Preview 流式传输 | 中 |
| `frontend/src/components/StreamingText.tsx` | 增强打字机动画效果 | 低 |
| `frontend/src/services/api.ts` | 支持 PATCH conversation title API | 低 |

---

## 12. 风险点

1. **Agent 均为 mock 实现** — 当前不调用真实 LLM，演示时可能需要接真实 LLM
2. **Gateway→Orchestrator 是 HTTP/SSE** — 契约要求目标 gRPC streaming，HTTP/SSE 可能在生产有性能问题
3. **持久化默认内存** — MemoryStore 重启丢失，SQLite 是 demo-only
4. **前端直接 hardcode Agent 列表** — 与后端 `api.listAgents()` 不一致时可能出现 Agent 找不到
5. **编排状态暴露** — OrchestrationCard 显示 AI 编排推理，可能包含不准确信息
6. **Web Preview 安全** — 如果渲染真实用户生成的 HTML，需要严格的 CSP 和 XSS 防护
7. **go.work 包含遗留模块** — `agents/` 和 `server/` 仍在 workspace 中，可能导致意外的依赖

---

## 13. 下一步建议

### 立即可以做的（不需要改代码）

1. 运行 `smoke-new-arch.sh` 验证当前服务是否可启动
2. 在浏览器中打开 `http://localhost:3000` 手动验证 UI 行为
3. 确认当前 mock agent 的响应格式是否满足前端渲染期望

### 第一轮修复（阶段1 - 演示必修）

1. 前端添加 "Auto / Smart" 编排选项
2. 增强 Loading/Streaming 体验感知
3. Web Preview 改为 sandbox iframe
4. 会话标题自动生成（前端取前 30 字或后端首条消息摘要）
5. 代码 Copy 按钮功能验证

### 需要更多信息的问题

1. **mock 测试注册模型包问题** — 需要运行 `go test ./services/... -count=1` 确认当前状态
2. **GOPROXY 问题** — 需要确认当前网络环境下 Go 模块是否可正常下载
3. **对话截断** — 需要复现具体场景（消息数量、内容大小）才能定位

---

## 关键问题快速回答

### 1. 前端入口在哪里？
`frontend/src/main.tsx` → 渲染 `<App />` → `frontend/src/App.tsx` → `<ChatLayout />` → `frontend/src/components/ChatLayout.tsx`

### 2. 后端聊天接口在哪里？
`POST /api/chat` → `services/gateway/httpapi/server.go:245` `handleChat()`

### 3. Orchestrator 在哪里？
`services/orchestrator/` — 入口 `cmd/orchestrator/main.go`，核心流处理 `httpapi/handler_run_stream.go:61`

### 4. Agent 注册在哪里？
三处硬编码：
- Gateway: `services/gateway/cmd/gateway/main.go:178-191`
- Orchestrator: `services/orchestrator/cmd/orchestrator/main.go:64-81`
- 前端: `frontend/src/lib/agents.ts:19-32`

### 5. Web Preview 现在到底是怎么工作的？
**不工作**。`WebPreview.tsx` 仅以 `<pre><code>` 显示原始 HTML 字符串，没有实际的沙箱渲染。Web Agent 返回的安全 HTML 片段被当作纯文本展示。

### 6. 哪几个问题最适合作为第一轮修复？
1. 自动编排入口（前端 2 文件）
2. Web Preview sandbox iframe（前端 1 文件）
3. 会话标题自动生成（前端 1 文件 + 后端 2 文件）
4. Loading/Streaming 体验增强（前端 1 文件）

### 7. 第一轮修复预计会改哪些文件？
- `frontend/src/lib/agents.ts` — 添加 auto 选项
- `frontend/src/components/ChatWindow.tsx` — 下拉框添加 auto
- `frontend/src/components/WebPreview.tsx` — iframe sandbox
- `frontend/src/stores/messageStore.ts` — Web Preview 边界检查 + streaming 增强
- `frontend/src/services/api.ts` — 支持会话标题更新
- `frontend/src/components/ConversationList.tsx` — 标题自动生成展示
- `services/gateway/store/store.go` — Conversation 添加 Title 字段
- `services/gateway/httpapi/server.go` — 创建会话时生成默认标题
