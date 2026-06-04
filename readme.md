# AgentHub - Multi-Agent Collaboration Platform

AgentHub 是一个基于 IM 交互范式的多 Agent 协作平台。用户通过对话与不同 AI Agent 交互，由 Orchestrator 理解意图、生成计划、调度 Agent 并聚合结果。

## 当前阶段

v1.0 Productization Stage（产品化与能力池接入阶段）

- 后端 Orchestrator vertical slice 已完成（五服务链路已跑通）
- 不再沿用 Phase 8 编号继续开发
- 后续 Milestone 参见 [productization-stage-guide.md](docs/refactor/productization-stage-guide.md)

---

## 当前新架构（主路径）

### 服务拓扑

```text
Frontend / Client
  → Gateway (services/gateway)
  → Orchestrator (services/orchestrator)
  → code-agent (services/agents/code-agent) / web-agent (services/agents/web-agent)
  → Orchestrator summary
  → Gateway SSE
  → Frontend
```

### 运行时链路

```text
Frontend / Client
  → Gateway Public REST (POST /api/agui/run)
  → Gateway → Orchestrator Internal API (stream)
  → Orchestrator RulePlanner → OrchestrationPlan
  → PlanValidator → SingleExecutor / OrderedParallelExecutor
  → A2A Dispatcher → code-agent / web-agent
  → Orchestrator summary aggregation
  → Gateway SSE transformation
  → Frontend SSE consumer
```

### 核心约束

- **Gateway 与 Orchestrator 是两个独立进程**，不得合并
- Frontend 只能通过 Gateway 公开 API 通信，不得直连 Orchestrator 或 Agent
- Gateway 不负责 Agent 编排，不直接调用 Child Agent，不直接调用 LLM Provider
- Orchestrator 是唯一编排层（Planner → Validator → Executor → Dispatcher → Registry）

---

## 新架构主路径

### `services/gateway`

对外公开入口。职责：REST API、SSE stream、用户鉴权、CORS、Conversation/Message 查询入口、调用 Orchestrator 内部 API、将内部事件转为前端 AG-UI 事件。

### `services/orchestrator`

唯一编排层。职责：RulePlanner、OrchestrationPlan 生成、PlanValidator 校验、Agent Registry 查询与 Health Check 过滤、single / ordered_parallel / sequential 执行策略、A2A Dispatcher、fallback / retry、多 Agent 结果聚合。

当前使用 **RulePlanner**（确定性规则），不依赖真实 LLM。

### `services/agents/code-agent`

当前已服务化的代码类 demo Agent。v0.1 mock / deterministic response，实现 `/health`、A2A Server、AgentCard。Dockerfile 已就绪。

### `services/agents/web-agent`

当前已服务化的 Web/UI 类 demo Agent。v0.1 mock / deterministic response，实现 `/health`、A2A Server、AgentCard。Dockerfile 已就绪。

### `docker-compose.new-arch.yml`

新架构主 compose，定义五服务拓扑（frontend-new → gateway-new → orchestrator-new → code-agent-new / web-agent-new）。

### `smoke-new-arch.sh`

新架构 smoke test，覆盖 health check、single code、single web、mixed ordered_parallel、log safety。

---

## 当前 Agent 状态

| Agent | 状态 | 响应方式 |
|-------|------|----------|
| code-agent (`services/agents/code-agent`) | 已服务化 | v0.1 deterministic mock |
| web-agent (`services/agents/web-agent`) | 已服务化 | v0.1 deterministic mock |

- `agents/` 下保留了大量旧 Agent 实现（vision-agent、file-agent、document-agent 等），这些是**旧能力池与未来服务化候选池**，尚未全部服务化
- 当前 CI / smoke 不依赖真实 LLM key
- Agent 服务化按能力价值和契约成熟度分批进行，推荐顺序见 [legacy-boundary.md](docs/refactor/legacy-boundary.md)

---

## 旧路径边界

