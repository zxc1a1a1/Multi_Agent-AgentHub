# ADR-V2-01：MySQL 权威 + Redis 辅助存储

- 日期：2026-10-29
- 状态：已采纳（P0）
- 背景：Gateway 默认 MemoryStore、可选 SQLite 镜像；Orchestrator 无持久执行账本。用户要求 Redis + MySQL 的新范围。
- 决策：MySQL 8.4 LTS 为 Gateway/Orchestrator/Memory/Runtime session 的权威存储；Redis 只做带版本缓存、短期事件副本与心跳。先提交 SQL + outbox，再异步投递 Redis；消费者按 `(runId, seq)` 去重。
- 备选：PostgreSQL + Redis（无当前需求证明迁移方言）；SQLite + Redis（不能解决多实例与主 Store 边界）；Redis-only（审批/终态/账本可靠性难解释）；SQL-only（缺短期广播/缓存，可作降级）。
- 后果：需要 outbox relay、去重、故障降级、集成测试与运维；SQL 不可用时停止接受新副作用；Redis 不可用时降级为 SQL 补读。
- 同步契约：`storage-profiles.md`、`mysql-schema.md`、`data-model.md`、`redis-usage.md`、`project-architecture.md`、`event-replay.md`。
