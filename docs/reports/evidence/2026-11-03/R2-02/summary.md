# R2-02 EventBus 接口与内存实现

- task: R2-02（`docs/plans/v3-gap-atomic-plan.md` §4.3）
- 对应缺口：GAP-10；P7-MQ-02 前置
- commit baseline: `a083e4df78886e68370c7719e607b1d3bf3a4e9d`（dev）
- 环境：Windows 10.0.19045 / `go version go1.26.5 windows/amd64`
- 范围：`pkg/runtime/eventbus` 的接口、封套编解码、内存传输、per-run 串行 dispatcher、指标钩子；**不含** Redis 适配器（R2-03）和生产线接线（R2-04）
- overall: **PASS**

## 1. 结果

| 指标 | 数值 |
|---|---|
| 测试 | 36 |
| PASS | 36 |
| FAIL | 0 |
| SKIP | 0 |
| `go vet ./eventbus/` | 0 问题 |

## 2. 覆盖

- 发布/消费往返，13 字段封套编解码，schemaVersion/attempt 派生规则；
- 重复投递只产生一次业务效果（按 eventId 去重）；
- 预算内重试成功；超预算进入内存 DLQ，DLQ 只含元数据、无 payload；
- 同一 run 严格保序、不同 run 可并发；并发消费不越序；
- context 取消后消息保持未 ack；Close 停止 goroutine 并拒绝新发布；
- handler panic 不 ack；非法封套被拒绝；maxAttempts 有上限；
- 封套 allowlist 与 `pkg/runtime/agui` 及契约 schema 一致；DLQ 元数据校验。

## 3. 交互覆盖（§5.5）

| 交互 | 记录文件 | 条数 | 覆盖 |
|---|---|---|---|
| MQ publish/consume/ack | events.jsonl | 36 | yes（test_case 粒度，含 retry/dlq/order/cancel） |
| 用户/浏览器 | - | 0 | not_applicable: 进程内库测试 |
| HITL/工具/记忆/Artifact | - | 0 | not_applicable: 与传输层无关 |
| 多实例 | - | 0 | not_applicable: 单进程内存实现 |

## 4. 回滚

- 删除 `pkg/runtime/eventbus` 内存实现相关文件；无调用点，运行时不变化。
- 说明：Redis 适配器随 R2-03 单独提交，本任务提交不包含 `redis.go`/`redis_integration_test.go` 与 go-redis 依赖。

## 5. 证据

- `docs/reports/evidence/2026-11-03/R2-02/`（commands.txt、events.jsonl、summary.md、logs/go-test.jsonl）
