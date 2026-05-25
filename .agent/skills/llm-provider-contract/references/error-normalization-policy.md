# Error Normalization Policy

Provider 错误必须归一化为 `SafeLLMError`。

## 推荐错误码

```text
LLM_BAD_REQUEST
LLM_UNAUTHORIZED
LLM_RATE_LIMITED
LLM_TIMEOUT
LLM_CONTEXT_CANCELLED
LLM_PROVIDER_UNAVAILABLE
LLM_STRUCTURED_OUTPUT_INVALID
LLM_STREAM_INTERRUPTED
LLM_INTERNAL
```

## 规则

- Provider 原始错误不得直接返回给用户。
- 用户可见 message 必须脱敏。
- 内部日志可记录 provider error code，但不得记录 secret。
- schema validation 失败是本地错误，不应伪装成 Provider 成功。
