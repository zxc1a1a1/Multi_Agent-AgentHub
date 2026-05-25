# Child Agent Lifecycle

## 1. 生命周期阶段

一个 Child Agent 接入 AgentHub 的生命周期：

```text
define AgentCard
→ expose endpoints
→ Registry discovery
→ health check
→ Planner selection
→ Orchestrator task submit
→ streaming output
→ artifact mapping
→ completed / failed
→ fallback if needed
```

## 2. define AgentCard

新增 Agent 前必须先定义 AgentCard。

至少包括：

- name
- description
- url
- version
- capabilities
- skills
- inputModes
- outputModes

## 3. expose endpoints

必须暴露：

```text
GET /.well-known/agent.json
GET /health
POST /a2a/tasks/sendSubscribe
```

## 4. Registry discovery

Registry 拉取 AgentCard，并缓存能力信息。

AgentCard 无效时，Agent 不参与编排。

## 5. health check

Registry 周期性调用 `/health`。

unhealthy Agent 不进入 Planner 候选列表。

## 6. Planner selection

Planner 根据 AgentCard.skills / outputModes 选择一个或多个 Agent。

Planner 不应依赖硬编码 agentName 判断能力。

## 7. task submit

Orchestrator 使用 A2A Client 提交 task。

必须传递：

- task id
- messages
- runId
- threadId
- traceId
- agentName

## 8. streaming output

Agent 输出：

- working
- text
- artifact
- completed 或 failed

## 9. artifact mapping

Artifact 由 ProtocolConverter 转成 AG-UI Tool Call。

Child Agent 不直接输出前端 Tool Call。

## 10. fallback

Agent failed 且 retryable 时，Orchestrator 可尝试 fallback。

fallback 状态应通过 AG-UI STATE_UPDATE 告知前端。
