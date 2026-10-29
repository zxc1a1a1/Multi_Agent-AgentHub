# Project Architecture Contract

**Status:** Active (v2)
**Owner:** AgentHub
**Last updated:** 2026-10-29（P0：加入 Redis、Runtime Harness、MemoryController 与事件重放）

## System topology（目标）

```text
Frontend ──REST + AG-UI SSE（可恢复游标）──> Gateway
                                             ├─ MySQL：会话/消息/请求幂等/投影
                                             └─ 内部 gRPC：start/subscribe/approval/cancel/status
                                                   ▼
                                             Orchestrator
                                             ├─ MainAgent（唯一 LLM 决策入口）
                                             ├─ Coordinator / Runtime Harness
                                             ├─ MemoryController
                                             ├─ Tool Broker
                                             ├─ MySQL：执行/审批/账本/checkpoint/events/outbox/Memory
                                             ├─ Redis：缓存/短期事件副本/心跳（非权威）
                                             └─ A2A → Child Agents → ADK / Tools / Sandbox
```

迁移期 HTTP/SSE 仅是兼容实现；目标协议是 gRPC。公开端仍是 AG-UI SSE。

## Current modules

```text
pkg/adk
pkg/runtime
services/gateway
services/orchestrator
services/agents/*
frontend
```

拟新增（P3）：

```text
pkg/runtime/harness
pkg/runtime/tools
pkg/runtime/memory
services/gateway/proto、services/orchestrator/proto（P4）
```

## Rules

1. Frontend talks only to Gateway；不连接 Orchestrator、Redis、MySQL、MCP 或 Child Agent。
2. Gateway 持久化会话/消息与投影，代理内部 gRPC；**不**规划、不语义路由、不直接调用 Agent（Orchestrator 已配置时）。
3. Orchestrator 是唯一编排决策/执行协调服务；Planner 只产出结构化结果。
4. Child Agents 是独立 A2A 服务；不直接改 Orchestrator 执行表。
5. ADK 保持纯引擎：无具体 provider、无项目业务逻辑、无 Redis/MySQL/MCP 客户端。
6. Runtime Harness 控制预算、权限、上下文、checkpoint、恢复与终态；不是第二套 Agent 循环。
7. MemoryController 统一长短期记忆读取/候选/确认/CAS/遗忘；模型不能直接改策略。
8. Legacy `server/` 与根目录 `agents/` 是参考实现，不新增目标路径工作。

## Profiles

| Profile | Purpose | DB | Gateway↔Orchestrator |
|---|---|---|---|
| Target | Architecture contract | MySQL + Redis | gRPC streaming |
| Demo compatibility | Temporary | SQLite allowed（显式标记） | HTTP/SSE allowed only if marked compatibility |

## Review blockers

A change is blocking if it:

- Imports Orchestrator into Gateway as in-process logic.
- Makes Frontend call Orchestrator/Child Agent/Redis/MySQL directly.
- Adds concrete LLM providers or storage clients under `pkg/adk`.
- Adds rule/keyword fallback or a second semantic router outside MainAgent.
- Adds new work to legacy `server/` or root `agents/` without a migration task.
- Treats Redis as authoritative for approvals, terminal state, budgets or Memory.
- Introduces a second Agent loop or bypasses Tool Broker for side effects.
