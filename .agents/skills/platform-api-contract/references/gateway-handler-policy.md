# Gateway Handler 策略

Gateway Handler 是公开 Platform API 的实现层，不是编排层。

允许：解析 HTTP request、用户鉴权、对象级权限校验、request schema 校验、调用应用服务或内部下游、返回统一 response envelope、记录 requestId / traceId。

禁止：直接选择 Agent、直接调用 Child Agent、直接调用 LLM Provider、直接生成 OrchestrationPlan、暴露 Orchestrator 内部 endpoint、返回 OpenAPI 未定义字段。
