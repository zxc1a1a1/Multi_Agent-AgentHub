# Contract Test 策略

## 1. 目的

本文定义项目 contract 和 JSON Schema 的测试规则。

## 2. 必须测试的 schema

至少覆盖：

```text
frontend-runtime-skills.schema.json
artifact.schema.json
execution-plan.schema.json
llm-provider.schema.json
observability-debugging.schema.json
testing-review.schema.json
```

## 3. 测试要求

每个 schema 必须包含：

- 合法样例。
- 非法样例。
- MVP implemented=true 样例。
- reserved / implemented=false 样例。
- 缺少必填字段样例。
- 字段类型错误样例。
- additionalProperties 错误样例。

## 4. Golden fixtures

协议转换和 schema 样例建议使用 golden fixtures。

示例：

```text
testdata/contracts/valid/
testdata/contracts/invalid/
testdata/protocol/
```

## 5. 禁止事项

不得：

- 改 schema 不改测试。
- 只测合法样例。
- schema validation 失败仍继续执行。
- 在 fixture 中放真实 secret。
