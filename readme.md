# AgentHub 2.0

AgentHub 是面向多会话 IM 交互的 Agent 协作平台。产品级事实源是 [AgentHub 2.0 PDR](docs/pdr/AgentHub_2.0_PDR.md)，Active Contract 索引见 [docs/contracts/README.md](docs/contracts/README.md)。当历史资料与这两处冲突时，以 PDR 和 Active Contract 为准。

## 项目状态

**2.0 Contract 已冻结，业务实现迁移中。**

`services/` 下的 Gateway、Orchestrator、CodeAgent 与 WebAgent 是当前可运行链路的一部分；它不等同于已完整实现 AgentHub 2.0 Target。下列能力须在文档中按 Target、Planned 或 In Progress 理解，除非实现与测试另有明确证据：动态 Agent 注册、LLM 规划与校验、PlanVersion 用户确认、统一 SQLite/WAL 持久化，以及完整的 Artifact/Preview 生命周期。

## 2.0 Target 架构

```text
Frontend -> Gateway -> Orchestrator -> Registered A2A Agents
                         |-> LLM Planner -> Plan Validator -> User Confirmation Gate
                         |-> Context Manager / Agent Registry / Result Synthesizer
```

- CodeAgent 和 WebAgent 是内置稳定参考 Agent，不是平台全部 Agent。
- 平台目标支持运行时动态注册符合 A2A 约束、通过校验且已授权的 Agent。
- Planner、Context Manager、Synthesizer、摘要器、标题生成器和 Preview Renderer 是平台内部模块，不是 Agent。
- Direct Mode 直接调用一个符合条件的 Agent，不创建多 Agent Plan。
- Manual Multi-Agent Mode 只可使用用户选择的 Agent；Auto Mode 从 Registry 可用目录中选择 Agent。
- Manual Multi-Agent 与 Auto 都必须由 LLM 生成 Plan、由本地 Validator 校验，并由用户确认准确 PlanVersion 后才能执行。
- 2.0 默认单机持久化目标为 SQLite/WAL；MySQL 与 gRPC 都不是隐含强制目标。
- 2.0 默认 Compose 目标为 Frontend、Gateway、Orchestrator、CodeAgent、WebAgent；远程 Agent 在运行时注册，不要求加入 Compose。

## Current Implementation 与 Target

当前仓库的 `docker-compose.new-arch.yml` 仍配置了十个 specialist Agent，且当前 Orchestrator 仍包含规则规划路径。这是迁移中的 Current Implementation，不表示“固定十个 Agent”是 2.0 目标架构。请勿据此新增固定 Agent 依赖或把内部编排模块注册为 Agent。

## 可用启动与检查命令

以下命令来自当前仓库的 `Makefile` 或已存在脚本；运行前请自行满足本地依赖，本文档不会安装依赖。

```bash
# 当前 new-arch Compose（实际文件当前仍为十 Agent 迁移态）
make docker-new-arch-up
make docker-new-arch-down
make docker-new-arch-logs

# 本地分别启动已有服务入口
make dev-gateway-new
make dev-code-agent-new
make dev-web-agent-new
make dev-frontend

# 当前 smoke / 环境诊断入口
make smoke-new-arch
make smoke-new-arch-sh
make doctor-new-arch
make doctor-new-arch-sh
```

更完整的现状、端口与验证边界见 [LOCAL_DEV_GUIDE.md](LOCAL_DEV_GUIDE.md)。旧 `server/`、根目录 `agents/`、旧 Compose 与旧设计材料仅为历史参考；请从 [docs/legacy](docs/legacy) 中查阅，而不要将其视为 Active 架构。
