# R2-03 R2-03 integration evidence

- task: R2-03
- testId: R2-03
- commit: 29cf7779295608081e90dc5daaabba860a45807a (dev)
- go: go version go1.26.5 windows/amd64
- started: 2026-11-03T03:05:05Z
- ended: 2026-11-03T03:05:09Z
- duration: 4s
- packages: ./pkg/runtime/eventbus/...
- run pattern: ^TestIntegration
- go test exit: 0

## results

| metric | count |
|---|---|
| total tests | 6 |
| PASS | 6 |
| FAIL | 0 |
| SKIP | 0 |
| incomplete | 0 |
| interaction rows | 0 |
| overall | PASS |

## per test

| test | status | elapsed(s) |
|---|---|---|
| github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/eventbus#TestIntegrationRedisPublishConsumeOrderAndAck | ok | 0.110 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/eventbus#TestIntegrationRedisDuplicateDeliveryHasOneEffect | ok | 0.100 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/eventbus#TestIntegrationRedisClaimRedeliversUnackedEntry | ok | 0.200 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/eventbus#TestIntegrationRedisMalformedEntryGoesToDlq | ok | 0.100 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/eventbus#TestIntegrationRedisStreamTrimIsBounded | ok | 0.150 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/pkg/runtime/eventbus#TestIntegrationRedisStreamsCarryTTL | ok | 0.150 |

## skipped tests (must be explained)

- none

## failed tests

- none

## interaction coverage (plan §5.5)

| interaction | record file | rows | coverage |
|---|---|---|---|
| test case result | events.jsonl | 6 | yes |
| middleware interactions | interactions.jsonl | 0 | not_applicable: no test in this selection emitted interaction rows |
| user actions | - | 0 | not_applicable: gate script runs no browser code |
| SSE / AG-UI events | - | 0 | not_applicable: no SSE client in this selection |

## redaction

- DSN and Redis address values are replaced with `<redacted-dsn>` / `<redacted-addr>` in all logs.
- Environment values are read only through `ENVIRON` inside awk, so they never reach argv or evidence files.

## 执行记录（计划 §9）

任务 ID：R2-03
对应缺口：GAP-10；P7-MQ-02
目标行为：提供 Redis Streams 适配器（XADD / XREADGROUP / XACK / XAUTOCLAIM / consumer group / pending / trim / 封套编解码），并在真实 Redis 7.2 上证明投递语义，不接生产链路。

### 交付内容

- `pkg/runtime/eventbus/redis.go`：`RedisConfig`（addr/password/db、`agenthub:{env}:run:events` 与 `{env}:dlq` 两个流名、group/consumer/ConsumerKey、`MaxLen`/`DLQMaxLen`、`ClaimMinIdle`/`ClaimPeriod`、`Block`、`Count`、`MaxAttempts`、`StreamTTL`/`DLQStreamTTL`）；
  `NewRedisPublisher`/`NewRedisConsumer`；`EnsureGroup` 容忍 BUSYGROUP；读循环 `XREADGROUP ... BLOCK ... STREAMS key >`；
  认领循环 `XPENDING` + `XAUTOCLAIM`；成功 `XACK`；畸形帧与超 `MaxAttempts` 写 DLQ 后 `XACK`；`Hooks` 指标钩子与 traceId 透传。
- `pkg/runtime/eventbus/redis_integration_test.go`：6 个 `TestIntegrationRedis*` 用例（无 `AGENTHUB_REDIS_TEST_ADDR` 时显式 SKIP 并写原因）。
- `pkg/runtime/go.mod`/`go.sum`：新增 `github.com/redis/go-redis/v9 v9.17.3`（与 `services/orchestrator/go.mod` 同一版本，未引入新第三方库），`go mod tidy` 后归入直接依赖块。

### 真实环境暴露并已修正的两点

1. **DLQ 与事件流没有过期时间**：实现只做了 `XADD MAXLEN ~`，而 `docs/contracts/redis-usage.md` §6 明确「禁止无 TTL 的业务 key」。
   补 `StreamTTL`（默认 1h，与现有 relay 一致）与 `DLQStreamTTL`（默认 30 天），发布与 DLQ 写入都改为 `TxPipeline` 的 XADD + EXPIRE，
   并新增 `TestIntegrationRedisStreamsCarryTTL` 断言两个 key 的 TTL 落在 `(0, 上界]`。
