# R0-04 Docker compose 构建与冒烟认证

- commit: 150019f2da917abcbaabc47f6595aa58bc442057
- compose project: agenthub-r004 / file: docker-compose.new-arch.yml / base url: http://127.0.0.1
- started: 2026-11-02T09:41:13Z / ended: 2026-11-02T09:41:25Z
- result: **PASS** (28 checks, 0 failures)
- requests issued: 9; 5xx: 0
- containers healthy: 16/16
- provider: `scripts/fake-llm-server.mjs`（测试专用假模型，无任何真实凭据；生产链路不在本任务认证范围）

## checks

| check | result | detail |
|---|---|---|
| all containers healthy | PASS | 16/16 services |
| GET /health via nginx | PASS | 200 |
| GET /api/agents lists code-agent | PASS |  |
| internal token absent from /api/agents | PASS |  |
| POST /api/conversations (single) | PASS | df1edf3fc163e1564c7b4eb7328678b1 |
| F-01 run started over SSE | PASS | run=run_ab27176b343834ac |
| F-03 plan reaches approval | PASS | plan=plan_1793612475178_2059 |
| F-03 approve accepted | PASS | status=200 |
| F-01 agent attribution in stream | PASS | 9 events |
| F-01 no RUN_ERROR | PASS |  |
| F-01 terminal state reached | PASS |  |
| nginx keeps SSE streaming content type | PASS |  |
| nginx disables buffering for SSE | PASS |  |
| internal token absent from SSE | PASS |  |
| F-01 messages persisted | PASS | 2 roles |
| POST /api/conversations (group) | PASS | 602ea961353ca828031e8c41a6182a55 |
| F-14 participants persisted as selected | PASS | [{"agentName":"code-agent","role":"owner","selected":true},{"agentName":"test-agent","role":"member","selected":true}] |
| F-02 group chat SSE stream started | PASS | 4 events, run=run_10424e62b24081ce |
| F-02 group plan reaches approval | PASS | plan=plan_1793612480257_8208 |
| F-14 non-selected agents excluded from plan | PASS | test-agent present, code-agent present |
| F-05 cancel pending group run | PASS | status=200 |
| F-05 cancelled state observed in stream | PASS |  |
| F-05 cancel emits no RUN_ERROR | PASS |  |
| F-02 history readable after cancel | PASS |  |
| artifact volume writable | PASS | drwxr-xr-x    3 app      app           4096 Nov  2 09:41 /data/artifacts |
| no tokens in captured traffic | PASS |  |
| secret pattern scan on evidence | PASS |  |
| no 5xx responses | PASS | 9 requests |


## interaction coverage (plan §5.5)

| interaction | record file | rows | coverage |
|---|---|---|---|
| 用户操作（创建会话/选择参与者/发送） | events.jsonl | 6 | yes（compose 层以等价 API 动作驱动；真实浏览器矩阵在 R5-04） |
| API 请求/响应 | events.jsonl | 5 | yes |
| SSE 连接/事件 | events.jsonl | 9 | yes（只记事件类型与计数，不记文本增量） |
| HITL 请求/审批/取消 | events.jsonl | 2 | yes（F-03 approve、F-05 cancel） |
| 工具授权/结果 | - | 0 | not_applicable: 假 LLM 计划不触发写工具审批 |
| 记忆召回 | - | 0 | not_applicable: MEMORY_MODE=session_only |
| Artifact | events.jsonl | 1 | yes（制品卷可写探针） |
| MQ | - | 0 | not_applicable: R2 未实现 |
| 多实例 | - | 0 | not_applicable: 默认 compose 单副本；多副本在 R0-06 |

证据文件：logs/compose-ps.txt、logs/http-transcript.txt、logs/f01-sse.txt、logs/f02-sse.txt、logs/f01-sse-headers.txt。
响应体只在本地临时文件解析，不入库；SSE 采样保留事件类型与计数，不含完整文本增量。

## 执行记录（计划 §9）

任务 ID：R0-04
对应缺口：GAP-03；F-01、F-02、F-14（并顺带认证 F-03 approve、F-05 cancel 的部署形态链路）
目标行为：`docker compose -f docker-compose.new-arch.yml` 能构建并拉起全栈，所有健康检查通过，并且**经真实后端**完成单 Agent 对话（含审批）与群聊（含参与者过滤与取消）。

### 拓扑与前置

- 16 个容器全部 healthy（`logs/compose-ps.txt`）：mysql、redis、10 个 child agent、orchestrator、gateway、frontend、nginx。
- 冒烟走浏览器同款入口：`http://127.0.0.1` → nginx → gateway → orchestrator → child agent。
- **模型用测试专用假 provider**：`scripts/fake-llm-server.mjs`（只实现 `POST /v1/messages`，忽略任何 key，不联网）。本任务不认证真实模型输出质量，那是 R5-05。凭据全部是一次性假值（`smoke-*`），只在 shell 与容器环境变量里，不进仓库。

### 真实环境暴露的两个部署缺陷（已修）

