# LLM Provider Contract

本文是 AgentHub LLM Provider 适配层的正式契约。

## 目标

统一所有 LLM 调用，避免业务代码直接绑定 Provider SDK 或 Provider 原始响应结构。

## 核心对象

- ProviderDefinition
- ModelDefinition
- LLMRequest
- LLMResponse
- LLMStreamEvent
- TokenUsage
- SafeLLMError
- RetryPolicy
- RateLimitPolicy
- FallbackPolicy
- PromptTemplate

## 核心规则

1. 所有模型调用必须通过 Provider Adapter。
2. Gateway Service 不直接调用 LLM Provider。
3. Provider secret 只能来自环境变量或 secret manager。
4. 结构化输出必须本地 schema validation。
5. 流式事件必须归一化。
6. Provider 错误必须脱敏。
7. fallback 必须满足能力兼容。
8. 本契约不绑定具体 Agent 名称。
