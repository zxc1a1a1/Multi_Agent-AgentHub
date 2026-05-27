# AgentHub 模块拆分迁移计划

## 1. 目标

本次重构目标是降低耦合，把当前项目逐步拆成以下独立模块，并形成稳定迁移路径：

- `pkg/adk`
- `pkg/runtime`
- `services/gateway`
- `services/orchestrator`
- `services/agents/*`
- `frontend`

核心定位如下：

- ADK 是纯引擎层。
- Runtime 是工程化框架层。
- Gateway 是前端唯一 HTTP/AG-UI 入口。
- Orchestrator 是统一编排大脑。
- Child Agent 通过 A2A 协议自治接入。

当前阶段不破坏旧系统，只并行建立新模块。

## 2. 迁移原则

- 先新增新模块，再迁移旧逻辑。
- 每个 Phase 都必须测试通过。
- 每次只迁移一个明确边界。
- 不在同一轮同时移动 Gateway、Orchestrator、Agent、Frontend。
- 不让 Gateway 承担编排职责。
- 不让 Orchestrator 直接处理前端 HTTP。
- 不让 Child Agent 依赖前端协议。
- 不让 `pkg/adk` 依赖 Runtime、业务配置、具体 LLM Provider。
- 不提交 `.env`、`exe`、真实密钥。
- 不使用 `git add .`。

## 3. 分阶段计划

### Phase 0：迁移护栏和新模块骨架

- 目标：建立迁移护栏文档、`go.work` 与 `pkg/adk` 最小骨架。
- 主要产物：`docs/refactor/*`、`go.work`、`pkg/adk/go.mod`、`pkg/adk/adk.go`。
- 不应修改的范围：旧 `server/`、`agents/`、`frontend/` 业务逻辑与运行链路。
- 验收命令：

```powershell
git status --short
git diff --stat
cd pkg/adk
go build ./...
cd ..
cd agents
go test ./...
cd ..
cd server
go test ./...
```

### Phase 1：pkg/adk core interfaces + Runner

- 目标：在 `pkg/adk` 中实现核心接口与 Runner 闭环。
- 主要产物：`Agent`、`Model`、`Tool`、`Plugin`、`SessionService`、`Runner`、核心单测。
- 不应修改的范围：`pkg/runtime`、`services/*`、`frontend` 与旧主线目录结构。
- 验收命令：

```powershell
cd pkg/adk
go test ./...
go test -race ./...
go build ./...
```

### Phase 2：pkg/adk/a2a adapter

- 目标：实现 A2A Server/Client 适配层与 AgentCard 基础能力。
- 主要产物：`pkg/adk/a2a/*`、A2A 契约测试、流式任务基础测试。
- 不应修改的范围：`services/gateway`、`services/orchestrator`、旧 `server/internal/*` 主链路。
- 验收命令：

```powershell
cd pkg/adk
go test ./a2a/...
go test ./...
```

### Phase 3：pkg/runtime config + registry + model

- 目标：建立 Runtime 配置装配、注册表与模型适配能力。
- 主要产物：`pkg/runtime/config`、`pkg/runtime/registry`、`pkg/runtime/model`。
- 不应修改的范围：`services/*` 生产入口与旧主线对外行为。
- 验收命令：

```powershell
cd pkg/runtime
go test ./...
go build ./...
```

### Phase 4：pkg/runtime session + context pruning

- 目标：实现 Session 持久化抽象与上下文裁剪策略。
- 主要产物：`pkg/runtime/session`、`pkg/runtime/context`、相关单测与回归测试。
- 不应修改的范围：前端事件协议、旧 Gateway/Orchestrator 对外接口。
- 验收命令：

```powershell
cd pkg/runtime
go test ./...
```

### Phase 5：pkg/runtime skill + agui + launcher

- 目标：完成 Skill 管理、AG-UI 转换器与 Launcher 组装能力。
- 主要产物：`pkg/runtime/skill`、`pkg/runtime/agui`、`pkg/runtime/launcher`。
- 不应修改的范围：旧 `frontend/src/agui/*` 运行逻辑与旧 `server` 主链路。
- 验收命令：

```powershell
cd pkg/runtime
go test ./...
go build ./...
```

### Phase 6：services/orchestrator

- 目标：落地独立 Orchestrator 服务并承接统一编排。
- 主要产物：`services/orchestrator` 独立入口、计划器、执行器、转换器。
- 不应修改的范围：Gateway 对外公开 API 行为。
- 验收命令：

```powershell
cd services/orchestrator
go test ./...
go build ./cmd/...
```

### Phase 7：services/gateway

- 目标：落地独立 Gateway 服务并统一对外 HTTP/AG-UI 接入。
- 主要产物：`services/gateway` 独立入口、内部客户端、协议映射与持久化入口。
- 不应修改的范围：Orchestrator 编排核心逻辑与 Child Agent 业务逻辑。
- 验收命令：

```powershell
cd services/gateway
go test ./...
go build ./cmd/...
```

### Phase 8：services/agents/code-agent 首个迁移

- 目标：完成第一个 Child Agent 从旧目录到 `services/agents/*` 的迁移闭环。
- 主要产物：`services/agents/code-agent`、A2A 接入、运行验证脚本。
- 不应修改的范围：其余旧 `agents/*` 未迁移 Agent 的运行行为。
- 验收命令：

```powershell
cd services/agents/code-agent
go test ./...
go build ./...
```

### Phase 9：frontend agentName / 多 Agent 事件适配

- 目标：前端适配多 Agent 消息归属与运行状态可视化。
- 主要产物：`frontend` 事件消费适配、UI 展示补强、前端测试补充。
- 不应修改的范围：后端契约外的旧业务逻辑重写。
- 验收命令：

```powershell
cd frontend
npm test -- --run
npm run build
```

### Phase 10：Integration + Docker + smoke test

- 目标：完成全链路集成、Compose 一键演示与验收回归。
- 主要产物：更新后的 `docker-compose.yml`、smoke test、交付文档。
- 不应修改的范围：已经通过验收的上游契约与测试基线。
- 验收命令：

```powershell
docker compose config
docker compose up -d
docker compose ps
```

## 4. 保持旧系统可运行

- 在 `services/*` 完整替换前，旧 `server`、`agents`、`frontend` 仍作为可运行主线保留。
- 新 `pkg/adk` 与 `pkg/runtime` 先独立测试。
- 旧 `agents/adk` 暂时不删除。
- 旧 `server/internal/orchestrator` 暂时不删除。
- 迁移完成前，`docker-compose` 仍以旧主线为准。

## 5. 回滚策略

- Phase 0/1 主要新增文件，回滚只需删除 `pkg/adk`、`docs/refactor` 和 `go.work` 变更。
- 不动旧业务代码，因此回滚风险低。
- 每个 Phase 单独提交，避免大提交难以回滚。
