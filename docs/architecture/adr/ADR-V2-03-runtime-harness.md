# ADR-V2-03：复用 ADK，增加薄 Runtime Harness

- 日期：2026-10-29
- 状态：已采纳（P0）
- 背景：`pkg/adk` 已有 Agent/Model/Tool/Plugin/Runner/Session；缺执行前 checkpoint、预算、审批、恢复和证据收集。
- 决策：保留纯 ADK，新增 `pkg/runtime/harness` 作为控制面（ContextBuilder/BudgetManager/LoopController/PolicyGate/CheckpointManager/RunController/EvidenceCollector）。同一模型/工具循环只有一个所有者；不在 Runner 外套第二套同级循环。
- 备选：整体迁 Eino ADK（需一次性验证全部事件/工具/A2A/存储）；Python LangGraph 服务（双语言发布与 RPC 成本）；自研通用工作流引擎（重复框架）。
- 后果：需要最小扩展 ADK 纯状态/存储接口；checkpoint 版本化；恢复不得重置预算；Redis/MySQL 细节留在适配层。
- 同步契约：`runtime-harness.md`、`adk-runtime.md`、`agent-config.md`、`project-architecture.md`。
