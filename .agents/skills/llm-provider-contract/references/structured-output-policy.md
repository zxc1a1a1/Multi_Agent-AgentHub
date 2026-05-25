# Structured Output Policy

结构化输出必须先校验，再执行。

## 三层校验

```text
Provider structured output
→ 本地 JSON parse
→ 本地 JSON Schema validation
```

## 规则

- JSON mode 不等于 schema validation。
- Provider 声称符合 schema，也必须本地校验。
- 校验失败不得执行下游动作。
- schema 必须版本化。
- 原始 LLM 输出不得直接作为计划、工具参数或配置使用。
