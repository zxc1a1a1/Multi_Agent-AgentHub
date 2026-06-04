# AgentHub v1.0 提交包索引

## 推荐评委阅读顺序

1. **[final-architecture-overview.md](./final-architecture-overview.md)** — 架构总览（含 Mermaid 图，5 分钟可懂）
2. **[final-feature-checklist.md](./final-feature-checklist.md)** — 功能完成清单（一目了然）
3. **[demo-walkthrough.md](./demo-walkthrough.md)** — 演示脚本与讲解词（6 个场景）
4. **[test-and-ci-evidence.md](./test-and-ci-evidence.md)** — 测试覆盖与 CI 证据
5. **[security-and-risk-review.md](./security-and-risk-review.md)** — 安全审计
6. **[v1.1-roadmap.md](./v1.1-roadmap.md)** — 后续规划

## 核心代码路径

```
services/
├── gateway/                              # Gateway Service（Go）
│   ├── cmd/gateway/main.go               # 启动入口（含 SQLite bootstrap）
│   ├── gateway.go                        # Gateway 装配（auth + CORS + httpapi）
│   ├── persistence_bootstrap.go          # SQLite 启动引导
│   ├── httpapi/
│   │   ├── server.go                     # HTTP API + SSE handler
│   │   ├── persistence_writer.go         # SSE event → SQLite 写入
│   │   ├── replay.go                     # 消息 replay 格式与转换
│   │   └── *_test.go                     # 测试文件
│   ├── internal/persistence/
│   │   ├── migrations/001_initial_schema.sql  # DDL（6 表 + 12 索引）
│   │   ├── migrate.go                    # Migration runner
│   │   └── sqlite/store.go               # SqliteStore（15 methods）
│   ├── store/                            # MemoryStore
│   ├── orchestratorclient/               # Gateway → Orchestrator client
│   └── Dockerfile                        # 容器构建文件
│
├── orchestrator/                         # Orchestrator Service（Go）
│   ├── planner/rule_planner.go           # RulePlanner（确定性规则）
│   ├── validator/plan_validator.go       # PlanValidator
│   ├── executor/                         # SingleExecutor + OrderedParallelExecutor
│   └── dispatcher/                       # A2A Dispatcher
│
└── agents/
    ├── code-agent/                       # code-agent 服务（Go）
    └── web-agent/                        # web-agent 服务（Go）

frontend/                                 # 前端（React + TypeScript）
├── src/
│   ├── components/                       # UI 组件（MessageBubble、CodePreview、WebPreview）
│   ├── stores/                           # Zustand 状态管理（messageStore、agentStore）
│   ├── agui/                             # AG-UI SSE 客户端
│   └── types/                            # TypeScript 类型定义
└── Dockerfile

docker-compose.new-arch.yml               # v1.0 主 compose（5 服务 + SQLite volume）
smoke-new-arch.sh                         # 自动化验收脚本（5 Levels）
.github/workflows/new-arch-smoke.yml      # CI workflow
```

## 核心文档路径

```
docs/
├── delivery/                             # 比赛交付包（本目录）
│   ├── README.md
│   ├── final-architecture-overview.md
│   ├── final-feature-checklist.md
│   ├── test-and-ci-evidence.md
│   ├── demo-walkthrough.md
│   ├── security-and-risk-review.md
│   ├── v1.1-roadmap.md
│   └── submission-package-index.md
│
├── refactor/                             # 实施过程文档
│   ├── current-architecture-state.md     # 当前架构状态
│   ├── legacy-boundary.md               # Legacy 与新架构边界定义
│   ├── productization-stage-guide.md    # 产品化阶段总指导
│   ├── persistence-implementation-plan.md # 持久化实施计划
│   ├── release-checklist-v1.0.md        # v1.0 发布检查清单
│   ├── demo-script-v1.0.md             # v1.0 演示脚本
│   ├── legacy-cleanup-final-review.md   # Legacy cleanup 审计
│   └── productization-final-polish-report.md # Step 4 收口报告
│
└── contracts/                            # 协议契约
```

## 演示入口

| 入口 | 命令/URL |
|------|----------|
| 一键启动 | `docker compose -f docker-compose.new-arch.yml up --build` |
| 前端 UI | http://localhost:3000 |
| Gateway health | http://localhost:8080/health |
| 自动化验收 | `bash ./smoke-new-arch.sh` |

## 测试入口

```bash
# Go 测试（所有服务）
go test ./services/gateway/... ./services/orchestrator/... ./services/agents/... -count=1

# 前端测试
cd frontend && npm test -- --run

# 前端构建
cd frontend && npm run build
```

## GitHub Actions 验收入口

- **Workflow**: `.github/workflows/new-arch-smoke.yml`
- **最新通过**: New Architecture Smoke #18（2026-06-05）
- **覆盖**: Go tests + compose config check + container smoke（5 Levels）+ log safety

## Legacy 路径说明

| 路径 | 定位 | v1.0 状态 |
|------|------|-----------|
| `server/` | Legacy Gateway（含内嵌 Orchestrator） | 保留，不扩展新功能 |
| `agents/` | 旧 Agent 能力池（~20 Agent） | 保留，分批服务化 |
| `docker-compose.yml` | Legacy compose（单 Agent + MySQL + LLM） | 保留，不基于此扩展 |

## v1.0 完成范围

✅ 五服务独立进程架构  
✅ Gateway → Orchestrator → Agent 完整链路  
✅ Multi-Agent ordered_parallel 编排  
✅ RulePlanner + PlanValidator 确定性编排  
✅ AG-UI / SSE 多 Agent 消息归属  
✅ SQLite persistence + migration + replay  
✅ Frontend Multi-Agent 独立气泡 + refresh recovery  
✅ Docker Compose 一键启动 + smoke 自动化验收  
✅ GitHub Actions CI 全链路验证  
✅ Error sanitization + 安全边界  
✅ 完整架构文档与演示材料  

## v1.1 延后范围

⏳ LLMPlanner（真实 LLM 计划生成）  
⏳ vision-agent 服务化  
⏳ Agent Registry / AgentCard 完整化  
⏳ Artifact object storage  
⏳ Run History UI  
⏳ Multi-user auth  
⏳ Automatic retry  
⏳ Observability dashboard  

---

- Created: 2026-06-05
- Step: AgentHub v1.0 Step 5 — Submission Package Index
