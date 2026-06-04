# AgentHub Legacy Boundary

## 1. Purpose

本文档定义 AgentHub 项目中 **new architecture 主路径** 与 **legacy 路径** 的明确边界。所有后续开发者必须据此判断新功能应该写在哪里、什么情况下可以触碰 legacy、什么情况下不能触碰 legacy。

## 2. What Is the New Architecture Main Path

New architecture 主路径是指基于 `services/` 的五服务独立进程拓扑：

```text
Frontend (frontend/)
  → Gateway Service (services/gateway)
  → Orchestrator Service (services/orchestrator)
  → Child Agent Service (services/agents/<agent-name>)
  → Orchestrator summary aggregation
  → Gateway SSE transformation
  → Frontend SSE consumer
```

核心特征：

- Gateway 与 Orchestrator 是 **两个独立进程**
- Gateway 只通过内部 API 调用 Orchestrator，不直接调 Agent
- Orchestrator 是唯一编排层（Planner → Validator → Executor → Dispatcher → Registry）
- Child Agent 通过 A2A 协议接入，暴露 `/health` 和 AgentCard
- 使用 `docker-compose.new-arch.yml` 作为 compose
- 使用 `smoke-new-arch.sh` 和 `.github/workflows/new-arch-smoke.yml` 作为验证

**新功能必须进入 new architecture 主路径。**

## 3. What Is the Legacy Path

Legacy 路径是指 MVP v0.1 时期的旧架构：

```text
Frontend → Gateway (server/) → Code-Agent (agents/code-agent/) → LLM
```

核心特征：

- Gateway（`server/`）包含内嵌 Orchestrator
- 单 Agent（code-agent），无多 Agent 编排
- Gateway 直接调用 Agent（无中间编排层）
- 依赖真实 LLM key（Anthropic / OpenAI）
- 使用 `docker-compose.yml` 作为 compose
- 依赖 MySQL 持久化

Legacy 路径的定位：**历史兼容与回归测试基线**，不作为新功能开发入口。

## 4. `agents/` vs `services/agents/`

| 维度 | `agents/` | `services/agents/` |
|------|-----------|-------------------|
| 定位 | 旧能力池与未来服务化候选池 | 已完成服务化的 Agent |
| 架构归属 | Legacy 路径 | New Architecture 主路径 |
| 被谁调用 | Legacy Gateway（`server/`） | Orchestrator（`services/orchestrator`） |
| 协议 | 旧 A2A 实现 | A2A 协议契约（`/health`、AgentCard、Streaming Task） |
| ADK | 旧 `agents/adk/` | 共享 `pkg/adk` |
| 是否可直接新增 | 否 | 是（按契约服务化） |
| Dockerfile | 指向 `agents/` 目录 | 指向 `services/agents/<name>/Dockerfile` |

**关键规则：旧 `agents/` 不能一次性全部搬迁到 `services/agents/`，必须按能力价值和契约成熟度分批服务化。**

当前已服务化（已在 `services/agents/`）：

- `code-agent` — v0.1 mock / deterministic response
- `web-agent` — v0.1 mock / deterministic response

仍在 `agents/` 的旧实现（等待服务化）：

- `vision-agent`、`file-agent`、`document-agent`、`ppt-agent`
- `security-agent`、`test-agent`、`review-agent`、`deploy-agent`
- `web-research-agent`、`artifact-agent`、`context-agent`
- `agent-builder-agent`、`custom-agent`、`diff-agent`
- `qa-acceptance-agent`、`release-agent`、`version-agent`
- `code-agent`（旧版）、`web-agent`（旧版）

## 5. `server/` vs `services/gateway` / `services/orchestrator`

| 维度 | `server/` | `services/gateway` + `services/orchestrator` |
|------|-----------|----------------------------------------------|
| 定位 | Legacy 主链路或历史兼容路径 | New Architecture 主路径 |
| Gateway | 内嵌 Orchestrator 逻辑 | 独立 Gateway Service，不含编排 |
| Orchestrator | 无独立进程 | 独立 Orchestrator Service 进程 |
| Agent 调用 | Gateway 直连 Agent | Gateway → Orchestrator → Agent |
| 多 Agent | 不支持 | 支持（single / ordered_parallel / sequential） |
| 编排 | Converter 内 A2A ↔ AG-UI 协议转换 | Planner → Validator → Executor → Dispatcher |
| 扩展方式 | 在 `server/` 内修改 | 在 `services/gateway` 或 `services/orchestrator` 内修改 |
| 新功能 | 禁止新增主功能 | 新功能默认入口 |

**关键规则：后续开发默认不应在 `server/` 增加新主功能。**

## 6. `docker-compose.yml` vs `docker-compose.new-arch.yml`

| 维度 | `docker-compose.yml` | `docker-compose.new-arch.yml` |
|------|---------------------|-------------------------------|
| 定位 | Legacy 或历史 compose | New Architecture 主 compose |
| Gateway | 从 `server/` 构建 | 从 `services/gateway/Dockerfile` 构建 |
| Agent | 从 `agents/code-agent/` 构建 | 从 `services/agents/code-agent/` 和 `services/agents/web-agent/` 构建 |
| Orchestrator | 无独立服务 | 从 `services/orchestrator/Dockerfile` 构建 |
| Agent 数量 | 1（code-agent） | 2+（code-agent + web-agent，可扩展） |
| LLM 依赖 | 需要真实 LLM key | 使用 deterministic mock，不依赖真实 LLM |
| 数据库 | MySQL | 无（后续持久化 Milestone 引入） |
| 新功能 | 禁止基于此扩展 | 新功能默认基于此扩展 |

