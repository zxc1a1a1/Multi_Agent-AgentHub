# PlannerInput 规则

PlannerInput 是 Planner 的标准输入，不区分 LLM、规则、mention 或 manual Planner。

## 输入来源

- 当前用户消息。
- 会话类型。
- 必要历史消息或历史摘要。
- 可用 Agent 能力摘要。
- 前端当前可处理 Runtime Capability 摘要。
- 用户 @mention。
- 用户或 UI 手动选择。
- 运行约束。
- traceId。

## 规则

- 不得把无限历史直接塞入 PlannerInput。
- 不得传入 API key、token、数据库连接对象。
- `availableAgents` 必须是摘要。
- `mentions` 是路由提示，不是无校验执行命令。
- `runtimeCapabilities` 只代表前端可处理能力，不代表 Agent 能力。
