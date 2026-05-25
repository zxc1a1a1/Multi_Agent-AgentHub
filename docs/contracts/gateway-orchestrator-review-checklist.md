# Gateway-Orchestrator Review Checklist

## 进程边界

- [ ] Gateway 和 Orchestrator 是两个独立服务。
- [ ] Gateway 没有 import Orchestrator 业务包。
- [ ] Orchestrator 有独立 main / health / port。
- [ ] Frontend 不能直接访问 Orchestrator。
- [ ] Child Agent 不反向依赖 Gateway 前端 API。

## 内部 API

- [ ] Gateway 通过 ORCHESTRATOR_URL 调用。
- [ ] 有服务间鉴权。
- [ ] 有 timeout。
- [ ] 传播 requestId / traceId / runId。
- [ ] 请求/响应 JSON 可序列化。
- [ ] 没有传 HTTP context / DB handle / Go channel。

## Gateway

- [ ] 只负责外部 HTTP / SSE / auth / persistence / request assembly。
- [ ] 没有 Agent 选择逻辑。
- [ ] 没有 fallback 决策。
- [ ] 没有直接调用 Child Agent。
- [ ] 没有解析 Child Agent 原始事件。

## Orchestrator

- [ ] 负责 planning / task / multi-agent / fallback。
- [ ] 不处理用户登录。
- [ ] 不写浏览器响应。
- [ ] 输出稳定 stream events。
- [ ] 返回 OrchestratorResult。
- [ ] fallback.mode 为正式枚举：none / same_capability_alternative / lower_risk_plan / single_agent_fallback / fail_fast。
- [ ] 未使用 legacy first_healthy_agent 作为 fallback.mode。
- [ ] same_capability_alternative 候选排序按 healthy 优先。
- [ ] 外部请求 planningMode 不含 fallback。
- [ ] fallback plan 由 Orchestrator 内部生成，关联原 plan（parentPlanId / fallbackOf）。

## 通用性

- [ ] 没有固定具体 Agent 名称。
- [ ] 支持 2+ Agent。
- [ ] 支持 single / ordered_parallel / sequential。
- [ ] 没有通过 agentName 推断能力。

## 取消超时

- [ ] 浏览器断连传播到 Orchestrator。
- [ ] Gateway timeout 关闭内部请求。
- [ ] 有 cancel endpoint 或等价机制。
- [ ] Orchestrator 停止未完成 task。
- [ ] 没有 goroutine / stream 泄漏。

## 安全

- [ ] service token 不进日志。
- [ ] 用户 token 不当作 service token。
- [ ] Orchestrator 不公网暴露。
- [ ] 错误脱敏。
