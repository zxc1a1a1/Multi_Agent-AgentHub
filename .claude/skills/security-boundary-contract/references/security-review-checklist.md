# Security Review Checklist

来源：`docs/contracts/security-review-checklist.md`。

- [ ] `.env` 与密钥未入库、未入日志。
- [ ] 鉴权与 Bearer 头符合规范。
- [ ] 错误脱敏。
- [ ] A2A endpoint 不暴露前端。
- [ ] `code_preview` 仅展示。
- [ ] commit 前完成敏感信息扫描。
