# Logging / Audit Policy

## 必须审计事件

- 登录失败
- 未授权访问
- service-to-service auth 失败
- permission denied
- Agent Registry 变更
- Agent health 异常
- LLM Provider key 更新
- high-risk tool call
- confirm_action
- file upload / download
- deploy
- run_command
- fallback / retry

## 日志规则

- 使用结构化日志。
- 记录 requestId / traceId / runId。
- 不记录 secret。
- 不默认记录完整 Prompt、完整用户隐私输入、完整 LLM 原始响应。
- 审计日志必须可追踪且脱敏。
