# Fallback / Retry Boundary

fallback / retry 是 v1.0 架构能力。

## 规则

- retry 必须有最大次数。
- fallback 必须有最大次数。
- fallback 不得选择不健康 Agent。
- fallback 必须选择能力兼容的目标。
- fallback 后必须记录实际执行 Agent。
- 用户可见提示必须安全、可理解。
- 主失败原因必须脱敏保留，不能完全吞掉。

## 边界

Gateway 不做 fallback 决策。Orchestrator 负责策略，Gateway 负责转发状态。
