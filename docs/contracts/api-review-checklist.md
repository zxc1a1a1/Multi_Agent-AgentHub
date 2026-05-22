# AgentHub REST API Review Checklist

用于评审 Frontend ↔ Gateway REST API Contract 与实现一致性。

## Contract 与 OpenAPI

- [ ] `docs/contracts/openapi.yaml` 是否已更新。  
- [ ] 每个接口是否都有稳定 `operationId`。  
- [ ] 是否只定义 Frontend ↔ Gateway 的 REST API。  

## 响应格式

- [ ] 成功响应是否符合 `code / data / message`。  
- [ ] 错误响应是否符合 `code / data:null / message`。  
- [ ] 分页响应是否符合 `list / total / page / pageSize`。  

## 鉴权与字段规范

- [ ] 鉴权是否使用 `bearerAuth`（`Authorization: Bearer <token>`）。  
- [ ] JSON 字段是否统一使用 `camelCase`。  

## 协议边界

- [ ] 是否没有把 AG-UI 事件写进 REST schema。  
- [ ] 是否没有把 A2A endpoint 暴露给 Frontend。  
- [ ] 是否没有把 `/internal/runs` 暴露给 Frontend。  

## 前后端一致性

- [ ] 前端是否不能手写 response 类型（应由 OpenAPI 生成）。  
- [ ] 后端是否不能返回未定义字段（必须与 OpenAPI 一致）。  

## MVP v0.1 API 检查项

MVP v0.1 必须实现：

- [ ] `GET /api/conversations`
- [ ] `POST /api/conversations`
- [ ] `GET /api/conversations/{id}/messages`
- [ ] `GET /api/agents`
- [ ] `POST /api/agui/run`

MVP v0.1 暂不强制实现，但可保留在 OpenAPI 中作为 planned：

- [ ] `GET /api/conversations/{id}`
- [ ] `PATCH /api/conversations/{id}`
- [ ] `DELETE /api/conversations/{id}`
- [ ] `GET /api/agents/{name}/card`
- [ ] `POST /api/agents/custom`
- [ ] `PATCH /api/agents/custom/{id}`
- [ ] `DELETE /api/agents/custom/{id}`
- [ ] `GET /api/artifacts/{id}`
- [ ] `GET /api/artifacts/{id}/preview`
- [ ] `POST /api/agui/run/{runId}/cancel`
- [ ] `POST /api/agui/run/{runId}/tool-result`

MVP v0.1 仍然必须检查：

- [ ] 成功响应保持 `code / data / message`。
- [ ] 错误响应保持 `code / data: null / message`。
- [ ] 列表响应保持 `list / total / page / pageSize`。
- [ ] 鉴权形式保持 `Authorization: Bearer <token>`。
- [ ] JSON 字段保持 camelCase。
- [ ] `POST /api/agui/run` 不展开 AG-UI event schema。
- [ ] A2A endpoint 不暴露成 Frontend REST API。
