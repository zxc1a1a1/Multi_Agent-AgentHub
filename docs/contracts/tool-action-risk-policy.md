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
- 高风险动作必须 confirm_action（HITL 确认）。
- 高风险动作必须审计。
- LLM 不得直接执行高风险动作。

## HITL confirmation flow

When the Orchestrator encounters a medium or high risk task:

1. Orchestrator emits a `tool_action_confirm` SSE event to the Gateway/Frontend.
2. Frontend renders a confirmation dialog showing action name, risk level, description, and parameters.
3. User confirms or rejects within the timeout window.
4. Frontend sends `POST /api/runs/{runId}/confirm` with `{ actionId, confirmed, rejectReason }`.
5. Gateway forwards the response to the Orchestrator.
6. Orchestrator proceeds (confirmed) or skips (rejected/timeout) the task.

The Orchestrator's validator (`validator.go:191`) gates high-risk tasks: they are
rejected for auto-execution when HITL is not enabled. Set `HITL_ENABLED=true` to
allow high-risk tasks with HITL confirmation.
