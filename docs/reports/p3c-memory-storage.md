# P3c 阶段报告：Memory 存储与 Controller（部分）

- 日期：2026-10-29
- 基线：`dev / 70e20ca`（P3a 完成后）
- 状态：**Memory 存储/策略/Controller 已落地；短期记忆快照、Memory 工具、管理 API/UI、Redis 缓存未完成**。

## 1. 交付

| 交付 | 路径 | 说明 |
|---|---|---|
| Memory 迁移 | `services/orchestrator/internal/persistence/mysql/migrations/002_memory.sql` | scopes/entries/revisions/mutations + 唯一约束与索引 |
| Memory Repository | `.../mysql/memory_repository.go` | EnsureScope、Recall（active/未过期）、Propose(candidate)、Confirm(CAS+epoch)、Update(新 revision+CAS)、Forget(tombstone+epoch+mutation) |
| MemoryController | `services/orchestrator/internal/memory/controller.go` | 模式/scope 授权、召回过滤与去重、token/条数配额、活跃条目配额、候选 24h 过期、CAS/遗忘 |
| Policy/接口 | `pkg/runtime/memory/policy.go`、`controller.go` | off/session_only/scoped_long_term；默认 session_only；Entry/Store/Controller 接口 |

## 2. 验证

```text
$ go test ./services/orchestrator/internal/memory/... -count=1
PASS（默认策略、scope 隔离、候选不可召回、确认、CAS 冲突、遗忘、配额、候选过期）

$ go test ./services/orchestrator/internal/persistence/mysql/... -count=1
PASS（sqlmock 账本用例）
SKIP 2 个 Memory real-MySQL 集成用例：AGENTHUB_MYSQL_TEST_DSN 未设置

$ go test ./pkg/runtime/... ./services/orchestrator/... -count=1
PASS（全部包 ok）
```

## 3. 未完成/未验证

1. 真实 MySQL 的 scope/epoch/CAS/遗忘事务未执行（DSN 门控 SKIP）。
2. 短期会话记忆：`session_context_snapshots` 未接入；摘要来源区间/lineage/工具配对未实现。
3. Memory 工具 `memory.search/read/propose` 未注册到 Tool Broker。
4. Gateway Memory 管理 API/UI 未实现（P4）。
5. Redis Memory 缓存与失效事件未实现（P4）；当前每次读 SQL，无缓存一致性问题但性能未验证。
6. 遗忘后的缓存/派生物清理 worker、purgeStatus 更新、备份恢复重放未实现。
7. 敏感内容检测、来源删除联动、审批绑定与幂等 mutation 未完整接线。

## 4. 结论

可声明“MemoryController 策略与 MySQL 存储/候选/确认/CAS/遗忘逻辑已实现并有单元测试”；不能声明 P3c 完成，也不能声明真实模型已正确召回记忆。
