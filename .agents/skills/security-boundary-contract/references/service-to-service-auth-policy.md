# Service-to-Service Auth Policy

## 范围

适用于 Gateway Service ↔ Orchestrator Service，以及 Orchestrator 与内部服务的调用。

## 硬规则

- Gateway 与 Orchestrator 必须分进程。
- Gateway → Orchestrator 必须有 service-to-service auth。
- 用户 token 不得作为 service token 透传。
- Orchestrator 的 /internal/** 不得暴露给 Frontend。
- 内部调用必须有 timeout、requestId、traceId、runId。

## 允许模式

- private network + INTERNAL_SERVICE_TOKEN
- mTLS
- service mesh identity
- 等价的内部身份机制

## 禁止

- 在日志中打印 service token。
- 在前端事件中返回 service token。
- 把内部 endpoint 写入公开 OpenAPI。
