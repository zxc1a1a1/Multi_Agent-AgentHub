# Security Test Policy

## 最小测试项

- 未鉴权访问公开 API 返回 401。
- 无权限访问他人资源返回 403 或安全 404。
- `/internal/**` 不可被 Frontend 访问。
- Gateway → Orchestrator 没有 service token 时失败。
- AgentCard 不包含 secret。
- Health Check 不暴露堆栈或环境变量。
- 错误响应不包含 stack trace / API key。
- web_preview iframe 有 sandbox。
- markdown 不执行危险 HTML。
- high-risk tool 未确认不得执行。
- fallback 不调用 disabled / unhealthy Agent。
- smoke test 包含关键安全检查。

## 完成标准

安全测试失败时不得认为相关功能完成。
