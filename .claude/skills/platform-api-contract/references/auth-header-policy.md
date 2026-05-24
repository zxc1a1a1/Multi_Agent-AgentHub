# Auth Header Policy

来源：`docs/contracts/openapi.yaml`、`docs/contracts/auth-policy.md`、`docs/contracts/security-boundaries.md`。

## 1. 规则

- 使用 `Authorization: Bearer <token>`。
- Token 不得放入 query string。
- 未授权返回 401，权限不足返回 403。

## 2. MVP 说明

- MVP 可使用固定 Token 或环境变量 Token。
- 该简化不改变 Bearer Header 规范。

## 3. 边界

- Frontend 只向 Gateway 发送 token。
- 不向 Child Agent 透传用户 token。
