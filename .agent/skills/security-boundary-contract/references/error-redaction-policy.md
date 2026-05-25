# Error Redaction Policy

## 对外允许字段

- errorCode
- safeMessage
- requestId
- runId

## 禁止泄露

- stack trace
- API key
- service token
- internal URL
- database DSN
- absolute path
- system prompt
- raw provider response
- object storage signed credential

## 规则

- Provider 原始错误不得直接返回用户。
- Orchestrator 内部错误必须映射为安全错误。
- Agent 错误必须脱敏后才能进入前端事件。
- Debug dump 也必须脱敏。