1. `frontend-new` 永远 unhealthy：`frontend/nginx.conf` 是 `listen 3000`（仅 IPv4），而 compose 健康检查打 `http://localhost:3000/`；容器内 `localhost` 解析到 `::1` → 连接被拒。由于 `nginx` 服务 `depends_on: frontend-new: service_healthy`，**整个入口层永远起不来**。修复：健康检查改打 `http://127.0.0.1:3000/`。
2. `nginx` 服务同样永远 unhealthy：`deploy/nginx/agenthub.conf` 是 `listen 80`（仅 IPv4），健康检查打 `http://localhost/health`。修复：改打 `http://127.0.0.1/health`。实测 `127.0.0.1/health -> OK`、`localhost/health -> Connection refused`，确认根因就是 IPv6 解析而非服务本身。

这两处只有真实 `compose up` 才能暴露；此前 Docker 侧只有静态 YAML 审计，因此从未被发现（GAP-03 的实质）。

### HITL 在部署形态下的调用约束（记录，供文档修正）

- Gateway 在设置 `ORCHESTRATOR_GRPC_URL` 时走 gRPC 控制面（`services/gateway/cmd/gateway/main.go:130`），`GRPCRunService.ConfirmRun` 会把 `runId/actionId/planId/confirmed` 透传给 orchestrator。
- **只带 `action` 不带 `actionId` 与 `confirmed` 的 confirm 请求会被上游拒绝**，Gateway 以 `502 {"error":"HITL confirmation failed"}` 返回（非 `HTTPError`，上游状态码被吞掉）。补上 `actionId` 后 approve 与 cancel 都返回 200。
- `docs/contracts/platform-api.md` 的 confirm 请求示例里没有 `actionId`，与实现不一致；本轮以真实链路证据为准，文档修正留到 R3-01 之后的契约整理，不在本任务顺手改契约。
- Gateway 没有 `GET /api/runs/{id}` 路由（实测 404），运行终态通过 SSE 流与 `/api/conversations/{id}/messages` 判定；冒烟脚本据此断言，不再打不存在的路由。

### 结果

- 命令：`bash scripts/smoke-compose.sh --skip-up --keep-up --out docs/reports/evidence/2026-11-02/R0-04`
- **28 项检查全部 PASS，退出码 0**；9 个 HTTP 请求，0 个 5xx。
- F-01：单 Agent 对话 `run_started → planning → awaiting_confirmation →（approve）→ 执行 → 终态`，流中出现 `code-agent` 归属、无 `RUN_ERROR`，消息落库 2 条（user + assistant）。
- F-03：`POST /api/runs/{runId}/confirm` action=approve → 200，运行继续执行到终态。
- F-02 + F-14：群聊会话参与者 `[code-agent(owner), test-agent(member)]` 持久化正确，计划事件里出现两位被选 Agent，取消后可读历史。
- F-05：群聊在 `awaiting_confirmation` 阶段 cancel → 200，流中出现 cancelled 终态且**没有** `RUN_ERROR`（符合契约）。
- 传输安全：`text/event-stream` 保留、SSE 头含缓冲关闭标记、内部 token 不出现在 `/api/agents` 与 SSE 中、证据目录密钥模式扫描 0 命中。
- 制品卷 `/data/artifacts` 对服务用户可写（容器以 `app` 用户运行）。

### 构建侧记录与未认证项

- 镜像构建用 `docker compose -p agenthub-r004 -f docker-compose.new-arch.yml build`（未加 `--pull`）：本地已有 `mysql:8.4`、`redis:7.2-alpine`、`nginx:1.27-alpine` 等基础镜像，`--pull` 需重新拉取约 1GB 基础层，本机网络下未纳成本轮认证范围。**“`build --pull` 干净拉取”这一项本轮未认证**，需在 CI（R0-05）覆盖。
- 镜像大小（本地 `docker images`）：orchestrator 52.3MB、gateway 50.7MB、frontend 75.6MB、各 agent ≈28MB。

### 改动文件（实际）

- `docker-compose.new-arch.yml`（两处健康检查端点修正）
- `scripts/smoke-compose.sh`（新增）
- `scripts/fake-llm-server.mjs`（新增，测试专用）
- 本证据目录

### 未改文件（确认）

- 未修改 Gateway/Orchestrator/Agent 业务代码与任何 handler；上述 HITL 参数约束只作为证据记录
- 未修改 legacy `server/`、`agents/`；未修改 CI workflow；未读取或写入 `.env`

### 契约/数据变更

- 无。compose 端口、服务依赖、卷与镜像未变，仅健康检查 URL 修正。

### 回滚方式

- `git revert` 本提交；`docker compose -p agenthub-r004 -f docker-compose.new-arch.yml down -v` 清理测试栈。

### 交互覆盖补充说明

§5.5 表中“用户操作”行是以 API 等价动作（创建会话、选择参与者、发送消息、点击审批/取消）驱动并记录的，`action=select`/`http_request` 即对应这些动作；真实浏览器点击、键盘与视口矩阵属于 R5-04，本任务不声称已覆盖。
