# A2A Review Checklist

## AgentCard

- [ ] 是否暴露 `/.well-known/agent.json`？
- [ ] 是否包含 name？
- [ ] 是否包含 description？
- [ ] 是否包含 url？
- [ ] 是否包含 version？
- [ ] 是否包含 capabilities？
- [ ] 是否包含 skills？
- [ ] 是否包含 inputModes？
- [ ] 是否包含 outputModes？
- [ ] skills.outputTypes 是否与 outputModes 兼容？
- [ ] 是否没有泄漏 secret？

## Health

- [ ] 是否暴露 `/health`？
- [ ] `/health` 是否返回有效状态？
- [ ] `/health` 是否不触发 LLM？
- [ ] `/health` 是否不泄漏环境变量？
- [ ] unhealthy Agent 是否不会进入 Planner？

## A2A Endpoint

- [ ] 是否暴露 `/a2a/tasks/sendSubscribe`？
- [ ] 是否由 Orchestrator / A2A Client 调用？
- [ ] Frontend 是否没有直接调用？
- [ ] Gateway Handler 是否没有直接调用？

## Streaming

- [ ] 是否输出 working？
- [ ] 是否输出 text？
- [ ] 是否支持 artifact？
- [ ] 是否输出 completed？
- [ ] 失败时是否输出 failed？
- [ ] completed / failed 后是否停止正常输出？

## Artifact

- [ ] Artifact type 是否被 AgentCard.outputModes 声明？
- [ ] ArtifactDraft 是否只包含 Child Agent 可提供的字段（`type`、`title`、`content`/`contentRefDraft`、`metadata`）？
- [ ] ArtifactDraft 是否没有包含 `artifactId`、`version`、`links.*`、`source.*`、`preview.*`、`status`、`createdAt` 等平台字段？
- [ ] Artifact metadata 是否足够？
- [ ] ArtifactDraft 是否能归一化为 Core Artifact？
- [ ] Child Agent 是否没有直接输出 AG-UI Tool Call？
- [ ] 是否没有把 ArtifactDraft 当作 Core Artifact？

## Orchestrator

- [ ] 是否从 Registry 获取 Agent？
- [ ] 是否使用 AgentCard 判断能力？
- [ ] 是否没有硬编码 agentName 判断能力？
- [ ] 是否通过 A2A Client 调用？
- [ ] 是否支持 fallback？

## 安全

- [ ] AgentCard 是否不含密钥？
- [ ] metadata 是否不含 token？
- [ ] 日志是否脱敏？
- [ ] 错误信息是否脱敏？
- [ ] Frontend 是否不能直接访问 Child Agent？
