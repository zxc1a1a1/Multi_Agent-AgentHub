# Task Endpoints

来源：`docs/contracts/a2a-task.md`、`docs/contracts/a2a-agent-card.md`。

MVP 必需：

- `GET /health`
- `POST /a2a/tasks/sendSubscribe`
- `GET /.well-known/agent.json`

Post-MVP 可扩展：

- `POST /a2a/tasks/send`
- `GET /a2a/tasks/{id}`
- `DELETE /a2a/tasks/{id}/cancel`（或兼容 `POST /a2a/tasks/{id}/cancel`）

规则：前端不得直接调用以上 A2A endpoint。
