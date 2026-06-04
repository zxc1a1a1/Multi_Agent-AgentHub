# Step 2-B / 2-C Frontend Multi-Agent UI Rendering 完成报告

## 1. 本步目标

验证并修复 AgentHub v1.0 Productization Stage Step 2-B：确认 Step 2-A 修复后的 AG-UI / SSE 事件在前端真实 UI 中正确渲染，修复 multi-agent UI rendering 问题，确保 single code / single web / mixed ordered_parallel 三种场景的 UI 展示正确。

## 2. 修改文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `frontend/src/stores/messageStore.ts` | 修改 | 新增 `sanitizeErrorText()` 函数，前端侧防御性过滤敏感信息 |
| `frontend/src/stores/messageStore.test.ts` | 修改 | 新增 5 个测试用例（sender 对象字段传播、错误脱敏） |
| `frontend/e2e/mocks.ts` | 修改 | 更新为 AG-UI v1.0 `sender` 对象格式，对齐真实 Gateway 输出 |
| `frontend/src/agui/skills.ts` | 修改 | 仅添加注释说明，三个 capability 已在 Step 2-A 补全 |
| `docs/refactor/frontend-multi-agent-ui-rendering-report.md` | 新增 | 本报告 |

**未修改（但审计覆盖）：**
- `pkg/runtime/agui/` — Step 2-A 已修复，当前正确运行
- `services/gateway/` — Step 2-A 已修复，当前正确运行
- `services/orchestrator/` — 无需修改
- `services/agents/` — 无需修改

## 3. UI 链路审计结论

### 3.1 SSE Parser（client.ts）

- ✅ 按 `\n\n` 拆分 SSE block
- ✅ 处理粘包（tail buffer）和拆包（跨 chunk 拼接）
- ✅ 安全跳过 malformed JSON
- ✅ `event:` / `data:` 行识别
- ✅ `event.type` 回退到 SSE event name

### 3.2 Message Store（messageStore.ts）

- ✅ 按 `messageId` 聚合 delta — `ensureAgentMessage()` 在 `messageId` 变化时创建新消息气泡
- ✅ 不同 sender / 不同 `messageId` 分离显示 — 每个唯一 `messageId` 对应独立 Message 对象
- ✅ `TEXT_MESSAGE_START` → 重置 `agentMsgId` / `agentContent` / `codeBlocks` / `webPreviews`
- ✅ `TEXT_MESSAGE_CONTENT` / `message` / `message.delta` → 追加 delta 到当前消息
- ✅ `TEXT_MESSAGE_END` / `message.end` → 完成流式消息
- ✅ `TOOL_CALL_START/ARGS/END` → 解析 tool call，按 `toolName` 分发到 `code_preview` / `web_preview`
- ✅ `RUN_STARTED` → 初始化 run 状态
- ✅ `RUN_FINISHED` → 停止 streaming
- ✅ `RUN_ERROR` / `error` → 标记消息为 failed，停止 streaming
- ✅ `STATE_UPDATE` / `state.delta` → 提取 phase/messageId，console.debug 日志
- ✅ **新增** `sanitizeErrorText()` — 前端侧过滤 API key、private key、sk-token、路径、panic/stack trace

### 3.3 Message Rendering（MessageBubble.tsx）

- ✅ 每个 Message 对象渲染为独立气泡（ChatWindow `messages.map`）
- ✅ 用户消息右对齐（蓝色），Agent 消息左对齐（灰色）
- ✅ senderName 展示（`message.senderName || getAgentDisplayName(message.agentName)`）
- ✅ Agent avatar 展示，streaming 时带绿色脉冲指示器
- ✅ Error 消息显示 "Failed to generate response" 指示器
- ✅ 不使用 `dangerouslySetInnerHTML` 渲染不可信内容（StreamingText 使用 react-markdown）

### 3.4 Sender 展示

- ✅ code-agent → 显示 "Code Agent"
- ✅ web-agent → 显示 "Web Agent"
- ✅ orchestrator → 显示 "Orchestrator"（`humanizeAgentName` 处理）
- ✅ `sender.type` / `sender.name` / `sender.displayName` 通过 `resolveEventSenderName` 提取
- ✅ 无 sender 对象时回退到 `agentName` 和 `knownDisplayNameMap`

### 3.5 mixed ordered_parallel 展示

