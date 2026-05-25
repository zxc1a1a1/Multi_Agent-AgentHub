# AgentHub 安全边界契约

## 目的

本文件是 AgentHub 安全边界的合同级说明，用于约束 Frontend、Gateway、Orchestrator、Child Agent、LLM、Registry、Artifact、Tool、文件、部署、日志和审计。

## 当前 Profile

- MVP v0.1 已完成，仅作为历史回归基线。
- Gateway 与 Orchestrator 必须分进程。
- v1.0 面向 2+ Agent、单聊 + 群聊、LLM 编排、Agent Registry、健康检查、丰富产物、fallback / retry。

## 安全边界总览

| 边界 | 可信等级 | 核心规则 |
|---|---|---|
| Frontend | 不可信输入来源 | 只能访问 Gateway 公开 API |
| Gateway | 公开入口 | 鉴权、对象级授权、转发 Orchestrator |
| Orchestrator | 内部编排 | 不暴露给前端，先校验再调度 |
| Child Agent | 不完全可信 | 只执行声明且授权能力 |
| LLM | 不可信建议来源 | 输出必须结构化校验 |
| Artifact | 不可信内容 | 不直接进主 DOM，高风险隔离 |
| Tool Action | 受控副作用 | 按风险分级，高危需确认 |

## 统一禁止项

- 公开 API 暴露 `/internal/**`。
- 日志中输出 API key、service token、数据库连接串。
- LLM 输出直接执行高危动作。
- AgentCard 包含 secret。
- HTML Artifact 直接注入主应用 DOM。
- 用户 token 被当作 service token 透传。
