# Step 3-E/3-F Replay & Audit 完成报告

## 1. 本轮目标

完成 AgentHub v1.0 Productization Stage 的 Step 3-E/3-F 合并轮：
- 修复 Message replay / Frontend refresh recovery
- 补齐 Failure/Audit 基础能力
- 小范围 legacy cleanup 审计

## 2. 新增 / 修改文件

### 新增文件

| 文件 | 说明 |
|------|------|
| `services/gateway/httpapi/replay.go` | ReplayMessage 类型定义 + sqlite.Message / store.Message → ReplayMessage 转换 |
| `services/gateway/httpapi/replay_test.go` | 10 个测试（转换、API 格式、失败审计、脱敏、向后兼容） |
| `frontend/src/stores/messageReplay.test.ts` | 5 个前端 replay 测试（3 agent 恢复、displayName 优先级、失败状态、run/step 关联） |

### 修改文件

| 文件 | 说明 |
|------|------|
| `services/gateway/httpapi/server.go` | 新增 `persistenceStore` 字段 + `WithPersistenceStore` Option；`handleConversationMessages` 优先从 SQLite replay，回退到 MemoryStore |
| `services/gateway/internal/persistence/sqlite/store.go` | 新增 `ListFailedRunsByConversation`、`ListFailedMessagesByConversation`、`ListArtifactsByConversation`、`ListArtifactsByMessageID` 四个审计查询方法 |
| `frontend/src/types/index.ts` | Message 类型新增 `runId`、`stepId`、`sseMessageId`、`errorCode`、`errorMessage` 字段 |
| `frontend/src/stores/messageStore.ts` | StoredMessage 新增 `senderDisplayName`、`runId`、`sseMessageId`、`stepId`、`status`、`errorCode`、`errorMessage`；`loadMessages()` senderName 优先级：displayName → senderName → agentName → author；status 从 API 读取而非硬编码 `'sent'` |
| `frontend/src/stores/messageStore.test.ts` | 新增 `import * as api` 引用 |

### 删除文件

无。所有历史 phase 报告通过 `legacy-cleanup-candidates.md` 交叉引用，不符合无引用删除条件。

## 3. Gateway Replay API 设计

### 3.1 ReplayMessage 格式

`GET /api/conversations/{id}/messages` 返回 `[]ReplayMessage`：

```json
{
  "id": "db-msg-1",
  "conversationId": "conv-1",
  "runId": "run-001",
  "stepId": "step-w",
  "sseMessageId": "msg-web",
  "senderType": "agent",
  "senderName": "web-agent",
  "senderDisplayName": "Web Agent",
  "agentName": "web-agent",
  "role": "assistant",
  "author": "web-agent",
  "content": "<section><h1>Login Page</h1></section>",
  "text": "<section><h1>Login Page</h1></section>",
  "status": "sent",
  "errorCode": "",
  "errorMessage": "",
  "artifacts": null,
  "createdAt": "2026-06-04T00:00:01Z",
  "updatedAt": "2026-06-04T00:00:01Z"
}
```

### 3.2 字段兼容策略

- `author` + `text` 保留（旧前端可用）
- `content` 与 `text` 等同（新前端优先用 content）
- `senderDisplayName` 由 `knownDisplayName()` 映射：web-agent → "Web Agent", code-agent → "Code Agent", orchestrator → "Orchestrator", assistant → "Assistant"
- `artifacts` 从 `metadata_json.artifacts` 解析，不存在时省略
- `status` 从 DB 读取（sent / failed）
- `errorMessage` 来自已脱敏的 DB 存储

### 3.3 读取优先级

1. Server 配置了 `WithPersistenceStore(sqlStore)` → 从 SQLite 读取，返回 ReplayMessage 格式
2. 未配置 → 从 MemoryStore 读取，转换后返回 ReplayMessage 格式（向后兼容）

### 3.4 未改变行为

- `POST /api/chat` — 不变
- `GET /api/conversations` / `POST /api/conversations` — 不变
- `GET /api/agents` — 不变
- `GET /health` — 不变
- MemoryStore 仍为默认存储路径

## 4. Frontend Refresh Recovery

### 4.1 loadMessages() 更新

senderName 的解析优先级：
1. `raw.senderDisplayName` — Gateway replay 返回的人类可读名称
2. `raw.senderName` — 原始 sender 名称
3. `raw.agentName` — agent 标识
4. `raw.author` — 旧格式兼容

status 解析：
- `raw.status === 'failed'` → `'failed'`
- `raw.status === 'streaming'` → `'streaming'`
- 其他 → `'sent'`

### 4.2 恢复效果

- mixed ordered_parallel 刷新后：3 条独立 agent 消息（web-agent / code-agent / orchestrator），不合并
- 用户消息：senderType='user'，正确识别
- 旧格式兼容：author='assistant' + text='merged' → 1 条 agent 消息，displayName='Assistant'

## 5. Artifact Replay 基础支持

- `parseArtifactsFromMetadata()` 从 `metadata_json.artifacts` 提取 JSON 数组
- ReplayMessage.Artifacts 字段为 `json.RawMessage`，前端 `parseArtifacts()` 可直接解析
- 当前 PersistenceWriter 未写入 artifacts 到 metadata_json（留待后续步骤）

## 6. Failure / Audit 基础能力

