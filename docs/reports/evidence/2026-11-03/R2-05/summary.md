# R2-05 幂等消费者与 ack/claim

- task: R2-05（`docs/plans/v3-gap-atomic-plan.md` §4.3）
- 对应缺口：GAP-10；P7-MQ-03；MQ-01/MQ-02
- 环境：Windows；MySQL 8.4.13（真实测试库）；`go1.26.5`
- overall: **PASS**（5/5，0 SKIP）

## 1. 结果

| 测试 | 结果 | 覆盖 |
|---|---|---|
| TestIntegrationIdempotentProcessHasOneEffect | PASS | 同一 eventId 两次 → 业务效果 1 次，第二次返回 duplicate |
| TestIntegrationConcurrentDuplicateHasOneEffect | PASS | 两个并发消费者 → 恰好 1 个 duplicate，效果 1 次 |
| TestIntegrationFailedHandlerRollsBackAndRetries | PASS | handler 失败回滚业务写入，记录 failed；重试成功 1 次 |
| TestIntegrationSameEventDifferentPayloadIsConflict | PASS | 同 eventId 不同 payload → ErrIdempotencyConflict，不追加效果 |
| TestIntegrationWrapHandlesDuplicateEventBusMessage | PASS | 封套 Handler 包装，重复消息只产生一次业务效果 |

## 2. 实现

- 迁移：`services/orchestrator/.../migrations/008_event_consumer_idempotency.sql` 与
  `services/gateway/store/migrations/004_event_consumer_idempotency.sql`（同表，幂等 DDL）。
- 表：PK `(consumer_key, event_id)`，字段 `event_hash/status/result_ref/attempts/时间`。
- `pkg/runtime/eventbus/sqlidem.Process`：
  - 事务内 `INSERT ... ON DUPLICATE KEY UPDATE attempts=attempts+1` + `SELECT ... FOR UPDATE`；
  - 已 `done` 且 hash 相同 → duplicate（ack，不执行）；hash 不同 → conflict；
  - handler 与幂等行共用同一事务：业务效果和 `done` 同提交；
  - handler 失败先回滚业务事务，再用独立事务记录 `failed`（不提交半成品效果）。
- `Wrap` 把该语义适配为 `eventbus.Handler`；返回 nil 时传输层才 ack，失败则保持未 ack 供重投。
- 崩溃窗口：commit 后、ack 前崩溃 → 重投时 `done` 命中，只 ack 不重复执行。

## 3. 交互覆盖（§5.5）

| 交互 | 记录文件 | 条数 | 覆盖 |
|---|---|---|---|
| MQ consume/ack/duplicate/conflict | events.jsonl | 5 | yes |
| MySQL 事务/唯一键 | logs/go-test.jsonl | 5 | yes |
| 用户/SSE/HITL/工具/记忆/Artifact | - | 0 | not_applicable: 消费者层测试 |
| 多实例 | - | 0 | not_applicable: 并发 goroutine；进程级见 R2-08 |

## 4. 未包含

- 尚未接到 Gateway 投影 worker（R2-07）；transport 侧 maxAttempts/DLQ 由 R2-06。
- `attempts` 记录仅作审计；重试预算仍以消费者 PEL attempt 为准。
- 测试使用独立的 `r205_effects` 表模拟业务写入。

## 5. 回滚

- 停止消费者即可；表只增不删，迁移向前兼容。

## 6. 证据

- `docs/reports/evidence/2026-11-03/R2-05/`（commands.txt、events.jsonl、summary.md、logs/go-test.jsonl）
