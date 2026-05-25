# LLMRequest / LLMResponse

所有模型调用必须使用统一请求与响应结构。

## LLMRequest 要求

- 必须包含 requestId。
- 应包含 traceId。
- messages 必须结构化。
- metadata 不得包含 secret。
- structuredOutput.enabled 时必须提供 schema 或 schema 引用。

## LLMResponse 要求

- 必须包含实际 providerName / modelId。
- 不得直接包含 Provider 原始 response。
- structured 字段必须通过本地 schema validation。
- error 必须是 SafeLLMError。

## 兼容规则

如果 Provider 不返回 usage，应标记为 unknown，不得填 0 假装已知。
