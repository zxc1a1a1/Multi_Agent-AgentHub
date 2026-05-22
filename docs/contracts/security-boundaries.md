# Security Boundaries Contract

## 1. 文档目的

本文档定义 AgentHub 的安全边界。

核心原则：

```text
Frontend 不可信
Child Agent 不完全可信
用户输入不可信
LLM 输出不可信
Artifact 内容不可信
文件上传不可信
工具执行不可信
错误信息不能泄漏内部细节
高危操作必须 confirm_action
密钥只能来自环境变量或安全密钥管理
```

## 2. MVP v0.1 范围

MVP v0.1 允许：

- 固定 Token 或环境变量 Token。
- 不做完整用户注册登录。
- 不做完整 JWT 签发 / 刷新。
- A2A 运行在 Docker 内部网络。
- 暂不开放 file_upload。
- 暂不开放 run_command。
- 暂不实现部署发布。
- 只实现 `code_preview`。

MVP v0.1 仍然必须：

- Frontend 只能调用 Gateway。
- Gateway 鉴权 `/api/*`。
- Token 不得放 query string。
- 错误不得泄漏 stack trace / token / API key。
- LLM API key 只能来自环境变量。
- Child Agent 日志不得打印 API key。
- `code_preview` 只展示代码，不执行代码。
- A2A endpoint 不暴露给 Frontend。
- Gateway handler 不直接调用 A2A endpoint。

## 3. Frontend 边界

Frontend 允许：

- 调用 Gateway REST API。
- 连接 Gateway AG-UI SSE。
- 执行已注册 Frontend Runtime Skill。
- 展示 `code_preview`。

Frontend 不允许：

- 直接调用 Orchestrator。
- 直接调用 Child Agent。
- 直接调用 A2A endpoint。
- 直接访问 LLM Provider。
- 保存 LLM API key。
- 自动执行危险 Tool Call。
- 绕过 `confirm_action`。

## 4. Gateway 边界

Gateway 负责：

- 对外鉴权。
- CORS。
- requestId / traceId。
- REST API 权限校验。
- AG-UI endpoint 鉴权。
- 用户资源访问控制。
- 错误脱敏。

Gateway 不负责：

- 保存明文 API key。
- 直接执行 LLM 生成命令。
- 直接调用 Child Agent。
- 直接信任 Frontend 传来的 AgentName / toolName。
- 在错误中输出内部堆栈。

## 5. Orchestrator 边界

Orchestrator 负责：

- 校验 Agent 是否来自 Registry / 配置白名单。
- 校验 Frontend 声明的 Tools / Skills 是否白名单。
- 将 A2A 错误脱敏后转为 RUN_ERROR。
- 不把用户 token 传给 Child Agent。
- 不把敏感 system prompt 传给不可信 Agent。

Orchestrator 不允许：

- 直接暴露给 Frontend。
- 直接处理浏览器鉴权。
- 直接信任 LLM 输出的 AgentName。
- 把 Gateway Authorization token 放进 A2A metadata。

## 6. Child Agent 边界

Child Agent 必须：

- 只通过 A2A endpoint 被 Orchestrator 调用。
- 暴露 AgentCard，但不泄漏密钥。
- 不打印 LLM API key。
- 不打印完整敏感 prompt。
- 不访问超出 workspace 的文件。
- 不执行未授权命令。
- Artifact 输出必须经过 Orchestrator / Artifact Contract / Frontend Runtime Skill 处理。

## 7. A2A 内部通信

MVP v0.1：

- A2A 可运行在 Docker 内部网络。
- 不暴露给公网。
- 不暴露给 Frontend。
- Gateway handler 不直接调用。

Post-MVP：

- A2A 调用必须使用 service-to-service token 或 mTLS。
- A2A metadata 不得携带用户 token。
- A2A 请求必须携带 traceId。
- A2A endpoint 必须限流。
- A2A 错误必须脱敏。

## 8. Artifact 安全

- `code_preview` 只展示代码，不执行。
- `web_preview` 必须使用 iframe sandbox。
- HTML 内容必须隔离。
- 文件下载必须鉴权。
- 私有 Artifact URL 必须短期有效。
- Artifact metadata 不得包含密钥。
- deploy Artifact / deploy action 必须走 `confirm_action`。

## 9. 错误脱敏

AG-UI `RUN_ERROR` 不得包含：

- stack trace。
- API key。
- token。
- 数据库连接串。
- 内部路径。
- 内部服务地址。
- 完整 system prompt。

REST ErrorResponse 只能包含：

```text
code
data: null
message
```
