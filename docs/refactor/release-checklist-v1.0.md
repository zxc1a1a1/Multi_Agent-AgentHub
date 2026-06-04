# Release Checklist v1.0

## v1.0 已完成能力

### 核心架构

- [x] Gateway Service 独立进程（对外 HTTP API + SSE stream）
- [x] Orchestrator Service 独立进程（RulePlanner + PlanValidator + Executor）
- [x] code-agent 服务化（health check + A2A Server + AgentCard + Dockerfile）
- [x] web-agent 服务化（health check + A2A Server + AgentCard + Dockerfile）
- [x] Frontend → Gateway → Orchestrator → Agent 完整链路

### 编排能力

- [x] RulePlanner 确定性计划生成（web/code 关键词检测）
- [x] PlanValidator 计划校验
- [x] SingleExecutor 单 Agent 执行
- [x] OrderedParallelExecutor 多 Agent 有序并行
- [x] Orchestrator summary 聚合

### 前端能力

- [x] IM 聊天 UI（对话列表 + 消息区）
- [x] Multi-Agent SSE 消费（独立气泡，不合并）
- [x] senderName / senderDisplayName 展示
- [x] code preview 渲染
- [x] web preview 渲染
- [x] Page refresh 消息恢复（replay）
- [x] Error 安全展示（脱敏）

### 持久化能力

- [x] SQLite schema（6 表 + 12 索引）
- [x] Migration runner（idempotent）
- [x] SqliteStore CRUD（Conversation/Run/RunStep/Message/Artifact）
- [x] PersistenceWriter（SSE event → SQLite 实时写入）
- [x] Message replay（GET /api/conversations/{id}/messages）
- [x] Failure audit 查询（ListFailedRuns/ListFailedMessages）
- [x] 环境变量切换 memory/sqlite 存储模式

### 测试覆盖

- [x] Gateway Go tests（httpapi + persistence + sqlite + auth + sse + middleware）
- [x] Orchestrator Go tests（planner + validator + executor + dispatcher + registry）
- [x] Agent Go tests（code-agent + web-agent）
- [x] Frontend Vitest tests（messageStore + messageReplay）
- [x] New Architecture Smoke（GitHub Actions CI）

### DevOps

- [x] docker-compose.new-arch.yml（5 服务 + SQLite volume）
- [x] smoke-new-arch.sh（health + single + mixed + replay + log safety）
- [x] GitHub Actions CI（Go tests + compose config check + smoke）
- [x] Gateway 无直连 Agent URL 防护（CI 断言）
- [x] Dockerfile for gateway + orchestrator + agents + frontend

### 文档

- [x] README.md（新架构说明 + 启动方式 + 测试命令）
- [x] productization-stage-guide.md
- [x] current-architecture-state.md
- [x] legacy-boundary.md
- [x] persistence-implementation-plan.md
- [x] productization-final-polish-report.md
- [x] demo-script-v1.0.md
- [x] release-checklist-v1.0.md（本文档）
- [x] legacy-cleanup-final-review.md

## v1.1 延后能力

| 能力 | 原因 |
|------|------|
| LLMPlanner（真实 LLM 计划生成） | 需要 API key + 成本管理 + 安全审查 |
| vision-agent 服务化 | 多模态支持需要额外设计 |
| 自动 retry 机制 | 需要 retry policy + backoff + 监控 |
| Run History UI | 前端交互设计未定 |
| 对象存储（S3/MinIO） | v1.0 产物为 mock 小数据 |
| 多用户认证系统 | 需要 auth service + JWT + RBAC |
| 监控/告警 | 需要 observability stack |
| 数据库备份/恢复 | 运维层面需求 |
| AgentCard/Registry 契约化 | 需要跨团队对齐 |
| 群聊 @mention 路由 | 需要 UX 设计 + 协议扩展 |

## 交付物检查

- [x] `docker-compose.new-arch.yml` — 一键启动五服务拓扑
- [x] `smoke-new-arch.sh` — 自动化验收脚本
- [x] `.github/workflows/new-arch-smoke.yml` — CI 自动化
- [x] `README.md` — 项目说明与启动指南
- [x] `docs/refactor/` — 完整架构与实现文档
- [x] Go 测试覆盖（gateway + orchestrator + agents）
- [x] Frontend 测试覆盖（messageStore + messageReplay）

## 安全检查

- [x] 无真实 API key 写入仓库
- [x] 无 DATABASE_URL 写入代码
- [x] 无 .env 提交
- [x] 无 frontend/dist 提交
- [x] 无 node_modules 提交
- [x] 无 .exe 提交
- [x] CI 使用 mock/deterministic 响应
- [x] Error 消息已脱敏（TextStreamFilter + sanitizeErrorText）
- [x] Gateway 不直连 Agent（CI 断言防护）

## 发布前确认

- [ ] GitHub Actions New Architecture Smoke #17 通过
- [ ] 本地 `go test ./services/... -count=1` 通过
- [ ] 本地 `npm test -- --run` 通过
- [ ] 本地 `npm run build` 通过
- [ ] docker compose up 可正常启动
- [ ] 演示场景全部可执行

---

- Created: 2026-06-04
- Step: AgentHub v1.0 Release Checklist
