# REST API Policy

来源：`docs/contracts/openapi.yaml`、`docs/contracts/api-review-checklist.md`、`docs/skill/platform-api-contract/SKILL.md`。

## 1. 边界

- 本契约只覆盖 Frontend ↔ Gateway REST API。
- 不展开 AG-UI 事件语义。
- 不暴露 Gateway-Orchestrator 内部接口与 A2A endpoint。

## 2. 统一格式

- 成功：`code / data / message`。
- 错误：`code / data:null / message`。
- 分页：`list / total / page / pageSize`。

## 3. MVP 必需接口（v0.1）

- `GET /api/conversations`
- `POST /api/conversations`
- `GET /api/conversations/{id}/messages`
- `GET /api/agents`
- `POST /api/agui/run`（仅 HTTP 形态）
