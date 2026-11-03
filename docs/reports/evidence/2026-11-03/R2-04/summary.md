# R2-04 Outbox relay v2

- task: R2-04（`docs/plans/v3-gap-atomic-plan.md` §4.3）
- 对应缺口：GAP-10；P7-MQ-02
- commit baseline: `4cb2243`（dev）
- 环境：Windows 10.0.19045；真实 Redis 7.2.17（db 3 隔离）；`go1.26.5`
- overall: **PASS**（单测 4/4，集成 5/5，0 SKIP）

## 1. 结果

| 套件 | 测试 | 结果 |
|---|---|---|
| 单元 | TestRelayEventBusPublishesEnvelopesInOrder | PASS |
| 单元 | TestRelayEventBusPublishFailureLeavesOutboxPending | PASS |
| 单元 | TestRelayEventBusLeaseLostStopsPublishing | PASS |
| 单元 | TestRelayEventBusEventIDStableAfterMarkLoss | PASS |
| 集成 | TestIntegrationRelayEventBusWritesSharedStream（新增） | PASS |
| 集成 | TestIntegrationRelayPreservesPerRunOrder（legacy） | PASS |
| 集成 | TestIntegrationRelayRedeliversWhenMarkIsLost（legacy） | PASS |
| 集成 | TestIntegrationRelayKeepsOutboxPendingOnPublishFailure（legacy） | PASS |
| 集成 | TestIntegrationRelayRebuildsStreamFromSQLAfterDataLoss（legacy） | PASS |

## 2. 行为

- `EVENT_BUS=legacy`（默认）：完全走原 per-run stream，行为不变（5 个 legacy 用例证明）。
- `EVENT_BUS=redis_streams`：relay 用 13 字段封套发布到 `agenthub:{env}:run:events`；
  `eventId=evt_<runId>_<seq>_0`，`idempotencyKey=<runId>:<seq>:0`；
  conversationId 取 `executions.conversation_id`，测试替身回退到 runId。
- publish 成功后才 `MarkOutboxPublished`；失败只 `MarkOutboxAttempt` + 指数退避，outbox 保持 pending。
- publish 后 mark 前崩溃：下一轮重复投递，封套 `eventId` 不变，消费者可按 `(consumerKey,eventId)` 去重。
- 单写者租约：`OUTBOX_RELAY_LEASE=1` 时用 MySQL `GET_LOCK('agenthub_outbox_relay')` 的专用连接；
  `Valid()` 每次检查 `IS_USED_LOCK = CONNECTION_ID()`，租约丢失时 `RunOnce` 返回 `ErrLeaseLost` 且 `Run` 停止。
- 另一个实例已持有租约时，本实例 relay 进入 standby 且**不启动发布**。

## 3. 交互覆盖（§5.5）

| 交互 | 记录文件 | 条数 | 覆盖 |
|---|---|---|---|
| MQ publish/outbox mark/backoff | events.jsonl | 9（test_case 粒度） | yes |
| Redis 共享 stream / 旧 per-run stream | logs/*.jsonl | 2 | yes |
| 用户/浏览器/HITL/工具/记忆/Artifact | - | 0 | not_applicable: 后端 relay 测试 |
| 多实例 | - | 0 | not_applicable: 租约单测；进程级多副本见 R0-06，MQ 多副本见 R2-08 |

## 4. 未包含 / 边界

- MySQL 命名锁未做真实 DB 的并发/失效集成测试（`AdvisoryLock` 实现 + 单测租约语义）；R2-08 补进程级。
- standby 不做自动重新竞选；leader 崩溃后需要重启 standby 或后续 R2-06/R2-08 增加选主循环。
- 消费者幂等表、DLQ 指标、Gateway 投影消费分别属于 R2-05/R2-06/R2-07。
- 生产线默认仍是 `legacy`；未在默认部署中开启 v2。

## 5. 回滚

- `EVENT_BUS=legacy`（默认）；或 revert 本提交，旧 per-run stream 不受影响。

## 6. 证据

- `docs/reports/evidence/2026-11-03/R2-04/`（commands.txt、events.jsonl、summary.md、logs/unit.jsonl、logs/integration.jsonl）
- 契约状态更新：`docs/contracts/event-bus.md` §9.1
