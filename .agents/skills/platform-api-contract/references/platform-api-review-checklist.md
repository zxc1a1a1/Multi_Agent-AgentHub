# Platform API Review Checklist

## 边界

- 是否只定义 Frontend ↔ Gateway API？
- 是否没有暴露 `/internal/**`？
- 是否没有暴露 Orchestrator Service endpoint？
- 是否没有暴露 Child Agent endpoint？

## 通用性

- 是否没有写死具体 Agent 名称？
- Agent 能力是否来自摘要字段？
- Conversation 是否支持 `single | group`？
- Message 是否支持多 Agent sender？

## OpenAPI

- 是否更新 `docs/contracts/openapi.yaml`？
- 是否使用 OpenAPI 3.1？
- operationId 是否稳定？
- request / response / error schema 是否完整？
- securitySchemes 是否声明？

## 前端和 Gateway

- 是否由 OpenAPI 生成类型？
- 是否没有手写 API response 类型？
- Handler 是否不做 Agent 编排？
- Handler 是否不直接调用 Child Agent 或 LLM Provider？
- Handler 是否不返回 OpenAPI 未定义字段？
