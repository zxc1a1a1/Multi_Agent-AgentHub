# Step 3-B/3-C Persistence Foundation 完成报告

## 1. 本步目标

完成 SQLite schema / migration runner / SqliteStore 基础实现。本轮只建立 persistence foundation，**未接入 Gateway runtime**，未替换 MemoryStore，未修改 /api 行为。

## 2. 新增 / 修改文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `services/gateway/internal/persistence/migrations/001_initial_schema.sql` | **新增** | 6 表 DDL + 12 索引 + schema_migrations 表 |
| `services/gateway/internal/persistence/migrate.go` | **新增** | Migration runner（embed.FS + schema_migrations 幂等） |
| `services/gateway/internal/persistence/migrate_test.go` | **新增** | Migration 测试（建表、幂等、索引验证、nil 防御） |
| `services/gateway/internal/persistence/schema_test.go` | **新增** | Schema 结构验证（pragma_table_info 字段检查） |
| `services/gateway/internal/persistence/sqlite/models.go` | **新增** | 6 个实体 Go 结构体（Conversation/Message/Run/RunStep/Artifact） |
| `services/gateway/internal/persistence/sqlite/store.go` | **新增** | SqliteStore 基础实现（Conversation CRUD、Message append/list、Run/RunStep/Artifact 写入） |
| `services/gateway/internal/persistence/sqlite/store_test.go` | **新增** | Store 测试（Conversation+Message 4 消息独立验证、Run+RunStep 3 步骤验证、Artifact metadata placeholder） |
| `services/gateway/go.mod` | 修改 | 新增 `modernc.org/sqlite` 依赖（CGO-free SQLite driver） |
| `services/gateway/go.sum` | **新增** | 依赖锁文件 |
| `go.work.sum` | 修改 | Workspace sum 更新 |
| `docs/refactor/step-3b-3c-persistence-foundation-report.md` | **新增** | 本报告 |
| `docs/refactor/persistence-implementation-plan.md` | 修改 | 补充 Step 3-B/3-C 完成状态 |

## 3. Schema 概览

### 3.1 六张业务表

| 表 | 用途 | 关键字段 |
|----|------|----------|
| `conversations` | 对话会话 | id, title, status, created_at, updated_at |
| `conversation_participants` | 对话参与者 | conversation_id, participant_type, participant_name |
| `runs` | 执行运行 | id, conversation_id, status, planning_mode, error_code |
| `run_steps` | 执行步骤 | run_id, task_id, step_index, agent_name, status |
| `messages` | 聊天消息 | conversation_id, message_id, sender_type, sender_name, content, status |
| `artifacts` | 产物元数据 | artifact_type, preview_type, content_ref (content 不存) |

### 3.2 索引（12 个）

- `conversations(updated_at)`
- `conversation_participants(conversation_id)`
- `runs(conversation_id, started_at)`
- `run_steps(run_id, step_index)` + `run_steps(task_id)`
- `messages(conversation_id, created_at)` + `messages(run_id)` + `messages(step_id)` + `messages(message_id)`
- `artifacts(conversation_id)` + `artifacts(run_id)` + `artifacts(message_id)`

## 4. Migration Runner 设计

- **驱动**: `database/sql` + `embed.FS` 嵌入 `migrations/` 目录
- **幂等**: `schema_migrations` 表记录已应用的 migration 文件名，重复执行自动跳过
- **事务**: 每个 migration 文件在独立事务中执行，失败时回滚
- **排序**: 按文件名升序执行（`001_initial_schema.sql` → ...）
- **内存测试**: 所有测试使用 `:memory:` SQLite 数据库
- **依赖**: `modernc.org/sqlite`（纯 Go，无 CGO，无编译依赖）

## 5. SqliteStore 基础实现

### 5.1 已实现方法

| 方法 | 说明 |
|------|------|
| `NewStore(db *sql.DB) *Store` | 接受已打开的 *sql.DB，不自己打开连接 |
| `CreateConversation(ctx, Conversation)` | 创建对话，自动生成 ID 和时间戳 |
| `GetConversation(ctx, id)` | 获取单个对话 |
| `ListConversations(ctx)` | 列出所有对话（按 updated_at DESC） |
| `AppendMessage(ctx, Message)` | 插入消息（含 sender_type/sender_name/agent_name/status） |
| `ListMessages(ctx, conversationID)` | 列出对话的所有消息（按 created_at ASC） |
| `CreateRun(ctx, Run)` | 创建 Run 记录 |
| `GetRun(ctx, id)` | 获取单个 Run |
| `CreateRunStep(ctx, RunStep)` | 创建 RunStep 记录（含 task_id/agent_name/step_index） |
| `ListRunSteps(ctx, runID)` | 列出 Run 的所有步骤（按 step_index ASC） |
| `CreateArtifactMetadata(ctx, Artifact)` | 创建 Artifact 元数据（不存 content，用 content_ref） |

