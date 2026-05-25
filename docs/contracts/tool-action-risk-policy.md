# Tool Action Risk Policy

## 风险字段

- actionType
- riskLevel
- requiredPermission
- requiresConfirmation
- timeoutMs
- auditRequired
- allowedScope

## 高风险动作

- run_command
- deploy
- file_overwrite
- external_publish
- credential_update
- workspace_delete

## 规则

- 参数必须 schema validation。
- 高风险动作必须 confirm_action。
- 高风险动作必须审计。
- LLM 不得直接执行高风险动作。
