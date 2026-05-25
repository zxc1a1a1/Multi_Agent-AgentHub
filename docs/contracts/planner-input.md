# PlannerInput Contract

PlannerInput 是 Planner 的标准输入。

## 字段

| 字段 | 说明 |
|---|---|
| `runId` | 当前 Run ID |
| `conversationId` | 当前对话 ID |
| `conversationType` | `single` 或 `group` |
| `userMessage` | 当前用户输入 |
| `historySummary` | 可选历史摘要 |
| `messages` | 受控历史消息 |
| `availableAgents` | 当前可用 Agent 能力摘要 |
| `runtimeCapabilities` | 前端可处理能力摘要 |
| `mentions` | 用户显式 @mention |
| `manualSelectedAgents` | 手动选择的 Agent |
| `planningMode` | 路由模式 |
| `constraints` | 最大 task、最大 retry 等约束 |
| `traceId` | 追踪 ID |

## 安全规则

- 不得包含 API key、token、数据库连接串。
- 不得包含完整敏感 system prompt。
- 历史消息必须裁剪或摘要化。
- `availableAgents` 只传能力摘要。
