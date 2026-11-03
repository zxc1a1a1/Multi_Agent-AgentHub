# R0-06 多副本运行时认证

- commit: 0678163a847474edbbe04a810f0bd3c13b726a6d
- started: 2026-11-03T03:26:59Z / ended: 2026-11-03T03:28:13Z
- result: **PASS**（13 项断言，0 项失败）
- 实例：orchestrator-a=127.0.0.1:19080/grpc 19090，orchestrator-b=:19081/19091，gateway-a=:19100→A，gateway-b=:19101→B
- 共享中间件：同一 MySQL（`agenthub_test`，经 agenthub-r006-mysql 容器查询）与同一 Redis（db 0），子 Agent 复用运行中的 compose 栈
- provider：`scripts/fake-llm-server.mjs`（测试专用），因此本任务认证的是编排/投影/租约语义，不是模型质量

## 断言结果

| 场景 | 结果 | 证据 |
|---|---|---|
| build orchestrator + gateway binaries | PASS |  |
| four processes up (2 orchestrator + 2 gateway) | PASS | A/B on 19100/19101 |
| F-06 cross-instance confirm executes exactly once | PASS | holder=LAPTOP-J2Q85T28-31532 terminals=1 sql_run_finished=1 dup=200 |
| F-06 signal consumed once in SQL | PASS | total=1 |
| F-06 both gateways project identical history | PASS | roles=2 |
| F-05 cross-instance cancel stops the run once | PASS | status=cancelled cancels=0 refs=0 |
| F-05 no dangling task refs | PASS |  |
| F-10 gateway-a restarted without projection | PASS | worker=0 live=0 |
| F-10 surviving gateway completes without duplicates | PASS | status=completed finish=1 user=1 assistant=1 events=4 |
| F-11 child agent blocked to keep the run executing | PASS | agenthub-r004-code-agent-new-1 paused |
| F-11 subscriber loss leaves the durable run executing | PASS | owner=LAPTOP-J2Q85T28-31532 |
| F-11 lease taken over by the surviving instance | PASS | lease=LAPTOP-J2Q85T28-31532->none status=completed fencing=2->2 parked=executing |
| F-11 takeover produces exactly one terminal | PASS | status=completed |

## 交互覆盖（计划 §5.5）

| 交互 | 记录文件 | 条数 | 覆盖 |
|---|---|---|---|
| 用户操作 | - | 0 | not_applicable: 进程级冒烟，无浏览器；真实浏览器矩阵在 R5-04 |
| API 请求/响应 | events.jsonl | 2 | yes |
| SSE 连接/事件 | logs/f06-gateway-a-stream.txt 等 | 0 条连接记录 | yes（每个场景一份流文件，含事件类型） |
| HITL 请求/信号/Claim | events.jsonl + timeline.jsonl | 2 | yes（每条带 instanceId/role） |
| 工具授权/结果 | - | 0 | not_applicable: 假 LLM 计划不触发写工具审批 |
| 记忆召回 | - | 0 | not_applicable: MEMORY_MODE=session_only |
| Artifact | - | 0 | not_applicable: 本场景不产制品；MinIO 跨实例读在 R0-07 |
| MQ | - | 0 | not_applicable: R2 未接入运行链路 |
| 多实例 | events.jsonl + timeline.jsonl | 7 | yes（lease_owner 直接取自 MySQL executions 行） |
| 故障注入 | timeline.jsonl | 2 | yes（kill gateway-a、kill lease-holding orchestrator） |

## 已知边界

- F-08（取消传播到真实 A2A 任务）只能证明 SQL 侧 `run_cancels` 落库与 `run_task_refs` 归零；
  子 Agent 侧是否真的取消了远端 task，需要在日志里按 runId 追证据，属 R5-02 混沌验收范围，本任务不声称已认证。
- F-09（SSE 断线后 events/status → resume）按计划归 R5-02，这里没有认证。

## 修复项（本任务内）

1. **迁移竞态**：两个 Orchestrator 同时启动会同时执行迁移，后插入 `schema_migrations` 的实例报
   `Duplicate entry '001_execution_ledger.sql'`，进程退出。修复为 MySQL 命名锁
   `GET_LOCK('agenthub_schema_migrations')` 串行化，Gateway 迁移使用同一把锁；记录插入改为
   `ON DUPLICATE KEY UPDATE`。
   回归测试：`TestIntegrationConcurrentMigrationsSerialize`（真实 MySQL，持锁期间不得出现
   `schema_migrations` 表）。变异验证：把 `GET_LOCK` 替换为 `SELECT 1` 后该测试 FAIL
   （`RunMigrations wrote schema state while another replica held the lock`）；恢复后 PASS。
2. **F-10 投影接管**：新增测试/运维开关 `GATEWAY_PROJECTION_WORKER`、`GATEWAY_LIVE_PROJECTION`
   （默认 1）。Gateway A 关闭两路投影、只写 public journal；Gateway B 的 recovery worker 完成
   投影，得到 user=1、assistant=1、`run_finished=1`，无重复。
3. **F-11 租约接管**：`/internal/orchestrator/runs/{id}/resume` 接受持有者崩溃后停留在
   `executing` 的运行；`AcquireLease` 仍是唯一裁决，有效租约返回 409，因此不会双执行。
   脚本在租约 TTL 过期后 resume，以 owner 变更/终态/`run_finished` 任一作为接管证据。
   gRPC 订阅模型下，订阅者断开不会取消 durable run（`context.WithoutCancel`），因此本场景
   记录为 “subscriber loss leaves the durable run executing”，而不是伪造 paused。

## 验证命令（补充）

```bash
AGENTHUB_MYSQL_TEST_DSN=<test dsn> go test ./services/orchestrator/internal/persistence/mysql/ \
  -run TestIntegrationConcurrentMigrationsSerialize -count=1   # PASS（变异后 FAIL）
go test ./services/gateway/... -count=1                          # PASS
go test ./services/orchestrator/httpapi/ ./services/orchestrator/internal/persistence/mysql/ -count=1  # PASS
```

