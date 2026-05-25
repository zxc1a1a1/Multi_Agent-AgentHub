# Frontend Boundary

Frontend 负责用户交互，不负责后端编排。

## 负责

- IM 聊天 UI。
- 单聊与群聊交互。
- @mention 输入体验。
- 多 Agent 消息展示。
- SSE / stream 消费。
- 产物预览。
- 编排过程提示。

## 禁止

- 直接访问 Orchestrator。
- 直接访问 Child Agent。
- 实现 Agent 选择策略。
- 实现 fallback / retry。
- 直接调用 LLM Provider。
- 手写与公开 API 契约不一致的数据结构。

## 产物展示

Frontend 可根据 Runtime Capability 渲染 Artifact，但不得把组件逻辑反向写入后端架构。
