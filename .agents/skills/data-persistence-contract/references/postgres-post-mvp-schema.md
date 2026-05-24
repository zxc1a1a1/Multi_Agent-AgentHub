# PostgreSQL Post-MVP Schema

来源：`docs/contracts/postgres-schema.md`。

## 1. 定位

- PostgreSQL 是 Post-MVP 事实源方向（非 MVP 必需）。
- Redis / Object Storage 与之配套。

## 2. 规划对象

- conversation_participants
- run_steps
- approvals
- agent_health_checks

说明：以上为 Post-MVP 规划，不应在 MVP bug 修复任务中强行实现。
