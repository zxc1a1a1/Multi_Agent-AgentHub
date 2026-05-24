# API Review Checklist

来源：`docs/contracts/api-review-checklist.md`。

- [ ] OpenAPI 是否已更新且字段一致。
- [ ] 是否仅定义 Frontend ↔ Gateway REST API。
- [ ] 响应 / 错误 / 分页格式是否统一。
- [ ] 鉴权是否使用 Bearer Header。
- [ ] 是否未暴露 `/internal/*` 与 A2A endpoint。
- [ ] 前端是否使用生成类型。
- [ ] 后端是否无未定义字段返回。
- [ ] 是否满足 MVP 必需接口。
