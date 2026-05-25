# Usage / Cost Policy

LLM 调用应记录 usage 与成本等级，用于观测和调优。

## 推荐字段

```ts
type TokenUsage = {
  inputTokens?: number
  outputTokens?: number
  totalTokens?: number
  cachedInputTokens?: number
  reasoningTokens?: number
  costClass?: 'free' | 'low' | 'medium' | 'high' | 'unknown'
}
```

## 规则

- Provider 不返回 usage 时，标记 unknown，不得填 0。
- fallback 后应记录每次 attempt 的 usage 摘要。
- 成本信息不得作为唯一安全控制。
