# ADR-V2-04：分层 Memory 与 MemoryController

- 日期：2026-10-29
- 状态：已采纳（P0）
- 背景：现有 `pkg/adk/session.go`、`pkg/runtime/pruning` 只覆盖会话 CRUD 与上下文裁剪，不是长期记忆；Gateway MemoryStore 不是事实源。
- 决策：工作记忆（进程内+checkpoint）、短期会话记忆（MySQL session_events + 版本化 snapshot）、执行 checkpoint（MySQL）、长期记忆（MySQL entries/revisions/scopes/mutations）分层。MemoryController 统一召回、候选、确认、CAS、过期与遗忘；默认 `session_only`，`scoped_long_term` 需显式授权。Redis 仅缓存。
- 备选：直接把所有历史塞 prompt（预算不可控）；引入 Mem0/Letta/Hindsight 服务（数据出域与 Go/MySQL 边界不符）；让模型自动永久保存（无用户控制）。
- 后果：需要 scope/epoch/CAS/lineage、候选确认、purgeStatus 与在途授权边界；删除记忆不等于删除原始聊天/备份。
- 同步契约：`memory.md`、`data-model.md`、`mysql-schema.md`、`runtime-harness.md`、`redis-usage.md`。
