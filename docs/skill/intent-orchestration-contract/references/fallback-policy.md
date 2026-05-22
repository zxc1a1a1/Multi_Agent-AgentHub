# Fallback 策略

## 1. 目的

本文定义意图编排失败或任务失败时的 fallback 规则。

MVP 阶段不实现 fallback。

## 2. fallback 类型

正式开发阶段可以支持：

- agent fallback。
- skill fallback。
- deterministic routing fallback。
- user clarification fallback。
- partial result fallback。

## 3. fallback 配置

fallback 必须声明：

- fallbackAgent。
- fallbackSkill。
- fallback 条件。
- fallback 最大次数。
- 是否继承上下文。
- 是否继承 partial artifacts。
- 失败后的最终错误。
- trace 记录方式。

## 4. 失败场景

常见失败场景：

| 场景 | 处理 |
|---|---|
| Agent 不存在 | 尝试 fallbackAgent 或返回错误 |
| Agent disabled | 尝试 fallbackAgent 或返回错误 |
| skill 不存在 | 不执行，返回 contract error |
| planner validation failed | 不执行，可 deterministic fallback |
| A2A call failed | 可 fallback 或 RUN_ERROR |
| task timeout | 可 fallback 或 RUN_ERROR |
| parallel partial failed | 聚合成功部分，标记失败部分 |
| sequential dependency failed | 下游任务不得执行 |

## 5. 禁止事项

不得：

- 在 MVP 阶段启用 fallback。
- 无限 fallback。
- fallback 到不存在的 Agent。
- fallback 到 AgentCard 未声明的 skill。
- 隐藏 fallback 失败。
- 不记录 fallback trace。
