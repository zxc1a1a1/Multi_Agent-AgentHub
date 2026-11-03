# R1-05 purge worker 与审计状态机

- task: R1-05（`docs/plans/v3-gap-atomic-plan.md` §4.2）
- 对应缺口：GAP-14；M-18
- 环境：Windows；MySQL 8.4.13（真实测试库）
- overall: **PASS**（单元 3/3；真实 MySQL 集成 1/1）

## 1. 状态机

```
pending / failed --claim(SKIP LOCKED)--> running --purge--> done
                                             \--error--> failed (purge_after 退避)
                                             \--attempt>=5--> dead（不自动重放）
```

- 迁移 `010_memory_purge.sql`：`purge_attempts/purge_after/purge_error/purged_at` + due 索引。
- Worker 用 `SELECT ... FOR UPDATE SKIP LOCKED` 批量 claim；`mutation_id` 幂等。
- 清理范围：`memory_revisions.content/source_refs`、`memory_entries.source_refs` 清空；
  tombstone（id/status/时间）保留；`memory_scopes.epoch + 1`。
- Redis recall cache 以 epoch 为键，epoch 递增后旧键不再命中；不物理删除旧键（TTL 到期）。
- 重试指数退避（1s 起，上限 5min），最大 5 次后 `dead`，**不自动重放**。
- 运维：`POST /internal/memory/mutations/{id}/retry` 把 failed/dead 显式置回 pending；
  `GET /internal/memory/mutations/{id}` 返回状态/attempts/时间，不返回内容。
- 指标：`PurgeMetrics{Purged, Failed, Dead}`（只计数）；日志只记录 mutationId/attempt/class。

## 2. 测试

- 单元：成功批量、失败重试、达到上限进 dead、claim 失败上报。
- 集成（真实 MySQL）：
  - forget → worker 清理 revision/entry source refs，epoch 递增，tombstone=revoked；
  - mutation 变为 done、attempts=1、purged_at 非空；
  - 再次 PurgeOnce 为幂等 no-op；
  - dead → 显式 RetryMutation → worker 再次处理；
  - forget-scope 清理该 scope 全部 revision 内容。
- 回归：`internal/memory`、`internal/persistence/mysql`、`httpapi` 全部通过。

## 3. 交互覆盖（§5.5）

| 交互 | 记录文件 | 条数 | 覆盖 |
|---|---|---|---|
| 记忆 forget/mutation/purge | events.jsonl | 4 | yes（状态/次数，无内容） |
| 用户/API/SSE/HITL/MQ | - | 0 | not_applicable: 后台 worker |
| 内容 | - | 0 | 不记录记忆正文 |

## 4. 边界

- **明确不删除**原始聊天记录、备份、下游日志；只停止召回并清空记忆内容。
- Worker 默认关闭（`MEMORY_PURGE_WORKER=1` 才启动）。
- 重试 HTTP 端点已实现但未单独做 HTTP 层测试（repository 层已测）。
- 多副本同时运行 worker 由 SKIP LOCKED 保证不重复 claim；进程级验收归 R5-02。

## 5. 回滚

- `MEMORY_PURGE_WORKER=0` 停止 worker；pending 状态保留，不丢请求。

## 6. 证据

- `docs/reports/evidence/2026-11-03/R1-05/`（commands.txt、events.jsonl、summary.md、logs/go-test.jsonl）
