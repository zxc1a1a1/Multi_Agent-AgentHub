# Provider Registry

Provider Registry 是 LLM Provider 配置与状态的事实源。

## 目标

- 统一管理 Provider。
- 避免业务代码写死 Provider SDK。
- 避免真实 secret 进入配置文件。
- 支持 enabled / disabled / experimental / deprecated 状态。

## 推荐字段

```ts
type ProviderDefinition = {
  providerName: string
  providerType: 'openai' | 'anthropic' | 'gemini' | 'openai_compatible' | 'local'
  status: 'enabled' | 'disabled' | 'experimental' | 'deprecated'
  baseURL?: string
  apiKeyEnv?: string
  defaultModel?: string
  timeoutMs: number
  supportsStreaming: boolean
  supportsStructuredOutput: boolean
  supportsToolUse: boolean
}
```

## 规则

- `providerName` 必须唯一。
- 配置只保存 `apiKeyEnv`，不保存真实 API key。
- disabled Provider 不得被请求选择。
- experimental Provider 不得作为默认路径。
- openai-compatible Provider 必须独立声明能力，不能假定与 OpenAI 官方完全一致。