- ✅ Web-agent 输出 → 独立气泡，`messageId: "msg-web"`, `sender: {type: "agent", name: "web-agent"}`
- ✅ Code-agent 输出 → 独立气泡，`messageId: "msg-code"`, `sender: {type: "agent", name: "code-agent"}`
- ✅ Orchestrator summary → 独立气泡，`messageId: "msg-summary"`, `sender: {type: "orchestrator", name: "orchestrator"}`
- ✅ 三个气泡不合并，各自独立内容
- ✅ 测试覆盖：`creates 3 separate agent messages for web-agent, code-agent, orchestrator`

### 3.6 run_error 展示

- ✅ Gateway 层（filter.go）过滤 secrets/tokens/paths/stack traces
- ✅ **新增**前端层（sanitizeErrorText）防御性过滤敏感信息
- ✅ UI 显示 "Failed to generate response" 通用失败指示
- ✅ 不泄露 internal service URL / token / stack trace / panic / fatal
- ✅ 测试覆盖：4 个错误脱敏测试（secrets/path/panic/sk-token）

### 3.7 Preview 组件选择依据

- ✅ WebPreview/CodePreview 选择基于 `toolName`（`code_preview` / `web_preview`），不基于 `agentName`
- ✅ `artifact.type`（`code` / `webpage` / `html`）用于 artifact.delta 处理
- ✅ 无 `agentName` 硬编码路由到组件
- ✅ `isSupportedAgentName` 仅用于 display name 回退，不用于组件选择
- ✅ WebPreview 使用 Safe Mode（`<pre><code>` 文本化展示），无 iframe
- ✅ CodePreview 使用 highlight.js 语法高亮（预信任输出，有测试验证 `<script>` 标签不执行）

## 4. 修复内容

### 4.1 前端错误脱敏（新增）

在 `messageStore.ts` 新增 `sanitizeErrorText()` 函数，作为防御性第二层过滤：
- 过滤 API key 赋值（`OPENAI_API_KEY=value` → `[redacted]`）
- 过滤 private key blocks（`-----BEGIN ... PRIVATE KEY-----` → `[redacted]`）
- 过滤 sk- 前缀 token（`sk-proj-...` → `[redacted]`）
- 过滤文件路径（`C:\Users\...`、`/usr/local/...` → `[path]`）
- 替换 panic/stack trace 为通用 "internal error"

### 4.2 E2E Mock 对齐（更新）

将 e2e mock 中的旧格式字段（`senderName`、`agentName`、`error: string`）更新为 AG-UI v1.0 标准格式：
- `sender: {type, name, displayName}` 对象
- `error: {code, message}` 对象
- `delta` 字段（而非 `content`）用于 TEXT_MESSAGE_CONTENT

### 4.3 测试扩展（新增 5 个测试用例）

messageStore.test.ts 新增测试：
1. `propagates sender object fields to agent message` — 验证 sender 对象字段传播
2. `propagates orchestrator sender type for summary messages` — 验证 orchestrator summary
3. `strips sensitive secrets from error message text` — 验证 API key 脱敏
4. `strips file paths from error message text` — 验证路径脱敏
5. `replaces panic/stack trace with generic error message` — 验证 panic 替换
6. `strips sk-prefixed token from error message` — 验证 sk-token 脱敏

## 5. 旧路径 / 旧文件处理

本步未删除旧文件。

`docs/refactor/` 下有 27 个文件，大量是历史 Phase 报告。这些不属于 Step 2-B 范围，记录为后续清理候选：

- `phase-0-readiness-report.md` — 历史 Phase 0 报告
- `phase-8-*.md` — 历史 Phase 8 报告（4 个文件）
- `phase-9-*.md` — 历史 Phase 9 报告（2 个文件）
- `phase-10-*.md` — 历史 Phase 10 报告（3 个文件）
- `new-arch-demo-*.md` — 历史 Demo 文档（3 个文件）
- `agent-capability-inventory.md`、`agent-consolidation-plan.md`、`agent-reference-scan-report.md` — 可能过期

建议后续统一清理，但**不在本步执行**。

## 6. 测试结果

### 6.1 Frontend 测试

```
npm test -- --run → 31 tests passed (5 files)
  - agui/client.test.ts: 6 tests ✅
  - stores/agentStore.test.ts: 2 tests ✅
  - stores/messageStore.test.ts: 15 tests ✅ (原 10 + 新增 5)
  - components/WebPreview.test.tsx: 1 test ✅
  - components/CodePreview.test.tsx: 7 tests ✅
```

### 6.2 Frontend 构建

```
npm run build → tsc -b && vite build ✅
  - TypeScript 编译无错误
  - Vite 构建成功（2049 modules transformed）
```

