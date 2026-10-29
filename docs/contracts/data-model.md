# Data Model Contract

**Status:** Active (v2)
**Owner:** AgentHub
**Last updated:** 2026-10-29（P0：数据所有权与 Memory/执行账本分层）

## Scope

- Gateway 拥有浏览器会话/消息与运行投影。
- Orchestrator 拥有执行、审批、预算、工具账本、checkpoint、事件 journal/outbox 与长期记忆。
- Runtime 拥有 Agent session、事件与短期摘要。
- Redis 只保存可重建辅助数据。

## Current target database

MySQL 8.4 LTS 是 Gateway、Orchestrator、Memory 与 Runtime session 的目标存储。SQLite 仅允许显式 demo 兼容配置，不得作为生产主入口或恢复依据。

## Ownership

### Gateway（`agenthub_gateway`）

- `conversations`
- `messages`
- `chat_requests`（请求幂等）
- `run_views`
- `event_projection_offsets`
- `artifact_views`

### Orchestrator（`agenthub_orchestrator`）

- `executions`
- `task_executions`
- `plan_revisions`
- `approvals`
- `tool_calls`
- `checkpoints`
- `execution_events`
- `outbox`
- `agent_catalog`

### MemoryController（`agenthub_memory`）

- `memory_scopes`
- `memory_entries`
- `memory_revisions`
- `memory_mutations`

### Runtime session（`agenthub_runtime`）

- `sessions`
- `session_events`
- `session_context_snapshots`

## Separation rules

- Orchestrator 不写 Gateway 会话表；通过内部 API/gRPC 获取有界历史。
- Child Agent 不写 Gateway/Orchestrator 表；工具结果通过内部协议回报。
- Gateway 不直接操作 Memory 表；只代理 Orchestrator MemoryController。
- Runtime session storage 必须满足 `adk.SessionService`。
- 模块间引用只存 ID/版本，不复制对方整行数据。
- 同一逻辑 schema 内不使用多套 truth；缓存/投影可重建且带来源版本。

## 保留与删除

- Memory 删除 ≠ 删除原始聊天/备份；操作结果区分 `recallDisabled` 与 `purgeStatus`。
- Session/checkpoint/Memory/备份有独立保留策略；读取不自动延长保留期。
- 备份恢复后先重放 Memory 删除标记与最新策略，再开放召回。
- 详细字段见 `mysql-schema.md`，行为见 `memory.md` 与 `runtime-harness.md`。
