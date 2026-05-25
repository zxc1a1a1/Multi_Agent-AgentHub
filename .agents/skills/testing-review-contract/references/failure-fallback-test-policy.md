# Failure / Fallback Test Policy

v1.0 必须测试降级和失败路径。

## 必测场景

- LLM timeout。
- Planner invalid JSON。
- Agent unhealthy。
- Agent call failed。
- all agents failed。
- retrying STATE_UPDATE。
- max retry attempts。
- partial success 不被污染。

## 判定

失败必须返回 safe error，不得泄漏内部栈、token、provider raw error。
