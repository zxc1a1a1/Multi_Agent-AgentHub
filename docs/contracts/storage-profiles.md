# Storage Profiles Contract

**Status:** Active (v2)
**Owner:** AgentHub
**Last updated:** 2026-10-29（P0：MySQL 权威 + Redis 辅助；取消 MemoryStore/SQLite 双轨目标）

## Profiles

| Profile | Use | Status |
|---|---|---|
| `mysql` | Gateway 业务表、Orchestrator 执行账本、Memory、Runtime session 的权威存储 | **target（生产默认）** |
| `redis` | 带版本的能力/Memory 缓存、短期事件副本、心跳 | **target（辅助，永不权威）** |
| `sqlite-demo` | 本地/演示兼容；不得作为生产主路径或恢复依据 | temporary / legacy |
| `memory` | 单元测试与短暂 Child Agent runtime | active for tests only |
| `object-storage` | 大文件 artifact | future |

## 权威性规则

| 所有者 | 数据 | 权威性 |
|---|---|---|
| Gateway | conversations、messages、chat_requests、run_views、event_projection_offsets、artifact_views | 会话/用户消息权威；运行与 Agent 消息是可重建投影 |
| Orchestrator | executions、task_executions、plan_revisions、approvals、tool_calls、checkpoints、execution_events、outbox、agent_catalog | 执行/审批/预算/账本权威 |
| Orchestrator MemoryController | memory_entries、memory_revisions、memory_scopes、memory_mutations | 长期记忆/策略/epoch/遗忘唯一权威 |
| Runtime | sessions、session_events、session_context_snapshots | Agent 会话与短期摘要 |
| Redis | 版本化缓存、事件短期副本、心跳 | 可重建辅助；不能决定审批、终态或记忆可用性 |

## Rules

1. 目标启动图使用 MySQL Repository；不得以 `MemoryStore` 决定历史会话是否存在。
2. SQLite 只允许显式 demo 配置；文档不得把它写成目标。
3. 不新增全局 writer；每次写入使用独立连接/事务与条件更新。
4. SQL 不可用时停止接受需要记账的新副作用；Redis 不可用时可降级为 SQL 有界补读。
5. 不跨模块直接写对方表；同一 MySQL 实例内使用独立逻辑 schema/账号。
6. 缓存不得绕过权限、删除、epoch 或 revision 校验；TTL 不是遗忘正确性保证。
