# R0-02 R0-02 integration evidence

- task: R0-02
- testId: R0-02
- commit: 08c68042ff47731c968aa16a3c0da6d946bf7fb5 (dev)
- go: go version go1.26.5 windows/amd64
- started: 2026-11-02T09:04:40Z
- ended: 2026-11-02T09:05:22Z
- duration: 42s
- packages: ./services/orchestrator/... ./services/gateway/...
- run pattern: ^TestIntegration
- go test exit: 0

## results

| metric | count |
|---|---|
| total tests | 33 |
| PASS | 33 |
| FAIL | 0 |
| SKIP | 0 |
| incomplete | 0 |
| interaction rows | 0 |
| overall | PASS |

## per test

| test | status | elapsed(s) |
|---|---|---|
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql#TestIntegrationDuplicateRequestIsIdempotent | ok | 0.270 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql#TestIntegrationConcurrentSeqAndStaleFencing | ok | 0.860 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql#TestIntegrationCheckpointFencing | ok | 0.330 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql#TestIntegrationApprovalConsumedOnce | ok | 0.570 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql#TestIntegrationToolResultUnknown | ok | 0.570 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi#TestIntegrationMemoryAdminAPI | ok | 2.470 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql#TestIntegrationFailedStateChangeLeavesNoHalfResult | ok | 0.510 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql#TestIntegrationPlanApprovalLifecycle | ok | 0.610 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/memory#TestIntegrationRedisRecallCacheUsesEpoch | ok | 1.670 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql#TestIntegrationMemoryLifecycle | ok | 1.560 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi#TestIntegrationRunInjectsAuthorizedMemory | ok | 0.910 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/internal/persistence/mysql#TestIntegrationMemoryScopeIsolation | ok | 1.070 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi#TestIntegrationRunLedgerLifecycleMySQL | ok | 1.360 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi#TestIntegrationOutboxRelayMySQLRedis | ok | 0.780 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi#TestIntegrationLeaseRenewalAndStaleFencing | ok | 0.560 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi#TestIntegrationRunSignals | ok | 0.340 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi#TestIntegrationLeaseReleaseAllowsImmediateTakeover | ok | 0.350 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi#TestIntegrationRunTaskRefsAndCancel | ok | 0.300 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi#TestIntegrationHITLConfirmIdempotency | ok | 0.430 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi#TestIntegrationRouteDecisions | ok | 0.110 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi#TestIntegrationRouteMetrics | ok | 0.180 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/orchestrator/httpapi#TestIntegrationTwoOrchestratorsHITLMySQL | ok | 2.080 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/migration#TestIntegrationSQLiteToMySQLMigration | ok | 0.630 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/projection#TestIntegrationWorkerRecoversAgentTurns | ok | 1.630 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/projection#TestIntegrationTwoWorkersDoNotDuplicateTurns | ok | 1.320 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store#TestIntegrationSQLiteToMySQLFieldMapping | ok | 0.320 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store#TestIntegrationMySQLProjectionOffsetIsIdempotent | ok | 0.670 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store#TestIntegrationMySQLStoreCRUD | ok | 0.670 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store#TestIntegrationRecordChatRequestIdempotent | ok | 0.390 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store#TestIntegrationMySQLStoreSurvivesReopen | ok | 0.580 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store#TestIntegrationConversationIsolation | ok | 0.640 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store#TestIntegrationPublicEventJournal | ok | 0.510 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store#TestIntegrationMySQLConversationParticipants | ok | 0.740 |

## skipped tests (must be explained)

- none

## failed tests

- none

## interaction coverage (plan §5.5)

| interaction | record file | rows | coverage |
|---|---|---|---|
| test case result | events.jsonl | 33 | yes |
| middleware interactions | interactions.jsonl | 0 | not_applicable: no test in this selection emitted interaction rows |
| user actions | - | 0 | not_applicable: gate script runs no browser code |
| SSE / AG-UI events | - | 0 | not_applicable: no SSE client in this selection |

## redaction

- DSN and Redis address values are replaced with `<redacted-dsn>` / `<redacted-addr>` in all logs.
- Environment values are read only through `ENVIRON` inside awk, so they never reach argv or evidence files.

