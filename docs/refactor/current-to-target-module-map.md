# 当前目录到目标模块映射

| 当前目录/文件 | 当前职责 | 目标模块 | 迁移方式 | 当前阶段是否迁移 | 备注 |
|---|---|---|---|---|---|
| `agents/adk` | 现有 Agent 运行时与 A2A 基础能力 | `pkg/adk` + `pkg/runtime` | 先抽接口再拆实现，按能力分批迁移 | 否 | 本轮只建新骨架，不移动旧代码 |
| `agents/code-agent` | 代码类 Child Agent | `services/agents/code-agent` | 以首个迁移样板方式平移并适配 A2A | 否 | Phase 8 再迁移 |
| `agents/web-agent` | 网页类 Child Agent | `services/agents/web-agent` | 复用 code-agent 迁移模板迁移 | 否 | 后续批次迁移 |
| `agents/document-agent` | 文档类 Child Agent | `services/agents/document-agent` | 统一接入新 Runtime 与 A2A | 否 | 后续批次迁移 |
| `agents/vision-agent` | 视觉类 Child Agent | `services/agents/vision-agent` | 按能力声明和输出模式迁移 | 否 | 后续批次迁移 |
| `agents/*-agent` | 其余业务 Agent 集合 | `services/agents/*` | 逐个 Agent 小步迁移，禁止批量硬搬 | 否 | 本轮不处理任何旧 Agent |
| `server/cmd/server` | 旧单进程入口 | `services/gateway/cmd` + `services/orchestrator/cmd` | 按进程边界拆成两个独立入口 | 否 | 本轮不动旧入口 |
| `server/internal/handler` | 旧 HTTP/AG-UI 处理逻辑 | `services/gateway/internal/handler` | 先契约对齐，再迁移实现 | 否 | 本轮不改旧 handler |
| `server/internal/orchestrator` | 旧编排逻辑 | `services/orchestrator/internal/*` | 先抽象计划/执行边界，再迁移 | 否 | 本轮不改编排逻辑 |
| `server/internal/a2a` | 旧 A2A 客户端/类型 | `pkg/adk/a2a` + `services/orchestrator/internal` | 协议层与业务层分拆迁移 | 否 | 本轮不改 A2A 调用链 |
| `server/internal/model` | 旧 AG-UI/消息模型 | `pkg/runtime/agui` + `services/gateway/internal/model` | 模型归一化后按消费侧拆分 | 否 | 本轮不动旧 model |
| `frontend` | 前端界面与 AG-UI 消费 | `frontend`（保持独立） | 保持独立，仅在 Phase 9 做适配 | 否 | 本轮不改前端 |
| `docs/contracts` | 协议与契约事实源 | `docs/contracts`（保持） | 按阶段增量更新 | 否 | 本轮不改 contracts 主体 |
| `docs/integration` | 集成方案与联调文档 | `docs/integration`（保持） | 与集成阶段同步更新 | 否 | 本轮不改 integration |
| `docs/reports` | 测试与验收报告 | `docs/reports`（保持） | 按阶段补充报告 | 否 | 本轮只新增 `docs/refactor` |
| `docker-compose.yml` | 旧主线一键编排 | `docker-compose.yml`（Phase 10 更新） | 全模块稳定后统一切换 | 否 | 本轮禁止改动 |