### 6.1 SqliteStore 新增审计方法

| 方法 | 说明 |
|------|------|
| `ListFailedRunsByConversation(ctx, conversationID)` | 按对话查询 status='failed' 的 Run |
| `ListFailedMessagesByConversation(ctx, conversationID)` | 按对话查询 status='failed' 的 Message |
| `ListArtifactsByConversation(ctx, conversationID)` | 按对话查询所有 Artifact |
| `ListArtifactsByMessageID(ctx, messageID)` | 按消息查询 Artifact |

### 6.2 错误脱敏

- PersistenceWriter 只接收已脱敏的错误（AG-UI translator filter 已过滤）
- Replay API 不额外暴露内部错误详情
- 测试验证：replay response 不包含 sk-token、OPENAI_API_KEY、panic、stack trace、文件路径

### 6.3 未实现（留待后续）

- 自动 retry
- 监控/告警
- 复杂 Run History UI
- 实时重试触发

## 7. Legacy Cleanup 审计

### 7.1 审计结论

所有 `docs/refactor/phase-*.md` 文件通过 `legacy-cleanup-candidates.md` 和 `frontend-multi-agent-ui-rendering-report.md` 交叉引用，不符合"明确无引用"的删除条件。

### 7.2 不删除的文件

- `server/` 目录 — 保留
- `agents/` 目录 — 保留
- `docker-compose.yml` — 保留
- 所有 `docs/refactor/*` — 有交叉引用
- `AgentHub_重构与模块解耦阶段报告.md` — 用户自有文档
- `AGENTS.md` — 项目配置
- `readme.md` — 项目 README

## 8. 测试结果

### 8.1 Go 测试

```
go test ./services/gateway/... -count=1 → 11 packages all pass
```

新增 replay 测试：
- `TestSqliteToReplayMessages` — 4 消息转换验证
- `TestMemoryStoreToReplayMessages` — 向后兼容转换
- `TestReplaySenderDisplayName` — 8 个 display name 映射
- `TestReplayParseArtifactsFromMetadata` — artifact 解析 + 边界条件
- `TestHandleConversationMessagesWithPersistence` — Handler 级 SQLite replay
- `TestHandleConversationMessagesWithoutPersistence` — Handler 级 MemoryStore 兼容
- `TestHandleConversationMessagesWithPersistenceNotFound` — 空对话返回空列表
- `TestListFailedRunsByConversation` — 失败 Run 查询
- `TestListFailedMessagesByConversation` — 失败 Message 查询
- `TestFailedMessageErrorSanitized` — 错误脱敏验证
- `TestReplayJSONFormatIsBackwardCompatible` — JSON 格式向后兼容

### 8.2 Frontend 测试

```
npm test -- --run → 47 tests passed (7 files)
```

新增 5 个 replay 测试：
1. 3 agent 独立消息恢复（4 条 total）
2. senderDisplayName 优先于 senderName
3. 向后兼容 author='assistant'
4. 失败状态 + errorCode/errorMessage 传递
5. runId/stepId/sseMessageId 传递

### 8.3 Frontend 构建

```
npm run build → tsc -b && vite build ✓ (TypeScript 编译无错误)
```

## 9. 安全检查

| 检查项 | 结果 |
|--------|------|
| `git diff` 中真实 API key / token | 无 |
| `git diff` 中 sk- 开头真实长 key | 无 |
| `git status` 中 frontend/dist | 无 |
| `git status` 中 node_modules | 无 |
| `git status` 中 .env | 无 |
| `git status` 中 .exe | 无 |
| `.claude/settings.local.json` 修改 | 未修改 |
| DATABASE_URL 写入 | 无 |

## 10. 禁止项自查

| 禁止项 | 状态 |
|--------|------|
| 不允许 Gateway 直连 Agent | 遵守 |
| 不允许 Frontend 直连 Orchestrator | 遵守 |
| 不允许修改 Orchestrator runtime | 遵守 |
| 不允许修改 services/agents/** | 遵守 |
| 不允许修改 docker-compose.new-arch.yml | 遵守 |
| 不允许修改 .github/workflows/new-arch-smoke.yml | 遵守 |
| 不允许修改 smoke-new-arch.sh | 遵守 |
| 不允许弱化 smoke 断言 | 遵守 |
| 不允许接真实 LLM | 遵守 |
| 不允许引入真实 API key | 遵守 |
| 不允许提交 .env / token / DATABASE_URL / 私钥 / dist / node_modules / .exe | 遵守 |
| 不允许修改 .claude/settings.local.json | 遵守 |
| 不允许 git add . | 遵守 |
| 不允许 commit | 遵守 |
| 不允许 push | 遵守 |

## 11. 下一步建议

1. 提交 Step 3-E/3-F 变更并通过 GitHub Actions New Architecture Smoke 验证
2. 在 compose 中启用 SQLite（`WithPersistenceStore(store)`）使 replay 在集成环境中生效
3. 考虑在 PersistenceWriter 中同步写入 artifact metadata，使 replay 可恢复预览块
4. 考虑将 `WithPersistenceWriter` 和 `WithPersistenceStore` 合并为一个统一的 persistence Option

---

- Created: 2026-06-04
- Step: AgentHub v1.0 Productization Stage Step 3-E/3-F
- Status: 完成，待审核
