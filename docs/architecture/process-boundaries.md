# Process Boundaries

Gateway 与 Orchestrator 必须是两个独立进程。

架构 Review 时必须检查：

- Gateway 是否只通过内部 API 调用 Orchestrator。
- Gateway 是否没有 import Orchestrator 业务包。
- Orchestrator 是否有独立启动、健康检查和内部接口。
- Frontend 是否不能直接访问 Orchestrator。
