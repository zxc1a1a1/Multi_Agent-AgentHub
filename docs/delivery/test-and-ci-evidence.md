# AgentHub v1.0 测试与 CI 证据

## 1. 本地测试命令

### Go 测试（Gateway + Orchestrator + Agents）

```bash
cd services/gateway
go test ./... -count=1
```

结果：**11 packages all pass**（gateway, auth, cmd/gateway, config, httpapi, persistence, sqlite, orchestratorclient, runservice, sse, store）

```bash
cd services/orchestrator
go test ./... -count=1
```

结果：**all packages pass**（planner, validator, executor, dispatcher, registry）

```bash
cd services/agents/code-agent
go test ./... -count=1
```

结果：**all packages pass**

```bash
cd services/agents/web-agent
go test ./... -count=1
```

结果：**all packages pass**

### 前端测试

```bash
cd frontend
npm test -- --run
```

结果：**47 tests passed (7 files)**
- messageStore.test.ts (17 tests)
- messageReplay.test.ts (5 tests)
- agentStore.test.ts (2 tests)
- client.test.ts (6 tests)
- MessageBubble.test.tsx (9 tests)
- CodePreview.test.tsx (7 tests)
- WebPreview.test.tsx (1 test)

### 前端构建

```bash
npm run build
```

结果：**tsc -b && vite build ✓**（TypeScript 编译无错误）

## 2. GitHub Actions Smoke 历史

| Smoke # | 日期 | 状态 | 说明 |
|---------|------|------|------|
| #14 | 2026-06-04 | ✅ PASS | Step 3-D: PersistenceWriter 写入路径 |
| #15 | 2026-06-04 | ✅ PASS | Step 3-D: 补充修复 |
| #16 | 2026-06-04 | ✅ PASS | Step 3-E/3-F: Replay 回放 + 审计基础 |
| #17 | 2026-06-04 | ❌ FAIL | SQLite `/data` 权限不足，Gateway 启动失败 |
| #18 | 2026-06-05 | ✅ PASS | Step 4 修复：Dockerfile mkdir/chown + Bootstrap MkdirAll |

## 3. #17 失败根因与修复

### 失败现象

```
gateway store mode: sqlite db path: /data/agenthub.db
gateway startup failed: sqlite bootstrap failed: ping sqlite db: unable to open database file (14)
```

### 根因

Gateway 容器内 `app` 用户（非 root）无权限在 `/data/` 创建 SQLite 数据库文件。Docker named volume `gateway-data` 挂载到 `/data` 时，目录 owner 可能为 `root`。

### 修复（commit: `fb6c772`）

1. **Dockerfile**：在 `USER app` 之前添加 `RUN mkdir -p /data && chown -R app:app /data`
2. **persistence_bootstrap.go**：`sql.Open` 前对父目录执行 `os.MkdirAll(parentDir, 0755)`
3. **测试**：新增 4 个 bootstrap 单元测试（空路径、嵌套目录创建、纯文件名、MkdirAll 失败）

### 验证

GitHub Actions New Architecture Smoke #18 绿色通过，Gateway 成功启动并在 `/data/agenthub.db` 创建 SQLite DB。

## 4. Smoke 覆盖内容（smoke-new-arch.sh）

| Level | 测试内容 | 验证目标 |
|-------|----------|----------|
| Level 0 | Docker/Compose Preflight | Docker daemon、compose config 有效性 |
| Level 1 | Service Health | code-agent/web-agent/orchestrator/gateway 健康检查 |
| Level 2 | Single Agent | code-agent + web-agent 单 Agent SSE 流式输出 + sender 归属 |
| Level 3 | Mixed Ordered Parallel | web-agent + code-agent + orchestrator 三消息检测 + 顺序验证 |
| Level 4 | Replay / Persistence | GET /api/conversations/{id}/messages 返回独立 Agent 消息 + senderType/senderName/runId/status 字段 |
| Level 5 | Log Safety | 无 panic / fatal / API key 泄露 |

## 5. CI Workflow 配置

- **Workflow**: `.github/workflows/new-arch-smoke.yml`
- **触发**: push to branch `y` / pull_request / workflow_dispatch
- **并行 Go 测试**: gateway + orchestrator + code-agent + web-agent
- **Compose config check**: 验证 gateway 不包含 `AGENT_CODE_URL`、`AGENT_WEB_URL` 等直连 Agent URL
- **Container smoke**: docker compose up --build + smoke-new-arch.sh
- **Docker daemon**: ubuntu-latest runner

## 6. 本地 Docker 说明

当前开发环境（Windows 10 Home + Git Bash shell）无 Docker Desktop，无法本地运行 `docker compose` 和 `smoke-new-arch.sh`。Runtime smoke 依赖 GitHub Actions 云端 ubuntu-latest runner。本地 Go 测试和前端测试均可在无 Docker 环境下运行。

## 7. 测试覆盖总结

| 维度 | 覆盖 |
|------|------|
| Gateway REST API | Go 集成测试 + smoke Level 1-4 |
| Gateway SSE | smoke Level 2-3 |
| Gateway auth/CORS | Go 测试（token leak test） |
| Orchestrator Planner | Go 单元测试 |
| Orchestrator Validator | Go 单元测试 |
| Orchestrator Executor | Go 单元测试 + smoke Level 2-3 |
| Agent A2A Server | Go 测试 |
| Frontend 组件渲染 | Vitest（7 test files） |
| Frontend 消息状态管理 | Vitest（messageStore + messageReplay） |
| Persistence CRUD | Go 测试（SqliteStore + PersistenceWriter + replay） |
| Migration runner | Go 测试（table creation + idempotency + indexes） |
| Error sanitization | Go 测试 + 前端测试 |
| Log safety | smoke Level 5 |
| Gateway no-direct-agent | CI compose config check |
| SQLite container volume | smoke #18 |

---

- Created: 2026-06-05
- Step: AgentHub v1.0 Step 5 — Test & CI Evidence
