# LLM Debugging

## 目的

定义 LLM 请求、流式输出、结构化输出和 Provider 错误的排障字段。

## 必须字段

- `llmRequestId`
- `providerName`
- `modelId`
- `useCase`
- `requestTimeoutMs`
- `durationMs`
- `tokenInput`
- `tokenOutput`
- `finishReason`
- `errorCode`
- `retryAttempt`
- `fallbackAttempt`

## 禁止记录

- 完整 prompt
- 完整用户输入
- 完整 Provider raw response
- API key
- Authorization header

## 规则

- structured output validation 失败必须有独立错误码。
- Provider timeout 与本地 context cancelled 必须区分。
- retry 与 fallback 必须记录 attempt。
- token usage 可以统计，不得携带敏感内容。
