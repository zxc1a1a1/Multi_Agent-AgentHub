# Step 3-D Gateway Runtime Write Path 完成报告

## 1. 本步目标

将 Step 3-B/3-C 的 persistence foundation 接入 Gateway runtime write path，使 Run / RunStep / Message 在 SSE 事件流中实时写入 SQLite，同时保持现有 MemoryStore 行为和 SSE/API 输出不变。

## 2. 新增 / 修改文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `services/gateway/httpapi/persistence_writer.go` | **新增** | PersistenceWriter：消费 AG-UI 事件，写入 Run/RunStep/Message 到 SQLite |
| `services/gateway/httpapi/persistence_writer_test.go` | **新增** | 12 个测试（Single Code/Web、Mixed OrderedParallel、RunError、Sanitize、Nil safety、SSE 不变性验证、Update 方法等） |
| `services/gateway/httpapi/server.go` | 修改 | Server 新增 optional `persistenceWriter` 字段 + `WithPersistenceWriter` Option；handleChat 在 SSE loop 中调用 PersistenceWriter |
| `services/gateway/internal/persistence/sqlite/store.go` | 修改 | 新增 `UpdateRunStatus`、`UpdateRunStepStatus`、`UpdateMessageContentAndStatus`、`ListRunsByConversation` 四个方法 |
| `docs/refactor/step-3d-gateway-runtime-write-path-report.md` | **新增** | 本报告 |
| `docs/refactor/persistence-implementation-plan.md` | 修改 | 补充 Step 3-D 完成状态 |

## 3. Runtime Write Path 设计

### 3.1 Server 扩展

```go
type Server struct {
    // ...existing fields...
    persistenceWriter *PersistenceWriter  // optional, nil-safe
}
```

通过 `WithPersistenceWriter(w *PersistenceWriter) Option` 注入。当 `persistenceWriter == nil`（默认），handleChat 行为与之前完全一致。

### 3.2 handleChat 写入点

```
1. Save user message to MemoryStore (existing)
2. Save user message to PersistenceWriter → SQLite (new, optional)
3. SSE loop:
   a. Translate adk.Event → []agui.Event
   b. Accumulate text for MemoryStore (existing)
   c. PersistenceWriter.HandleEvent(ctx, conversationID, item) (new, optional)
   d. Write SSE event to client (existing)
4. After loop: save merged assistant message to MemoryStore (existing)
```

PersistenceWriter 写入失败不阻塞 SSE 流，不向用户暴露内部错误。

## 4. PersistenceWriter 行为

| Event | 行为 |
|-------|------|
| `RUN_STARTED` | Create Run (status=running, id=event.RunID) |
| `TEXT_MESSAGE_START` | 1. Finalize 上一条 message（flush buffer, status=sent）2. 新 taskId → Create RunStep (status=running) 3. Create Message (status=streaming, sender 来自 event.Sender) |
| `TEXT_MESSAGE_CONTENT` | Buffer delta（不写 DB） |
| `TEXT_MESSAGE_END` | Flush buffer → Update message content + status=sent；Update step status=completed |
| `RUN_ERROR` | Update Run/Step/Message → status=failed, error fields（error_message 来自 agui.Event.Error，已由 translator 脱敏） |
| `RUN_FINISHED` | Update Run → status=completed, finished_at=now |

### 4.1 Ordered Parallel 多消息分离

每个 `TEXT_MESSAGE_START` 携带独立的 `messageId`。当 `messageId` 变化时：
1. Finalize 当前 message（flush buffer + status=sent）
2. 新 `taskId` → 创建新 RunStep
3. 创建新 Message

结果：3 条 agent Message（web-agent / code-agent / orchestrator）独立存在，不合并。

### 4.2 线程安全

`PersistenceWriter` 内部使用 `sync.Mutex`，所有 HandleEvent 调用线程安全。

## 5. SqliteStore 新增方法

