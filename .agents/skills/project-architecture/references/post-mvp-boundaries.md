# Post-MVP Boundaries

来源：`docs/architecture/overview.md`、`docs/architecture/service-boundaries.md`、`docs/contracts/postgres-schema.md`。

## 1. Post-MVP 规划（非 MVP 必需）

- Orchestrator 拆分为独立服务。
- 多 Agent 编排（parallel / sequential / fallback）。
- `web-agent`、`doc-agent`、custom agent。
- PostgreSQL + Redis + Object Storage 完整数据层。
- 更完整 JWT / 权限体系、审计、限流。

## 2. 禁止提前实现

以下能力若无明确指令，不应在 MVP 任务中提前落地：

- 群聊与复杂多 Agent 编排。
- Agent Registry 的完整生产实现。
- 高复杂缓存层与对象存储全链路。
- 完整用户体系与复杂 RBAC。
