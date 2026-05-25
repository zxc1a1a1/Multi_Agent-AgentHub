# Model Registry

Model Registry 是模型能力、限制与使用策略的事实源。

## 推荐字段

```ts
type ModelDefinition = {
  modelId: string
  providerName: string
  status: 'enabled' | 'disabled' | 'deprecated'
  capabilities: {
    streaming: boolean
    structuredOutput: boolean
    jsonSchema: boolean
    toolUse: boolean
    vision: boolean
    reasoning: boolean
  }
  limits: {
    maxInputTokens?: number
    maxOutputTokens?: number
    defaultTimeoutMs?: number
  }
  costClass: 'free' | 'low' | 'medium' | 'high' | 'unknown'
  useCases?: string[]
}
```

## 规则

- 业务代码不得直接写死模型名。
- 模型能力必须由 registry 声明。
- Provider 支持某能力，不代表所有模型都支持。
- fallback 只能选择能力兼容模型。
- disabled Model 不得被新请求选择。
