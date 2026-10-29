# P2 阶段报告：MySQL 主存储与执行账本（进行中）

- 日期：2026-10-29
- 基线：`dev / ae28a96`（P1 完成后）
- 状态：**部分完成（PARTIAL）**。执行账本与 Gateway MySQL Store 已落地；真实 MySQL 集成、Session MySQL 与 Orchestrator 运行接线未完成。

## 1. 本轮交付

| 交付 | 路径 | 说明 |
|---|---|---|
| Orchestrator 执行账本迁移 | `services/orchestrator/internal/persistence/mysql/migrations/001_execution_ledger.sql` | executions/task/plan/approvals/tool_calls/checkpoints/events/outbox/catalog |
| 执行账本 Repository | `services/orchestrator/internal/persistence/mysql/repository.go` | requestId 幂等、seq 分配、状态+事件+outbox 同事务、fencing、租约、checkpoint、审批一次性消费、工具 unknown |
| MySQL 连接/迁移 | `db.go`、`migrate.go` | 固定 UTC、连接池、嵌入迁移 |
| Gateway MySQL Store | `services/gateway/store/mysql.go` + `migrations/001_gateway.sql` | 会话/消息权威存储、请求幂等记录、运行视图/投影位置表 |
| Gateway 启动切换 | `services/gateway/cmd/gateway/main.go` | 默认 `mysql`；`memory`/`sqlite` 仅显式 demo/test；缺 DSN 启动失败 |
| 单元测试 | `repository_test.go`、`mysql_test.go` | sqlmock 覆盖幂等、seq、fencing、终态、审批过期/参数不匹配、消息写入 |
| 集成测试（门控） | `integration_test.go`（两模块） | 设置 `AGENTHUB_MYSQL_TEST_DSN` 才运行 |

## 2. 验证记录

```text
$ go build ./services/gateway/... ./services/orchestrator/...
exit=0

$ go test ./services/gateway/... ./services/orchestrator/... -count=1
PASS（各包均 ok；详见本轮输出）

$ go test ./services/orchestrator/internal/persistence/mysql/ -count=1 -v
PASS 9 个 sqlmock 用例
SKIP 5 个 real-MySQL 集成用例：AGENTHUB_MYSQL_TEST_DSN 未设置

$ go test ./services/gateway/store/... -count=1
PASS 4 个 sqlmock 用例
SKIP 2 个 real-MySQL 集成用例：AGENTHUB_MYSQL_TEST_DSN 未设置
```

## 3. 未执行/未通过项（不得视为 P2 完成）

1. **真实 MySQL 事务/唯一约束/并发测试未执行**：本机 Docker daemon 未运行，且未发现可用本地 `mysqld`；未启动外部服务，也未被授权启动 Docker Desktop。门控测试代码已存在，但 SKIP 不等于 PASS。
2. **会话重启后读历史**未在真实 MySQL 上验证；Gateway 启动已切到 MySQL Store，但没有运行中的实例做端到端验证。
3. **Orchestrator 运行未接入执行账本**：Repository 已存在，handler/Coordinator 尚未写入 executions/events/outbox（计划归入 P3b 整合）。
4. **Runtime Session MySQL 适配**未开始（`pkg/runtime/session` 仍是现有实现）。
5. **SQLite→MySQL 合成数据转换测试**未编写；真实数据导入留 P7。
6. **原始 SQLite 路径缺陷回归**（跨 run 状态污染、重复内容追加、空消息终态）尚未在新投影上补测试。

## 4. 结论

P2 门禁 **未通过**，当前只能声明“执行账本接口与 Gateway MySQL 主 Store 已实现并有单测”；不能声明真实 MySQL 事务/并发/重启恢复已验证。下一步：获得可用 MySQL 实例后执行门控集成测试，再完成 Session 适配与 Orchestrator 接线。
