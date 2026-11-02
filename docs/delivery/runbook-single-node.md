# AgentHub 单机运行手册（Runbook）

- 状态：适用于 `docker-compose.new-arch.yml` 单机拓扑；多副本/对象存储/OTel 尚未纳入。
- 原则：不记录真实密钥；命令中的 `<...>` 由执行者从环境注入。
- 事实源：`docs/contracts/*`、`docs/plans/v3-atomic-implementation-plan.md`、源码与测试。

## 1. 拓扑与端口

| 组件 | Compose 服务 | 宿主端口 | 健康检查 |
|---|---|---|---|
| Gateway | `gateway-new` | `8080` | `GET /health` |
| Orchestrator HTTP | `orchestrator-new` | `8090` | `GET /health` |
| Orchestrator gRPC | `orchestrator-new` | `9090` | token 必需 |
| Frontend | `frontend-new` | `3000` | `GET /` |
| MySQL | `mysql` | `3306` | `mysqladmin ping` |
| Redis | `redis` | `6379` | `redis-cli ping` |
| 10 个 Child Agent | `*-agent-new` | `8081`–`8089` | `GET /health` |

Orchestrator 的制品元数据写入命名卷 `orchestrator-artifacts`，容器内路径
`/data/artifacts`（`ARTIFACT_STORE_DIR`）。卷不存在时 Docker 会创建；没有该卷的
临时容器重启后制品不可读。

## 2. 启动 / 停止 / 状态

```bash
# 启动（在仓库根目录；.env 必须存在且不提交）
docker compose -f docker-compose.new-arch.yml up --build -d

# 状态与健康
docker compose -f docker-compose.new-arch.yml ps
curl -fsS http://127.0.0.1:8080/health
curl -fsS http://127.0.0.1:8090/health

# 日志（单服务/全部）
docker compose -f docker-compose.new-arch.yml logs -f gateway-new
docker compose -f docker-compose.new-arch.yml logs --no-color --tail=200

# 停止但保留卷（数据不丢）
docker compose -f docker-compose.new-arch.yml down

# 停止并删除数据卷（危险：会删除会话/账本/制品）
docker compose -f docker-compose.new-arch.yml down -v
```

## 3. 运行排查

内部接口需要 `Authorization: Bearer <INTERNAL_SERVICE_TOKEN>`；token 只从环境注入。

```bash
# 运行状态 / 事件回放 / 路由决策
curl -fsS -H "Authorization: Bearer $INTERNAL_SERVICE_TOKEN" \
  http://127.0.0.1:8090/internal/orchestrator/runs/<runId>/status
curl -fsS -H "Authorization: Bearer $INTERNAL_SERVICE_TOKEN" \
  "http://127.0.0.1:8090/internal/orchestrator/runs/<runId>/events?after=0&limit=200"
curl -fsS -H "Authorization: Bearer $INTERNAL_SERVICE_TOKEN" \
  http://127.0.0.1:8090/internal/orchestrator/runs/<runId>/route-decisions

# 路由/预算指标（window 支持 24h / 90m，或 since=RFC3339）
curl -fsS -H "Authorization: Bearer $INTERNAL_SERVICE_TOKEN" \
  "http://127.0.0.1:8090/internal/orchestrator/metrics/routing?window=24h"
```

前端 SSE：`POST /api/chat` 使用 `text/event-stream`。反向代理必须关闭响应缓冲
（`proxy_buffering off`、`X-Accel-Buffering: no`），否则前端表现为“卡住后一次性输出”。

## 4. 故障处置

### 4.1 MySQL 不可用

- 现象：Gateway/Orchestrator 启动日志出现持久化不可用；`/health` 可能仍 200。
- 处置：先恢复 MySQL（`docker compose ps mysql`、查看卷 `mysql-data`），再重启
  `orchestrator-new`/`gateway-new`。迁移由服务启动时幂等执行（`schema_migrations`）。
- 不要手工删除 `schema_migrations` 记录来“修复”迁移；先备份再定位失败语句。

### 4.2 Redis 不可用

- Redis 只承担缓存/relay/短期流；SQL 账本仍是事实源。
- 处置：重启 Redis 后观察 outbox relay 重投；重复投递由 `(runId, seq)` 去重。
- 若 Memory 功能报错，检查 `REDIS_URL` 与缓存 epoch；不要用清空 Redis 代替备份。

### 4.3 Gateway 崩溃/重启

