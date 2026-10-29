# P2 阶段报告：MySQL 主存储与执行账本

- 日期：2026-10-29
- 基线：`dev / f978cf5`（P3c 部分完成后）
- 状态：**存储层与真实 MySQL 门禁通过；跨服务 requestId→accept 接线仍未完成（P3b）**。

## 1. 交付

| 交付 | 路径 | 说明 |
|---|---|---|
| Orchestrator 执行账本迁移 | `services/orchestrator/internal/persistence/mysql/migrations/001_execution_ledger.sql` | executions/task/plan/approvals/tool_calls/checkpoints/events/outbox/catalog |
| 执行账本 Repository | `.../mysql/repository.go` | requestId 幂等、seq 分配、状态+事件+outbox 同事务、fencing、租约、checkpoint、审批一次性消费、工具 unknown |
| Memory 迁移/Repository | `.../mysql/migrations/002_memory.sql`、`memory_repository.go` | scope/entry/revision/mutation；候选/确认/CAS/遗忘 |
| Gateway MySQL Store | `services/gateway/store/mysql.go` + `migrations/001_gateway.sql` | 会话/消息权威存储、请求幂等记录、空消息拒绝、迁移导入 |
| Runtime Session MySQL schema | `pkg/runtime/session/migrations/001_mysql_schema.sql`、`mysql_schema.go` | sessions/session_events/session_context_snapshots |
| Gateway 启动切换 | `services/gateway/cmd/gateway/main.go` | 默认 `mysql`；缺 DSN 启动失败；memory/sqlite 仅显式 demo/test |
| 测试 | 单测 + DSN 门控集成测试 | 账本、Store、Session、SQLite→MySQL 字段映射 |

## 2. 真实 MySQL 验证（本次已执行）

环境：Docker `mysql:8.4`，隔离数据库 `agenthub_test`，测试 DSN 通过 `AGENTHUB_MYSQL_TEST_DSN` 显式注入；容器密码为一次性测试值，未写入仓库。

```text
$ AGENTHUB_MYSQL_TEST_DSN=... go test ./pkg/adk/... ./pkg/runtime/... ./services/gateway/... ./services/orchestrator/... -count=1
全部包 PASS
```

关键用例：

```text
TestIntegrationDuplicateRequestIsIdempotent              PASS
TestIntegrationConcurrentSeqAndStaleFencing              PASS
TestIntegrationCheckpointFencing                         PASS
TestIntegrationApprovalConsumedOnce                      PASS
TestIntegrationToolResultUnknown                         PASS
TestIntegrationFailedStateChangeLeavesNoHalfResult       PASS
TestIntegrationMemoryLifecycle                           PASS
TestIntegrationMemoryScopeIsolation                      PASS
TestIntegrationMySQLStoreCRUD                            PASS
TestIntegrationRecordChatRequestIdempotent               PASS
TestIntegrationMySQLStoreSurvivesReopen                  PASS  (关闭连接→重开→读历史→继续写)
TestIntegrationConversationIsolation                     PASS
TestIntegrationSQLiteToMySQLFieldMapping                 PASS  (合成 SQLite→MySQL 字段映射)
TestIntegrationMySQLSessionLifecycle                     PASS  (重启后 session/events/state 保留)
```

期间修复的真实缺陷：

- `ConsumeApprovalAndDispatch` 用 `RowsAffected` 判断 fencing，预算值未变化时 0 行被误判为 stale；改为事务内 `SELECT ... FOR UPDATE` 校验 fencing。
- `RenewLease` 同样存在“时间值未变化→0 行”误判；改为锁行校验 owner/fencing/status。
- `GetOrCreate` 的 duplicate 判断改为大小写不敏感，兼容 MySQL `Duplicate entry`。

## 3. 仍未完成

1. **Gateway requestId → Orchestrator accept 接线**：Gateway 已能写 `chat_requests`，Orchestrator 已有 `AcceptRun`；HTTP handler/main 尚未注入执行 Repository，因此跨服务超时重试/查询未验证（计划归入 P3b）。
2. Orchestrator 运行尚未写 executions/checkpoint/events/outbox；当前只是 Repository 层。
3. 真实用户 SQLite 数据导入未执行（P7，需授权）；仅完成合成映射测试。
4. 旧 SQLite 路径的“空助手消息终态”在 Gateway 投影层尚未专门回归；MySQL Store 已拒绝空消息作为新入口。

## 4. 结论

P2 的存储/事务/唯一约束/并发/重启恢复门禁在真实 MySQL 8.4 上通过；**不能**声明 P2 全部完成，因为跨服务 accept 与运行接线仍属于后续整合。
