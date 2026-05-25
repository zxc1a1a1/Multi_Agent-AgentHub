# AgentHub v1.0 Test Matrix

| 测试类型 | 主要对象 | 关键断言 |
|---|---|---|
| Schema | Plan / Event / Artifact / AgentCard / LLM | valid 通过，invalid 拒绝 |
| Unit | Planner / Registry / Converter | 确定性、无外部依赖 |
| Component | MessageBubble / WebPreview / Markdown | 用户可见行为正确 |
| Store | messageStore / conversationStore | 多会话、多 Agent、cancel 状态正确 |
| Integration | Gateway / Orchestrator / Registry | 服务边界正确 |
| Cross-process | Gateway ↔ Orchestrator | URL 调用、stream、timeout、cancel |
| Replay | SSE / Internal Stream | 顺序、buffer、malformed、cancel |
| Failure | fallback / retry | 有上限、有提示、不污染成功消息 |
| Security | Auth / sandbox / redaction | 不泄漏、不越权 |
| Smoke | Docker Demo | 启动、health、run、group、artifact |
