# AgentHub Testing Review Contract

## 1. 契约目标

本契约定义 AgentHub v1.0 的测试、Review、CI 和交付质量门禁。它将 Sprint v1.0 的多 Agent、群聊、LLM 编排、Registry、健康检查、丰富产物、降级重试、Docker Demo 和 AI 协作文档要求转化为可执行质量规则。

## 2. 质量门禁总览

| Gate | 必须覆盖 | 阻断合并 |
|---|---|---|
| Schema Validation | 所有跨边界 schema | 是 |
| Contract Tests | Public API / Internal API / Plan / Agent / Artifact / LLM | 是 |
| Unit Tests | Planner、Registry、Converter、Fallback、Safe Error | 是 |
| Frontend Tests | Store、Streaming、Artifact Preview、Error Boundary | 是 |
| Cross-process Tests | Gateway ↔ Orchestrator | 是 |
| Security Regression | Auth、Redaction、Sandbox、Secret Scan | 是 |
| Smoke Tests | Docker、Health、Run、Group、Fallback | Release 必须 |
| Demo Review | 3 分钟核心路径 | Release 必须 |

## 3. 分进程要求

Gateway 与 Orchestrator 必须作为两个独立进程测试。任何以同进程函数调用替代内部 API / stream 的测试，只能作为单元测试，不得视为分进程集成测试。

## 4. 通用 Agent 要求

测试不得把 `code-agent`、`web-agent`、`doc-agent` 写成唯一合法 Agent。示例 Agent 只能作为 Sprint Profile fixture。长期测试应基于 Agent capability、health、outputModes 和 schema。

## 5. 完成定义

- 所有必过 Gate 通过。
- 相关 contract / schema / fixture 已更新。
- v1.0 Demo path 未破坏。
- 无真实 secret 出现在代码、fixture、日志或 CI 输出中。
