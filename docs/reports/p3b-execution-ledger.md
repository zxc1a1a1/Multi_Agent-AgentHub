# P3b 阶段报告：执行账本、checkpoint 与预算接线（部分）

- 日期：2026-10-29
- 基线：`dev / 2e71954`（P2 真实 MySQL 门禁通过）
- 状态：**运行入口已接入 SQL 账本、计划版本、checkpoint、模型预算、计划审批表与最小恢复入口；工具工单审批与已完成调用复用未完成**。

## 1. 交付

| 交付 | 路径 | 说明 |
|---|---|---|
| 执行存储接口 | `services/orchestrator/execution/store.go` | AcceptRun/AcquireLease/AppendState/SaveCheckpoint/SavePlan |
| MySQL 适配 | `services/orchestrator/execution/mysql.go` | 包装执行账本 Repository |
| 运行账本接线 | `services/orchestrator/httpapi/run_ledger.go` | 预算、requestId、租约、状态、checkpoint、plan revision |
| Handler 生命周期 | `services/orchestrator/httpapi/handler_run_stream.go` | planning → awaiting_approval → executing → completed/failed/cancelled；断连记 paused |
| 启动接线 | `services/orchestrator/cmd/orchestrator/main.go` | `ORCHESTRATOR_DATABASE_URL`/`DATABASE_URL` 存在时启用账本并跑迁移 |
| 模型预算 | `planner/planner.go`、`main_agent.go` | 模型调用/修复共享 run 预算；耗尽时停止 |
| 计划审批 | `approvals` 表 + `ResolveApproval` | 绑定 run/plan/revision；一次性消费；过期不可批准；revise supersede |
| 恢复入口 | `httpapi/resume.go`、`LoadResume` | paused→AcquireLease→恢复预算→校验计划/边界→executing→终态 |
| 测试 | `run_ledger_test.go`、`run_ledger_mysql_test.go`、planner 预算测试 | 存储失败阻止执行、真实 MySQL 生命周期/审批、恢复 |

## 2. 验证

```text
$ go test ./services/orchestrator/httpapi/... ./services/orchestrator/planner/... -count=1
PASS

$ AGENTHUB_MYSQL_TEST_DSN=... go test ./services/orchestrator/httpapi/ -run TestIntegrationRunLedgerLifecycleMySQL -v
PASS
- executions.status=completed, current_seq=3
- plan_revisions=1, checkpoints>=2
- execution_events seq 1..3，终态事件 run_finished
- outbox 与事件一一对应（3 条）
```

单元用例：

```text
TestRunLedgerPersistsConversationalLifecycle   PASS
TestRunLedgerStoreFailureBlocksExecution       PASS
TestMainAgentBudgetBlocksModelCalls            PASS
TestMainAgentRepairBudgetIsEnforced            PASS
TestIntegrationPlanApprovalLifecycle           PASS（真实 MySQL：批准幂等、关闭拒绝、过期不可批准）
TestRunLedgerPersistsPlanApproval              PASS（确认请求→approvals=approved）
TestResumePausedRunFromCheckpoint              PASS（paused→executing→completed，checkpoint 保存）
TestResumeRejectsNonPausedRun                  PASS（409）
```

## 3. 未完成

1. **恢复不重用已完成 task 结果**：当前从计划重新执行；checkpoint 只用于状态/预算，不是逐 task 游标。
2. **审批**：计划审批已入表并一次性消费；**工具工单审批**（toolCallId 绑定、预算/派发同事务）仍未接入实际工具执行。
3. **工具工单审批**、Tool Broker 远程许可、审批消费与预算/派发同事务尚未接入 Handler（当前 Worker 尚未执行 Orchestrator 原生工具）。
4. **子 Agent/工具重试预算**未计入同一个 Budget；目前只有 MainAgent 模型/修复。
5. **取消**：用户取消写 cancelled 并关闭审批；远端 A2A 取消与断连 durable run 仍未完成。
6. **恢复预算**：使用 checkpoint 的 usage + 默认 limits；run profile/原 limits 未持久化，不能证明跨配置恢复。
7. **恢复鉴权**：internal token 可调用 resume；没有更细的主体/审批授权。

## 4. 结论

可声明“SQL 生命周期、checkpoint、plan approval、模型预算与最小 resume 已接线并在真实 MySQL 验证”；不能声明 P3b 完成（已完成调用复用、工具许可、断连 durable run 缺失）。
