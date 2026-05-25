# Run API 策略

推荐路径：

- `POST /api/runs`
- `GET /api/runs/{runId}`
- `POST /api/runs/{runId}/cancel`

兼容路径：

- `POST /api/agui/run`
- `POST /api/agui/run/{runId}/cancel`

新增 API 优先使用 `/api/runs`。Run 创建不展开 Orchestrator 内部计划。取消 Run 不应返回 500。
