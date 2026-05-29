# services/agents/code-agent

这是新架构下的 `code-agent v0.1` 最小迁移骨架（Phase 8.1~8.3）。

## 当前范围

- 基于 `pkg/adk` 实现最小 `CodeAgent`（mock/minimal）。
- 基于 `pkg/adk/a2a` 暴露最小 A2A Server：
  - `GET /health`
  - `GET /.well-known/agent.json`
  - `POST /`
  - `POST /a2a/tasks/sendSubscribe`
- 提供最小 code tools：
  - `generate_code_snippet`
  - `explain_code_snippet`
- 提供 `httptest` round-trip 集成测试（Client / RemoteAgent）。

## 关键说明

- 本模块当前不接真实 LLM。
- 不读取 `.env`，不依赖真实密钥。
- 不访问真实外网，不连接真实数据库。
- 旧 `agents/code-agent` 仍保留，未删除。

## 当前声明 skills

- `code_generation`
- `code_explanation`

## 后续工作（不在本轮）

- 对接真实 provider。
- 扩展正式 artifact 流程。
- 增加 Dockerfile 与容器交付。
- 完成 Gateway/Orchestrator 的最小链路对接与测试。