### 6.3 Go 测试

未修改 Go 文件，无需运行 Go 测试。

### 6.4 本地 Docker

本地 Docker 环境未使用。runtime 替代验收依赖 **GitHub Actions New Architecture Smoke**。push 后必须确认 Actions 绿色通过。

## 7. 安全检查

| 检查项 | 结果 |
|--------|------|
| `git diff` 中真实 API key / token | ❌ 无 — 匹配项为旧文件删除、测试 mock fake token、正则 pattern 名称 |
| `git diff` 中 sk- 开头的真实长 key | ❌ 无 — 匹配项为测试 mock 中的假 token |
| `git status` 中 frontend/dist | ❌ 无 |
| `git status` 中 node_modules | ❌ 无 |
| `git status` 中 .env | ❌ 无 |
| `git status` 中 .exe | ❌ 无 |
| `.claude/settings.local.json` 修改 | ❌ 未修改 |
| `.exe` 文件（全项目扫描） | ❌ 无 |

## 8. 禁止项自查

| 禁止项 | 状态 |
|--------|------|
| 不退回 Gateway 直连 Agent | ✅ 遵守 — Gateway 仅通过 OrchestratorRunService 调用 Orchestrator |
| 不让 Frontend 直连 Orchestrator | ✅ 遵守 — 前端仅调 `/api/*` |
| 不让 Frontend 直连 Agent | ✅ 遵守 — 无 Agent URL 引用 |
| 不绕过 Orchestrator | ✅ 遵守 |
| 不绕过 PlanValidator | ✅ 遵守 |
| 不合并 mixed ordered_parallel 为一个气泡 | ✅ 遵守 — 按 messageId 分离气泡 |
| 不通过 agentName 硬编码决定 WebPreview/CodePreview | ✅ 遵守 — 基于 toolName/artifact.type |
| 不进入持久化 | ✅ 遵守 — 未修改数据库或 schema |
| 不新增 Artifact schema | ✅ 遵守 |
| 不服务化 vision-agent | ✅ 遵守 |
| 不引入 LLMPlanner | ✅ 遵守 |
| 不引入真实 LLM key | ✅ 遵守 |
| 不依赖真实 OCR | ✅ 遵守 |
| 不弱化 CI / smoke | ✅ 遵守 |
| 不修改 legacy server/ 作为主路径 | ✅ 遵守 |
| 不在旧 agents/ 下新增主功能 | ✅ 遵守 |
| 不删除仍被引用的旧文件 | ✅ 遵守 |
| 不提交 .env / token / DATABASE_URL / 私钥 / dist / node_modules / .exe | ✅ 遵守 |
| 不修改 .claude/settings.local.json | ✅ 遵守 |
| 不执行 git clean | ✅ 遵守 |
| 不使用 git add . | ✅ 遵守 |

---

## Step 2-C: UI 验收证据补强与旧路径清理候选收口

### 1. 本步目标

在 Step 2-B 已修复基础上，补强前端 multi-agent UI 的验收证据：
- 增加 MessageBubble 组件测试（sender 展示验证）
- 增加 mixed ordered_parallel e2e mock（3 Agent 独立消息场景）
- 增加 preview 选择不依赖 agentName 的 store 测试
- 执行旧路径引用扫描，记录清理候选

### 2. 修改文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `frontend/src/components/MessageBubble.test.tsx` | **新增** | 9 个组件测试（sender 标签、错误状态、preview 块） |
| `frontend/e2e/mocks.ts` | 修改 | 新增 `buildMixedOrderedParallelSSE()` mock（3 Agent 场景） |
| `frontend/src/stores/messageStore.test.ts` | 修改 | 新增 2 个 preview 选择测试（toolName-based，非 agentName） |
| `docs/refactor/legacy-cleanup-candidates.md` | **新增** | 旧路径清理候选记录 |
| `docs/refactor/frontend-multi-agent-ui-rendering-report.md` | 修改 | 追加 Step 2-C 内容 |

### 3. UI 验收覆盖矩阵

