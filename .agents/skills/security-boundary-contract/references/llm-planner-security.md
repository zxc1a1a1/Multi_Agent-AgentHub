# LLM Planner Security

## 原则

LLM 输出和 Planner 输出都不可信。

## 规则

- Planner 输出必须 JSON parse + schema validation。
- Planner 输出必须校验 Agent、capability、output 和 permission。
- LLM 不得创造不存在的 Agent 或 capability。
- LLM 不得绕过对象级授权。
- LLM 不得绕过 confirm_action。
- Prompt 不得包含 API key、service token、数据库密码或完整内部拓扑。
- 不可信 Agent 不得接收系统级 Prompt 或 Provider key。
- fallback plan 必须重新校验。

## 失败处理

- 结构化输出失败不得执行下游动作。
- validation 失败必须返回安全错误或进入受控 fallback。
