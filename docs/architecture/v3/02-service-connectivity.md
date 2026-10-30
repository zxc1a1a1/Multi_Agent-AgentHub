# 02 服务联通与协议边界

- 状态：待评审
- 目标：明确“谁和谁说话、用什么协议、用什么身份、断在哪里”

## 1. 协议矩阵

| 链路 | 协议 | 端口/路径 | 身份 | 备注 |
|---|---|---|---|---|
| Browser → Gateway | HTTP + SSE | `:8080/api/*`、`/api/chat`、`/api/runs/{id}/events/stream` | `AGENTHUB_API_TOKEN`（可选） | POST SSE，支持 cursor |
| Gateway → Orchestrator | gRPC | `:9090` StartRun/SubscribeRun/Status/Approval/Cancel/ListAgents | 内部网络 | 聊天主路径 |
| Gateway → Orchestrator | HTTP | `:8080/internal/orchestrator/*` | `GATEWAY_INTERNAL_TOKEN` | 回放/管理兼容 |
| Orchestrator → Child Agent | A2A JSON-RPC | `/` `tasks/sendSubscribe` | 内部网络 | 结果是事件流 |
| Orchestrator/Agent → Model | HTTPS | provider API | 环境变量密钥 | 超时/错误脱敏 |
| Orchestrator → MySQL | MySQL | `mysql:3306` | `DATABASE_URL` | 账本/事件/审批 |
| Orchestrator → Redis | RESP | `redis:6379` | `REDIS_URL` | outbox relay/cache |
| Gateway → MySQL | MySQL | `mysql:3306` | `DATABASE_URL` | 会话/消息/投影 |
| Tool Runtime → MCP | stdio/HTTP | 由 adapter 决定 | allowlist | 不得自动安装 |
| Tool Runtime → Sandbox | Docker API | 本地 socket | 受控 executor | 非 root/无网络 |
| WeCom/Feishu → Gateway | HTTPS callback | `/api/integrations/*` | 签名/Token | 见 08/09 |

## 2. 端口

| 服务 | 容器端口 | 宿主端口（示例） |
|---|---|---|
| Gateway | 8080 | 8080 |
| Orchestrator HTTP | 8080 | 8090 |
| Orchestrator gRPC | 9090 | 9090 |
| code-agent | 8080 | 8081 |
| web-agent | 8080 | 8082 |
| MySQL | 3306 | 3306 |
| Redis | 6379 | 6379 |

## 3. 核心时序

### 3.1 启动一次运行

```text
Browser POST /api/chat
  -> Gateway 生成 runId/requestId，写 chat_requests
  -> gRPC StartRun
  -> Orchestrator 建账本/租约，调用 Planner
  -> 计划落库，等待审批（如需要）
  -> Gateway SubscribeRun 消费公开事件
  -> Browser 收 SSE（seq/subIndex）
```

### 3.2 审批

```text
Browser POST /api/runs/{id}/confirm
  -> Gateway ConfirmRun
  -> gRPC SubmitApproval
  -> Orchestrator 校验 plan/action/revision/participants
  -> 一次性消费审批，执行计划
```

### 3.3 事件与恢复

- 公开事件必须有稳定 `(runId, seq, subIndex)`。
- 浏览器保存 `(runId, seq, subIndex)`；断线后先读 SQL 回放，再继续。
- `RUN_FINISHED`/`RUN_ERROR` 为终态；SSE 断开不等于取消 run。
- 显式取消走 `POST /api/runs/{id}/cancel`，不得靠断连表达取消。

## 4. 鉴权与隔离

1. 浏览器只携带 `AGENTHUB_API_TOKEN`，不接触内部 token/模型 key。
2. Gateway-Orchestrator 使用独立内部 token；生产必须开启。
3. A2A、gRPC、MCP 端口不得暴露到公网。
4. IM 回调必须验签/解密，校验时间窗与重放；只接受配置的 CorpID/AppID。
5. 工具调用参数哈希与审批绑定，参数变化即拒绝。

## 5. 联通验收清单

- [ ] `curl /health` 每个服务返回 ok
- [ ] gRPC `ListAgents` 返回预期 Agent
- [ ] 浏览器 `/api/chat` 能收到 `RUN_STARTED` → 文本 → `RUN_FINISHED`
- [ ] `/api/runs/{id}/status` 与 `/events` 可读
- [ ] 审批 `/confirm` 返回 200 且只消费一次
- [ ] 取消 `/cancel` 在运行中返回 200；待审批阶段返回 200
- [ ] 断线后重连按 `(seq, subIndex)` 无重复/无缺口
- [ ] MySQL/Redis 重启后 SQL 历史仍可读

## 6. 已知边界

- gRPC SubscribeRun 当前以 SQL 轮询（约 200ms）实现，不是 Redis 实时推送。
- 外部 IM 渠道尚未实现，需新增 Gateway adapter。
- 浏览器刷新恢复已实现；工具/制品卡片重建未验证。
- DeepSeek Flash 带 tools 返回 HTTP 400，工具默认关闭；详见 07。
