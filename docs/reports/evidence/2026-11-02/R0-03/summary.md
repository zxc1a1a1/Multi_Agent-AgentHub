# R0-03 R0-03 integration evidence

- task: R0-03
- testId: R0-03
- commit: 8c6fb2cb78dd247627f6a3e800a2f336e82f1340 (dev)
- go: go version go1.26.5 windows/amd64
- started: 2026-11-02T09:11:47Z
- ended: 2026-11-02T09:11:53Z
- duration: 6s
- packages: ./services/orchestrator/relay/... ./services/orchestrator/internal/memory/...
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
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/memory#TestIntegrationRedisRecallCacheUsesEpoch | ok | 1.660 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/relay#TestIntegrationRelayRedeliversWhenMarkIsLost | ok | 0.020 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/relay#TestIntegrationRelayKeepsOutboxPendingOnPublishFailure | ok | 0.200 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/relay#TestIntegrationRelayRebuildsStreamFromSQLAfterDataLoss | ok | 0.020 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/relay#TestIntegrationRelayPreservesPerRunOrder | ok | 0.020 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/memory#TestIntegrationRedisCacheTTLAndOutageDegradesToSource | ok | 0.900 |

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

任务 ID：R0-03
对应缺口：GAP-02；M-13；P-03 基础
目标行为：outbox relay 的发布/重试/重投与记忆召回缓存的 TTL/降级在真实 Redis 7.2 上得到认证，并确认 Redis 不可用时 SQL 仍是权威。

### 环境

- `redis:7.2-alpine` 一次性容器，启动参数 `--appendonly yes`（仅测试）；`AGENTHUB_REDIS_TEST_ADDR` 由操作者显式 export。
- Redis 版本、AOF 状态与命令记录在 `redis-restart-check.txt`；地址与凭据不写入证据。
- 用例统一使用 DB 3（`redis.Options{DB: 3}`），避免与同一实例上其它包的用例互相干扰。

### 结果

- 命令：`bash scripts/run-integration-evidence.sh --test-id R0-03 --plan-task R0-03 --packages "./services/orchestrator/relay/... ./services/orchestrator/internal/memory/..." --run '^TestIntegration'`
- **6 个用例：6 PASS / 0 FAIL / 0 SKIP**（退出码 0）
- `REDIS_TEST_CONTAINER=... bash scripts/r0-03-redis-restart-check.sh` -> PASS（真实 `docker restart` 前后 `XLEN` 均为 1，AOF 保留）

| 场景 | 用例 | 结果 |
|---|---|---|
| publish 成功但 mark 前崩溃 → 重复投递，仍可按 `(runId, seq)` 去重 | `TestIntegrationRelayRedeliversWhenMarkIsLost` | PASS（2 次 pass 各 2 条，stream 4 条、去重后 2 个键） |
| Redis 发布失败 → outbox 保持 pending、只记首个 attempt、后续 seq 不越序 | `TestIntegrationRelayKeepsOutboxPendingOnPublishFailure` | PASS |
| stream 数据丢失（等价于无持久化重启）→ 由 SQL outbox 重建且顺序一致 | `TestIntegrationRelayRebuildsStreamFromSQLAfterDataLoss` | PASS（含 key 命名与 TTL ≤1h 契约断言） |
| 同一 run 事件顺序 | `TestIntegrationRelayPreservesPerRunOrder` | PASS（seq 1..5 严格递增） |
| 记忆召回缓存 epoch 失效（M-13） | `TestIntegrationRedisRecallCacheUsesEpoch` | PASS |
| 缓存 key TTL 上界 + Redis 不可达时回源 SQL | `TestIntegrationRedisCacheTTLAndOutageDegradesToSource` | PASS（TTL ≤5min 符合 `docs/contracts/redis-usage.md` §6） |

### 新增/改动文件

- `services/orchestrator/relay/redis_integration_test.go`（新增 4 个真实 Redis 集成用例 + 连接 helper）
- `services/orchestrator/relay/relay_test.go`（`fakeStore` 增加 `suppressMark`，模拟发布后、标记前崩溃窗口）
- `services/orchestrator/internal/memory/cache_redis_test.go`（新增 TTL 与停机降级用例）
- `scripts/r0-03-redis-restart-check.sh`（真实重启持久性检查，命令在容器内执行）

### 未改文件（确认）

- 未修改 `relay/relay.go`、记忆 controller/repository、Gateway、Frontend、compose、legacy 目录
- 未引入消费者或 consumer group（消费者语义按计划留给 R2）
- 未把 Redis 变成权威存储：用例断言的正是“丢数据后由 SQL 重建”

### 契约一致性

- stream key 断言为 `agenthub:test:<workspace>:run:<runId>:events`，与 `redis-usage.md` 和现有 `relay.streamKey` 实现一致。
- 缓存 TTL 断言不超过 5 分钟；事件 stream TTL 断言不超过 1 小时且有显式 `MAXLEN`（实现中 `maxLen=10000`、`Approx=true`）。
- 明确记录：当前 relay **没有消费者/consumer group/PEL**，因此本任务只认证发布侧与缓存侧；队列语义属 R2，未在此声称。

### 失败与修复过程（首轮）

- 首轮 1 FAIL：`TestIntegrationRelayKeepsOutboxPendingOnPublishFailure`。根因是用例 helper 用同一个“故意只读超时”的客户端做可达性探测，导致 setup 直接失败而非验证发布失败。修复：可达性用独立健康客户端探测。属于测试 helper 缺陷，未改动业务代码。

### 回归风险

- 无运行时行为变化；仅新增测试与脚本。`fakeStore` 的 `suppressMark` 默认 false，旧用例语义不变。

### 回滚方式

- `git revert` 本提交；无迁移、无配置变化。

### 交互覆盖（计划 §5.5）

| 交互 | 记录文件 | 条数 | 覆盖 |
|---|---|---|---|
| 用户操作 | - | 0 | not_applicable: 无浏览器参与 |
| API 请求/响应 | - | 0 | not_applicable: 本任务在 relay/memory 层，不经 HTTP handler |
| SSE 连接/事件/断线 | - | 0 | not_applicable: relay 只写 stream，当前无消费者 |
| HITL 请求/信号/Claim | - | 0 | not_applicable: 审批链路在 R0-02 认证 |
| 工具授权/结果 | - | 0 | not_applicable: 不涉及工具 |
| 记忆召回 | events.jsonl | 2（epoch 失效用例、TTL/降级用例） | yes（只记录用例级结果与缓存键数量断言，不记录记忆内容） |
| Artifact | - | 0 | not_applicable: 无制品 |
| MQ（publish/consume） | events.jsonl + logs/go-test.jsonl | 4（relay 用例） | yes（发布、重投、失败保持 pending、数据丢失后重建；消费侧明确未实现） |
| 多实例 | - | 0 | not_applicable: 单 relay 实例；双实例验收在 R0-06 |
| 中间件故障 | redis-restart-check.txt + 用例 | 2（发布失败、真实重启） | yes |
