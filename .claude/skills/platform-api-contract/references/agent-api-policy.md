# Agent API 策略

Agent API 只返回前端展示和选择所需的摘要信息。

推荐字段：`id`、`name`、`displayName`、`description`、`status`、`health`、`capabilities`、`inputModes`、`outputModes`、`tags`、`version`、`updatedAt`。

不得写死 `code-agent`、`web-agent`、`doc-agent`。前端不得通过 Agent 名称推断能力。
