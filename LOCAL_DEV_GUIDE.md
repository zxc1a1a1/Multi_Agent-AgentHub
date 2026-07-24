# AgentHub 本地开发指南

## 先读状态

产品级事实源是 [PDR](docs/pdr/AgentHub_2.0_PDR.md)，Active Contract 索引是 [docs/contracts/README.md](docs/contracts/README.md)。当前状态为 **2.0 Contract 已冻结，业务实现迁移中**。

当前 `docker-compose.new-arch.yml` 仍是十个 specialist Agent 的迁移态 Compose；它不是 2.0 Target 的默认五服务拓扑。2.0 Target 为 Frontend、Gateway、Orchestrator、CodeAgent、WebAgent，远程 A2A Agent 通过运行时注册接入。不要将当前 Compose 的十 Agent 配置解释为固定目标架构。

## 已存在的启动入口

下列命令只记录仓库中已经存在的入口，不会安装依赖，也不代表未实现的 2.0 能力已经完成。

```bash
# 当前 new-arch Compose
make docker-new-arch-up
make docker-new-arch-down
make docker-new-arch-logs
make docker-new-arch-config

# 本地服务入口
make dev-gateway-new
make dev-code-agent-new
make dev-web-agent-new
make dev-frontend
```

`make dev-new-arch` 只并行启动 Gateway、CodeAgent 与 WebAgent；它没有启动 Orchestrator，因此不应被描述为完整 2.0 链路。

## 当前可用检查

```bash
make smoke-new-arch
make smoke-new-arch-sh
make doctor-new-arch
make doctor-new-arch-sh
```

Smoke 的可用范围由当前脚本决定；不要将其结果扩展解释为动态注册、PlanVersion 确认或完整 SQLite/WAL 迁移已经验收。

## Provider 与凭据

`.env.example` 只提供空占位符。Provider 或 Agent 凭据由运行环境注入，绝不能写进 `VITE_*`、示例配置、日志或文档。Mock/确定性路径应当不依赖生产凭据。

## 2.0 行为边界

- CodeAgent、WebAgent：内置稳定参考 Agent。
- 动态 Agent：Target/In Progress；仅符合 A2A 约束、已校验且被授权的 Agent 才可进入可用目录。
- Direct：调用单个符合条件的 Agent，不是多 Agent Plan。
- Manual Multi-Agent：LLM 只能在用户选定集合中生成计划。
- Auto：LLM 从 Registry 可用目录中选择 Agent。
- Manual Multi-Agent 与 Auto：LLM 生成、Validator 校验、用户确认精确 PlanVersion 后才可执行。
- Planner、Context Manager、Synthesizer：Orchestrator 内部模块，不是 Agent。
