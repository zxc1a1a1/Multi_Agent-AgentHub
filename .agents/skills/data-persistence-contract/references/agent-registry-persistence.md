# Agent Registry 持久化规则

## 目的

AgentHub 需要保存 Agent 注册信息、能力摘要、健康状态和最近错误，支持 2+ Agent 与后续扩展。

## agents 表

关键字段：

```text
name
display_name
description
url
version
status
health
agent_card
skills
input_modes
output_modes
last_check_at
last_error_code
```

## agent_health_checks 表

关键字段：

```text
agent_name
agent_url
status
latency_ms
error_code
error_message
checked_at
```

## 规则

- `name` 必须唯一。
- 不固定具体 Agent 名称。
- Agent 是否可用不能只靠环境变量。
- 最近健康状态必须能被查询。
- 健康检查错误必须脱敏。
- AgentCard 可以保存摘要，但不得保存 secret。
- Agent 禁用时不应被正常调度。
- `url` 和 `agent_url` 字段是 Registry / Orchestrator 内部使用的调用地址，不得通过 Public API 暴露给 Frontend。

## Agent.status（生命周期/启用状态）

```text
enabled
disabled
experimental
deprecated
```

`disabled` 属于 `Agent.status`，不属于 `Agent.health`。

## Agent.health（当前健康状态）

由 Registry 周期性探测 `/health` 并归一化后写入：

```text
/health.status = ok       → healthy
/health.status = degraded → degraded
timeout / non-2xx / invalid response → unhealthy
未探测                        → unknown
```

枚举：

```text
healthy
degraded
unhealthy
unknown
```

## agent_health_checks.status（归一化后健康检查记录）

```text
healthy
degraded
unhealthy
unknown
```

## 禁止

- 把 Agent URL 写死在业务逻辑里且不落库。
- 把 API key 写入 AgentCard。
- 把健康检查结果只写日志。
- 只允许一个 Agent。
- 把 Agent URL 通过 Public API 暴露给 Frontend。
