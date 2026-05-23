# PR Review Checklist

## 1. 目的

本文定义 AgentHub PR / 代码变更 review checklist。

## 2. Contract-first

检查：

- 是否改了 contract？
- 如果改了实现，contract 是否先更新？
- 是否更新对应 schema？
- 是否更新 references？
- 是否补了合法和非法样例？

## 3. 协议边界

检查是否影响：

- AG-UI event lifecycle。
- A2A Task / Streaming。
- Artifact schema / mapping。
- ExecutionPlan validation。
- LLM Provider adapter。
- Frontend Runtime Skill registry。
- DB persistence。

## 4. 测试

检查：

- 是否补了 unit test？
- 是否补了 protocol conversion test？
- 是否补了 contract test？
- 是否补了 frontend test？
- 是否补了 E2E smoke？
- 是否覆盖错误路径？

## 5. 安全

检查：

- 是否泄漏 API key？
- 是否泄漏 token？
- 是否暴露 provider raw error？
- 是否暴露 stack trace？
- 是否绕过 auth？
- 是否引入危险操作？
- 是否需要用户确认？

## 6. 高风险禁止项

不得接受：

- React 直连子 Agent。
- Gateway handler 写复杂编排。
- Orchestrator 直接生成 React 组件参数。
- 子 Agent 绕过 A2A。
- 大型 Artifact 进入 AG-UI token 流。
- Tool Call args 绕过 schema 校验。
