# Fallback / Retry 规则

fallback / retry 是通用编排能力，不绑定具体 Agent。

## 策略

正式 `fallback.mode` 枚举：

```text
none
same_capability_alternative
lower_risk_plan
single_agent_fallback
fail_fast
```

`same_capability_alternative` 的候选排序规则：
- 必须优先选择 healthy Agent。
- healthy 优先是选择算法，不是独立的 `fallback.mode`。
- legacy `first_healthy_agent` 语义已归入此排序规则。

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
