# Security Review Checklist

## 事实源

- 是否参考 `SPRINT-v1.0-Plan.md`？
- 是否把 MVP v0.1 降级为 Historical Profile？
- 是否没有把 Sprint 示例固化为长期 Agent 名称限制？

## 分进程

- Gateway 与 Orchestrator 是否分进程？
- Frontend 是否不能直连 Orchestrator？
- Orchestrator `/internal/**` 是否没有暴露给前端？
- Gateway → Orchestrator 是否有 service-to-service auth？
- 用户 token 是否没有被当成 service token 透传？

## 公开 API

- `/api/**` 是否鉴权？
- 是否有对象级授权？
- token 是否不在 query string？
- 错误是否脱敏？
- 是否有 rate limit？

## Agent

- AgentCard 是否不泄密？
- Registry 是否只接受可信来源？
- Health Check 是否不暴露内部错误？
- Orchestrator 是否只调用 enabled + healthy Agent？
- 用户自建 Agent 是否有隔离策略？

## LLM

- LLM 输出是否只作为不可信建议？
- Planner 输出是否 schema validation？
- LLM 是否不能绕过权限 / confirm_action？
- system prompt / secret 是否不传给不可信 Agent？
- fallback plan 是否重新校验？

## Artifact / Runtime

- code preview 是否只展示不执行？
- markdown 是否防 XSS？
- web preview 是否 iframe sandbox？
- download 是否鉴权？
- Artifact metadata 是否不含 secret？

## Tool / 高危动作

- run_command 是否 sandbox / whitelist / timeout？
- deploy / overwrite / external publish 是否 confirm_action？
- 高危动作是否审计？
- LLM 是否不能直接执行高危动作？

## Secret

- API key 是否只来自 env / secret manager？
- `.env.example` 是否无真实值？
- 日志 / trace / Artifact / AgentCard 是否无 secret？

## 测试

- 未鉴权是否返回 401？
- 无权限是否返回 403？
- 错误是否无 stack trace？
- AgentCard 是否无密钥？
- iframe 是否有 sandbox？
- high-risk tool 是否需要 confirm_action？
- smoke test 是否包含安全检查？
