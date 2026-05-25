# Planner Test Policy

Planner 测试必须确定性，不得依赖真实 LLM。

## 必测场景

- 合法 plan。
- 非法 JSON。
- 不存在 Agent。
- 不存在 capability。
- no healthy agents。
- @mention。
- group conversation。
- ordered_parallel。
- sequential dependsOn。
- fallback plan 重新校验。

## Fake LLM

Fake LLM 必须支持固定输出、错误输出、超时输出和 malformed 输出。
