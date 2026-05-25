# Auth Policy

## Public Auth

Frontend → Gateway 必须使用用户身份认证。认证令牌只能通过 Authorization header 传递。

## Internal Auth

Gateway → Orchestrator 必须使用 service-to-service auth。service token 与用户 token 必须分离。

## Object-level Authorization

以下资源必须做对象级授权：

- conversation
- message
- run
- artifact
- tool call
- uploaded file
- agent configuration

## 错误语义

- 未认证：401。
- 已认证但无权限：403 或安全 404。
- 不得泄露他人资源是否存在的敏感细节。
