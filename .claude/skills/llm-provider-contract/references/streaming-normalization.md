# Streaming Normalization 规则

Provider 原始 stream event 不得直接进入 ADK Runtime、A2A 或 AG-UI。

长期统一事件类型：

```text
message_start
delta_text
tool_call_start
tool_call_delta
tool_call_end
usage_delta
message_end
error
```

MVP 阶段只强制：

```text
delta_text
message_end
error
```

`LLMStreamEvent` 建议包含：

```text
type
textDelta
toolCallId
toolName
toolArgsDelta
usage
finishReason
provider
model
providerMetadata
traceId
```

禁止：

- 将 Anthropic / OpenAI / Gemini 原始事件直接传给 ADK Runtime。
- 将 Provider 原始事件直接转成 AG-UI 事件。
- 在 streaming event 中暴露 API key 或 provider 原始敏感错误。
- 忽略 context cancellation。