## 执行记录（计划 §9）

任务 ID：R0-02（业务修复拆分为 R0-02a）
对应缺口：GAP-01；测试矩阵 F-03、F-04、F-05、F-07、F-13
目标行为：让运行账本、HITL 审批、取消信号、outbox、投影偏移、route_decisions 在真实 MySQL 8.4 上被认证，而不是长期停留在 SKIP。

### 认证结果

- 命令：`bash scripts/run-integration-evidence.sh --test-id R0-02 --plan-task R0-02 --packages "./services/orchestrator/... ./services/gateway/..." --run '^TestIntegration' --timeout 25m`
- 环境：本机 Docker 一次性容器 `mysql:8.4`（专用库 `agenthub_test`，独立端口）+ `redis:7.2-alpine`（AOF 开启）；凭据仅存在于容器与调用 shell 的环境变量，未写入仓库。
- 结果：**33 个 TestIntegration 用例，33 PASS，0 FAIL，0 SKIP**（门禁退出码 0）。
- 场景映射：F-13 重复 requestId → `TestIntegrationDuplicateRequestIsIdempotent`；F-03/F-04 审批与 revision → `TestIntegrationPlanApprovalLifecycle`、`TestIntegrationHITLConfirmIdempotency`；F-05 取消 → `TestIntegrationRunTaskRefsAndCancel`；F-07 工具账本 → `TestIntegrationApprovalConsumedOnce`、`TestIntegrationToolResultUnknown`；outbox 同事务与投影幂等 → `TestIntegrationFailedStateChangeLeavesNoHalfResult`、`TestIntegrationMySQLProjectionOffsetIsIdempotent`；跨实例 signal SQL 语义 → `TestIntegrationRunSignals`、`TestIntegrationTwoOrchestratorsHITLMySQL`。
- MySQL 重启后终态/审批仍可读：`restart-check.txt`（写入 `completed|3|approved` → `docker restart` → 读回同一行）-> PASS。
- 迁移重复执行幂等：每个用例都调用 `RunMigrations`，整轮 33 个用例在同一库上重复执行 3 轮无报错 -> PASS。

### 真实环境暴露的缺陷（本机长期 SKIP 因而从未验证过）

诊断跑证据见 `../R0-02-baseline/summary.md`：首轮真实 MySQL 下 33 个用例中 6 个 FAIL、0 SKIP。

1. **R0-02a 业务缺陷：HITL 首次 confirm 永远拿不到认领权。** `BeginConfirm` 用 `INSERT IGNORE` 写入 `processing` 行后再读回，只用 `created_at` 两分钟阈值判断“是否别人在处理”，无法区分“是我刚插入的”。首次请求即返回 `ErrConfirmInFlight`，对外表现为 409 `CONFIRM_IN_FLIGHT`，真实 MySQL 下审批主链路不可用（跨实例用例同样卡死）。修复：用 `RowsAffected==1` 判定本次调用即为建行者并直接认领；已存在且未过期的 `processing` 才判为在途。回归证据：修复前 `TestIntegrationHITLConfirmIdempotency`、`TestIntegrationTwoOrchestratorsHITLMySQL` FAIL，修复后 PASS。
2. **R0-02a 业务缺陷：重复 signal 丢失原 id。** `AppendRunSignal` 命中唯一键时返回 `id=0`，调用方无法把重放当作同一信号。修复：重放查询并返回原行 id（与 `AcceptRun` 的 dedup 语义一致）。回归证据：`TestIntegrationRunSignals` 由 FAIL 转 PASS。
3. **测试缺陷（已修）：relay 用例不带租约写状态。** `AppendStateChange` 设计上要求持有租约（`FencingVersion<=0` 直接 `ErrStaleFencing`），用例传 0。修复：先 `AcquireLease` 再用返回的 fencing 写入，与运行时一致。
4. **测试缺陷（已修）：租约续期用例自相矛盾。** 先以 `time.Minute` 续期再 sleep 150ms 期望过期接管；服务端正确拒绝。修复：续期 TTL 改为 40ms，断言“过期后可接管 + fencing 递增 + 旧 owner 续期被拒”不变。
5. **测试隔离缺陷（已修）：`TestIntegrationRouteMetrics` 断言全表聚合。** `RouteDecisionMetrics` 按表聚合，其他用例留下的 `route_decisions` 使 `TotalRuns` 变成 4。修复：在专用测试库内先清空该表再断言（未降低断言）。
6. **真实 MySQL 与 fake 的差异（已修）：JSON 列读回被格式化。** `hitl_confirmations.response_json` 是 JSON 列，MySQL 读回为 `{"status": "accepted"}`，用例做逐字节比较而失败。修复：改为解析 JSON 后断言字段值（仍断言 202 与 status=accepted，不弱化契约）。

