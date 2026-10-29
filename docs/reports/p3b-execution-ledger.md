# P3b 阶段报告：执行账本、checkpoint 与预算接线（部分）

- 日期：2026-10-29
- 基线：`dev / 2e71954`（P2 真实 MySQL 门禁通过）
- 状态：**运行入口已接入 SQL 账本、计划版本、checkpoint 与模型预算；暂停/恢复与工具工单审批未完成**。

## 1. 交付

| 交付 | 路径 | 说明 |
|---|---|---|
| 执行存储接口 | `services/orchestrator/execution/store.go` | AcceptRun/AcquireLease/AppendState/SaveCheckpoint/SavePlan |
| MySQL 适配 | `services/orchestrator/execution/mysql.go` | 包装执行账本 Repository |
| 运行账本接线 | `services/orchestrator/httpapi/run_ledger.go` | 预算、requestId、租约、状态、checkpoint、plan revision |
| Handler 生命周期 | `services/orchestrator/httpapi/handler_run_stream.go` | planning → awaiting_approval → executing → completed/failed/cancelled；断连记 paused |
| 启动接线 | `services/orchestrator/cmd/orchestrator/main.go` | `ORCHESTRATOR_DATABASE_URL`/`DATABASE_URL` 存在时启用账本并跑迁移 |
| 模型预算 | `planner/planner.go`、`main_agent.go` | 模型调用/修复共享 run 预算；耗尽时停止 |
| 测试 | `run_ledger_test.go`、`run_ledger_mysql_test.go`、planner 预算测试 | 存储失败阻止执行、真实 MySQL 生命周期 |

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
```

## 3. 未完成

1. **暂停/恢复端点**未实现；断连只写 paused，不能从 checkpoint 恢复执行。
2. **计划审批表接线**：审批状态仍以 in-memory PendingPlan 为主，未写 `approvals`；revise 时旧审批未显式 supersede。
3. **工具工单审批**、Tool Broker 远程许可、审批消费与预算/派发同事务尚未接入 Handler（当前 Worker 尚未执行 Orchestrator 原生工具）。
4. **子 Agent/工具重试预算**未计入同一个 Budget；目前只有 MainAgent 模型/修复。
5. **取消**：用户取消写 cancelled，但远端 A2A 取消与终态一致性仍沿用旧逻辑；断连不取消 durable run 尚未实现。
6. **恢复校验**（revision/hash/expiry/fencing）没有 resume 入口，无法端到端证明。

## 4. 结论

可声明“运行入口的 SQL 生命周期、checkpoint、plan revision 与模型预算已接线并在真实 MySQL 验证”；不能声明 P3b 完成（审批/暂停恢复/工具许可缺失）。
