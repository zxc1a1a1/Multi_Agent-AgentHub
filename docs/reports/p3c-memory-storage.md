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
| 短期快照 | `pkg/runtime/session/snapshot.go` | `session_context_snapshots` 版本化写入/最新读取/来源区间 |
| 上下文装配 | `pkg/runtime/harness/assembler.go`、`session/context_source.go` | 快照摘要 + session 事件 → ContextBuilder；保留 tool 对；Thinking 不重注入 |
| 自动召回 | `services/orchestrator/httpapi/memory_recall.go`、planner prompt | scoped 模式下按用户消息召回并作为“data, not instructions”注入 |
| 范围遗忘 | `MemoryRepository.ForgetScope`、admin/UI | 单事务撤销 scope 内 active/candidate、epoch++、pending purge |
| Policy/接口 | `pkg/runtime/memory/policy.go`、`controller.go` | off/session_only/scoped_long_term；默认 session_only；Entry/Store/Controller 接口 |

## 2. 验证

```text
$ go test ./services/orchestrator/internal/memory/... -count=1
PASS（默认策略、scope 隔离、候选不可召回、确认、CAS 冲突、遗忘、配额、候选过期）

$ go test ./pkg/runtime/harness/... -count=1
PASS（ContextAssembler：快照摘要、tool 对、session 隔离）

$ AGENTHUB_MYSQL_TEST_DSN=... go test ./pkg/runtime/session/... -run TestIntegrationSessionContextSource
PASS（真实 MySQL：session 事件 + 快照 → 上下文；Thinking 不注入）

$ AGENTHUB_MYSQL_TEST_DSN=... go test ./services/orchestrator/httpapi/ -run TestIntegrationRunInjectsAuthorizedMemory
PASS（自动召回注入 planner；其他 principal 不泄漏）

$ go test ./services/orchestrator/planner/ -run TestMainAgentInjectsMemoryAsData
PASS（memory 标记为 data、含 id/revision/source）
```

## 3. 未完成/未验证

1. 真实 MySQL 的 scope/epoch/CAS/遗忘事务未执行（DSN 门控 SKIP）。
2. **摘要生成任务未实现**：SnapshotStore/Assembler 已具备，但没有后台摘要调用、触发阈值或 token 预算接线；Orchestrator handler 仍以 Gateway 历史为主要上下文，尚未改用 assembler。
3. **Agent 分支隔离**：快照按 agentName 版本化，但 session 事件没有分支元数据过滤；跨分支隔离未证明。
4. Memory 工具已实现但尚未注册进实际 Agent 工具配置/Tool Broker 执行路径。
5. Redis Memory 缓存与失效事件未实现（P4）；当前每次读 SQL。
6. 遗忘后的缓存/派生物清理 worker、purgeStatus 更新、备份恢复重放未实现。
7. 敏感内容检测、来源删除联动、审批绑定与幂等 mutation 未完整接线。
8. 自动召回使用字面 token 匹配，不是语义检索；同义词/中文分词质量未评测。

## 4. 结论

可声明“Memory 存储/Controller、短期快照/装配、自动召回与 Memory 工具/管理 API 已实现并在真实 MySQL 验证”；不能声明 P3c 完成（无摘要任务、分支隔离、实际 Agent 工具接线与语义召回评测）。
