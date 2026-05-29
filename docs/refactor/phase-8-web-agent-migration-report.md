# Phase 8.5~8.7 Web Agent 最小迁移完成报告

## 1. 本轮目标

本轮目标是新增 `services/agents/web-agent`，完成最小 ADK Agent、最小 web tools、A2A Server、round-trip 集成测试、main 入口和 Dockerfile。

## 2. 新增/修改文件

- `go.work`
- `services/agents/web-agent/go.mod`
- `services/agents/web-agent/agent.go`
- `services/agents/web-agent/agent_test.go`
- `services/agents/web-agent/tools.go`
- `services/agents/web-agent/tools_test.go`
- `services/agents/web-agent/server.go`
- `services/agents/web-agent/server_test.go`
- `services/agents/web-agent/integration_test.go`
- `services/agents/web-agent/cmd/web-agent/main.go`
- `services/agents/web-agent/cmd/web-agent/main_test.go`
- `services/agents/web-agent/Dockerfile`
- `services/agents/web-agent/README.md`
- `docs/refactor/phase-8-web-agent-migration-report.md`
- `services/gateway/runservice/remote_agent_test.go`（仅新增 generic web-agent 测试）

## 3. WebAgent 设计说明

- 当前是 `v0.1 minimal/mock agent`，先保证新架构接入骨架可运行，再迭代真实能力。
- `Generate` 从 `req.Contents` 提取最后一个用户 `TextPart`，没有用户文本时返回错误。
- 普通请求返回固定 mock 文本；页面类请求返回简短安全 HTML 片段。
- 当前不接真实 LLM，避免引入 provider 依赖和密钥注入风险。
- `Generate` 不读取文件、不执行命令、不联网、不读取 `.env`。
- HTML 输出通过最小安全约束过滤危险内容，避免 `script`、`iframe`、`on*`、`javascript:`。

## 4. Web Tools 设计说明

- `generate_html_snippet`
  - 输入：`title`、`description`
  - 输出：最小安全 HTML 片段
  - 参数非法时返回 `ToolResult{IsError:true}`
- `summarize_ui_request`
  - 输入：`request`
  - 输出：结构化 UI 摘要（`title`、`components`、`constraints`）
  - 参数非法时返回 `ToolResult{IsError:true}`
- 两个工具都不调用 LLM，不执行外部命令，不写文件，不读 `.env`，不联网。
- HTML 输出应用最小安全约束，不输出 unsafe HTML。

## 5. A2A Server 设计说明

- `NewHandler(cfg ServerConfig) (http.Handler, adk.SessionService, error)`
- 使用 `adk.NewMemorySessionService()` 作为 session service
- 使用 `adk.NewRunner(agent, session, adk.WithTools(...))`
- 使用 `a2a.NewServer(agentCfg, runner)` 暴露 A2A HTTP handler
- AgentCard 声明：
  - `Name`: `web-agent`
  - `Description`: `web UI and HTML generation`
  - `Version`: `0.1.0`
  - `Skills`: `web_generation`、`html_generation`、`ui_summarization`
  - `InputModes`: `text`、`image_ref`、`extracted_text`、`vision_analysis`
  - `OutputModes`: `text`、`webpage`、`html`、`artifact_ref`
  - `Streaming`: `false`
- endpoint：
  - `GET /health`
  - `GET /.well-known/agent.json`
  - `POST /`
  - `POST /a2a/tasks/sendSubscribe`

说明：`vision_analysis` 与 `webpage` 目前只在能力声明层暴露，v0.1 仍是最小 mock 输出。

## 6. Round-trip 测试说明

- 使用 `httptest` 启动 A2A server（不启动真实端口）。
- 覆盖 `a2a.Client.Send` round-trip。
- 覆盖 `a2a.RemoteAgent.Generate` round-trip。
- 覆盖 AgentCard endpoint 和 run endpoint。
- 覆盖 unsafe HTML 约束。
- 覆盖 no legacy import（未依赖旧 `agents/web-agent`）。

## 7. main/Dockerfile 说明

- `main.go` 读取：
  - `WEB_AGENT_ADDR`（默认 `:8080`）
  - `WEB_AGENT_PUBLIC_URL`（默认 `http://localhost:8080`）
- `buildHandler(publicURL string)` 可单测；测试不调用 `main`，不监听真实端口。
- Dockerfile 使用多阶段构建：
  - builder: `golang:1.22-alpine`
  - runtime: `alpine:3.20`
  - 构建目标：`./cmd/web-agent`
  - 暴露 `8080`
- 未修改 `docker-compose.yml`。

## 8. Gateway generic RunService 验证

本轮新增 `TestRemoteAgentRunService_GenericWebAgent`（`services/gateway/runservice/remote_agent_test.go`）验证：

- 使用 `httptest` mock A2A server 模拟 web-agent
- 不 import `services/agents/web-agent`
- 同一个 `RemoteAgentRunService` 可适配 web-agent
- 断言 `Author=web-agent`，且返回 `TextPart` 包含 HTML 文本
- 未修改 Gateway 生产行为

## 9. 验证结果

- `services/agents/web-agent`
  - `go test ./... -v` 通过
  - `go build ./...` 通过
- `services/agents/code-agent`
  - `go test ./... -v` 通过
  - `go build ./...` 通过
- `services/gateway`
  - `go test ./... -v` 通过
  - `go build ./...` 通过
- `pkg/adk`
  - `go test ./...` 通过
  - `go build ./...` 通过
- `pkg/runtime`
  - `go test ./...` 通过
  - `go build ./...` 通过
- `agents`
  - `go test ./...` 通过
- `server`
  - `go test ./...` 通过
- `git status --short`
  - 预期存在未提交改动（含本轮新增 `services/agents/web-agent`、报告文件与可选 gateway 测试）
  - 也存在你此前已存在的未提交改动（保留未回退）
- `git diff --stat`
  - tracked diff 显示 `go.work` 和已有 tracked 文件（你先前变更）增量
  - 本轮新文件为 untracked，体现在 `git status --short`
- 安全扫描结果
  - 未发现真实 API Key / Token / `DATABASE_URL` / 私钥内容
  - 命中的敏感关键词均为测试占位文本、正则规则或文档规则说明（非真实密钥）
  - `sk-` 命中为测试 mock 字符串（`sk-THIS_IS_A_MOCK_SECRET_TOKEN_12345`）
- exe 检查结果
  - 未发现新增 `*.exe`

## 10. 保持不变的范围

- 未修改旧 `agents/web-agent`
- 未修改旧 `agents/code-agent`
- 未删除任何旧 Agent
- 未修改 `server/`
- 未修改 `frontend/`
- 未修改 `docker-compose.yml`
- 未修改 `Makefile`
- 未修改根 `go.mod/go.sum`
- 未读取/修改 `.env`
- 未提交
- 未使用 `git add`
- 未调用真实 LLM

## 11. 风险与注意事项

- 当前 `web-agent v0.1` 仍是最小 mock 实现，不是最终生产能力。
- 当前未接真实 model provider。
- 当前未输出正式 `webpage artifact`，仅返回文本/HTML snippet。
- 当前未接 frontend `web_preview`。
- 当前未接 Orchestrator。
- 旧 `agents/web-agent` 仍保留。
- 后续需要做 frontend agentName 与 `web_preview` 适配。

## 12. 下一步建议

下一步可以进入 Phase 9：frontend agentName、多 Agent SSE 与 web_preview/code_preview 最小适配；Orchestrator 编排继续暂缓。

