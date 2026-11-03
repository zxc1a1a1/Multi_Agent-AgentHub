# R2-06 重试、DLQ 与指标

- task: R2-06（`docs/plans/v3-gap-atomic-plan.md` §4.3）
- 对应缺口：GAP-10；P7-MQ-04
- 环境：Windows；`go1.26.5`；内存传输（真实 Redis 重试/DLQ 见 R2-03/R2-08）
- overall: **PASS**（4/4，0 SKIP）

## 1. 结果

| 测试 | 结果 | 覆盖 |
|---|---|---|
| TestMemoryRetryThenDlq | PASS | 超 maxAttempts 进 DLQ，不再自动重放 |
| TestMemoryRetryWithinBudget | PASS | 预算内重试后成功 |
| TestMemoryRetryDlqAndPrometheusMetrics | PASS | 重试/DLQ 计数 + Prometheus 文本全部 series |
| TestCountersPrometheusCountsPublishFailure | PASS | 被拒绝的 publish 计入 error，不计入 ok |

## 2. 指标契约（Prometheus 文本）

```
eventbus_publish_total{outcome="ok|error"}
eventbus_consume_total{result="ok|duplicate|failure"}
eventbus_ack_total{result="ok|error"}
eventbus_retry_total
eventbus_dlq_total
eventbus_pending
```

- 标签只含 outcome/result，不含事件内容、payload、错误文本（EB-DEL-09）。
- `MetricsHooks` 是适配器唯一上报缝；Redis adapter 已有 publish/consume/ack/retry/dlq/pending 钩子。
- DLQ 条目只有 `eventId/consumerKey/runId/seq/type/attempts/errorClass/时间`，无 payload；
  进入 DLQ 后只 ack，不存在自动重放代码。

## 3. 交互覆盖（§5.5）

| 交互 | 记录文件 | 条数 | 覆盖 |
|---|---|---|---|
| MQ retry/DLQ/metrics | events.jsonl | 4 | yes |
| 用户/浏览器/API/HITL/记忆/Artifact | - | 0 | not_applicable: 库级指标测试 |
| 多实例 | - | 0 | not_applicable: 内存传输；Redis 版见 R2-03/R2-08 |

## 4. 未包含 / 边界

- 未增加 `GET /internal/eventbus/stats` HTTP 端点（计划标注为可选）；指标目前由
  `Counters.Prometheus()` 输出，尚未挂到服务 metrics 路由，因为生产消费者还未接入（R2-07）。
- 指标为进程内计数；多副本聚合需由 Prometheus 抓取各实例。
- 真实 Redis 的 maxAttempts/DLQ 语义已在 R2-03 集成测试覆盖；进程级故障矩阵归 R2-08。

## 5. 回滚

- 关闭消费者/不启用 EVENT_BUS；DLQ 数据保留供排查。

## 6. 证据

- `docs/reports/evidence/2026-11-03/R2-06/`（commands.txt、events.jsonl、summary.md、logs/go-test.jsonl）
