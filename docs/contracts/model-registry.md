# Model Registry

Model Registry 记录模型能力、限制、状态和适用场景。

## 必填信息

- modelId
- providerName
- status
- capabilities
- costClass

## 能力字段

- streaming
- structuredOutput
- jsonSchema
- toolUse
- vision
- reasoning

## 规则

- 模型能力不得由代码猜测。
- fallback 必须选择能力兼容的模型。
- disabled 模型不得被新请求选择。
- Provider 能力不自动继承到所有模型。
