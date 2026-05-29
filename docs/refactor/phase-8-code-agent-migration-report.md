# Phase 8.1~8.3 Code Agent 最小迁移完成报告

## 1. 本轮目标

本轮目标是新增 `services/agents/code-agent`，完成最小 ADK Agent、最小 code tools、最小 A2A Server，以及基于 `httptest` 的 A2A round-trip 集成测试。

## 2. 新增/修改文件

- `go.work`
- `services/agents/code-agent/go.mod`
- `services/agents/code-agent/agent.go`
- `services/agents/code-agent/agent_test.go`
- `services/agents/code-agent/tools.go`
- `services/agents/code-agent/tools_test.go`
- `services/agents/code-agent/server.go`
- `services/agents/code-agent/server_test.go`
- `services/agents/code-agent/integration_test.go`
- `services/agents/code-agent/README.md`
- `docs/refactor/phase-8-code-agent-migration-report.md`

说明：`go.work.sum` 本轮未变化。

## 3. CodeAgent 设计说明

- 当前为 `v0.1 minimal/mock`，先保证新架构接入骨架可运行，再迭代真实能力。
- `Generate` 只从 `req.Contents` 提取最后一个用户 `TextPart`；没有用户文本时返回错误。
- 普通请求返回固定 mock 文本；检测到明显代码请求时返回简短示例代码文本。
- 不读取文件、不执行命令、不访问网络、不读取 `.env`，不调用真实 LLM。

## 4. Code Tools 设计说明

- `generate_code_snippet`
  - 输入：`language`、`description`
  - 输出：模板化最小代码片段
  - 不调用 LLM、不执行外部命令
- `explain_code_snippet`
  - 输入：`code`
  - 输出：简单解释（行数 + 语义摘要）
  - 不执行代码、不写文件
- 两个工具参数非法时返回 `ToolResult{IsError:true}`，且不 panic。

## 5. A2A Server 设计说明

- 使用 `adk.NewMemorySessionService()` 提供最小 session。
- 使用 `adk.NewRunner(agent, session, adk.WithTools(...))` 注入 tools。
- 使用 `a2a.NewServer(config, runner)` 暴露最小 A2A HTTP handler。
- AgentCard 声明：
  - `name`: `code-agent`
  - `description`: `code generation and code explanation`
  - `version`: `0.1.0`
  - `skills`: `code_generation`、`code_explanation`
  - `inputModes`: `text`、`code`
  - `outputModes`: `text`、`code`、`artifact_ref`
  - `streaming`: `false`
- 暴露 endpoint：
  - `GET /health`
  - `GET /.well-known/agent.json`
  - `POST /`
  - `POST /a2a/tasks/sendSubscribe`

## 6. Round-trip 测试说明

- 使用 `httptest.NewServer`，不启动真实监听端口。
- 覆盖 `a2a.Client.Send` round-trip。
- 覆盖 `a2a.RemoteAgent.Generate` round-trip。
- 验证响应包含 `TextPart`，并验证不暴露 `ThinkingPart`。
- 不依赖 Gateway、Orchestrator、旧 `agents/code-agent`。

## 7. 验证结果

- `services/agents/code-agent`
  - `go test ./... -v`：通过（25/25）
  - `go build ./...`：通过
- `pkg/adk`
  - `go test ./...`：通过
  - `go build ./...`：通过
- `pkg/runtime`
  - `go test ./...`：通过
  - `go build ./...`：通过
- `services/gateway`
  - `go test ./...`：通过
  - `go build ./...`：通过
- `agents`
  - `go test ./...`：通过
- `server`
  - `go test ./...`：通过
- `git status --short`
  - 变更仅见 `go.work`（tracked 修改）与本轮新增未跟踪文件（`services/agents/code-agent/*`、`docs/refactor/phase-8-code-agent-migration-report.md`）
  - 其余已有未跟踪文档（如 `phase-8-preflight-report.md` 等）保持原样
- `git diff --stat`
  - `go.work | 1 +`
- 安全扫描
  - `git diff` 中未发现真实 `sk-` 长 key、未发现真实 `DATABASE_URL`、未发现私钥内容
  - 在新增测试文件中出现的是安全占位符字符串（用于防泄漏测试），非真实凭据
  - 现有 `docs/refactor/refactor-risk-checklist.md` 命中的是规则关键词说明文本
- `exe` 检查
  - 未发现新增 `*.exe`

## 8. 保持不变的范围

- 未修改旧 `agents/code-agent`
- 未删除任何旧 Agent
- 未修改 `server/`
- 未修改 `frontend/`
- 未修改 `docker-compose.yml`
- 未修改 `Makefile`
- 未修改根 `go.mod/go.sum`
- 未读取、未创建、未修改 `.env`
- 未执行 `git add`
- 未执行 `git commit`
- 未调用真实 LLM API

## 9. 风险与注意事项

- 当前 `code-agent v0.1` 仅为最小 mock 实现，不是最终生产能力。
- 当前未接真实 model provider。
- 当前未输出正式 code artifact，仅返回文本/code snippet。
- 当前未接 Gateway/Orchestrator 真实运行链路。
- 旧 `agents/code-agent` 仍保留。
- 后续需要继续完成 Gateway -> RemoteAgent -> code-agent 最小链路集成。

## 10. 下一步建议

下一步可以进入 Phase 8.4：为新 code-agent 增加 Dockerfile 与 Gateway RemoteAgentRunService 最小链路测试，但仍不删除旧 code-agent。
