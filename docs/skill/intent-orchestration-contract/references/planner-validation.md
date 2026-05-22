# Planner Validation 规则

## 1. 目的

本文定义 LLM planner 输出的结构化校验规则。

MVP 阶段不启用 LLM planner。

正式开发阶段启用 LLM planner 前，必须实现本规则。

## 2. 输出格式

LLM planner 必须输出 JSON。

输出必须符合：

```text
docs/contracts/execution-plan.schema.json
```

如果 provider 支持 structured outputs，应优先使用结构化输出能力。

即使使用 structured outputs，Orchestrator 仍必须执行本地 schema validation。

## 3. 输入约束

planner prompt 只能包含允许被选择的 Agent 和 skills。

必须从以下来源构造 planner 可选能力：

- Agent Registry。
- AgentCard.skills。
- 当前 conversation context。
- 用户消息。
- 安全策略。

## 4. 校验流程

校验流程：

```text
LLM output
→ parse JSON
→ schema validation
→ strategy validation
→ agent registry validation
→ AgentCard.skills validation
→ safety validation
→ executable plan
```

任一阶段失败，不得执行。

## 5. 失败处理

planner 输出无效时：

- 不执行 plan。
- 记录结构化日志。
- 返回用户安全错误。
- 可回退到 deterministic routing，但必须明确记录 fallback reason。

## 6. 禁止事项

不得：

- 执行自然语言 planner 输出。
- 执行 JSON parse 失败的输出。
- 执行 schema validation 失败的输出。
- 让 LLM 创造 Agent。
- 让 LLM 创造 skill。
- 让 LLM 绕过安全策略。
- 把 planner prompt 中的 secret 写入日志。
