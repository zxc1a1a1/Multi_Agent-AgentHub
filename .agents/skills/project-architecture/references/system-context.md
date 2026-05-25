# System Context

AgentHub 是 IM 聊天式多 Agent 协作平台。用户通过单聊或群聊对话发起任务，系统由 Orchestrator 生成计划、选择 Agent、调度任务并聚合结果。

## 核心对象

- Conversation：用户协作空间。
- Message：用户、系统或 Agent 的交流单元。
- Run：一次用户请求触发的执行过程。
- Plan：Orchestrator 生成的编排计划。
- Agent：能力提供服务。
- Artifact：非纯文本产物或可预览内容。
- Registry：Agent 能力与健康状态目录。

## v1.0 必须支持

- 2+ Agent。
- 单聊与群聊。
- LLM 意图编排。
- 多消息归属。
- Agent 健康检查。
- code、webpage、markdown 等产物示例。
- fallback 与 retry。

## 非目标

本文件不定义具体 API 字段、数据库表或前端组件实现。
