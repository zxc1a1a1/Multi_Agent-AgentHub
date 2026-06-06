# 前端问题 Round 1B 代码补强报告

Date: 2026-06-06
Branch: `g`
Status: **All 6 fixes implemented and verified**

## 1. 使用的 Skills

| Skill | 本轮约束 |
|-------|----------|
| `project-architecture` | 仅修改 `frontend/src`，禁止 touch `server/`、`agents/`、Docker 文件 |
| `frontend-runtime-skills-contract` | 前端只通过 Gateway 访问后端，不直连 Orchestrator / Agent |
| `gateway-orchestrator-contract` | 不改 Gateway/Orchestrator 代码 |
| `artifact-contract` | Web Preview 依赖 artifact type（webpage/html）、tool call name（web_preview/generate_html_snippet）判定 |
| `security-boundary-contract` | WebPreview sandbox 收紧，禁止脚本执行，CSP 注入纵深防御 |
| `testing-review-contract` | 每项修复补测试，保持已有 63 测试全通过 |
| `observability-debugging-contract` | 保持 `agentName`, `runId`, `traceId` 传递 |
| `code-style-and-conventions` | 无注释风格、无新路径侵入遗留代码 |
| `ai-collaboration-workflow` | 契约优先，字段变化记录到 FRONTEND_GATEWAY_FIELD_CHANGELOG.md |
| `commit-security-review` | diff 中无 secrets、无真实 API Key |

## 2. 本轮修复范围

本轮仅做代码层面补强，不接入真实 LLM，不进行人工联调。所有修改限制在 `frontend/src/` 目录。

## 3. 修改文件清单

| 文件 | 修改内容 | 是否涉及 Frontend ↔ Gateway 字段变化 |
|------|----------|-------------------------------------|
| `frontend/src/stores/messageStore.ts` | Web Preview 触发逻辑重写（SSE 来源判定 + artifact 证据）；STATE_UPDATE 新增 selectedAgent 字段提取；localStorage 标题持久化调用 | 是 — 前端对已有 SSE 字段的解析语义变化 |
| `frontend/src/stores/conversationStore.ts` | 加载会话时叠加 localStorage 本地标题 | 否 — 仅前端本地行为 |
| `frontend/src/lib/conversationTitles.ts` | **新建** — localStorage 会话标题持久化工具 | 否 |
| `frontend/src/lib/conversationTitles.test.ts` | **新建** — localStorage 工具测试 (7 测试) | 否 |
| `frontend/src/components/WebPreview.tsx` | sandbox 从 `allow-scripts` 改为 `""`；新增 CSP `<meta>` 注入；安全加固 | 否 — 纯展示/安全行为 |
| `frontend/src/components/WebPreview.test.tsx` | 新增 sandbox/CSP 测试 (2 新测试)；修复 JSDOM API 断言 | 否 |
| `frontend/src/components/ChatWindow.tsx` | "Thinking…" 从一个文字行升级为带头像的骨架气泡 | 否 — 纯展示变化 |
| `frontend/src/components/MessageBubble.tsx` | 用户消息内容增加 `break-words overflow-hidden` | 否 |
| `frontend/src/components/StreamingText.tsx` | Markdown 容器增加 `break-words overflow-x-hidden`；流式文本增加 `break-words` | 否 |
| `frontend/src/components/CodePreview.tsx` | pre 增加 `max-w-full`；code 增加 `break-words` | 否 |
| `frontend/src/components/OrchestrationCard.tsx` | OrchestrationInfo 新增 selectedAgent 字段；statusSummary 显示实际 Agent 名 | 否 — 纯展示变化 |
| `frontend/src/stores/messageStore.test.ts` | 新增 web preview auto-mode 测试 (5 新测试) | 否 |
| `docs/handoff/FRONTEND_GATEWAY_FIELD_CHANGELOG.md` | 新增 Round 1B 小节 | N/A |
| `docs/handoff/FRONTEND_FIX_ROUND1B_REPORT.md` | 本报告 | N/A |

## 4. Round 1B 补强项

| 编号 | 补强项 | 修复方式 | 自动化验证 |
|------|--------|----------|-----------|
| 一 | Auto 模式下 Web Preview 的触发边界 | SSE 事件 agentName 优先 → artifact 证据 → 工具调用证据 → UI 显式选择。Markdown code fence 中的 HTML 不再误触发。 | 5 个新测试覆盖：tool/web_preview + auto mode、artifact.delta/webpage + auto mode、SSE agentName=web-agent + auto mode、code fence HTML 不触发、code-agent HTML 不触发 |
| 二 | Web Preview sandbox 权限收紧 | sandbox="" (禁止脚本)、CSP `<meta>` 注入 (default-src 'none')、不引入大型 sanitization 依赖 | 2 个新测试：iframe sandbox 不包含 allow-scripts、srcDoc 包含 CSP meta |
| 三 | 会话标题 localStorage 持久化 | 新模块 `conversationTitles.ts`。Gateway 真实标题优先，本地标题兜底。防御 JSON 损坏、storage 不可用。 | 7 个新测试：读写、覆盖、overlay、Gateway 优先、JSON 损坏、空存储、类型过滤 |
| 四 | Streaming 等待态加固 | ChatWindow "Thinking..." 从单行文字升级为带头像 + 气泡骨架的可见占位卡片。不修改 messageStore，不创建 placeholder message。 | 已有 streaming 测试保持通过；发送后 streaming=true 立刻显示 skeleton |
| 五 | 长内容显示加固 | MessageBubble、StreamingText、CodePreview 增加 break-words/overflow-x-auto/max-w-full。不截断内容。 | 代码检查确认类存在；构建通过 |
| 六 | OrchestrationCard 实际 Agent 摘要 | STATE_UPDATE 中新增提取 selectedAgentName/selectedAgentDisplayName/taskAgentNames。折叠态显示"已选择 Code Agent"等 | 现有测试通过；新字段可选不影响旧逻辑 |

