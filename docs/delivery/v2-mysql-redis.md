# AgentHub v2 部署：MySQL + Redis + Harness

- 日期：2026-10-29
- 状态：**MySQL/Redis/Orchestrator/Gateway 最小栈冷启动与迁移已通过；完整 Agent+真模型 smoke 未执行**。
- 相关：[实施计划](../plans/agenthub-redis-sql-harness-plan.md)、[架构](../architecture/optimization-architecture.md)。

## 1. 拓扑

```text
Frontend(3000) → Gateway(8080)
                  ├─ MySQL(3306)：conversations/messages/chat_requests/run_views/projections
                  └─ Orchestrator(8090 HTTP/SSE 兼容)
                        ├─ MySQL：executions/approvals/tool_calls/checkpoints/events/outbox/Memory
                        ├─ Redis(6379)：outbox relay 短期 Stream 副本
                        └─ 10 个 Child Agents(8081-8090)
```

## 2. 版本锁定

| 组件 | 版本/镜像 | 备注 |
|---|---|---|
| Go | 1.26（`go.work`、Dockerfile 一致） | — |
| MySQL | `mysql:8.4@sha256:a0225d1e…8d5dd7` | 8.4 LTS，GPLv2 服务端 |
| Redis | `redis:7.2-alpine@sha256:2c933526…b96d84` | 7.2 BSD-3-Clause；7.4+ 许可不同 |
| MySQL 驱动 | `go-sql-driver/mysql v1.9.3` | MPL-2.0 |
| Redis 客户端 | `go-redis/v9 v9.17.3` | BSD-2-Clause |

调用方修改镜像/驱动前必须复核许可证与支持状态。

## 3. 启动

```bash
cp .env.example .env
# 在 .env 填入（不要提交）：
#   MYSQL_ROOT_PASSWORD=...
#   LLM_API_KEY=... LLM_MODEL=...
docker compose -f docker-compose.new-arch.yml up -d --build
```

- Orchestrator 在 `DATABASE_URL` 存在时启动迁移并启用执行账本；`REDIS_URL` 存在时启动 outbox relay。
- Gateway 默认 `AGENTHUB_GATEWAY_STORE=mysql`；缺少 `DATABASE_URL` 启动失败。
- 迁移在服务启动时执行；相同 migration 不会重复。
- MySQL/Redis 健康检查见 compose；Orchestrator 无模型配置会启动失败（LLM-only）。

## 4. 最小 smoke（启动后手动执行）

```bash
curl -fsS http://localhost:8080/health
curl -fsS http://localhost:8090/health
# 创建会话（按实际 API 字段）
curl -fsS -X POST http://localhost:8080/api/conversations -H 'Content-Type: application/json' -d '{"agentName":"code-agent"}'
# 发消息需要真实模型 key；成功后检查：
#   MySQL executions/execution_events/outbox
#   Redis: docker exec <redis> redis-cli --scan --pattern 'agenthub:*:run:*:events'

# 重启恢复 smoke（要求 MYSQL_ROOT_PASSWORD 已注入）
bash scripts/smoke-restart.sh
```

## 5. 数据迁移工具

```bash
# 先 dry-run 校验源库结构与计数（不写 MySQL）
go run ./services/gateway/cmd/migrate-sqlite --sqlite /path/legacy.db --dry-run
# 经明确授权后导入；DSN 由命令行显式传入，不读取 .env
go run ./services/gateway/cmd/migrate-sqlite --sqlite /path/legacy.db --mysql-dsn "user:pass@tcp(host:3306)/agenthub?parseTime=true&loc=UTC"
```

- 导入保留 conversation/message ID、标题、pinned、时间与 metadata 中的 userId/agentName。
- 幂等：重复运行不会重复消息。
- 真实数据导入前必须备份并在合成/脱敏副本演练；本工具不自动读取本地 data 目录。

## 6. 持久化、备份与回滚

- MySQL 使用 `mysql-data` 卷；**单机部署**。未完成跨节点共享存储验证前，不声明多节点 artifact/恢复能力。
- 备份：停写后 `mysqldump --single-transaction`；恢复前停止 Gateway/Orchestrator。
- 备份恢复后，P3c 的 Memory 删除标记/epoch 重放流程尚未实现；在完成前不得把恢复库直接开放召回。
- **代码回滚 ≠ 数据库回滚**。新 schema 已接受新写后，不得启动旧 SQLite/rule 模式；应暂停入口，使用向前修复或经验证的导出恢复。
- 破坏性清表、回退 migration、覆盖备份必须单独确认。

## 7. 已知未完成

1. 内部 gRPC 未实现，当前仍是 HTTP/SSE 兼容路径。
2. 浏览器断线恢复/游标、Memory 管理 UI、投影 offsets 未接线。
3. 最小栈（4 服务）已启动验证；完整 10-Agent 与真模型聊天/审批/回放 smoke 尚未执行。
4. 真实数据 SQLite→MySQL 导入未执行（仅合成字段映射测试）。
5. 多节点 artifact/Redis 故障演练未执行。
