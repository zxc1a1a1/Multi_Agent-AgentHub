# R0-06 多副本运行时认证

- commit: a083e4df78886e68370c7719e607b1d3bf3a4e9d
- started: 2026-11-02T11:14:39Z / ended: 2026-11-02T11:20:11Z
- result: **FAIL**（12 项断言，4 项失败）
- 实例：orchestrator-a=127.0.0.1:19080/grpc 19090，orchestrator-b=:19081/19091，gateway-a=:19100→A，gateway-b=:19101→B
- 共享中间件：同一 MySQL（`agenthub_test`，经 agenthub-r0-mysql 容器查询）与同一 Redis（db 0），子 Agent 复用运行中的 compose 栈
- provider：`scripts/fake-llm-server.mjs`（测试专用），因此本任务认证的是编排/投影/租约语义，不是模型质量

## 断言结果

| 场景 | 结果 | 证据 |
|---|---|---|
| build orchestrator + gateway binaries | PASS |  |
| four processes up (2 orchestrator + 2 gateway) | PASS | A/B on 19100/19101 |
| F-06 cross-instance confirm executes exactly once | PASS | holder=LAPTOP-J2Q85T28-33660 terminals=1 sql_run_finished=1 dup=200 |
| F-06 signal consumed once in SQL | PASS | total=1 |
| F-06 both gateways project identical history | PASS | roles=2 |
| F-05 cross-instance cancel stops the run once | PASS | status=cancelled cancels=0 refs=0 |
| F-05 no dangling task refs | PASS |  |
| F-10 surviving gateway completes without duplicates | FAIL | approve=200 status=completed finish=1 user=1 assistant=0; see logs/f10-diagnostics.txt |
| F-11 child agent blocked to keep the run executing | PASS | agenthub-r004-code-agent-new-1 paused |
| F-11 run parked as paused on client disconnect | FAIL | status=executing; resume only accepts paused |
| F-11 lease taken over by the surviving instance | FAIL | no verified takeover within 90s; lease=LAPTOP-J2Q85T28-33660->LAPTOP-J2Q85T28-33660 status=executing paused=executing |
| F-11 takeover produces exactly one terminal | FAIL | run_finished rows=0 lease=LAPTOP-J2Q85T28-33660->LAPTOP-J2Q85T28-33660 status=executing paused=executing |

## 失败详情

- F-10 surviving gateway completes without duplicates: approve=200 status=completed finish=1 user=1 assistant=0; see logs/f10-diagnostics.txt
- F-11 run parked as paused on client disconnect: status=executing; resume only accepts paused
- F-11 lease taken over by the surviving instance: no verified takeover within 90s; lease=LAPTOP-J2Q85T28-33660->LAPTOP-J2Q85T28-33660 status=executing paused=executing
- F-11 takeover produces exactly one terminal: run_finished rows=0 lease=LAPTOP-J2Q85T28-33660->LAPTOP-J2Q85T28-33660 status=executing paused=executing

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

## 执行记录（计划 §9）

任务 ID：R0-06
对应缺口：GAP-05；F-06、F-08、F-10、F-11
**任务结论：FAIL（不是“部分通过即通过”）。** 计划硬规则要求“不得把测试改成单实例、不得用内存 store 代替 MySQL、不得退化为单实例后写 PASS”。本任务如实记录：F-06 与 F-05 认证通过，F-10 与 F-11 暴露真实缺陷并拆出子任务。

### 实际拓扑（真实四进程，不是共享内存 store）

- `orchestrator-a` = 本机进程 `:19080` / gRPC `:19090`；`orchestrator-b` = `:19081` / `:19091`
- `gateway-a` = `:19100` → 只指向 A；`gateway-b` = `19101` → 只指向 B
- 共享同一 MySQL 8.4（`agenthub_test`）与同一 Redis 7.2；10 个 child agent 复用运行中的 compose 栈
- PID/端口见 `logs/instances.txt`；跨实例身份取自 MySQL `executions.lease_owner`（形如 `主机名-PID`），不是编造字段
- provider 为 `scripts/fake-llm-server.mjs`（测试专用假模型），因此认证的是编排/投影/租约语义，不含模型质量

### 通过的断言

| 断言 | 证据 |
|---|---|
| F-06 跨实例 confirm 且只执行一次 | `logs/f06-gateway-a-stream.txt`：`RUN_FINISHED` 恰好 1 条；SQL `execution_events` 中 `run_finished` 行数 = 1；重复同 idempotencyKey 的 confirm 返回 200 但不再产生第二个终态 |
| F-06 信号只被消费一次 | SQL `run_signals` total=1、consumed=1（`events.jsonl` 的 `hitl_confirm` 行带 instanceId/role） |
| F-06 两个 Gateway 投影结果一致 | 同一会话在 A、B 上读到的消息条数一致（roles=2） |
| F-05 跨实例取消待审批群聊计划 | `executions.status=cancelled`、`run_task_refs` 归零、`run_finished` 恰好 1 条，且无 `RUN_ERROR` |

