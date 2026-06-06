# Frontend Gateway Field Changelog — Round 1A

Last updated: 2026-06-06

## Changed Fields

### 1. `agentName` — request body (`POST /api/chat`)

| Aspect | Before | After |
|--------|--------|-------|
| Default value | `'code-agent'` | `'auto'` |
| Sent when | Always sent (even when 'code-agent') | Only sent when user selects a concrete agent ('code-agent' or 'web-agent'). Omitted entirely when 'auto'. |
| Effect on wire | `{ "agentName": "code-agent", ... }` | No `agentName` key in JSON body when auto-selected |

**Why:** The orchestrator already handles empty/missing agentName via keyword-based RulePlanner. Sending an explicit agentName forced a concrete agent selection even when the user wanted auto-orchestration. Omitting the field when 'auto' lets the orchestrator do its own keyword matching.

**Affected code:**
- `frontend/src/lib/agents.ts` — `DEFAULT_AGENT_NAME` changed from `'code-agent'` to `'auto'`
- `frontend/src/stores/messageStore.ts` — `sendMessage()` gating: `if (options?.agentName && options.agentName !== 'auto') { request.agentName = options.agentName }`

**Gateway impact:** None. Gateway already passes agentName through to orchestrator. No Gateway code changes needed.

### 2. `WebPreview` header label (no wire change)

**Note:** This is purely a frontend presentation change. The header text changed from "Web Preview (Safe Mode)" to "Web Preview". No SSE event fields changed.

### 3. `OrchestrationCard` collapsed by default (no wire change)

**Note:** Purely a frontend UX change. Orchestration info fields (`intent`, `reasoning`, `strategy`, `taskCount`, `plannerSource`, `plannerModel`) are unchanged. The card now defaults to collapsed with a one-line status summary.

## Unchanged Fields

All SSE event fields (`TEXT_MESSAGE_CONTENT`, `TOOL_CALL_*`, `STATE_UPDATE`, `RUN_FINISHED`, etc.) are unchanged. The `OrchestrationInfo` interface fields are unchanged. No breaking wire-format changes.

---

## Round 1B 变更摘要

本轮未修改 Frontend ↔ Gateway 的 REST 请求字段、REST 响应字段、SSE event type 或 SSE event data wire format。

但本轮存在以下前端行为 / 解析变化：

| 编号 | 接口/事件 | 方向 | 字段 | 变更类型 | 变更前 | 变更后 | 是否兼容旧逻辑 | 涉及文件 | 原因 |
|------|----------|------|------|----------|--------|--------|----------------|----------|------|
| 1 | `/api/chat` (SSE) | 接收 | `agentName` (event field) | 前端解析逻辑变化 | 仅用于消息气泡 senderName 解析 | 新增用于 Web Preview 边界判定：Auto 模式下优先使用 SSE 事件中的 agentName 判断是否来自 web-agent | 是 | `messageStore.ts` | Auto 模式不传 agentName，需要实际 SSE 来源判断 |
| 2 | `/api/chat` (SSE) | 接收 | `STATE_UPDATE` → `selectedAgentName`, `selectedAgentDisplayName`, `taskAgentNames` | 前端解析语义变化 | 未提取 | 新增提取用于 OrchestrationCard 摘要显示 | 是（新字段，旧逻辑忽略） | `messageStore.ts`, `OrchestrationCard.tsx` | 编排卡片显示实际选中的 Agent |
| 3 | `/api/conversations` | 接收 | `title` | 无 wire 字段变化，前端本地持久化行为变化 | 仅使用 Gateway 返回的 title | 在 Gateway 返回空/默认标题时，叠加 localStorage 中的本地标题 | 是 | `conversationStore.ts`, `lib/conversationTitles.ts` | 提升用户体验，不依赖 Gateway 标题持久化 |
| 4 | WebPreview | 前端展示 | `sandbox` attribute | 无 wire 字段变化，前端安全行为变化 | `sandbox="allow-scripts"` | `sandbox=""` (不允许脚本) | 是（纯展示层） | `WebPreview.tsx` | 收紧安全策略 |
| 5 | WebPreview | 前端展示 | `srcDoc` CSP injection | 无 wire 字段变化，前端安全行为变化 | 纯 HTML | 注入 CSP `<meta>` 标签 | 是（纯展示层） | `WebPreview.tsx` | 纵深防御 |
| 6 | `/api/chat` (SSE) | 接收 | `artifact.delta` → `type` | 前端解析逻辑变化 | artifact type 仅用于创建预览块 | 新增设置 `hasWebArtifactEvidence` 标志，用于 Auto 模式 Web Preview 触发判定 | 是 | `messageStore.ts` | Auto 模式需要 artifact 证据 |
| 7 | `/api/chat` (WebPreview) | - | Web Preview 内容提取 | 前端解析逻辑变化 | 仅检查 UI 下拉框 agentName | 同时检查 SSE 来源 agentName + artifact 证据 + 工具调用证据 | 是（更宽松、更准确） | `messageStore.ts` | 修复 Auto 模式下 Web Preview 可能不触发的问题 |

### REST 请求字段

延续 Round 1A：Auto 模式下 `agentName` 仍省略。无新增变化。

### REST 响应字段

无变化。

### SSE event type

无新增或删除 SSE event type。`STATE_UPDATE` 中新增解析 `selectedAgentName` 等可选字段，但不改变 wire format。

### SSE event data 字段

无新增 wire 字段。前端新增使用 `STATE_UPDATE` 中已有的 `selectedAgentName` 等可选字段（如有），以及使用 `artifact.delta` 中的 `type` 字段设置标志。

### 前端对已有 SSE 字段的解析语义变化

- `resolveEventAgentName` 提取的 agentName 新增用于 Web Preview 判定
- `artifact.delta` type=webpage/html 新增设置 `hasWebArtifactEvidence`
- `TOOL_CALL_*` web_preview/generate_html_snippet 新增设置 `hasWebArtifactEvidence`
- `STATE_UPDATE` 新增提取 `selectedAgentName`, `selectedAgentDisplayName`, `taskAgentNames`

### 无 wire 字段变化但行为变化

- WebPreview iframe sandbox 从 `allow-scripts` 改为 `""`
- WebPreview srcDoc 注入 CSP `<meta>` 标签（纵深防御）
- 会话标题增加 localStorage 客户端持久化层（Gateway 标题优先，本地标题兜底）
- ChatWindow "Thinking…" 从简单文字变为带头像的骨架气泡
- Markdown code fence 中的 HTML 不再触发 Web Preview
- 长文本/长 URL 增加 break-words / overflow-x-auto 样式
