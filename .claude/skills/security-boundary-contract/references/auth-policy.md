# Auth Policy

来源：`docs/contracts/auth-policy.md`。

- 使用 `Authorization: Bearer <token>`。
- Token 不得出现在 query string。
- MVP 可使用固定 Token（环境变量注入）。
- Post-MVP 可扩展 JWT/refresh/权限体系。
