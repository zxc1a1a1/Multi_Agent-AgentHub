# LLM Planner 校验规则

LLM Planner 可以参与意图编排，但不得直接驱动执行。

## 必须

- 输出结构化 JSON。
- 使用 schema 约束输出。
- 本地再次 schema validation。
- 校验 Agent / capability / expectedOutputs。
- 校验安全字段。

## 禁止

- 执行自然语言 plan。
- 相信 LLM 自称的 validated 状态。
- 让 LLM 直接调用 Agent。
- 在 prompt 中放入 API key、token、完整敏感 prompt。
- 把原始 LLM 长输出直接落库或展示给用户。
