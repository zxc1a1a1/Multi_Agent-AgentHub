# Registry / Health Test Policy

Agent Registry 与 Health Check 是 v1.0 当前门禁。

## 必测场景

- valid AgentCard accepted。
- invalid AgentCard rejected。
- duplicate agentName deterministic handling。
- unhealthy Agent excluded。
- disabled Agent not selected。
- health check timeout。
- `/api/agents` 只返回安全摘要。

## 通用性

测试必须使用通用 fixture，例如 `example-agent-a`、`example-agent-b`。
