# AgentHub Current Architecture State

## 1. Project Stage

AgentHub 当前处于 **v1.0 Productization Stage**（产品化与能力池接入阶段），不再沿用 Phase 8 编号。

前置阶段已完成：

1. 模块解耦与 Runtime 基础设施（`pkg/adk`、`pkg/adk/a2a`、`pkg/runtime`）
2. 子 Agent 能力池与多模态契约
3. 新架构 Orchestrator vertical slice（已跑通五服务链路）

当前阶段目标是把已跑通的新架构链路整理成可演示、可持久化、可扩展、可逐步接入更多 Agent 的平台基础。

## 2. Current Service Topology

当前新架构主链路（五服务拓扑）：

```text
Frontend / Client
  → Gateway (services/gateway)
  → Orchestrator (services/orchestrator)
  → code-agent (services/agents/code-agent) / web-agent (services/agents/web-agent)
  → Orchestrator summary
  → Gateway SSE
  → Frontend
```

运行时链路细节：

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

## 3. `services/*` — New Architecture Main Path

`services/` 是当前新架构主运行路径。后续所有新架构功能默认进入 `services/`。

### 3.1 `services/gateway`

Gateway Service 是 AgentHub 对外公开入口。职责包括：

- 对外 REST API（`/api/conversations`、`/api/agui/run` 等）
- 对外 SSE / stream endpoint（AG-UI 事件流）
- 用户鉴权与 CORS 边界
- requestId / traceId 注入
- Conversation / Message / Agent 摘要 / Artifact 元数据查询入口
- 接收前端 run 请求，调用 Orchestrator 内部 API
- 将 Orchestrator 内部事件转换为前端 AG-UI 事件
- 持久化用户消息和结果入口
- 将内部错误转换为前端安全错误

Gateway 禁止：

- 实现意图编排或 Agent 选择
- 直接调用 Child Agent
- 直接调用 LLM Provider
- 执行 fallback / retry 策略
- 合并多 Agent 业务结果

### 3.2 `services/orchestrator`

Orchestrator Service 是 AgentHub 唯一编排层。职责包括：

- 意图理解（RulePlanner，后续 LLMPlanner feature flag）
- OrchestrationPlan 生成
- PlanValidator 校验
- Agent Registry 查询与 Health Check 过滤
- Agent 选择（基于 capabilityIds / inputModes / outputTypes）
- single / ordered_parallel / sequential 执行策略
- A2A Dispatcher 调用 Child Agent
- fallback / retry
- 多 Agent 结果聚合
- 状态更新与 Artifact / ToolCall 引用归一

Orchestrator 禁止：

- 直接暴露给 Frontend
- 处理用户登录态
- 管理浏览器连接
- 定义公开 REST API response envelope

当前 Orchestrator 使用 `RulePlanner`（确定性规则），不使用真实 LLM。后续 LLMPlanner 通过 `ORCHESTRATOR_PLANNER_MODE` feature flag 受控接入。

### 3.3 `services/agents/code-agent`

当前状态：**已服务化，v0.1 mock / deterministic response**。

- 实现了 `/health`、A2A Server、AgentCard
- `Generate()` 返回确定性 mock response，不调用真实 LLM
- 符合 A2A 协议契约
- Dockerfile 已就绪，纳入 `docker-compose.new-arch.yml`
- 有 unit test 和 integration test

### 3.4 `services/agents/web-agent`

当前状态：**已服务化，v0.1 mock / deterministic response**。

- 实现了 `/health`、A2A Server、AgentCard
- `Generate()` 返回确定性 mock HTML response，不调用真实 LLM
- 内置 unsafe HTML token / sensitive token 过滤
- 符合 A2A 协议契约
- Dockerfile 已就绪，纳入 `docker-compose.new-arch.yml`
- 有 unit test 和 integration test

## 4. `agents/` — Legacy Agent Capability Pool

`agents/` 下保留了大量旧 Agent 实现，包括：

```text
vision-agent
file-agent
document-agent
ppt-agent
security-agent
test-agent
review-agent
deploy-agent
web-research-agent
artifact-agent
context-agent
agent-builder-agent
custom-agent
diff-agent
qa-acceptance-agent
release-agent
version-agent
code-agent (旧版)
web-agent (旧版)
```

定位：**旧能力池与未来服务化候选池**。

这些 Agent 不是当前新架构主运行链路的一部分。不能一次性全部搬迁到 `services/agents/`，必须按能力价值和契约成熟度分批服务化。

