# Frontend API Client 策略

Frontend 必须从 OpenAPI 生成 API 类型与 client。

不得手写 API response interface，不得直接 `fetch` OpenAPI 未定义路径，不得访问 `/internal/**`，不得访问 Orchestrator 真实地址。Mock 数据必须符合 OpenAPI schema。
