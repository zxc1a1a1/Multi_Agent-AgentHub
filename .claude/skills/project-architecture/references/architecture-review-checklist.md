# Architecture Review Checklist

来源：`docs/architecture/project-architecture-checklist.md`、`docs/architecture/service-boundaries.md`。

- [ ] Frontend 是否仅连接 Gateway。
- [ ] Gateway 是否未承担编排与 A2A 调度。
- [ ] Orchestrator 是否承担路由、A2A 调度、协议转换。
- [ ] Child Agent 是否通过 A2A 暴露能力。
- [ ] AG-UI / REST / A2A 是否无混用。
- [ ] MVP 简化是否未破坏长期分层。
- [ ] 是否避免提前实现 Post-MVP 能力。
- [ ] 文档变更是否同步到 `docs/architecture` / `docs/contracts`。
