# OpenAPI 唯一事实源

`docs/contracts/openapi.yaml` 是 Frontend ↔ Gateway Platform API 的唯一事实源。

## 规则

- 新增公开 API 前先更新 OpenAPI。
- 修改字段前先更新 schema。
- 删除字段前必须标记 deprecation 或完成兼容迁移。
- OpenAPI 使用 `openapi: 3.1.0`。
- 每个 operation 必须有稳定 `operationId`。
- Gateway Handler、Frontend API Client、Mock、Contract Test 都以 OpenAPI 为准。

## 禁止

- 前端直接调用 OpenAPI 未定义接口。
- Gateway 返回 OpenAPI 未定义字段。
- Mock 数据超出 schema。
- 把 Orchestrator 内部 API 放入公开 OpenAPI。
