# Service-to-Service Auth Policy

## 强制边界

Gateway Service 与 Orchestrator Service 必须分进程。Gateway 只能通过受保护的内部 API 或 internal stream 调用 Orchestrator。

## 最小要求

- 内部调用有服务凭证。
- 内部调用有 timeout。
- 内部调用有 requestId / traceId / runId。
- 内部凭证不进入日志、错误、Artifact、AgentCard、前端事件。

## gRPC 控制面（A1-07 已实现）

- Server 端 `grpcapi.Server.GRPCServer()` 安装 unary + stream interceptor；
  `AGENTHUB_INTERNAL_TOKEN`（`InternalToken`）非空时，缺失或错误的
  `authorization: Bearer <token>` 返回 `Unauthenticated`，处理器不执行。
- 客户端 `orchestratorclient.NewGRPCRunServiceWithAuth` 注入 token 与
  `x-agenthub-trace-id`；`StartRun`/`SubscribeRun`/`SubmitApproval`/`CancelRun`
  共用同一元数据。
- `StartRunRequest.trace_id` 为空时回退到 metadata trace；Gateway 每次 Run 生成
  `trace_<unixnano>` 并写入请求与 metadata，供 Orchestrator 日志串联。
- token 为空仅保留本地/测试兼容；生产启动必须同时配置 HTTP 与 gRPC token。
- 证据：`services/orchestrator/grpcapi/auth_test.go`（缺失/错误 token 被拒、
  trace 透传），`services/gateway/orchestratorclient/grpc_test.go`
  （客户端发送 auth/trace metadata）。

## 禁止

- Frontend 访问 Orchestrator。
- 用户 token 充当 service token。
- Orchestrator `/internal/**` 写入公开 API。
