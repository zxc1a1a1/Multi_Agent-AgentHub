# P7 阶段报告：部署与数据切换（部分）

- 日期：2026-10-29
- 基线：`dev / 01e3c3a`
- 状态：**compose 已加入 MySQL 8.4/Redis 7.2；镜像构建与最小栈冷启动/健康/迁移 smoke 通过；完整 Agent+真模型+浏览器验收未执行**。

## 1. 交付

| 项目 | 状态 | 证据 |
|---|---|---|
| compose MySQL/Redis 服务 | 完成 | `docker compose config` exit=0 |
| Gateway 主 Store=mysql | 完成 | compose 设置 `AGENTHUB_GATEWAY_STORE=mysql` + DATABASE_URL |
| Orchestrator 账本/relay | 完成 | compose 注入 DATABASE_URL/REDIS_URL |
| 版本锁定 | 完成 | MySQL/Redis digest、驱动版本写入交付文档 |
| 交付文档 | 完成 | `docs/delivery/v2-mysql-redis.md` |
| 镜像构建 | **PASS** | gateway/orchestrator 镜像构建成功 |
| 最小栈启动/健康/迁移 | **PASS** | MySQL/Redis/Orchestrator/Gateway；health ok；21 张表 |
| MySQL/Redis 重启恢复 | **PASS** | `scripts/smoke-restart.sh`：会话在 MySQL 重启后仍可读；Redis 重启不影响 SQL 权威历史 |
| SQLite→MySQL 迁移工具 | 完成（合成验证） | `cmd/migrate-sqlite`；测试使用临时 SQLite + 真实 MySQL，幂等；**真实用户数据未导入** |
| 回滚流程 | 已文档化 | 代码回滚≠数据库回滚 |

## 2. 验证命令

```text
$ MYSQL_ROOT_PASSWORD=<placeholder> docker compose -f docker-compose.new-arch.yml config
exit=0（仅缺 LLM 变量告警）

$ docker pull mysql:8.4        # digest a0225d1e…8d5dd7
$ docker pull redis:7.2-alpine # digest 2c933526…b96d84

$ docker compose -f docker-compose.new-arch.yml build gateway-new orchestrator-new
PASS

$ docker compose ... up -d --no-deps mysql redis orchestrator-new gateway-new
PASS（MySQL/Redis healthy；Orchestrator/Gateway healthy）

$ curl http://localhost:8080/health → {"service":"gateway","status":"ok"}
$ curl http://localhost:8090/health → {"service":"orchestrator","status":"ok"}
$ MySQL agenthub 表：agent_catalog, approvals, artifact_views, chat_requests,
  checkpoints, conversations, event_projection_offsets, execution_events,
  executions, memory_entries/mutations/revisions/scopes, messages, outbox,
  plan_revisions, run_views, schema_migrations, task_executions, tool_calls
$ POST /api/conversations → 200；MySQL conversations 行存在（userId=smoke-user）

$ bash scripts/smoke-restart.sh
created conversation=...
PASS: MySQL restart recovered conversation
PASS: Redis restart did not affect SQL-authoritative history
```

说明：Orchestrator 使用占位 provider/model 启动（未发起模型请求）；日志提示未启动的 8 个 Agent unhealthy，符合 `--no-deps` 最小栈预期。

## 3. 数据转换证据

```text
$ AGENTHUB_MYSQL_TEST_DSN=... go test ./services/gateway/migration/ -count=1 -v
PASS
- 临时 SQLite（真实 sqlite.Store）→ MySQL：会话/消息/用户/标题/pinned/时间字段
- 第二次 Apply 幂等，不重复消息
- CLI：services/gateway/cmd/migrate-sqlite（--sqlite --mysql-dsn --dry-run）
```

## 4. 未通过的门禁

1. 完整 10-Agent 栈与真实模型未启动；聊天/审批/工具/取消/回放 smoke 未执行。
2. MySQL/Redis 重启已通过最小恢复 smoke；Redis 清空/裁剪、MySQL 崩溃事务恢复未做。
3. 未备份/迁移真实数据；禁止把合成测试当作用户数据迁移证据。
4. 未验证多节点 artifact 存储；只允许声明单机持久卷。
5. 内部 gRPC、浏览器恢复、Memory 管理 UI 仍未实现，不能声明目标拓扑完整。

## 4. 结论

P7 的 compose/版本/文档/镜像构建与最小栈冷启动已通过；完整业务 smoke 与数据切换未完成，P7 未完成。
