# Fallback / Retry 规则

Fallback 和 Retry 是当前通用编排能力。

## fallback mode

- `none`
- `same_capability_alternative`
- `lower_risk_plan`
- `single_agent_fallback`
- `fail_fast`

## 规则

- 必须有最大尝试次数。
- 不得无限循环。
- 不得选择 unhealthy Agent。
- fallback plan 必须重新校验。
- fallback 后必须记录实际执行 Agent。
- 所有 fallback 失败时必须返回 SafeError。
