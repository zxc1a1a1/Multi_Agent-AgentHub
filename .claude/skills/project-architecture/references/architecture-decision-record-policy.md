# Architecture Decision Record Policy

重大架构变化必须记录 ADR 或等价决策文档。

## 触发条件

- 改变 Gateway / Orchestrator / Agent 进程边界。
- 新增跨服务协议。
- 替换主数据库。
- 引入消息队列。
- 改变 Artifact 存储方式。
- 改变公开 API 版本策略。
- 引入新的 Provider 类型。

## ADR 最小字段

- 标题。
- 日期。
- 状态。
- 背景。
- 决策。
- 备选方案。
- 后果。
- 需要同步更新的契约。
