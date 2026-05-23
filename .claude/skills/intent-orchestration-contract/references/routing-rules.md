# 路由规则

## 1. 目的

本文定义 Orchestrator 如何选择目标 Agent 和 skill。

路由结果必须写入 ExecutionPlan。

## 2. 长期路由优先级

长期路由优先级：

```text
1. 用户显式 @Agent
2. 当前 conversation 默认 Agent
3. 用户创建对话时选择的 Agent
4. LLM planner 基于 AgentCard 选择
5. fallback Agent
```

## 3. MVP 路由规则

MVP 阶段只使用：

```text
req.AgentName
或 conversation.agentName
```

MVP 阶段默认目标：

```text
code-agent
```

MVP 阶段默认 skill：

```text
code_generate
```

## 4. @Agent 规则

正式开发阶段启用 @Agent 后：

- @Agent 优先级高于 LLM planner。
- @Agent 必须能解析到 Agent Registry 中存在的 Agent。
- @Agent 指定的 Agent disabled 时不得执行。
- @Agent 指定的 Agent 不存在时应返回用户安全错误。

## 5. LLM planner 路由

LLM planner 只能从 Agent Registry / AgentCard 中选择 Agent 和 skill。

不得创造不存在的 Agent 或 skill。

## 6. 禁止事项

不得：

- 在 MVP 阶段启用复杂自动路由。
- 跳过 Agent Registry。
- 跳过 AgentCard.skills 校验。
- 让 Gateway handler 直接实现复杂路由。
- 让用户输入中的任意字符串直接成为 Agent 名称。
