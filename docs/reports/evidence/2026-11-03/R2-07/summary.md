# R2-07 Gateway 投影消费试点

- task: R2-07（`docs/plans/v3-gap-atomic-plan.md` §4.3）
- 对应缺口：GAP-10；P7-MQ-05；F-10
- 环境：Windows；真实 Redis 7.2.17（db 4 隔离）；`go1.26.5`
- overall: **PASS**（4/4，0 SKIP）

## 1. 结果

| 测试 | 结果 | 覆盖 |
|---|---|---|
| TestHandleEventRunsSQLRecoveryForTheRun | PASS | 队列消息只触发该 run 的 SQL recovery |
| TestHandleEventPropagatesRecoveryError | PASS | recovery 失败 → handler 返回错误，消息不 ack |
| TestMemoryBusDrivesSQLRecovery | PASS | 端到端：publish → consume → recovery 一次 |
| TestIntegrationEventBusConsumerDrivesRecovery | PASS | 真实 Redis：消费、recovery、ack 后 PEL=0 |

## 2. 设计

- `EVENTBUS_PROJECTION=1` + `REDIS_URL` 时，Gateway 启动 `projection` group 消费者；
  默认关闭，行为与现状一致（periodic SQL worker）。
- 消费者 handler 只调用 `Worker.RecoverRun(runID)`：**SQL public journal + projection offset
  仍是唯一事实源**，队列只是唤醒提示；重复/乱序消息不会重复投影（offset 幂等）。
- 失败（MySQL 不可用）→ handler 返回错误，消息保留 pending；periodic worker 仍按时兜底。
- 消费者使用 `assistant_message.v1` consumerKey、`gateway-<pid>` consumer、claim 30s/15s。
- 关闭时取消消费上下文并关闭 consumer/Redis 客户端；periodic worker 的关闭顺序不变。

## 3. 交互覆盖（§5.5）

| 交互 | 记录文件 | 条数 | 覆盖 |
|---|---|---|---|
| MQ consume/ack | events.jsonl | 4 | yes |
| SQL recovery / offset | events.jsonl + logs/go-test.jsonl | 4 | yes |
| 用户/API/SSE/HITL/工具/记忆/Artifact | - | 0 | not_applicable: 投影 worker 级测试 |
| 多实例 | - | 0 | not_applicable: R0-06 已认证双 worker；MQ claim 见 R2-03/R2-08 |

## 4. 未包含 / 边界

- 未用真实 Gateway 进程 + 真实 Redis + MySQL 跑完整 F-10 场景；F-10 的 SQL 路径已在 R0-06 PASS，
  本任务验证队列唤醒层。R2-08 负责进程级故障矩阵。
- Redis 停机时的回退由“periodic worker 独立运行”保证，未做停机注入测试（R2-08）。
- 队列消息是内部事件，不含 assistant 文本；投影内容始终来自 public journal。

## 5. 回滚

- `EVENTBUS_PROJECTION=0`（默认），恢复纯 SQL worker。

## 6. 证据

- `docs/reports/evidence/2026-11-03/R2-07/`（commands.txt、events.jsonl、summary.md、logs/go-test.jsonl）
- 契约状态：`docs/contracts/event-bus.md` §9.1
