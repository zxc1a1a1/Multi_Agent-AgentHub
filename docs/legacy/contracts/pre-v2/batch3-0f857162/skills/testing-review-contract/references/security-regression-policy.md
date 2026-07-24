# Security Regression Policy

安全回归是 CI 和 Review 的必过项。

## 必测项

- 未鉴权 401。
- 无权限 403。
- 错误脱敏。
- AgentCard 无 secret。
- fixtures/logs 无真实 key。
- iframe sandbox。
- markdown 不执行危险 HTML。
- high-risk action 需要 confirm_action。
- `/internal/**` 不出现在公开 OpenAPI。
