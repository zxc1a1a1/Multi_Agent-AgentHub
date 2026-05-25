# 能力校验规则

Plan Validation 必须验证每个 TaskPlan 是否能被目标 Agent 支持。

## capabilityId 来源

`TaskPlan.capabilityIds` 必须来自目标 Agent 的 `AgentCard.skills[].id`。不得凭自然语言临时生成未注册的 capabilityId，不得将 `toolName`、`artifact.type` 或 `outputMode` 作为 capabilityId。

## 校验项

- Agent 是否存在。
- Agent 是否 enabled。
- Agent 是否 healthy。
- `capabilityIds` 是否存在（必须在 `AgentCard.skills[].id` 中存在）。
- `expectedOutputs` 是否被目标能力支持。
- 输入类型是否兼容。
- 任务数量是否超过约束。

## 禁止

- 用 agentName 推断能力。
- 使用 LLM 编造的 skill / capability。
- 未校验能力就执行。
- 把 `toolName` 当作 capabilityId。
- 把 `artifact.type` 当作 capabilityId。
- 把 `outputMode` 当作 capabilityId。
