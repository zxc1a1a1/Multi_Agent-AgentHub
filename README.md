# Multi-Agent Framework Starter: React + Go

这是一个团队协作版多 Agent 项目骨架，技术主线是：

- **Go API / Agent Backend**：负责 Agent 编排、AG-UI 事件流、A2A 服务入口、A2UI 消息生成。
- **React Web**：负责聊天交互、实时事件展示、A2UI 声明式 UI 渲染。
- **ADK / A2A / AG-UI / A2UI**：先用清晰的协议适配层占位，后续逐步接入官方 SDK 或生产实现。

> 建议仓库先设为 Private，master/main 保留稳定版本，dev 作为开发集成分支；同时创建成员分支 `c`、`y`、`g`，每个具体任务仍然使用 `feature/<成员>/<任务>` 分支。

## 项目结构

```text
multi-agent-framework-go-react/
├── apps/
│   ├── api/                    # Go 后端：Agent API / AG-UI / A2A / A2UI
│   │   ├── cmd/server/main.go
│   │   ├── internal/
│   │   │   ├── agent/           # Root Agent 和子 Agent 编排
│   │   │   ├── config/          # 配置读取
│   │   │   ├── httpapi/         # HTTP 路由和 handler
│   │   │   └── service/         # 应用服务层
│   │   └── pkg/protocol/        # 协议模型：AG-UI / A2A / A2UI
│   └── web/                    # React 前端
│       ├── src/
│       │   ├── api/             # AG-UI 客户端
│       │   ├── components/      # A2UI renderer
│       │   └── App.tsx
├── docs/
│   ├── architecture.md
│   ├── git-workflow.md
│   └── learning-roadmap.md
├── scripts/
│   ├── init-repo.sh             # 初始化 master/dev/c/y/g
│   ├── create-branch.sh         # 从 dev 或成员分支创建 feature/xxx
│   └── sync-member-branch.sh    # 将 dev 同步到成员分支
├── docker-compose.yml
├── .env.example
├── .gitignore
└── Makefile
```

## 本地启动

### 1. 启动 Go 后端

```bash
cd apps/api
go mod tidy
go run ./cmd/server
```

默认端口：`http://localhost:8080`

测试：

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/.well-known/agent-card.json
```

### 2. 启动 React 前端

```bash
cd apps/web
npm install
npm run dev
```

默认端口：`http://localhost:5173`

## 当前已经内置的接口

| 接口 | 作用 |
|---|---|
| `GET /healthz` | 健康检查 |
| `GET /.well-known/agent-card.json` | A2A Agent Card 占位 |
| `POST /api/v1/agent-runs` | 创建一次 Agent Run |
| `POST /api/v1/agent-runs:stream` | AG-UI 风格 SSE 事件流 |

## 后续接入路线

1. 先跑通 React + Go 的本地闭环。
2. 在 `internal/agent` 中完善 Root Agent、Research Agent、Code Agent 等子 Agent。
3. 在 `pkg/protocol/agui` 中对齐 AG-UI 官方事件类型。
4. 在 `pkg/protocol/a2ui` 中对齐 A2UI v0.8/v0.9 schema。
5. 在 `pkg/protocol/a2a` 或 `internal/httpapi/a2a_handlers.go` 中接入官方 A2A Go SDK。
6. 在 `internal/agent` 或单独 adapter 中接入 Go ADK。

## Git 初始化

如果你已经创建了远程空仓库：

```bash
bash scripts/init-repo.sh git@github.com:your-org/your-repo.git
```

脚本会创建并推送：

```text
master
dev
c
y
g
```

之后团队成员从自己的成员分支拉任务分支：

```bash
# 成员 c
bash scripts/create-branch.sh feature/c/adk-base c

# 成员 y
bash scripts/create-branch.sh feature/y/frontend-layout y

# 成员 g
bash scripts/create-branch.sh feature/g/a2a-server g
```

## 分支规范

```text
master / main         稳定分支，只放可发布版本
dev                   开发集成分支
c                     成员 c 的长期分支
y                     成员 y 的长期分支
g                     成员 g 的长期分支
feature/<成员>/<任务> 新功能开发
bugfix/<成员>/<问题>  修 bug
hotfix/xxx            紧急修线上问题
```

推荐流程：

```text
feature/<成员>/<任务> → c/y/g → dev → master/main
```


## 成员分支同步

当 `dev` 有其他成员合入的新代码后，成员分支要定期同步：

```bash
bash scripts/sync-member-branch.sh c
bash scripts/sync-member-branch.sh y
bash scripts/sync-member-branch.sh g
```

详细流程见 `docs/git-workflow.md`。

## 后端接口与命名规范

本仓库已内置后端规范文档：

- `docs/backend-conventions.md`：Go 命名、RESTful URL、JSON 字段、错误响应、PR 检查项。
- `docs/api-design.md`：当前 API 说明和后续资源设计建议。

当前后端接口：

```text
GET  /healthz
GET  /.well-known/agent-card.json
POST /api/v1/agent-runs
POST /api/v1/agent-runs:stream
```

新增接口时遵循：资源名用复数名词，路径统一 `/api/v1`，不要在 URL 里写 `get/create/update/delete`，非 CRUD 的协议动作使用 `:stream`、`:cancel` 这类 custom method。
