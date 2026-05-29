# Phase 8.4 Code Agent Docker 与 Gateway 链路完成报告

## 1. 本轮目标

本轮完成了两件核心工作：

- 为新模块 `services/agents/code-agent` 新增可独立启动的 `main` 入口与 Dockerfile。
- 在 `services/gateway` 新增 `RemoteAgentRunService`，并通过 `httptest` 打通 Gateway `/api/chat` 到远端 A2A Agent 的最小 SSE 链路测试。

本轮未实现 Orchestrator/Planner/Executor，仅做单远端 Agent 适配。

## 2. 新增/修改文件

- `services/agents/code-agent/Dockerfile`（新增）
- `services/agents/code-agent/cmd/code-agent/main.go`（新增）
- `services/agents/code-agent/cmd/code-agent/main_test.go`（新增）
- `services/gateway/runservice/remote_agent.go`（新增）
- `services/gateway/runservice/remote_agent_test.go`（新增）
- `services/gateway/gateway_test.go`（补充 Gateway -> RemoteAgentRunService 集成测试）
- `docs/refactor/phase-8-code-agent-gateway-link-report.md`（新增）

说明：`go.work.sum` 本轮未变化。

## 3. code-agent main/Dockerfile 设计说明

- `main.go` 仅提供启动入口：
  - `CODE_AGENT_ADDR` 默认 `:8080`
  - `CODE_AGENT_PUBLIC_URL` 默认 `http://localhost:8080`
- 关键构造逻辑下沉为 `buildHandler(publicURL string) (http.Handler, error)`，测试仅调用 `buildHandler`，不调用 `main`，不监听真实端口。
- `main_test.go` 覆盖：
  - `buildHandler` 成功构建
  - `/health` 可用
  - `/.well-known/agent.json` 可用
  - `publicURL` 为空时回退默认地址
- `Dockerfile` 为多阶段构建：
  - builder：`golang:1.22-alpine`
  - 仅复制 `go.work/go.work.sum`、`pkg/adk`、`services/agents/code-agent`
  - 构建 `./cmd/code-agent`
  - runtime：`alpine:3.20`，暴露 `8080`，`CMD ["code-agent"]`
- 未修改 `docker-compose.yml`，符合本轮“只交付构建文件，不接 compose”的范围。
- 未复制 `.env`，未写入任何 secrets。

## 4. RemoteAgentRunService 设计说明

- 新增 `services/gateway/runservice/remote_agent.go`：
  - `RemoteAgentRunService`（单远端 Agent 适配器，不是 Orchestrator）
  - `NewRemoteAgentRunService(agentName, agentURL string, opts ...Option)` 含参数校验
  - `WithClient(*a2a.Client)` 支持注入测试 client
  - `Run(ctx, conversationID, userContent)` 实现 `httpapi.RunService`
- 运行逻辑：
  - 校验 `conversationID`、`userContent`
  - 提取用户文本后调用 `a2a.NewRemoteAgent(...).Generate(...)`
  - 用 `conversationID` 作为远端 `sessionId`
  - 将远端返回映射为单个 `adk.Event`：
    - `Author = agentName`
    - `Content.Role = assistant`
    - `Final = true`
  - 保留 `text/tool_call/tool_result`，过滤 `thinking`
  - 远端错误通过 `yield error` 返回，不 panic
- Gateway 生产代码未 import `services/agents/code-agent`，只依赖 `RunService` 抽象与 `pkg/adk/a2a`。

## 5. Gateway 链路测试说明

- 在 `services/gateway/gateway_test.go` 新增 3 个最小链路测试：
  - `TestGatewayRemoteAgentRunService_ChatSSE`
  - `TestGatewayRemoteAgentRunService_PersistsMessages`
  - `TestGatewayRemoteAgentRunService_ErrorSSE`
- 测试形态：
  - 使用 `httptest.NewServer` 构造 A2A mock server（不启动真实端口）
  - Gateway 走真实 HTTP handler：创建 conversation -> `POST /api/chat`
  - 校验 SSE 输出中包含远端返回文本/错误事件
  - 校验 `MemoryStore` 用户消息与 assistant 消息落库行为
  - 远端错误场景校验 SSE `error` 事件与脱敏行为

## 6. 验证结果

- `services/agents/code-agent`
  - `go test ./... -v`：通过（含新 `cmd/code-agent` 测试）
  - `go build ./...`：通过
- `services/gateway`
  - `go test ./... -v`：通过（含新 `runservice` 测试和 Gateway 链路测试）
  - `go build ./...`：通过
- `pkg/adk`
  - `go test ./...`：通过
  - `go build ./...`：通过
- `pkg/runtime`
  - `go test ./...`：通过
  - `go build ./...`：通过
- `agents`
  - `go test ./...`：通过
- `server`
  - `go test ./...`：通过
- `git status --short`
  - 当前可见：`go.work` 已有既存修改；本轮新增/修改文件位于允许范围内
- `git diff --stat`
  - tracked 变更显示 `services/gateway/gateway_test.go`（本轮）
  - `go.work` 变更为本轮前已存在
- 安全扫描
  - 未发现真实 `sk-` 长 token
  - 未发现真实私钥内容
  - 命中的敏感词来自测试占位值和规则文档关键字说明
- `exe` 检查
  - 未发现新增 `*.exe`

## 7. 保持不变范围

已确认本轮保持以下范围不变：

- 未修改旧 `agents/code-agent`
- 未删除任何旧 Agent
- 未修改 `server/`
- 未修改 `frontend/`
- 未修改 `docker-compose.yml`
- 未修改 `Makefile`
- 未修改根 `go.mod/go.sum`
- 未读取/创建/修改 `.env`
- 未提交代码
- 未执行 `git add`
- 未调用真实 LLM API

## 8. 风险与注意事项

- `services/agents/code-agent` 当前仍是 `v0.1` mock 能力，不是最终生产能力。
- 本轮 Dockerfile 仅完成构建交付，尚未接入 compose 编排。
- `RemoteAgentRunService` 是单 Agent 适配层，不承担编排职责。
- Gateway 链路测试已覆盖 SSE 与落库最小主路径，但尚未接真实 frontend 联调。
- 下一阶段仍需在多 Agent 场景下补充前端事件归属与展示适配。

## 9. 下一步建议

下一步可以进入 Phase 9：frontend agentName 与多 Agent SSE 适配，或继续迁移 web-agent；Orchestrator 编排继续暂缓。