推荐服务化顺序：vision-agent → file-agent → document-agent → test-agent → security-agent → ppt-agent → review-agent → deploy-agent。

## 5. `server/` — Legacy Path / Historical Compatibility Path

`server/` 代表旧主链路。当前定位：

- MVP v0.1 时期的 Gateway（含内嵌 Orchestrator）
- 历史兼容路径
- 不再作为新功能开发的主入口

后续开发默认不应在 `server/` 增加新主功能。除非明确是在做 legacy 兼容或迁移清理，否则新功能进入 `services/gateway`、`services/orchestrator`、`services/agents/*`。

## 6. Compose Files

### 6.1 `docker-compose.new-arch.yml` — New Architecture Main Compose

当前新架构 demo 主路径 compose。定义五服务拓扑：

- `frontend-new` — React 前端
- `gateway-new` — Gateway Service（依赖 orchestrator-new healthy）
- `orchestrator-new` — Orchestrator Service（依赖 code-agent-new、web-agent-new healthy）
- `code-agent-new` — services/agents/code-agent
- `web-agent-new` — services/agents/web-agent

所有服务有 healthcheck。Gateway 只通过 `ORCHESTRATOR_URL` 连接 Orchestrator，不持有 Agent URL。

配套验证：

- `smoke-new-arch.sh` — 本地 smoke test
- `.github/workflows/new-arch-smoke.yml` — CI smoke

### 6.2 `docker-compose.yml` — Legacy / Historical Compose

旧架构 compose。Gateway 从 `server/` 构建，Code-Agent 从 `agents/` 构建，依赖 MySQL，包含真实 LLM key 环境变量。

定位：**legacy 或历史兼容路径**。后续不应基于此 compose 开发新功能。

## 7. CI / Smoke Status

- CI 默认使用 **RulePlanner + mock/deterministic Agent response**
- CI 不依赖真实 LLM key、真实 OCR、外部文件解析服务
- CI smoke 覆盖：single code、single web、mixed ordered_parallel
- CI 通过 `new-arch-smoke` workflow 运行
- LLMPlanner 等需要外部依赖的功能通过 feature flag 控制

## 8. Development Constraints

### 8.1 Do Not Expand Gateway-Direct-to-Agent Path

禁止扩展以下路径：

```text
Gateway → code-agent (direct)
Gateway → web-agent (direct)
Gateway → any new Agent (direct)
```

Gateway 必须通过 Orchestrator 调用 Agent。

### 8.2 Do Not Add New Main Features in `server/`

默认不在 `server/` 增加新主功能。新功能进入 `services/gateway`、`services/orchestrator`、`services/agents/*`。

### 8.3 Do Not Hardcode Agent Names

不要通过 agentName（如 `"web-agent"`、`"code-agent"`）决定前端组件或编排路由。应使用：

- artifact.type
- previewType
- toolName
- runtime capability registry

### 8.4 Do Not Bypass Validator / Executor / Registry

禁止为了快速 demo 绕过 PlanValidator、Executor、Registry、A2A Dispatcher。

## 9. Runtime Infrastructure (`pkg/`)

`pkg/` 提供跨服务共享的 Runtime 基础设施：

- `pkg/adk` — ADK Runtime 接口（`Agent`、`GenerateRequest`、`GenerateResponse`、`Part` 等）
- `pkg/adk/a2a` — A2A 协议适配（A2A Server 封装）
- `pkg/runtime` — 通用 Runtime 工具

这些包被 `services/agents/code-agent`、`services/agents/web-agent` 和未来新 Agent 共享。

## 10. Frontend

`frontend/` 是 React + TypeScript 前端。当前状态：

- 通过 Vite 代理或 Gateway 公开 API 连接后端
- 消费 AG-UI SSE 事件流
- 前端不得直连 Orchestrator 或 Child Agent

## 11. Related Documents

- `docs/refactor/productization-stage-guide.md` — 产品化阶段总指导
- `docs/refactor/legacy-boundary.md` — Legacy 与新架构边界定义
- `docs/architecture/overview.md` — 架构总览
- `docs/architecture/service-topology.md` — 服务拓扑
- `docs/architecture/process-boundaries.md` — 进程边界硬规则
- `docs/contracts/` — 各层协议契约

## 12. Version

- Last updated: 2026-06-03
- Project stage: v1.0 Productization Stage
- Previous stage: Phase 8 (Orchestrator vertical slice)