## 5. Frontend ↔ Gateway 字段变化

详见：`docs/handoff/FRONTEND_GATEWAY_FIELD_CHANGELOG.md` Round 1B 小节。

### 本 轮 归 纳

- **REST 请求字段**：无变化（延续 Round 1A：Auto 模式仍省略 agentName）
- **REST 响应字段**：无变化
- **SSE event type**：无新增或删除
- **SSE event data 字段**：无新增 wire 字段。前端新增使用已有可选字段（STATE_UPDATE.selectedAgentName 等、artifact.delta.type）
- **前端对已有 SSE 字段的解析语义变化**：
  - `resolveEventAgentName` 结果新增用于 Web Preview 判定
  - `artifact.delta` type=webpage/html 新增设置 `hasWebArtifactEvidence`
  - `TOOL_CALL_*` web_preview/generate_html_snippet 新增设置 `hasWebArtifactEvidence`
  - `STATE_UPDATE` 新增提取 `selectedAgentName`, `selectedAgentDisplayName`, `taskAgentNames`
- **无 wire 字段变化但行为变化**：
  - WebPreview sandbox="" + CSP meta 注入
  - 会话标题 localStorage 客户端持久化
  - ChatWindow "Thinking…" 骨架气泡
  - Markdown code fence HTML 不再触发 Web Preview
  - 长内容 CSS 加固

## 6. 测试结果

| 命令 | 结果 | 说明 |
|------|------|------|
| `npm run test` | **63/63 通过** (8 文件) | 新增 14 个测试（5 消息存储 + 2 WebPreview + 7 localStorage） |
| `npx tsc --noEmit` | **Clean** | 无类型错误 |
| `npm run build` | **Success** | vite v6.4.2，2051 模块 |

测试文件分布：
- `src/agui/client.test.ts` — 6 tests
- `src/stores/agentStore.test.ts` — 2 tests
- `src/stores/messageReplay.test.ts` — 5 tests
- `src/stores/messageStore.test.ts` — 22 tests (+5 new)
- `src/lib/conversationTitles.test.ts` — 7 tests (new)
- `src/components/WebPreview.test.tsx` — 5 tests (+2 new)
- `src/components/CodePreview.test.tsx` — 7 tests
- `src/components/MessageBubble.test.tsx` — 9 tests

## 7. 已知限制

1. **Auto 模式下 Web Preview 依赖 SSE 元数据或 artifact 证据**：如果 orchestrator 完全不带 agentName/artifact.type 字段，且 web-agent 返回纯文本 HTML（无工具调用、无 artifact delta），则 Auto 模式下仍无法触发 Web Preview。这是一个极端边缘情况，需要 orchestrator 或 web-agent 至少提供一种信号。

2. **Web Preview sandbox="" 下不支持脚本交互**：预览中的任何 JS（事件处理、表单验证等）不会执行。这是安全决策，不是缺陷。

3. **会话标题 localStorage 是前端临时持久化**：不是 Gateway 权威持久化。不同浏览器/设备不共享。Gateway 标题优先。主要改进是同一浏览器内刷新不丢标题。

4. **人工联调尚未执行**：本 轮 所有修改通过自动化测试验证链路正确性，但未在真实服务环境下端到端联调。

5. **OrchestrationCard 的 selectedAgent 字段**：依赖 orchestrator 在 STATE_UPDATE 中发送这些可选字段。如果 orchestrator 不发送，卡片退化为 Round 1A 的策略摘要。

## 8. 非本轮范围 / 已澄清问题

1. **联调报告第 3 条纠正**：真实 LLM 测试失败是因为没有引入正确的 llmProvider 包，不是 mock 测试问题，也不是本轮前端修复范围。
2. **Docker COPY 整个 monorepo、GOPROXY** 是工程化 / 构建问题，不属于本轮前端修复。
3. **Agent 注册中心统一** 不是本轮范围。
4. **Orchestrator 重构** 不是本轮范围。
5. **Gateway 标题持久化** 不是本轮范围（本轮用 localStorage 轻量替代）。
6. **不接入真实 LLM**。
7. **不新增大型依赖**。
8. **未修改 server/、agents/、Dockerfile、docker-compose 文件**。