| 方法 | 说明 |
|------|------|
| `UpdateRunStatus(ctx, runID, status, errorCode, errorMessage, finishedAt)` | 更新 Run 状态/错误/完成时间 |
| `UpdateRunStepStatus(ctx, stepID, status, errorCode, errorMessage, finishedAt)` | 更新 RunStep 状态/错误/完成时间 |
| `UpdateMessageContentAndStatus(ctx, id, content, status, errorCode, errorMessage, updatedAt)` | 原子更新 Message content + status |
| `ListRunsByConversation(ctx, conversationID)` | 按对话列出所有 Run（按 started_at DESC） |

## 6. 测试覆盖

**12 个新增测试**，全部通过：

| 测试 | 验证内容 |
|------|----------|
| `TestPersistenceWriterSingleCode` | 1 Run + 1 Step + user + code-agent Message (sender_name/agent_name/content 正确) |
| `TestPersistenceWriterSingleWeb` | 1 Run + 1 Step + user + web-agent Message |
| `TestPersistenceWriterMixedOrderedParallel` | 1 Run + 3 Steps + 4 Messages (user + web + code + orchestrator，不合并) |
| `TestPersistenceWriterRunError` | Run/Step/Message status=failed，error fields 正确，content 保留 |
| `TestPersistenceWriterRunErrorSanitized` | error_message 不含 sk- token / panic / paths |
| `TestHandleChatWithPersistenceWriterDoesNotChangeSSE` | **Handler-level**: SSE 事件不变 + SQLite 写入正确 |
| `TestHandleChatWithoutPersistenceWriterPreservesLegacyBehavior` | writer=nil 时行为完全不变 |
| `TestHandleChatWithPersistenceWriterMultiAgentSSE` | **Handler-level**: ordered_parallel 4 条独立 Message + 3 RunStep |
| `TestPersistenceWriterNilIsSafe` | nil writer 所有方法不 panic |
| `TestPersistenceWriterErrorDoesNotPanic` | DB 写入失败（FK 违反）不 panic |
| `TestUpdateRunAndStepStatus` | UpdateRunStatus / UpdateRunStepStatus / UpdateMessageContentAndStatus 方法验证 |

**所有现有测试继续通过**（server_test.go 8 个测试零修改）。

## 7. 未做内容

- 未在 main / compose 中启用 SQLite（PersistenceWriter 仅通过 `WithPersistenceWriter` Option 注入）
- 未写真实 DB 文件（测试使用 `:memory:` SQLite）
- 未替换 MemoryStore（MemoryStore 仍是主存储）
- 未修改 `/api/conversations/{id}/messages` 返回格式（仍从 MemoryStore 读取）
- Message replay / frontend refresh recovery 留待 Step 3-E

## 8. 安全检查

| 检查项 | 结果 |
|--------|------|
| `git diff` 中真实 API key / token | 无 |
| `git diff` 中 sk- 开头真实长 key | 无 |
| `git status` 中 frontend/dist / node_modules / .env / .exe | 无 |
| `.claude/settings.local.json` 修改 | 未修改 |
| `.exe` 文件 | 无新增 |
| DATABASE_URL 写入 | 无 |

## 9. 禁止项自查

| 禁止项 | 状态 |
|--------|------|
| 不让 Gateway 直连 Agent | 遵守 |
| 不修改 Orchestrator runtime | 遵守 |
| 不修改 services/agents/** | 遵守 |
| 不修改 frontend/src/** | 遵守 |
| 不修改 docker-compose | 遵守 |
| 不修改 .github/workflows | 遵守 |
| 不修改 smoke-new-arch.sh | 遵守 |
| 不弱化 smoke 断言 | 遵守 |
| 不接真实 LLM / 不引入 API key | 遵守 |
| 不引入 MySQL / 不读取 .env | 遵守 |
| 不服务化 vision-agent / 不引入 LLMPlanner | 遵守 |
| 不修改 .claude/settings.local.json | 遵守 |
| 不 git add / 不 commit / 不 push | 遵守 |

## 10. 下一步建议

Step 3-B/3-C/3-D 一起提交并通过 GitHub Actions New Architecture Smoke 后，再进入 Step 3-E Message replay / frontend refresh recovery。