### 5.2 未实现（留待 Step 3-D）

- Gateway `store.Store` 接口适配器
- SSE 事件到 Message/Run Step/RunStep 的实时写入
- Message 状态更新（streaming → sent → failed）
- Run 完成/失败状态更新
- 错误脱敏写入
- `ListRunsByConversation`、`ListArtifacts` 等查询方法

### 5.3 与 MemoryStore 的关系

本轮未替换 MemoryStore。Gateway runtime 仍使用 `store.MemoryStore`。SqliteStore 在 `internal/persistence/sqlite` 包中独立存在，供 Step 3-D 接入。

## 6. 测试覆盖

| 测试 | 文件 | 验证内容 |
|------|------|----------|
| `TestMigrateCreatesTables` | `migrate_test.go` | 7 张表全部创建（6 业务 + schema_migrations） |
| `TestMigrateIsIdempotent` | `migrate_test.go` | 两次 RunMigrations 不报错，记录不重复 |
| `TestInitialSchemaIndexes` | `migrate_test.go` | 12 个索引全部创建 |
| `TestMigrateNilDB` | `migrate_test.go` | nil DB 返回错误 |
| `TestSchemaConformsToDesign` | `schema_test.go` | conversations/messages/runs/run_steps 字段存在性和 NOT NULL 约束 |
| `TestSqliteStoreConversationAndMessages` | `sqlite/store_test.go` | 4 条独立消息（user/web-agent/code-agent/orchestrator）不合并，sender_name 正确 |
| `TestSqliteStoreRunAndRunStep` | `sqlite/store_test.go` | 1 Run + 3 RunStep（web/code/summary），task_id/agent_name/step_index 正确 |
| `TestArtifactMetadataPlaceholder` | `sqlite/store_test.go` | Artifact metadata 写入，artifact_type/preview_type 正确，content_ref 为空 |
| `TestSqliteStoreConversationCRUD` | `sqlite/store_test.go` | Conversation 创建/获取/列表 CRUD 验证 |

## 7. 未接入 Runtime 的说明

本轮完成的 foundation 独立于 Gateway runtime：

- **SqliteStore 不实现 `store.Store` 接口** — 接口适配留待 Step 3-D
- **handleChat 不写入 SqliteStore** — 运行时仍使用 MemoryStore
- **不对 /api 行为产生任何影响** — 前端收发的消息格式不变
- **MemoryStore 完整保留** — 所有现有测试继续通过
- **无数据库文件生成** — 测试使用 `:memory:` SQLite

## 8. 安全检查

| 检查项 | 结果 |
|--------|------|
| `git diff` 中真实 API key / token | 无 |
| `git diff` 中 sk- 开头真实长 key | 无 |
| `git status` 中 frontend/dist / node_modules / .env / .exe | 无 |
| `.claude/settings.local.json` 修改 | 未修改 |
| `.exe` 文件 | 无新增 |
| DATABASE_URL 写入 | 无 — 测试使用 `:memory:` |

## 9. 禁止项自查

| 禁止项 | 状态 |
|--------|------|
| 不把 Gateway 默认改成使用 SqliteStore | 遵守 — MemoryStore 仍是默认 |
| 不删除 MemoryStore | 遵守 |
| 不修改 handleChat runtime 写入逻辑 | 遵守 — httpapi/server.go 未修改 |
| 不修改 Orchestrator runtime | 遵守 |
| 不修改 Frontend runtime | 遵守 |
| 不修改 smoke-new-arch.sh | 遵守 |
| 不弱化 New Architecture Smoke | 遵守 |
| 不修改 docker-compose.new-arch.yml | 遵守 |
| 不修改 .github/workflows/new-arch-smoke.yml | 遵守 |
| 不修改 legacy server/ 作为主路径 | 遵守 — 所有新代码在 services/gateway/internal/ |
| 不在旧 agents/ 下新增主功能 | 遵守 |
| 不服务化 vision-agent | 遵守 |
| 不引入 LLMPlanner | 遵守 |
| 不接真实 LLM | 遵守 |
| 不引入 MySQL | 遵守 — 使用 SQLite |
| 不读取 .env | 遵守 |
| 不写入真实 DATABASE_URL | 遵守 |
| 不提交 .env / token / DATABASE_URL / 私钥 / dist / node_modules / .exe | 遵守 |
| 不修改 .claude/settings.local.json | 遵守 |
| 不 git clean / git add . / commit / push | 遵守 |

## 10. 下一步建议

Step 3-B/3-C foundation 提交并通过 GitHub Actions New Architecture Smoke 后，再进入 Step 3-D Gateway runtime write path。