### 失败的断言与根因（真实缺陷，非测试写法问题）

1. **F-10 存活 Gateway 不接管投影**（FAIL）
   - 现象：kill `gateway-a` 后经 `gateway-b` 审批，SQL 侧运行完成（`status=completed`，journal 4 行，`run_finished` 恰好 1 行），
     但 `gateway-b` 的 `/api/conversations/{id}/messages` 在 40 秒轮询内始终只有 user 消息（`assistant=0`）。
   - 证据：`logs/f10-diagnostics.txt`（gateway-b 日志显示 “gateway projection worker started” 但对该 run 无产出；
     `event_projection_offsets` / `run_views` 查不到该 run 的行）。
   - 方向：Gateway 投影 worker 疑似只跟踪“本实例发起/正在流式”的 run，没有按 SQL journal 做全量尾随。
     这使多副本下“谁的客户端连着谁”决定用户能否看到回复，属可用性缺陷。
   - 拆出子任务 **R0-06a**：投影 worker 需以 SQL journal 为准尾随所有活跃 run（幂等由 offset 保证），
     验收标准即本用例：kill 提供 SSE 的 Gateway 后，另一实例必须在有界时间内产出同样的消息，且不重复。
2. **F-11 持有租约的 Orchestrator 崩溃后无人接管**（FAIL）
   - 现象：先 `docker pause` 子 Agent 让 run 真正处于执行中，再断开 SSE 客户端；`executions.status` 始终是 `executing`（未变成 `paused`），
     kill 掉持有租约的 `orchestrator-a` 后，在 90 秒窗口内 `lease_owner` 未变、无新终态；对 B 调
     `/internal/orchestrator/runs/{id}/resume` 返回 `{"error":"run is not paused","status":"executing"}`（`logs/f11-resume-stream.txt`）。
   - 根因链：`resume` 只接受 `status=paused`；`paused` 只在部分断连路径写入（`handler_run_stream.go:790`），执行中被阻塞时不会落 `paused`；
     生产默认 `leaseTTL = 15 分钟`（`httpapi/server.go:284`）且崩溃不会主动让租约失效；没有任何扫描器去接管“租约过期/持有者已死”的 run。
     结果：单实例崩溃 = 该 run 永久悬挂，只能等 TTL 且仍无人认领。
   - 本任务为使该场景在可接受时间内可测，新增了租约覆写（见下），把默认 15 分钟缩到 20 秒，缺陷仍然复现 → 证明这不是等待时间不够的问题。
   - 拆出子任务 **R0-06b**：需要（a）执行期断连也能落到可恢复状态（或允许接管 `executing` 且租约已过期的 run），
     （b）一个 orphan 扫描/接管循环（带 fencing 校验，禁止双执行），验收标准：kill 持有者后另一实例在 `leaseTTL + 一个轮询周期` 内接管且 `run_finished` 恰好 1 条。

### 为可测性新增的代码（本提交内）

- `services/orchestrator/httpapi/server.go`：新增 `WithLeaseTTL(ttl)`，同时按比例设 `leaseRenewInterval = ttl/3`。
- `services/orchestrator/cmd/orchestrator/main.go`：读取 `ORCHESTRATOR_LEASE_TTL`（Go duration），非法值只记日志并忽略。
- **默认行为不变**：不设置该环境变量时仍是 15 分钟；这是本任务唯一的运行时改动，且只是把已硬编码的值变成可配置。

### 未做（不声称）

- 未验证子 Agent 侧远端 A2A task 是否真被取消（F-08 只认证 SQL 侧）；未认证 SSE resume（F-09，归 R5-02）。
- 未在 Docker 内起多副本（本任务用本机进程 + compose 内的 child agent）；compose 侧多副本与 `--scale` 语义留待 R0-06a/b 修好后一并验收。

### 改动文件（实际）

- `scripts/run-multi-replica-smoke.sh`（新增）
- `services/orchestrator/httpapi/server.go`、`services/orchestrator/cmd/orchestrator/main.go`（租约覆写）
- `docs/reports/evidence/2026-11-02/R0-06/`（events.jsonl、timeline.jsonl、logs/*、本记录）

### 回归风险

- 运行时仅新增可选环境变量；未设置时行为不变，`go test ./...`（orchestrator 全量）PASS。
- 脚本自身幂等：每次新建 run/conversation id，进程与容器句柄在 trap 中清理。

### 回滚方式

- `git revert` 本提交：删除覆写选项与脚本，恢复硬编码 15 分钟租约；无数据变更。

### 交互覆盖补充（§5.5 之外必须说明的两点）

- `instanceId` / `role` 来自 SQL `executions.lease_owner` 与脚本进程标签，逐条写在 `events.jsonl`、`logs/timeline.jsonl`；缺失即视为证据不完整。
- 计划要求的“时间线 JSONL 记录 start/confirm/cancel/kill/recover/terminal”见 `logs/timeline.jsonl`，其中 kill 事件状态为 `error`，
  recover 未发生（这正是 F-11 FAIL 的事实），没有伪造恢复记录。
