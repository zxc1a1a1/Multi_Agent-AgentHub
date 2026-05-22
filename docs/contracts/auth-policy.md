# Auth Policy

## 1. 文档目的

本文档定义 AgentHub 鉴权策略，包括 MVP 固定 Token、Post-MVP JWT、A2A service-to-service auth 和资源权限校验。

## 2. MVP v0.1

MVP 使用：

```text
Authorization: Bearer <fixed-token>
```

规则：

- Token 来自环境变量。
- Token 不得硬编码到代码。
- Token 不得出现在 query string。
- 无 Token 请求返回 401。
- Token 错误返回 401。
- 错误响应不得泄漏内部细节。

## 3. Post-MVP JWT

Post-MVP 可扩展：

- 登录。
- JWT access token。
- refresh token。
- 用户权限。
- 会话权限。
- Artifact 下载权限。
- Agent 管理权限。

规则：

- JWT signing secret 来自安全密钥管理或环境变量。
- JWT 不得打印日志。
- refresh token 必须有过期和撤销策略。
- 不允许把 JWT 转发给 Child Agent。

## 4. A2A 内部鉴权

MVP：

```text
Docker 内部网络隔离 + 不暴露给 Frontend
```

Post-MVP：

```text
service-to-service token 或 mTLS
```

规则：

- A2A metadata 不得携带用户 token。
- A2A endpoint 不得暴露给 Frontend。
- Gateway handler 不直接调用 A2A endpoint。
- Orchestrator / A2A Client 是唯一调用边界。

## 5. 资源权限

需要权限控制的资源：

- conversations
- messages
- runs
- artifacts
- agents
- tool_calls
- approvals

规则：

- 用户只能访问自己的 conversation。
- Artifact 下载必须校验权限。
- 高危操作必须 confirm_action。
- 自建 Agent Post-MVP 必须有所有权和权限隔离。

## 6. CORS / CSRF

MVP：

- Gateway 只允许前端开发域名。
- AG-UI endpoint 需要鉴权。
- Token 不放 query string。

Post-MVP：

- CORS 白名单。
- SameSite cookie 策略。
- CSRF token 或 Bearer token 明确策略。
- 不允许 `Access-Control-Allow-Origin: *` 配合 credentials。