| 路径 | 定位 |
|------|------|
| `server/` | Legacy 路径（MVP v0.1 Gateway + 内嵌 Orchestrator），不再作为新功能主入口 |
| `agents/` | 旧 Agent 能力池与未来服务化候选池，一次性全部搬迁 |
| `docker-compose.yml` | Legacy compose（单 Agent + MySQL + 真实 LLM），不基于此扩展新功能 |
| Gateway 直连 Agent | 旧路径，**禁止扩展**。新功能必须走 Gateway → Orchestrator → Agent |

新功能默认进入 `services/gateway`、`services/orchestrator`、`services/agents/*`。

---

## 快速启动（新架构）

```bash
# 一键启动新架构五服务
docker compose -f docker-compose.new-arch.yml up --build

# 运行验收 smoke test
./smoke-new-arch.sh

# 运行新架构 doctor 检查
./doctor-new-arch.sh
```

启动后访问：
- **前端界面**：http://localhost:3000
- **Gateway API**：http://localhost:8080
- **Orchestrator**：http://localhost:8090
- **Code-Agent**：http://localhost:8081
- **Web-Agent**：http://localhost:8082

---

## 测试命令

```bash
# 新架构核心服务测试（deterministic，不依赖外部 LLM）
go test ./services/orchestrator/... ./services/gateway/... ./services/agents/code-agent/... ./services/agents/web-agent/... -count=1
```

---

## 后续开发入口

当前处于 AgentHub v1.0 Productization Stage。后续 Milestone 主线：

1. CI / Smoke 正式化
2. 前端 Multi-Agent SSE 展示
3. Conversation / Run / Message 持久化
4. Artifact 与 Runtime Preview 落地
5. vision-agent 服务化与多模态入口
6. AgentCard / Registry 契约化
7. LLMPlanner 受控接入（feature flag）

详见：
- [productization-stage-guide.md](docs/refactor/productization-stage-guide.md) — 产品化阶段总指导
- [current-architecture-state.md](docs/refactor/current-architecture-state.md) — 当前架构状态
- [legacy-boundary.md](docs/refactor/legacy-boundary.md) — Legacy 与新架构边界定义

---

## Legacy / Historical Notes

以下内容来自 MVP v0.1 时期，仅作为历史参考。当前主路径见上方新架构说明。

<details>
<summary>MVP v0.1 历史架构（点击展开）</summary>

### 历史架构概览

```
Frontend (React + AG-UI SSE)
       │
       ▼
Gateway (Go/Gin, port 8080)
  └─ Orchestrator (内嵌, A2A↔AG-UI 协议转换)
       │
       ▼  A2A Protocol (a2a-go/v2)
Code-Agent (Go, port 8081)
  └─ Anthropic Claude API (流式)
       │
       ▼
MySQL 8 (持久化)
```

### 历史环境依赖

| 依赖 | 版本要求 | 说明 |
|------|----------|------|
| Go | >= 1.26 | 后端 + Agent 编译 |
| Node.js | >= 18 | 前端开发 |
| npm | >= 9 | 前端包管理 |
| Docker & Docker Compose | latest | 一键部署 |
| MySQL | 8.0 | 数据库 |

### 历史项目结构

```
├── frontend/                  # React 前端
│   ├── src/
│   │   ├── components/        # UI 组件
│   │   ├── agui/              # AG-UI SSE 客户端
│   │   ├── stores/            # Zustand 状态管理
│   │   └── services/          # REST API 调用
│   └── package.json
│
├── server/                    # Legacy Go Gateway + 内嵌 Orchestrator
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── handler/           # HTTP handlers
│   │   ├── orchestrator/      # 编排 + A2A↔AG-UI 协议转换
│   │   ├── a2a/               # A2A 客户端
│   │   ├── store/             # MySQL 数据操作
│   │   └── model/             # 数据模型
│   └── go.mod
│
├── agents/                    # 旧 Agent 能力池
│   ├── adk/                   # 旧 ADK Runtime
│   │   ├── context.go
│   │   ├── server.go
│   │   └── llm.go
│   ├── code-agent/            # 旧版 code-agent
│   │   ├── main.go
│   │   ├── handler.go
│   │   └── config.yaml
│   └── go.mod
│
├── services/                  # 新架构主路径（当前）
│   ├── gateway/
│   ├── orchestrator/
│   └── agents/
│       ├── code-agent/
│       └── web-agent/
│
├── docs/                      # 架构文档 + 契约
│   ├── contracts/
│   ├── architecture/
│   └── refactor/
│
├── docker-compose.yml         # Legacy compose
├── docker-compose.new-arch.yml # 新架构 compose
└── smoke-new-arch.sh           # 新架构 smoke
```

