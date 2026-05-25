# 路由优先级规则

路由优先级用于把用户显式意图、会话上下文和自动 Planner 统一到一个结构化计划。

推荐优先级：

1. 用户或 UI 手动选择。
2. 显式 @mention。
3. 会话 direct target。
4. auto planner。
5. fallback planner。

## 规则

- 高优先级不是无条件执行。
- 不存在、禁用或不健康的目标必须拒绝或 fallback。
- 多个目标可生成 ordered_parallel 或 sequential。
- 所有路由结果必须统一成为 OrchestrationPlan。
