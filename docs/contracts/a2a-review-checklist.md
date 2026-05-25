# A2A Review Checklist

## 1. 适用范围

Review 以下内容时使用本清单：

- 新增 Child Agent
- 修改 AgentCard
- 修改 `/health`
- 修改 `/a2a/tasks/sendSubscribe`
- 修改 A2A Client
- 修改 Agent Registry
- 修改 Orchestrator 调用 Agent 逻辑
- 修改 Artifact 输出和转换

## 2. AgentCard

- [ ] 是否暴露 `GET /.well-known/agent.json`？
- [ ] 是否包含 `name`？
- [ ] 是否包含 `description`？
- [ ] 是否包含 `url`？
- [ ] 是否包含 `version`？
- [ ] 是否包含 `capabilities`？
- [ ] 是否包含 `skills`？
- [ ] 是否包含 `inputModes`？
- [ ] 是否包含 `outputModes`？
- [ ] `skills[].outputTypes` 是否与 `outputModes` 兼容？
- [ ] 是否没有泄漏 API key / token / system prompt？

## 3. Health

- [ ] 是否暴露 `GET /health`？
- [ ] healthy 时是否返回 200？
- [ ] `/health` 是否不触发 LLM？
- [ ] `/health` 是否不执行昂贵工具？
- [ ] unhealthy Agent 是否不会进入 Planner 候选列表？

## 4. A2A Endpoint

- [ ] 是否暴露 `POST /a2a/tasks/sendSubscribe`？
- [ ] 是否支持结构化 messages？
- [ ] 是否支持 metadata.runId / threadId / traceId / agentName？
- [ ] metadata 是否不含 secret？
- [ ] 是否由 Orchestrator / A2A Client 调用？
- [ ] Frontend 是否没有直接调用？
- [ ] Gateway Handler 是否没有直接调用？

## 5. Streaming

- [ ] 是否输出 `working`？
- [ ] 是否输出 `text`？
- [ ] 是否支持 `artifact`？
- [ ] 是否输出 `completed`？
- [ ] 失败时是否输出 `failed`？
- [ ] completed / failed 后是否停止正常输出？

## 6. Artifact

- [ ] Artifact type 是否被 AgentCard.outputModes 声明？
- [ ] Artifact 是否包含 title？
- [ ] Artifact 是否包含 content？
- [ ] Artifact metadata 是否足够？
- [ ] Artifact 是否被 `artifact-contract` 支持？
- [ ] Artifact 是否能映射到 Frontend Runtime Skill？
- [ ] Child Agent 是否没有直接输出 AG-UI Tool Call？

## 7. Registry

- [ ] 是否支持多个 Agent？
- [ ] 是否能拉取 AgentCard？
- [ ] 是否能缓存 AgentCard？
- [ ] 是否能进行健康检查？
- [ ] 是否把 unhealthy Agent 排除出 Planner？
- [ ] 是否不硬编码 code/web 等固定 Agent 名称？

## 8. Orchestrator

- [ ] 是否从 Registry 获取 Agent？
- [ ] 是否基于 AgentCard 判断能力？
- [ ] 是否没有通过 agentName 判断能力？
- [ ] 是否通过 A2A Client 调用？
- [ ] 是否支持 fallback？
- [ ] 是否把 A2A event 转换成 AG-UI event？
- [ ] 是否没有把 A2A event 原样给 Frontend？

## 9. 安全

- [ ] Frontend 是否不能直接访问 Child Agent？
- [ ] Gateway Handler 是否不能直接访问 Child Agent？
- [ ] AgentCard 是否不含密钥？
- [ ] health 是否不泄漏配置？
- [ ] logs 是否脱敏？
- [ ] errors 是否脱敏？
