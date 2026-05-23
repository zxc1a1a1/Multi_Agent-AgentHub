# Structured Output 规则

结构化输出适用于：

- ExecutionPlan。
- Tool args。
- JSON object generation。
- classification。
- routing。
- metadata extraction。

规则：

- 如果 Provider 支持官方 structured output，应优先使用。
- 如果 Provider 只支持 JSON mode，不得把 JSON mode 视为 schema adherence。
- 如果 Provider 只支持 JSON Schema 子集，Adapter 必须明确记录限制。
- 无论 Provider 是否支持 structured output，AgentHub 必须执行本地 schema validation。

流程：

```text
LLM output
→ parse JSON
→ schema validation
→ domain validation
→ executable object
```

禁止：

- 把 JSON mode 当成严格 schema 校验。
- 跳过本地 schema validation。
- 让 LLM 输出直接执行。
- 让 structured output 包含 secret。
