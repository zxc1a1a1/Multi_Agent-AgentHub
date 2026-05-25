# E2E / Smoke Test Policy

E2E 只覆盖关键路径，Smoke Test 作为交付门禁。

## Docker Smoke Gate

- compose config valid。
- frontend / gateway / orchestrator / child agents / database 启动。
- health check 通过。
- `/api/agents` 返回安全摘要。
- simple run 返回 stream。
- group run 有多 Agent 消息。
- fallback 场景显示安全提示。

## Demo E2E

必须验证单聊、群聊、@mention、代码预览、网页预览、Markdown、刷新持久化。
