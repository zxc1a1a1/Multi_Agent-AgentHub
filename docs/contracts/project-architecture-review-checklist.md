# Architecture Review Checklist

## 事实源

- 是否参考 v1.0 Sprint Profile？
- 是否把 MVP v0.1 放入 Historical Profile？
- 是否没有把 Sprint 示例固化为长期限制？

## 进程边界

- Gateway 与 Orchestrator 是否是两个独立进程？
- Frontend 是否只访问 Gateway？
- Orchestrator 是否不暴露给 Frontend？
- Gateway 是否没有 import Orchestrator 业务包？
- Gateway 是否没有直接调用 Child Agent？
- Gateway 是否没有直接调用 LLM Provider？

## 通用性

- 是否没有写死 code-agent / web-agent / doc-agent？
- 是否把 code/web/markdown 作为 Sprint 示例，而不是长期全集？
- 新增 Agent 是否只需声明能力，不需要改总架构？

## Conversation

- 是否支持 single / group？
- 是否支持 participants / senderName？
- @mention 是否只是路由提示？
- 一个 run 是否可以产生多条 Agent message？

## Orchestrator

- 是否负责 LLM 编排？
- 是否负责 Agent 选择？
- 是否负责 Registry / Health 过滤？
- 是否负责 fallback / retry？
- 是否负责结果聚合？

## Artifact

- 非文本产物是否通过 Artifact？
- 大产物是否不进入普通文本流？
- Frontend Runtime Capability 是否负责渲染？
- 新增产物类型是否不破坏总架构？

## 交付

- 是否体现 docker compose Demo？
- 是否体现 smoke test？
- 是否体现 README / 架构图 / AI 协作文档？
- 是否先 Contract，再 Mock，再真实集成？
