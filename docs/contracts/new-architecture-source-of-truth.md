# New Architecture Source of Truth

**Status:** Active (v2)
**Owner:** AgentHub
**Last updated:** 2026-10-29（P0）

## Purpose

防止后续工作漂回旧 `server/` + 根 `agents/` 布局，或退回规则路由/单一存储。

## Active module boundaries

| Module | Path | Responsibility | Forbidden responsibility |
|---|---|---|---|
| ADK engine | `pkg/adk` | Agent/Model/Tool/Plugin/Runner/Session 接口、Event、A2A adapter | YAML、具体 provider、业务 handler、Redis/MySQL/MCP 客户端 |
| Runtime framework | `pkg/runtime` | Config、registries、providers、session、pruning、skills、AG-UI、launcher | 业务 Agent 逻辑 |
| Runtime Harness | `pkg/runtime/harness`（P3） | ContextBuilder、BudgetManager、LoopController、Checkpoint、RunController | 语义路由、第二套 Agent 循环 |
| Runtime Tools | `pkg/runtime/tools`（P3） | ToolDescriptor、Broker、原生工具、MCP adapter | 绕过审批/账本的副作用 |
| Runtime Memory | `pkg/runtime/memory`（P3） | MemoryController/Policy、存储适配、召回/候选/遗忘 | 模型直接改策略、跨 scope 读取 |
| Gateway | `services/gateway` | 公开 HTTP/SSE、鉴权、会话/消息/投影、内部 gRPC client | Planning、语义路由、工具执行、直连 Child Agent |
| Orchestrator | `services/orchestrator` | MainAgent、Coordinator、Harness、Memory、Tool Broker、A2A、事件 | 浏览器 API、会话存储、Redis 权威状态 |
| Child Agents | `services/agents/*` | Agent 行为、A2A server、AgentCard、受限工具 | Frontend 协议、Gateway/Orchestrator 表 |
| Frontend | `frontend` | React UI、AG-UI reducer、游标恢复、skill 渲染 | 直连 Orchestrator/Agent/数据库 |

## Legacy path rule

New development MUST NOT target these paths unless doing migration/removal:

```text
server/**
agents/**
```

## Current known gaps (P0 记录)

- Gateway 目标启动仍默认 MemoryStore，SQLite 镜像不是 MySQL 主存储。
- Orchestrator 启动仍默认 rule 模式，MainAgent 存在关键词回退。
- 无 MySQL 执行账本、checkpoint、outbox、事件 journal。
- 无 Redis 接线、无内部 gRPC start/subscribe/control。
- 无 Harness/MemoryController/Tool Broker 实现。
- Gateway→Orchestrator 目标协议为 gRPC；HTTP/SSE 仅迁移兼容。
- Canonical Docker topology 当前未含 Redis；P7 增加。

以上差距按 `docs/plans/agenthub-redis-sql-harness-plan.md` 分阶段关闭，不能在文档中提前标记完成。
