# Model Registry 规则

Model Registry 用于声明 Provider、模型名称和模型能力。

长期 Provider 类型：

```text
openai
anthropic
gemini
openai_compatible
local
```

模型能力字段：

```text
provider
model
enabled
supportsStreaming
supportsStructuredOutput
supportsJsonSchema
supportsToolUse
supportsVision
maxInputTokens
maxOutputTokens
costClass
defaultTimeoutMs
```

规则：

- 不得在业务代码中硬编码模型能力。
- 不得因为 Provider 支持某能力，就默认所有模型支持该能力。
- 不得因为 OpenAI-compatible Provider 暴露 OpenAI 风格 API，就默认其支持 OpenAI 官方所有能力。
- Provider Adapter 必须根据官方文档和实际测试声明能力。

MVP 可以只注册一个 Provider 和一个模型。
