# Fallback / Retry 规则

fallback / retry 是通用编排能力，不绑定具体 Agent。

## 策略

```text
none
same_capability_alternative
first_healthy_agent
fail_fast
```

## 规则

- fallback 不得选择不可用 Agent。
- fallback 不得无限重试。
- maxAttempts 必须明确。
- retry/fallback 必须输出 state_update。
- fallback 后的 message / task / result 必须记录实际执行 Agent。
- 所有候选失败时，run 必须 failed。
- 错误必须脱敏。

## 禁止

- 用具体 Agent 名称写死 fallback。
- 对同一个失败原因无限重试。
- fallback 时丢失 traceId / runId。
- fallback 成功后隐藏原始失败状态。