### 改动文件（实际）

- `services/orchestrator/internal/persistence/mysql/repository.go`（R0-02a：`BeginConfirm`、`AppendRunSignal`）
- `services/orchestrator/httpapi/run_ledger_mysql_test.go`（用例 3–6 的修复与隔离）
- `scripts/r0-02-mysql-restart-check.sh`（重启持久性检查，凭据只从容器环境读取）
- 本证据目录与 `../R0-02-baseline/`

### 未改文件（确认）

- 未修改 Gateway、Frontend、Runtime/AG-UI、Docker/compose、legacy `server/`、`agents/`
- 未修改迁移文件与任何 `.sql`；无表结构变更
- 未修改 CI workflow；未读取或写入任何 `.env`

### 契约/数据变更

- 无 schema 变更。行为澄清：HITL confirm 首次请求可认领（原来必然 409）；signal 重放返回原 id（原来返回 0）。

### 回归风险

- `BeginConfirm` 的“两分钟 abandonment 阈值”仅作用于非建行者，逻辑不变；跨实例并发认领语义由 `TestIntegrationTwoOrchestratorsHITLMySQL` 覆盖。
- `AppendRunSignal` 重放路径多一次按唯一键的 SELECT，仅在重复提交时发生。

### 回滚方式

- `git revert` R0-02a 与 R0-02 两个提交即可回到原状态；无数据迁移，无需清理表结构。

### 交互覆盖（计划 §5.5）

| 交互 | 记录文件 | 条数 | 覆盖 |
|---|---|---|---|
| 用户操作 | - | 0 | not_applicable: 本任务是服务端集成门禁，无浏览器参与 |
| API 请求/响应 | events.jsonl | 33 用例（含 `TestIntegrationHITLConfirmIdempotency`、`TestIntegrationTwoOrchestratorsHITLMySQL` 的 HTTP handler 往返） | yes（用例级结果；HTTP 状态在被测进程内断言） |
| SSE 连接/事件/断线 | - | 0 | not_applicable: 该选择器不含 SSE 客户端用例；跨实例用例走 `/runs/stream` handler 但不断言事件流 |
| HITL 请求/信号/Claim | events.jsonl | 3（`TestIntegrationHITLConfirmIdempotency`、`TestIntegrationRunSignals`、`TestIntegrationTwoOrchestratorsHITLMySQL`） | yes（请求、认领、重放、跨实例 confirm 均有对应用例行） |
| 工具授权/结果 | events.jsonl | 3（`ApprovalConsumedOnce`、`ToolResultUnknown`、`PlanApprovalLifecycle`） | yes |
| 记忆召回 | events.jsonl | 5（memory 生命周期/scope 隔离/缓存 epoch） | yes（属 R1-01 深入范围） |
| Artifact | - | 0 | not_applicable: 本选择器不产生制品 |
| MQ | - | 0 | not_applicable: R2 未实现 |
| 多实例 | events.jsonl | 1（`TestIntegrationTwoOrchestratorsHITLMySQL`，实例角色 orchestrator-a 持有/orchestrator-b 确认） | yes（进程内双 Server 实例；真实双进程验收在 R0-06） |
| MySQL 事务/幂等 | events.jsonl + restart-check.txt | 全量 + 1 次真实重启 | yes |

说明：本任务的 `events.jsonl` 由门禁脚本按用例产出行级记录（计划 §4.1 R0-02 只要求“每个场景有 testId、命令、结果、耗时”）；更细粒度的 §5.2 交互行需要在被测代码里加事件导出，作为 R0-06/R1-01 的前置在对应任务中补齐，不在此伪造。
