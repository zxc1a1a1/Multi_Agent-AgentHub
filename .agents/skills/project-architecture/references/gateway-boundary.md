# Gateway Boundary

Gateway Service 是 Frontend 的唯一后端入口。

## 负责

- REST API。
- 前端 stream。
- 用户鉴权。
- requestId / traceId。
- 会话、消息、Agent 摘要、Artifact 元数据查询。
- 接收 run 请求。
- 调用 Orchestrator 内部 API。
- 转发 Orchestrator 流式事件。
- 持久化入口。
- 安全错误映射。

## 禁止

- 实现 LLM 编排。
- 生成 OrchestrationPlan。
- 根据关键词选择 Agent。
- 直接调用 Child Agent。
- 直接调用 LLM Provider。
- 执行 fallback / retry。
- 返回内部服务对象。

## 边界判断

如果代码在回答“应该调用哪个 Agent、如何拆任务、如何 fallback”，它不应该位于 Gateway。
