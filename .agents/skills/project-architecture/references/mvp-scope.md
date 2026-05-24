# MVP Scope

来源：`docs/architecture/overview.md`、`docs/architecture/service-boundaries.md`、`docs/contracts/openapi.yaml`（MVP 必需接口段落）。

## 1. MVP v0.1 必须跑通

```text
用户发消息
→ Gateway
→ Orchestrator
→ code-agent
→ A2A 流式响应
→ AG-UI 流式回复
→ code_preview
```

## 2. MVP 允许的简化

- Orchestrator 嵌入 Gateway 进程（但边界不合并）。
- 当前数据库采用 MySQL 8。
- 固定 Token 或环境变量 Token。
- Agent 可由配置静态注册（`code-agent`）。

## 3. MVP 必需能力

- `POST /api/agui/run`。
- 最小会话与消息查询接口。
- A2A `sendSubscribe` 主链路。
- `code` Artifact 映射 `code_preview`。
