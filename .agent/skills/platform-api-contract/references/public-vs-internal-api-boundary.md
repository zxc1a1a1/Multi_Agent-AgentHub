# 公开 API 与内部 API 边界

Platform API 只定义 Frontend ↔ Gateway Service。

公开 API 必须位于 `/api/**`。

以下 API 不得进入公开 OpenAPI：

```text
/internal/**
/orchestrator/**
/a2a/**
/.well-known/agent.json
/a2a/tasks/**
```

Gateway 与 Orchestrator 必须分进程。Frontend 不知道 Orchestrator URL，也不持有 service-to-service token。
