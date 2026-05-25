# v1.0 Sprint Profile

## 目的

本文件用于承接 `SPRINT-v1.0-Plan.md` 中的当前交付目标，避免把 Sprint 示例误写成长期架构硬规则。

## 当前 Sprint 目标

v1.0 Profile 面向：

- 2+ Agent。
- LLM 意图编排。
- 单聊与群聊。
- Agent Registry 与健康检查。
- 结构化多轮消息。
- 丰富产物预览。
- fallback / retry 提示。
- single 与 ordered_parallel 语义。
- docker compose Demo。
- smoke test。
- README、架构图、AI 协作文档。

## 示例与长期规则的区别

Sprint 可以使用代码类 Agent、网页类 Agent、代码预览、网页预览、Markdown 渲染作为示例。

长期架构不得把这些示例写成固定枚举。新增 Agent、产物类型或 Provider 时，应通过能力声明、Registry、Artifact、Runtime Capability 和对应契约扩展。

## 与分进程设定的关系

即使 Sprint 文档中出现同进程目录或历史实现路径，当前架构仍以 Gateway Service 与 Orchestrator Service 分进程为准。
