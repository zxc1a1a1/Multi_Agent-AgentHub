# Frontend API Client Policy

来源：`docs/contracts/openapi.yaml`、`docs/contracts/api-client-generation.md`、`docs/skill/platform-api-contract/SKILL.md`。

## 1. 类型来源

- 前端 API 类型必须来自 OpenAPI 生成结果。
- 禁止长期手写后端 response 类型。

## 2. 修改顺序

```text
先改 openapi.yaml
→ 更新生成类型
→ 修改前端调用
→ 修改后端实现
→ 补 contract test
```

## 3. 禁止事项

- 调用未定义接口。
- 私自增加 response 字段。
- 混入 AG-UI / A2A 内部字段到 REST DTO。