- 已在 journal 中的事件由投影 worker 从 `run_public_events` 恢复；只投影
  `AGENT_TURN_FINISHED`/`RUN_FINISHED` 的完整轮次。
- 处置：重启 `gateway-new`；观察日志 `gateway projection: recovered N turn message(s)`。
- 若前端历史缺最后一轮，先查 `/events` 是否已有该轮 finish 事件，再查投影偏移
  `event_projection_offsets`。
- 已知限制：`AGENT_TURN_FINISHED` 之前的崩溃轮次不会落库（A1-09c 未完成）。

### 4.4 Orchestrator 重启/租约

- 每次 run 获取 15 分钟租约，handler 返回时释放；崩溃后最迟 TTL 过期可被接管。
- 处置：重启后用 `/status` 查看 `fencingVersion/status`；`paused` 运行可
  `POST /internal/orchestrator/runs/<runId>/resume`（需要有效 checkpoint/budget）。
- 预算损坏/超限会拒绝 resume（fail-closed），不会重置预算。

### 4.5 审批丢失/重复

- 跨实例确认写 `run_signals`，同 `(run_id, idempotency_key)` 重放不会重复消费；
  `hitl_confirmations` 缓存响应。重复确认返回同一响应或 409 `CONFIRM_IN_FLIGHT`。
- 处置：查 `run_signals`/`hitl_confirmations`/`approvals` 状态；不要手工改
  `status='pending'` 绕过审批，先确认运行是否已终态。

### 4.6 取消失败

- 取消请求写 `run_cancels`；远端任务引用在 `run_task_refs`，任一实例可对 A2A
  任务发取消。执行实例轮询取消请求并终止本地 context。
- 处置：`/status` 确认终态；若仍执行，检查 `run_task_refs` 中 agent URL 是否可达，
  以及该 Agent 的 `/health`。

### 4.7 制品不可读/磁盘满

- 检查 `orchestrator-artifacts` 卷挂载与 `ARTIFACT_STORE_DIR` 权限（容器内 `app` 用户）。
- 磁盘满时先扩容/清理旧制品；不要删除整个卷来“释放空间”，会丢失全部元数据。
- 跨节点读取未实现（A6-03 S3/MinIO 未完成），不要把单机卷当成共享存储。

## 5. 备份与恢复

```bash
# MySQL 逻辑备份（密码从环境读取，不写入命令历史）
docker compose -f docker-compose.new-arch.yml exec -T mysql \
  mysqldump -uroot -p"$MYSQL_ROOT_PASSWORD" --single-transaction agenthub > backup.sql

# 制品卷备份
docker run --rm -v agenthub_orchestrator-artifacts:/data -v "$PWD":/backup alpine \
  tar czf /backup/artifacts.tgz -C /data .
```

恢复顺序：先 MySQL（停 Gateway/Orchestrator 写入），再制品卷，最后启动服务。
恢复后跑一遍 `go test ./...` 不替代生产校验；用 `/status`、`/events`、
`/metrics/routing` 抽查。

## 6. 回滚

1. 记录当前镜像/提交：`git rev-parse HEAD`、`docker compose images`。
2. 回滚代码后 `docker compose ... up --build -d`；数据库迁移是向前兼容的，
   不删除旧列；不要 `down -v`。
3. 若迁移导致启动失败：停止服务、保留卷，恢复到迁移前备份，再回滚镜像。
4. 回滚后检查：health、一次对话、审批、取消、制品读取。

## 7. 安全操作

- Token/密码只经环境变量或 Secret 注入；不要写入仓库、日志、Issue/PR。
- 轮换 `INTERNAL_SERVICE_TOKEN`/`AGENTHUB_API_TOKEN` 后同时重启 Gateway 与
  Orchestrator；gRPC 无 token 请求应被拒绝。
- 不把 `.env` 复制进容器镜像；`.env.example` 只保留空值/占位符。
- 发现密钥泄露：立即轮换、清理日志、按 `docs/contracts/secret-management.md` 处置。

## 8. 未覆盖（不要按本手册宣称已支持）

- 多 Orchestrator 副本整体认证（A6-05）、S3/MinIO（A6-03）、Nginx 模板与
  自动备份（A6-01）、OTel/告警（A6-06/A6-07）。
- MySQL/Redis 集成测试在本机可能 SKIP；CI 门禁 `.github/workflows/v3-integration-gates.yml`
  在配置了 service 时执行，SKIP 即失败。
