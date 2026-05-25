# Registry Discovery

## 1. 定位

Agent Registry 负责管理所有 Child Agents。

它是 Planner / Orchestrator 的 Agent 能力来源。AgentCard 注册、刷新、健康检查的权威消费方是 Registry / Orchestrator。

Gateway 不得直接持有 Agent URL、不得基于 Agent URL 执行调度。Gateway 可以读取 Registry 脱敏摘要用于前端展示（`/api/agents`），但摘要不得包含内部调用 URL。

## 2. 职责

Registry 必须：

- 从配置读取多个 Agent URL。
- 拉取 `/.well-known/agent.json`。
- 校验 AgentCard。
- 调用 `/health`。
- 缓存 AgentCard。
- 缓存 healthy 状态。
- 向 Planner 提供 healthy agents。
- 向 Gateway API 提供 Agent 摘要（脱敏，不含内部 URL）。

## 3. 配置示例

```yaml
agents:
  - name: code-like-agent
    url: http://code-agent:8081
  - name: web-like-agent
    url: http://web-agent:8082
  - name: document-like-agent
    url: http://doc-agent:8083
```

这些名称只是示例，不是硬约束。

## 4. 健康检查

- Registry 应周期性调用 `/health`。
- unhealthy Agent 不进入 Planner 候选列表。
- Agent 恢复健康后可以重新进入候选列表。
- 健康状态变化应记录日志。

## 5. 边界

- Frontend 不直接访问 Registry 内部状态。
- Frontend 通过 Gateway API 查询 Agent 摘要。
- Gateway 可以展示 Registry 摘要，但不得基于 Agent URL 执行调度。
- Gateway 不得直接持有 Child Agent 调用地址。
- Registry 不调用 LLM。
- Registry 不处理用户任务。

## 6. Review 要点

- 是否支持多个 Agent。
- 是否不写死 Agent 名称。
- 是否能处理 AgentCard 拉取失败。
- 是否能处理健康检查失败。
- 是否能为 Planner 提供能力列表。
- Agent URL 是否只由 Registry / Orchestrator 持有，未被 Gateway 直接使用。
- Gateway 的 Agent 摘要是否脱敏，不包含内部调用 URL。
- `CapabilitySummary.id` 是否是 `AgentCard.skills[].id` 的公开投影（非独立 ID）。
- 是否没有把 `toolName`、`artifact.type`、`outputMode` 当作 capabilityId。