| 验收场景 | 测试层 | 覆盖状态 |
|----------|--------|----------|
| single code → code-agent 显示 | store + component + e2e mock | ✅ |
| single web → web-agent 显示 | store + component + e2e mock | ✅ |
| mixed ordered_parallel → 3 独立消息 | store + e2e mock | ✅ |
| 不同 sender 不合并为同一气泡 | store (messageId 分离) | ✅ |
| run_error 安全展示（不泄露敏感信息） | store（5 个脱敏测试） | ✅ |
| preview 选择基于 toolName，非 agentName | store（2 个工具名测试） | ✅ |
| sender 对象字段传播（type/name/displayName） | store（2 个 sender 测试） | ✅ |
| MessageBubble sender 标签渲染 | component（4 个 label 测试） | ✅ |
| MessageBubble 错误指示器 | component（2 个状态测试） | ✅ |
| MessageBubble preview 块渲染 | component（2 个 preview 测试） | ✅ |

### 4. 新增测试详情

#### 4.1 MessageBubble 组件测试（9 tests）

`frontend/src/components/MessageBubble.test.tsx`:
1. code-agent sender label 显示 "Code Agent"
2. web-agent sender label 显示 "Web Agent"
3. orchestrator sender label 显示 "Orchestrator"（独立气泡）
4. 用户消息无 sender 标签（仅 avatar）
5. failed 状态显示 "Failed to generate response"
6. codeBlocks 渲染 CodePreview
7. webPreviews 渲染 WebPreview Safe Mode
8. senderName 缺失时回退到 agentName display
9. streaming 状态不显示错误指示器

#### 4.2 Preview 选择测试（2 tests，messageStore）

`frontend/src/stores/messageStore.test.ts`:
1. `code_preview` toolCall 在 web-agent 上下文仍创建 CodePreview（证明基于 toolName）
2. `web_preview` toolCall 在 code-agent 上下文仍创建 WebPreview（证明基于 toolName）

#### 4.3 E2E mock 扩展

`frontend/e2e/mocks.ts`:
- 新增 `buildMixedOrderedParallelSSE()` — 模拟 web-agent → code-agent → orchestrator summary 三种独立消息
- 使用不同 messageId 和 sender 对象

### 5. 旧路径清理候选

已新增 `docs/refactor/legacy-cleanup-candidates.md`，记录：

- **保留（9 个）**：当前活跃使用的架构/边界/guide 文档
- **归档候选（16 个）**：Phase 0-10 历史报告、Demo 文档、Agent 能力清单
- **审查候选（3 个）**：agent-capability-inventory、agent-consolidation-plan、agent-reference-scan-report

本轮**未删除任何文件**。历史报告在 `docs/refactor/` 内部存在交叉引用链，统一归档需要独立策略。

### 6. 测试结果（Step 2-C）

```
npm test -- --run → 42 tests passed (6 files) ✅
  - agui/client.test.ts:          6 tests
  - stores/agentStore.test.ts:     2 tests
  - stores/messageStore.test.ts:  17 tests (原 15 + 新增 2)
  - components/CodePreview.test.tsx: 7 tests
  - components/WebPreview.test.tsx:  1 test
  - components/MessageBubble.test.tsx: 9 tests (新增)
```

```
npm run build → tsc -b && vite build ✅
  - TypeScript 编译无错误
  - Vite 构建成功
```

Go 文件未修改，无需 Go test。本地 Docker 未使用，runtime 替代验收依赖 GitHub Actions New Architecture Smoke。

### 7. 安全检查（Step 2-C）

| 检查项 | 结果 |
|--------|------|
| `git diff` 中真实 API key / token | ❌ 无 |
| `git diff` 中 sk- 开头真实长 key | ❌ 无（仅有 fake test token：`sk-abc123...`） |
| `git status` 中 frontend/dist / node_modules / .env / .exe | ❌ 无 |
| `.claude/settings.local.json` 修改 | ❌ 未修改 |
| `.exe` 文件 | ❌ 无 |

### 8. 禁止项自查（Step 2-C 追加）

所有 Step 2-B 禁止项仍然遵守，Step 2-C 新增约束：

| 禁止项 | 状态 |
|--------|------|
| 不修改 pkg/runtime/agui | ✅ 遵守 |
| 不修改 services/gateway | ✅ 遵守 |
| 不修改 services/orchestrator | ✅ 遵守 |
| 不修改 CI / smoke | ✅ 遵守 |
| 不修改 docker-compose | ✅ 遵守 |
| 不大规模删除 server/ 或 agents/ | ✅ 遵守 — 仅记录候选 |
| 不删除仍被引用的文件 | ✅ 遵守 — 所有候选均被交叉引用 |

---

## 9. 下一步建议

Step 2-B / 2-C 一起提交并通过 GitHub Actions New Architecture Smoke 后，才能进入 Conversation / Run / Message 持久化或 Artifact 相关工作。