## 7. Where New Features Should Be Written

| 功能类型 | 进入位置 |
|---------|---------|
| Gateway 对外 API | `services/gateway/httpapi/` |
| Gateway SSE 转换 | `services/gateway/sse/` |
| Gateway 数据持久化 | `services/gateway/store/` |
| Gateway → Orchestrator 通信 | `services/gateway/orchestratorclient/` |
| Orchestrator 编排逻辑 | `services/orchestrator/planner/`、`services/orchestrator/executor/`、`services/orchestrator/dispatcher/` |
| Orchestrator 内部 API | `services/orchestrator/httpapi/` |
| Agent Registry | `services/orchestrator/registry/` |
| 新 Agent 服务化 | `services/agents/<agent-name>/` |
| 共享 Runtime 基础设施 | `pkg/adk/`、`pkg/runtime/`（谨慎扩展） |
| 前端组件与 SSE 消费 | `frontend/` |
| 架构文档 | `docs/architecture/` |
| 协议契约 | `docs/contracts/` |
| 重构/迁移文档 | `docs/refactor/` |

## 8. When Legacy Can Be Modified

以下情况允许修改 legacy（`server/`、`agents/`、`docker-compose.yml`）：

1. **安全修复**：legacy 路径存在安全漏洞（如 secret 泄露、注入风险）。
2. **针对性的迁移清理**：将某个旧 Agent 服务化后，清理 `agents/<name>/` 中对应的旧代码。
3. **旧 CI 修复**：修复立即影响开发或持续构建过程的问题。
4. **明确的遗留兼容任务**：用户明确要求修改 legacy 路径以保持历史兼容性。

## 9. When Legacy Cannot Be Modified

以下情况禁止修改 legacy：

1. **新增主功能**：新功能无论如何不得在 `server/` 或旧 `agents/` 中新增。
2. **扩展 Gateway 直连 Agent**：禁止在 legacy gateway 中增加新的 Agent 直连路径。
3. **在旧 Agent 中增加新能力**：如果某个能力需要新增，应该通过在 `services/agents/<name>/` 中服务化该 Agent 来实现，而非在 `agents/<name>/` 中打补丁。
4. **基于 `docker-compose.yml` 构建新 demo**：禁止基于旧 compose 扩展新服务。
5. **引入新的外部依赖到 legacy**：禁止在 legacy 路径中引入新的 LLM Provider、新的数据库、新的消息队列等。
6. **将 legacy 作为新功能参考实现**：禁止以 `server/` 或旧 `agents/` 的实现为蓝本编写新功能。

## 10. Agent Service Migration Rules

旧 `agents/` 中的 Agent 不能一次性全部搬迁。必须遵守：

1. **按优先级**：vision-agent → file-agent → document-agent → test-agent → security-agent → ppt-agent → review-agent → deploy-agent
2. **按契约**：每个新 Agent 服务化必须同时满足以下契约：
   - `a2a-agent-contract` — A2A 协议、AgentCard、Health Check
   - `adk-runtime-contract` — Agent 内部 ADK Runtime 实现
   - `testing-review-contract` — 测试与质量门禁
   - `security-boundary-contract` — 安全边界
3. **按产物**：如果 Agent 输出 Artifact，还必须满足：
   - `artifact-contract` — Artifact schema 与存储
   - `frontend-runtime-skills-contract` — 前端 Runtime Capability 渲染
4. **目录规范**：新 Agent 服务化必须进入 `services/agents/<agent-name>/`，目录结构符合 `adk-runtime-contract` 规定。
5. **Mock first**：新 Agent 优先以 deterministic mock 接入，后续通过 feature flag 切换真实 LLM。

## 11. Hard Rules

以下规则不可违反：

1. **Gateway 不能重新成为 Agent 编排层** — 即使为了快速 demo，也不能让 Gateway 直接选择或调用 Agent。
2. **Frontend 不能直连 Orchestrator 或 Agent** — Frontend 只能通过 Gateway 公开 API 通信。
3. **Gateway 与 Orchestrator 必须是两个独立进程** — 不得合并为同一进程。
4. **Orchestrator 不能直接暴露给浏览器** — Orchestrator 只有内部 API。
5. **不得绕过 Validator** — 所有 Plan 必须经过 PlanValidator 校验。
6. **不得通过 agentName 路由** — 使用 capabilityIds / inputModes / outputTypes 进行能力匹配。
7. **CI 不得依赖真实 LLM key** — CI 必须保持 deterministic。

## 12. Related Documents

- `docs/refactor/current-architecture-state.md` — 当前架构状态
- `docs/refactor/productization-stage-guide.md` — 产品化阶段总指导
- `docs/architecture/process-boundaries.md` — 进程边界硬规则
- `docs/architecture/service-topology.md` — 服务拓扑
- `docs/contracts/a2a-agent-card.md` — A2A Agent Card 契约
- `docs/contracts/adk-runtime.md` — ADK Runtime 契约

## 13. Version

- Last updated: 2026-06-03
- Applies to: AgentHub v1.0 Productization Stage