### 历史快速开始（Legacy Compose）

```bash
cp .env.example .env
# 编辑 .env，填入 ANTHROPIC_API_KEY
make docker-up
```

### 历史开发命令

```bash
make install       # 安装所有依赖
make dev           # 启动所有服务（并行）
make dev-frontend  # 仅启动前端
make dev-server    # 仅启动 Gateway
make dev-agent     # 仅启动 Code-Agent
make docker-up     # Docker 一键启动
make docker-down   # 停止并清理 Docker
make build-check   # 验证所有模块编译通过
```

### 历史环境变量说明

**基础配置：**

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `DATABASE_URL` | `root:agenthub123@tcp(localhost:3306)/agenthub?...` | MySQL 连接字符串 |
| `GATEWAY_PORT` | `8080` | Gateway 服务端口 |
| `CODE_AGENT_PORT` | `8081` | Code-Agent 服务端口 |
| `AGENT_CODE_URL` | `http://localhost:8081` | Gateway 连接 Agent 的地址 |

**LLM Provider 配置：**

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `LLM_PROVIDER` | `anthropic` | 选择 Provider：`anthropic` 或 `openai` |

Anthropic 配置：`ANTHROPIC_API_KEY`、`ANTHROPIC_MODEL`、`ANTHROPIC_BASE_URL`

OpenAI 配置：`OPENAI_API_KEY`、`OPENAI_MODEL`、`OPENAI_BASE_URL`

### 历史技术栈

| 层 | 技术 |
|----|------|
| 前端 | React 18 + TypeScript + Tailwind CSS + Vite |
| 状态管理 | Zustand |
| 代码高亮 | highlight.js |
| Gateway | Go + Gin |
| 协议库 | a2a-go/v2 (A2A 官方协议) |
| LLM | Anthropic Claude API |
| 数据库 | MySQL 8.0 |
| 部署 | Docker Compose |

### 历史 API 测试用例（curl）

**AgentCard — Agent 自描述发现：**

```bash
curl -s http://localhost:8081/.well-known/agent.json | jq .
```

**A2A 协议 — Gateway → Code-Agent：**

```bash
curl -s -N -X POST http://localhost:8081/ \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tasks/sendSubscribe",
    "id": "test-001",
    "params": {
      "message": {
        "role": "user",
        "parts": [
          {"type": "text", "text": "用 Go 写一个 hello world"}
        ]
      }
    }
  }'
```

**AG-UI 协议 — 前端流式展示：**

```bash
curl -s -N -X POST http://localhost:8080/api/agui/run \
  -H "Content-Type: application/json" \
  -d '{
    "threadId": "test-conv-001",
    "runId": "test-run-001",
    "messages": [
      {"role": "user", "content": "用 Go 写一个 hello world"}
    ],
    "tools": [
      {"name": "code_preview"}
    ]
  }'
```

### 历史常见问题

**Q: Anthropic API Key 从哪里获取？**
A: 访问 https://console.anthropic.com/ 创建 API Key。

**Q: 可以使用其他模型吗？**
A: 可以。通过 `LLM_PROVIDER` 切换 Anthropic / OpenAI 兼容服务（DeepSeek、Ollama 等）。

**Q: 本地开发时前端如何连接 Gateway？**
A: Vite 配置了 `/api` 代理到 `http://localhost:8080`，无需额外配置 CORS。

**Q: Docker 启动后 MySQL 初始化失败？**
A: 执行 `make docker-down` 清理 volume 后重新 `make docker-up`。

</details>
