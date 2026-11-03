# R2-08 MQ 故障、重复与顺序验收

- commit: 1aef0a07e3ab27d27b7913ff62b2fae262e57ed3
- started: 2026-11-03T04:22:22Z / ended: 2026-11-03T04:23:26Z
- result: **PASS**（8 项断言，0 项失败）
- Redis: 7.2-alpine，AOF on，独立容器 agenthub-r208-redis (127.0.0.1:16380)
- MySQL: 真实测试库（sqlidem 用例）

## 断言结果

| 场景 | 结果 | 证据 |
|---|---|---|
| dedicated Redis 7.2 with AOF up | PASS | 127.0.0.1:16380 |
| eventbus-transport | PASS | pass=21 skip=3 |
| relay-outbox | PASS | pass=22 skip=6 |
| sqlidem | PASS | pass=6 skip=0 |
| gateway-projection | PASS | pass=14 skip=1 |
| restart seed leaves one unacked PEL entry | PASS | env=r208_1793679742 |
| Redis restarted with AOF | PASS | rto=4s |
| unacked PEL entry reclaimed after restart | PASS | no duplicate effect |

## 场景映射

| MQ 场景 | 覆盖 |
|---|---|
| MQ-01 重复投递只有一次业务效果 | sqlidem + eventbus-transport |
| MQ-02 ack/claim 不重复执行 | restart-claim + sqlidem |
| MQ-03 超 maxAttempts 进 DLQ、不自动重放 | eventbus-transport (malformed/DLQ) |
| MQ-04 指标 | R2-06 证据 |
| MQ-05 投影消费 | gateway-projection |
| MQ-06 同 run 顺序 | eventbus-transport (order) |
| P-06 consumer kill / claim | restart-claim |

## SKIP 明细

- `eventbus-transport` 的 3 个 skip：2 个是包级 skip（`pkg/runtime`、`pkg/runtime/proto/orchestratorv1` 无匹配测试），
  1 个是 `TestIntegrationRedisPendingSurvivesRestart` 在未设置 phase 时按设计跳过；该用例随后在
  seed/claim 两阶段真实执行并 PASS。
- `relay-outbox` / `gateway-projection` 的 skip 均为 `cmd/*` 等无测试文件的包级 skip。
- 关键 MQ 用例 0 SKIP。

## 交互覆盖（§5.5）

| 交互 | 记录文件 | 条数 | 覆盖 |
|---|---|---|---|
| MQ publish/consume/ack/claim/DLQ | events.jsonl + logs/*.jsonl | 8 | yes |
| 故障注入（Redis restart） | events.jsonl + logs/restart-*.jsonl | 3 | yes（RTO=4s） |
| 用户/SSE/HITL/工具/记忆/Artifact | - | 0 | not_applicable: 中间件故障矩阵 |
| 多实例 | - | 0 | not_applicable: 双 consumer 在 restart-claim 内模拟 |

## 边界
- 未做真实进程级 consumer SIGKILL；claim 由‘不 ack + 第二个 consumer’模拟。
- Redis AOF 在 stop/start 间保留； 会丢数据，脚本只在 teardown 删除容器。
- 未覆盖 Redis AOF off 场景（计划 MQ 场景中的 AOF on/off 只做了 on）。
