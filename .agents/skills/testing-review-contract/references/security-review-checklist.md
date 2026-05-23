# 安全 Review Checklist

## 1. 目的

本文定义 AgentHub 安全 review 检查项。

安全 review 参考 OWASP Web Security Testing Guide 的方向，但以项目 contract 为准。

## 2. MVP 检查项

MVP 至少检查：

- 固定 Token 鉴权是否生效。
- 未授权请求是否被拒绝。
- 输入是否做基本校验。
- 错误是否脱敏。
- API key 不进入日志。
- API key 不进入前端响应。
- API key 不进入 Artifact。
- Provider 原始错误不直接暴露。
- Tool Call args 校验失败不执行。
- CodePreview 不执行代码。

## 3. 正式开发检查项

正式开发阶段增加：

- 用户权限模型。
- Artifact 下载鉴权。
- Object Storage signed URL 不泄漏。
- file_upload / deploy / shell 等危险操作需要确认。
- CORS / CSRF 策略。
- Rate limit。
- Audit log。
- Secret rotation。
- Dependency vulnerability scan。

## 4. 禁止事项

不得：

- 在测试或日志中出现真实 secret。
- 把 stack trace 暴露给用户。
- 前端直连子 Agent。
- 子 Agent 绕过 A2A。
- 大型 Artifact 进入 AG-UI token 流。
- 未确认执行危险操作。
