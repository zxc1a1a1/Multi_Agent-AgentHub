# Gateway-Orchestrator Review Checklist

来源：`docs/contracts/gateway-orchestrator-review-checklist.md`、`docs/contracts/gateway-orchestrator.md`。

- [ ] Gateway 与 Orchestrator 边界清晰。
- [ ] MVP 同进程实现未破坏模块边界。
- [ ] 事件顺序符合 AG-UI 契约。
- [ ] 取消/超时路径有安全降级。
- [ ] 未向前端暴露 `/internal/*`。
- [ ] 错误映射脱敏。
