# REST 资源策略

Platform API 应使用稳定、可理解、可分页、可测试的资源模型。

## 推荐资源

- `/api/conversations`
- `/api/conversations/{conversationId}`
- `/api/conversations/{conversationId}/messages`
- `/api/agents`
- `/api/agents/{agentName}`
- `/api/runs`
- `/api/runs/{runId}`
- `/api/artifacts/{artifactId}`

## 禁止

- 路径中写死具体 Agent 名称。
- 路径中暴露内部协议名。
- 使用 `/internal/**` 作为前端公开 API。
- 用一个万能 `/api/action` 承载所有平台行为。
