# Phase 9.1~9.3 Frontend AgentName 与多 Agent SSE 适配完成报告

## 1. 本轮目标
本轮在不改后端架构的前提下，完成 frontend 对 `agentName` 的最小支持：可选 `code-agent/web-agent`、发送到 Gateway `POST /api/chat`、并在消息区展示多 Agent 消息归属与最小 code/web 预览入口。

## 2. 新增/修改文件
- `frontend/src/lib/agents.ts`（新增）
- `frontend/src/types/index.ts`
- `frontend/src/agui/client.ts`
- `frontend/src/agui/client.test.ts`
- `frontend/src/stores/messageStore.ts`
- `frontend/src/stores/messageStore.test.ts`
- `frontend/src/stores/conversationStore.ts`
- `frontend/src/services/api.ts`
- `frontend/src/components/ChatWindow.tsx`
- `frontend/src/components/ConversationList.tsx`
- `frontend/src/components/MessageBubble.tsx`
- `frontend/src/components/WebPreview.tsx`（新增）
- `frontend/src/components/WebPreview.test.tsx`（新增）
- `frontend/e2e/mocks.ts`
- `frontend/e2e/mvp-happy-path.spec.ts`
- `docs/refactor/phase-9-frontend-agent-routing-report.md`（新增）

## 3. Agent 选择设计
- 采用静态 `AGENT_OPTIONS`，定义在 `frontend/src/lib/agents.ts`。
- 默认 Agent 为 `code-agent`（`DEFAULT_AGENT_NAME`）。
- 最小支持两项：`code-agent`、`web-agent`。
- 本轮不接动态 `/api/agents` 拉取，原因是 Phase 9.1~9.3 目标是“最小前端路由适配”，优先保证聊天主链路稳定与兼容，避免引入额外依赖与异步状态复杂度。

## 4. /api/chat 请求适配
- `runAgent` 请求已由 `/api/agui/run` 改为 `/api/chat`。
- 请求体改为：
  - `conversationId`
  - `message`
  - `agentName?`（可选）
- `agentName` 保持 optional：调用 `sendMessage(conversationId, content)` 时不传；调用 `sendMessage(..., { agentName })` 时按选择传入。
- 未修改任何后端代码。

## 5. SSE 消息归属
- 前端事件处理同时兼容两类事件：
  - 旧事件：`TEXT_MESSAGE_*`、`TOOL_CALL_*`、`RUN_*`
  - 新事件：`message`/`message.delta`/`message.end`、`tool.call`、`artifact.delta`、`error`
- 归属字段解析优先级：
  - `senderName` / `agentName`
  - `author`
  - `metadata.agentName` / `stateDelta.agentName`
  - fallback 到当前请求的 `selectedAgentName`
- UI 展示：
  - `code-agent` 显示 `Code Agent`
  - `web-agent` 显示 `Web Agent`
- 错误信息仅显示安全文案，不展示内部 URL、堆栈、token。

## 6. code_preview / web_preview 最小策略
- `code-agent`：
  - 保留原有 `code_preview` tool 入口，继续渲染 `CodePreview`。
  - 同时兼容 `artifact.delta(type=code)` 的最小映射。
- `web-agent`：
  - 新增 `WebPreview` 只读卡片（Safe Mode）。
  - 支持 `web_preview` tool payload 与 `artifact.delta(type=webpage/html)` 最小映射。
  - 如仅返回 HTML 文本，也会在 `web-agent` 场景下尝试提取为只读预览。
- 安全策略：
  - 不使用 `dangerouslySetInnerHTML`
  - 不执行 script
  - 不注入主 DOM 执行事件
  - 本轮采用“降级展示”为主，未启用可执行 iframe 预览

## 7. 测试与验证
- frontend 测试：
  - 命令：`npm test -- --run`
  - 结果：通过（4 files, 20 tests）
- frontend build：
  - 命令：`npm run build`
  - 结果：通过（Vite build 成功，存在大 bundle warning，不影响本轮目标）
- e2e mock：
  - 已更新 `frontend/e2e/mocks.ts`，覆盖：
    - 不传 `agentName` 默认 code-agent
    - `agentName=code-agent`
    - `agentName=web-agent`
    - unknown agent 返回 error SSE
  - 未执行真实 e2e 浏览器回放（本轮仅完成 mock 与用例最小更新）
- 后端回归：
  - `services/gateway`：`go test ./...` 通过，`go build ./...` 通过
  - `services/agents/code-agent`：`go test ./...` 通过，`go build ./...` 通过
  - `services/agents/web-agent`：`go test ./...` 通过，`go build ./...` 通过
- `git status --short`：仅 frontend 与本报告相关文件变更，另有未跟踪 `.claude/settings.local.json`（未改动）
- `git diff --stat`：当前变更集中在 frontend 聊天链路、SSE、测试与 mock；无后端协议文件改动
- `dist/node_modules` 检查：
  - `frontend/dist` 未进入 git 变更列表
  - `node_modules` 未进入 git 变更列表

## 8. 保持不变范围
- 未修改 `agents/code-agent`（旧）
- 未修改 `agents/web-agent`（旧）
- 未删除任何旧 Agent
- 未修改 `server/`
- 未修改 `services/agents/code-agent`
- 未修改 `services/agents/web-agent`
- 未修改 `docker-compose.yml`
- 未修改 `Makefile`
- 未修改根 `go.mod` / `go.sum`
- 未读取/修改 `.env`
- 未提交代码
- 未执行 `git add`
- 未调用真实 LLM
- 未提交 `frontend/dist`
- 未提交 `node_modules`

## 9. 风险与注意事项
- 当前前端 Agent 列表为静态配置，后续可接动态 Registry。
- 当前实现不是 Orchestrator 群聊编排，仅做 Gateway `/api/chat` 最小路由适配。
- 当前 web 预览采用最小安全降级，不做可执行渲染。
- 后续可在稳定后引入 Gateway `/api/agents` 动态能力，或接入 Orchestrator Registry。
- 后续可扩展正式 Artifact 卡片体系（与 runtime skill registry 深度对齐）。

## 10. 下一步建议
下一步可以进入 Phase 9.4：前后端联调新 Gateway /api/chat 与多 Agent SSE Demo；Orchestrator 编排继续暂缓。
