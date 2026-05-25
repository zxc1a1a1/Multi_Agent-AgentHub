# Fallback Policy

Fallback 是 Provider / Model 选择层的降级能力，不等于 retry。

## 模式

```text
same_provider_different_model
different_provider_same_capability
lower_cost_model
fail_fast
```

## 规则

- fallback 只能选择 enabled Provider / Model。
- fallback 必须满足 requiredCapabilities。
- fallback 不得无限循环。
- fallback 后必须记录实际 providerName / modelId。
- structured output 请求不得 fallback 到不支持 schema 的模型，除非显式降级并重新校验。
