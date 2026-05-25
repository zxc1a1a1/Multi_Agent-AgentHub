# Process Boundary Policy

## 硬规则

Gateway Service 与 Orchestrator Service 必须是两个独立进程。

## Gateway 禁止

- import Orchestrator 业务包。
- 在 handler 中执行 planning。
- 根据用户意图选择 Agent。
- 直接调用 Child Agent。
- 直接调用 LLM Provider。
- 执行 fallback / retry 策略。

## Orchestrator 禁止

- 直接暴露给前端。
- 处理用户登录态。
- 管理浏览器连接。
- 依赖 Gateway handler 内部类型。

## 迁移期说明

旧代码中如存在同进程实现，只能作为 legacy adapter。新增功能必须向分进程服务边界迁移。