2. **XADD 必须使用 `~` 近似裁剪**：`Approx: true` 已固定，`MAXLEN=0` 时回退到契约推荐值（100000 / 10000）而不是无限增长。

### 结果

- 命令：`bash scripts/run-integration-evidence.sh --test-id R2-03 --plan-task R2-03 --packages "./pkg/runtime/eventbus/..." --run '^TestIntegration' --timeout 6m`
- 环境：真实 `redis:7.2-alpine`（AOF 开启的专用测试容器），地址由操作者 export；证据内一律 `<redacted-addr>`。
- **6 个用例：6 PASS / 0 FAIL / 0 SKIP**（门禁退出码 0）。

| 契约规则 | 用例 | 结果 |
|---|---|---|
| 同 run 顺序 + 事务后 ack（EB-ORD/EB-DEL） | `TestIntegrationRedisPublishConsumeOrderAndAck` | PASS |
| at-least-once 重复投递只产生一次业务效果（EB-IDM） | `TestIntegrationRedisDuplicateDeliveryHasOneEffect` | PASS |
| 未 ack 条目由 `XAUTOCLAIM` 重投（EB-DEL-05） | `TestIntegrationRedisClaimRedeliversUnackedEntry` | PASS |
| 畸形帧进 DLQ 且 DLQ 只存元数据（EB-SEC/EB-DEL-04） | `TestIntegrationRedisMalformedEntryGoesToDlq` | PASS |
| `MAXLEN ~` 有界裁剪（EB-CAP） | `TestIntegrationRedisStreamTrimIsBounded` | PASS |
| 事件流与 DLQ 均有界 TTL（redis-usage §6） | `TestIntegrationRedisStreamsCarryTTL` | PASS |

- 其它验证：`go build ./...`、`go test ./eventbus/... -count=1`（含 41 个用例的内存实现 + 契约测试）、`gofmt -l` 无输出、`go vet` 无告警。

### 未做与不声称

- **未接入生产**：`services/orchestrator/relay` 仍写旧的 per-run 流；接入在 R2-04（feature flag，默认 legacy）。
- 未做消费者幂等表与 migration：R2-05；未做重试指标暴露与运维接口：R2-06；未做故障/重复/顺序端到端验收：R2-08。
- 进程内去重 map 不作为唯一幂等手段（契约 EB-IDM-06），真实跨进程幂等依赖 R2-05 的 `(consumer_key,event_id)` 表。

### 改动文件（实际）

`pkg/runtime/eventbus/redis.go`、`pkg/runtime/eventbus/redis_integration_test.go`、
`pkg/runtime/eventbus/{eventbus.go,memory.go,dispatch.go,metrics.go}`（R2-02 同批）、
`pkg/runtime/go.mod`、`pkg/runtime/go.sum`、本证据目录

### 回滚方式

删除 `redis.go` 与 `redis_integration_test.go`，`go mod tidy` 收回依赖；现有 relay 不受影响。

### 交互覆盖（计划 §5.5）

| 交互 | 记录文件 | 条数 | 覆盖 |
|---|---|---|---|
| 用户操作 | - | 0 | not_applicable: 中间件适配层，无 UI |
| API 请求/响应 | - | 0 | not_applicable: 不经 HTTP |
| SSE / AG-UI 事件 | - | 0 | not_applicable: 适配器只处理封套，不产生 SSE |
| HITL | - | 0 | not_applicable: 信号消费属 R2-05/R2-07 |
| 工具授权/结果 | - | 0 | not_applicable |
| 记忆召回 | - | 0 | not_applicable |
| Artifact | - | 0 | not_applicable: payload 仅允许 artifact 引用，已由封套校验测试 |
| MQ publish/consume/ack/claim/DLQ | events.jsonl + logs/go-test.jsonl | 6 | yes（每条含 testId/命令/结果/耗时） |
| 多实例 | - | 0 | not_applicable: 单进程双消费者；跨实例接管在 R2-08 |
| 中间件 | events.jsonl | 6 | yes（真实 Redis 7.2） |
