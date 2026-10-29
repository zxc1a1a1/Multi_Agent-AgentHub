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
| SQLite→MySQL 真实数据导入 | 未执行 | 仅合成映射测试；需用户授权 |
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
```

说明：Orchestrator 使用占位 provider/model 启动（未发起模型请求）；日志提示未启动的 8 个 Agent unhealthy，符合 `--no-deps` 最小栈预期。

## 3. 未通过的门禁

1. 完整 10-Agent 栈与真实模型未启动；聊天/审批/工具/取消/回放 smoke 未执行。
2. 未验证 Redis 重启/清空、MySQL 重启后的恢复。
3. 未备份/迁移真实数据；禁止把合成测试当作用户数据迁移证据。
4. 未验证多节点 artifact 存储；只允许声明单机持久卷。
5. 内部 gRPC、浏览器恢复、Memory 管理 UI 仍未实现，不能声明目标拓扑完整。

## 4. 结论

P7 的 compose/版本/文档/镜像构建与最小栈冷启动已通过；完整业务 smoke 与数据切换未完成，P7 未完成。
