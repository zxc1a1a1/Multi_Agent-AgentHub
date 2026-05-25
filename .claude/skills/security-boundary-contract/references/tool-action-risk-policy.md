# Tool Action Risk Policy

## Tool Action 字段

- actionType
- riskLevel
- requiredPermission
- requiresConfirmation
- timeoutMs
- auditRequired
- allowedScope

## 风险等级

低风险：只读、无外部副作用。

中风险：可能生成文件、下载内容或访问受限资源。

高风险：可能执行命令、部署、覆盖文件、发布外部资源、修改凭证或删除数据。

## 规则

- Tool 参数必须 schema validation。
- 高风险动作必须 confirm_action。
- 高风险动作必须审计。
- LLM 不得直接执行高风险动作。
