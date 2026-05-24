# Module Boundaries

来源：`docs/architecture/overview.md`、`docs/architecture/service-boundaries.md`、`docs/architecture/project-architecture-checklist.md`。

## 1. 分层边界

```text
Frontend
Gateway
Orchestrator
Child Agent
Data Layer
```

- Frontend 只连接 Gateway，不直接调用 Orchestrator / Child Agent。
- Gateway 负责 REST API、AG-UI SSE、鉴权、会话与消息持久化入口。
- Orchestrator 负责意图路由、A2A 调度、协议转换、结果聚合。
- Child Agent 通过 A2A 暴露能力，不直接服务浏览器。
- Data Layer 负责事实源持久化，不能由 SSE 缓冲或内存替代。

## 2. 协议边界

- Frontend ↔ Gateway：REST + AG-UI。
- Gateway ↔ Orchestrator：内部 Contract（可 MVP 同进程模块调用）。
- Orchestrator ↔ Child Agent：A2A。

## 3. 常见越界

- 在 `handler` 中实现编排逻辑。
- 在 Frontend 直接调用 A2A。
- 在 Gateway 直接解析并决定 Frontend Skill 映射策略。
