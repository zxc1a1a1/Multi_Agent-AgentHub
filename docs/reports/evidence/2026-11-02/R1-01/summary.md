# R1-01 R1-01 integration evidence

- task: R1-01
- testId: R1-01
- commit: 0271a3803de4d83361ef12d755010510cd48b40e (dev)
- go: go version go1.26.5 windows/amd64
- started: 2026-11-02T09:22:08Z
- ended: 2026-11-02T09:22:24Z
- duration: 16s
- packages: ./services/orchestrator/internal/memory/... ./services/orchestrator/internal/persistence/mysql/... ./services/orchestrator/httpapi/...
- run pattern: ^TestIntegrationMemory|^TestIntegrationRedis|^TestIntegrationRunInjectsAuthorizedMemory
- go test exit: 0

## results

| metric | count |
|---|---|
| total tests | 11 |
| PASS | 11 |
| FAIL | 0 |
| SKIP | 0 |
| incomplete | 0 |
| interaction rows | 0 |
| overall | PASS |

## per test

| test | status | elapsed(s) |
|---|---|---|
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/memory#TestIntegrationRedisRecallCacheUsesEpoch | ok | 1.690 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql#TestIntegrationMemoryLifecycle | ok | 1.600 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/memory#TestIntegrationRedisCacheTTLAndOutageDegradesToSource | ok | 0.920 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql#TestIntegrationMemoryScopeIsolation | ok | 1.230 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/memory#TestIntegrationMemoryCandidateConfirmAndRecall | ok | 2.310 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi#TestIntegrationMemoryAdminAPI | ok | 2.410 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/memory#TestIntegrationMemoryExpiredCandidateIsRejected | ok | 0.730 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi#TestIntegrationRunInjectsAuthorizedMemory | ok | 0.880 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/memory#TestIntegrationMemoryScopeIsolationOnMySQL | ok | 1.250 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/memory#TestIntegrationMemoryQuotaAndBudgets | ok | 1.780 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/memory#TestIntegrationMemoryForgetScopeRevokesEveryEntry | ok | 2.670 |

## skipped tests (must be explained)

- none

## failed tests

- none

## interaction coverage (plan §5.5)

| interaction | record file | rows | coverage |
|---|---|---|---|
| test case result | events.jsonl | 11 | yes |
| middleware interactions | interactions.jsonl | 0 | not_applicable: no test in this selection emitted interaction rows |
| user actions | - | 0 | not_applicable: gate script runs no browser code |
| SSE / AG-UI events | - | 0 | not_applicable: no SSE client in this selection |

## redaction

- DSN and Redis address values are replaced with `<redacted-dsn>` / `<redacted-addr>` in all logs.
- Environment values are read only through `ENVIRON` inside awk, so they never reach argv or evidence files.

## 执行记录（计划 §9）

任务 ID：R1-01
对应缺口：M-08、M-10、M-11、M-13、M-14（GAP-01/GAP-02 记忆部分）
目标行为：长期记忆的 scope 隔离、candidate 不可召回、confirm/revision CAS、forget/forget-scope、epoch 缓存失效、配额与注入预算，全部在真实 MySQL + Redis 上成立，而不是只在 fake backend 单测里成立。

### 结果

- 命令：`bash scripts/run-integration-evidence.sh --test-id R1-01 --plan-task R1-01 --packages "./services/orchestrator/internal/memory/... ./services/orchestrator/internal/persistence/mysql/... ./services/orchestrator/httpapi/..." --run '^TestIntegrationMemory|^TestIntegrationRedis|^TestIntegrationRunInjectsAuthorizedMemory'`
- **11 个用例：11 PASS / 0 FAIL / 0 SKIP**（退出码 0）

