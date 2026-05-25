# Security Test Policy

## 必测项

- 未鉴权返回 401。
- 无权限返回 403 或安全 404。
- `/internal/**` 不可被前端访问。
- Gateway → Orchestrator 无 service token 时失败。
- AgentCard 无 secret。
- Health Check 不暴露环境变量或堆栈。
- 错误响应无 stack trace / API key。
- iframe 有 sandbox。
- high-risk tool 需要 confirm_action。
- fallback 不调用 disabled / unhealthy Agent。
