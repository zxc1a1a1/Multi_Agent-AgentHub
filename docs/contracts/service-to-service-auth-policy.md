# Service-to-Service Auth Policy

## 强制边界

Gateway Service 与 Orchestrator Service 必须分进程。Gateway 只能通过受保护的内部 API 或 internal stream 调用 Orchestrator。

## 最小要求

- 内部调用有服务凭证。
- 内部调用有 timeout。
- 内部调用有 requestId / traceId / runId。
- 内部凭证不进入日志、错误、Artifact、AgentCard、前端事件。

## 禁止

- Frontend 访问 Orchestrator。
- 用户 token 充当 service token。
- Orchestrator `/internal/**` 写入公开 API。