| 场景 | 用例 | 结果 |
|---|---|---|
| M-08 candidate 不可召回 → confirm 后可召回 | `TestIntegrationMemoryCandidateConfirmAndRecall` | PASS |
| confirm CAS 重放冲突（`ErrMemoryRevisionConflict`）、过期 revision 不得覆盖内容 | 同上 | PASS |
| M-11 forget 停止召回且写入 `purge_status` 审计行 | 同上 | PASS（当前值为 `pending`，物理清理属 R1-05） |
| M-10 过期 candidate 确认被拒（`ErrCandidateStale`） | `TestIntegrationMemoryExpiredCandidateIsRejected` | PASS |
| M-14 跨 principal / 跨 workspace 零召回，`Get` 不越界 | `TestIntegrationMemoryScopeIsolationOnMySQL`、`TestIntegrationMemoryScopeIsolation` | PASS |
| M-09/M-12 `MaxActivePerScope` 超限 `ErrQuota`、`MaxEntryBytes` 超限 `ErrInvalid`、`MaxInjectedEntries` 截断注入 | `TestIntegrationMemoryQuotaAndBudgets` | PASS |
| forget-scope 批量 revoke 且保留可审计墓碑（`status=revoked`） | `TestIntegrationMemoryForgetScopeRevokesEveryEntry` | PASS |
| M-13 Redis epoch 失效、缓存 key TTL ≤5min、Redis 不可达回源 SQL | `TestIntegrationRedisRecallCacheUsesEpoch`、`TestIntegrationRedisCacheTTLAndOutageDegradesToSource` | PASS |
| 真实库上的既有记忆回归 | `TestIntegrationMemoryLifecycle`、`TestIntegrationMemoryAdminAPI`、`TestIntegrationRunInjectsAuthorizedMemory` | PASS |

### 真实库上发现的语义（此前只有 fake 单测，容易误判）

- `Confirm` **不推进** `current_revision`：candidate 确认后 revision 仍为 1，因此 `Update(expectedRevision=1)` 是合法路径，第二次同 revision 才是冲突。fake 单测里 “确认后 stale update 冲突” 的直觉在真实库上不成立，用例已按真实语义写成“更新成功后重放旧 revision 必须冲突，且内容保持第二次写入值”。
- `memory_mutations` 主键是 `mutation_id`，没有自增 `id` 列；审计查询必须按 `operation` 过滤，不能用 `ORDER BY id`。
- `Policy.Validate` 要求 `MaxCoreEntries <= MaxInjectedEntries`，收紧注入预算时必须同时收紧 core 上限，否则策略本身被拒（与 R1-07 相关：`MaxCoreEntries` 目前只被校验，召回未使用）。
- forget 只写 `purge_status='pending'`，内容/revision 仍在库中：用例显式断言这一点，避免把“已停止召回”误说成“已物理删除”。

### 改动文件（实际）

- `services/orchestrator/internal/memory/controller_mysql_test.go`（新增：真实 MySQL 记忆控制面用例 + 专用 helper 与逐 principal 清理）
- 本证据目录

### 未改文件（确认）

- 未修改 `controller.go`、`memory_repository.go`、迁移文件、httpapi handler、Gateway、Frontend、legacy 目录
- 没有业务代码改动：本轮真实环境暴露的都是测试自身假设或文档语义问题，未发现记忆侧产品缺陷

### 契约/数据变更

- 无。使用既有 `002_memory.sql`；`RunMigrations` 在每个用例前重复执行保持幂等。

### 日志与脱敏

- 用例只在断言失败时打印条目 ID 与数量；不断言也不记录记忆正文，记忆内容仅存在于测试专用库的临时行，用例结束按 principal 清理。
- 证据目录不含 DSN/Redis 地址值（脚本内置扫描，命中即退出 1）。

### 回归风险

- 仅新增测试；无运行时行为变化。测试数据按 `r1-01-<ns>` principal 隔离并自清理，不污染共享库。

### 回滚方式

- `git revert` 本提交即可；无迁移、无配置项。

### 交互覆盖（计划 §5.5）

| 交互 | 记录文件 | 条数 | 覆盖 |
|---|---|---|---|
| 用户操作 | - | 0 | not_applicable: 无浏览器；MemoryPanel 侧在 R1-03 覆盖 |
| API 请求/响应 | events.jsonl | 2（`TestIntegrationMemoryAdminAPI`、`TestIntegrationRunInjectsAuthorizedMemory` 走 HTTP handler） | yes（用例级结果；状态码在 handler 内断言） |
| SSE 连接/事件/断线 | - | 0 | not_applicable: 本用例集不建立 SSE 订阅 |
| HITL 请求/信号/Claim | - | 0 | not_applicable: 记忆路径无审批信号 |
| 工具授权/结果 | - | 0 | not_applicable: 不涉及工具调用 |
| 记忆召回 | events.jsonl + 用例断言 | 11（每个用例都含 propose/confirm/recall/forget 交互，断言只记录条数与 memoryId） | yes |
| Artifact | - | 0 | not_applicable: 无制品 |
| MQ | - | 0 | not_applicable: R2 未实现 |
| 多实例 | - | 0 | not_applicable: 单实例记忆控制面；多实例验收在 R0-06 |
| 缓存/中间件 | events.jsonl | 2（epoch 失效、Redis 不可达降级） | yes |
