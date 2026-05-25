# Agent API 策略

Agent API 只返回前端展示和选择所需的摘要信息。

推荐字段：`id`、`name`、`displayName`、`description`、`status`、`health`、`capabilities`、`inputModes`、`outputModes`、`tags`、`version`、`updatedAt`。

`status` 与 `health` 分离：
- `status`：生命周期/启用状态（`enabled` / `disabled` / `experimental` / `deprecated`）。
- `health`：当前健康状态（`healthy` / `degraded` / `unhealthy` / `unknown`）。
- `disabled` 属于 `status`，不属于 `health`。

AgentSummary 是脱敏公开摘要，**不得包含内部调用 URL**（如 Agent service name、Agent URL、内部端口）。Agent URL 属于 Registry / Orchestrator 内部配置，不得通过 Public API 暴露给 Frontend。不得暴露内部 healthcheck 细节（如原始探针 latency、连续失败次数等）。

`CapabilitySummary.id` 是 `AgentCard.skills[].id` 的公开摘要投影，非独立 ID。`capabilityIds` 的事实源是 `AgentCard.skills[].id`。

不得写死 `code-agent`、`web-agent`、`doc-agent`。前端不得通过 Agent 名称推断能力。

禁止：
- 把 `toolName` 当作 capabilityId。
- 把 `artifact.type` 当作 capabilityId。
- 把 `outputMode` 当作 capabilityId。
