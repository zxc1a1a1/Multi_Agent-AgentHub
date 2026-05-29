# Phase 8.8 Gateway AgentName 路由完成报告

## 1. 本轮目标
本轮在不引入 Orchestrator / Planner / Executor 的前提下，新增 Gateway 侧最小静态注册与路由能力：
- 新增 `StaticAgentRegistry`，静态注册 `code-agent` 与 `web-agent`。
- 新增 `RoutingRunService`，按显式 `agentName` 路由到对应 `RemoteAgentRunService`。
- `/api/chat` 新增可选 `agentName`，未传时默认走 `code-agent`。

## 2. 新增/修改文件
- `services/gateway/runservice/registry.go`（新增）
- `services/gateway/runservice/registry_test.go`（新增）
- `services/gateway/runservice/routing.go`（新增）
- `services/gateway/runservice/routing_test.go`（新增）
- `services/gateway/runservice/remote_agent_test.go`（修改，去除对 `httpapi` 的测试耦合）
- `services/gateway/httpapi/server.go`（修改，`/api/chat` 支持可选 `agentName`）
- `services/gateway/httpapi/server_test.go`（修改，新增 `/api/chat` 路由与兼容测试）
- `services/gateway/gateway_test.go`（修改，新增 multi-agent 路由集成测试）
- `docs/refactor/phase-8-gateway-agent-routing-report.md`（新增）

## 3. StaticAgentRegistry 设计说明
- 注册项字段：
  - `Name`：agentName（必填）
  - `URL`：远端 A2A URL（必填）
  - `Description`：描述（可选）
  - `OutputModes`：输出模式（可选）
- 该注册表是静态配置容器，不承担编排职责，不是 Orchestrator。
- 不做 health check：本阶段只做最小路由验证，健康探测留给后续 Registry/Orchestrator 阶段。
- 不访问网络：仅做入参校验、内存存储、查询与排序。
- 对 `code-agent` / `web-agent` 的支持由静态注册项提供，不依赖具体 Agent 包导入。

## 4. RoutingRunService 设计说明
- 通过 `AgentNameSelector` 选择目标 agentName；默认 selector 从 context 读取 `agentName`。
- 新增 context helper：
  - `WithAgentName(ctx, agentName)`
  - `AgentNameFromContext(ctx)`
- 默认路由为 `code-agent`：
  - selector 为 `nil` 或返回空字符串时，回退到 `defaultAgentName`。
- 显式路由到 `web-agent`：
  - selector 返回 `web-agent` 时，调用对应 `RemoteAgentRunService`。
- unknown agent 处理：
  - `RoutingRunService` 返回安全错误 `requested agent is not available`。
  - Gateway SSE 层统一输出 `runner_error` + `assistant run failed`，不泄露内部细节。
- Gateway 生产代码未 import 具体 Agent 包，只通过 URL 调用远端 A2A。

## 5. /api/chat agentName 兼容说明
- `agentName` 为可选字段（`optional`）。
- 老请求（不传 `agentName`）保持兼容，默认走 `code-agent`。
- 新请求可显式传 `code-agent` 或 `web-agent`。
- `agentName` 通过 context 传入 RunService，不混入用户文本，不污染消息内容。
- 未修改 frontend，未修改旧 `server/`。

## 6. 测试说明
- `StaticAgentRegistry` 单测覆盖：
  - 创建成功、空 name、空 URL、重复 name、Get/List、稳定排序、`.env` 无依赖。
- `RoutingRunService` 单测覆盖：
  - 构造参数校验、默认 code 路由、web 路由、空 selector 回退默认、unknown agent 安全错误、远端错误、thinking 脱敏、`.env` 无依赖、无具体 Agent import。
- `/api/chat` 单测覆盖：
  - 不传 `agentName` 向后兼容；
  - `agentName=code-agent` / `web-agent` 正确路由；
  - `agentName=unknown` 返回安全 SSE error；
  - 路由后 user/assistant 持久化不破坏。
- Gateway 集成测试新增：
  - `TestGateway_StaticAgentRegistry_MultiAgentChatSSE`（双 mock A2A + SSE + 持久化 + unknown 安全错误）。

## 7. 验证结果
- `services/gateway`
  - `go test ./... -v`：通过
  - `go build ./...`：通过
- `services/agents/code-agent`
  - `go test ./... -v`：通过
  - `go build ./...`：通过
- `services/agents/web-agent`
  - `go test ./... -v`：通过
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

Git 与差异检查：
- `git status --short`：存在预期的未提交改动（包含历史 Phase 8 文件与本轮新增/修改文件）
- `git diff --stat`：当前 tracked diff 主要集中在 `services/gateway` 与 `go.work`，本轮新增文件处于未跟踪状态，未执行 `git add`

安全扫描结果：
- `git diff` 关键字扫描：未发现真实密钥/真实 token/真实数据库连接串/私钥内容
- 目录扫描命中均为测试 mock 或文档规则描述（如 `OPENAI_API_KEY`、`DATABASE_URL`、`sk-THIS_IS_A_MOCK_SECRET_TOKEN_12345`），非真实凭据
- `git ls-files .env`：无输出（`.env` 未被跟踪）
- `*.exe` 检查：无新增 exe 文件

## 8. 保持不变范围
- 未修改旧 `agents/code-agent`
- 未修改旧 `agents/web-agent`
- 未删除任何旧 Agent
- 未修改 `server/`
- 未修改 `frontend/`
- 未修改 `docker-compose.yml`
- 未修改 `Makefile`
- 未修改根 `go.mod` / `go.sum`
- 未读取/创建/修改 `.env`
- 未提交代码
- 未使用 `git add`
- 未调用真实 LLM API

## 9. 风险与注意事项
- `StaticAgentRegistry` 是当前阶段的最小静态注册表，不具备动态发现/健康检查能力。
- `RoutingRunService` 是显式 `agentName` 路由，不是意图识别或智能编排。
- 当前未实现 health check 驱动的可用性过滤。
- 当前未实现 capability-based planner。
- 当前 frontend 尚未传递/展示 `agentName` 的完整交互链路。
- 后续 Phase 9 可补齐 frontend 侧 agentName 选择与 senderName/preview 展示。

## 10. 下一步建议
下一步可以进入 Phase 9：frontend agentName、多 Agent SSE 与 web_preview/code_preview 最小适配；Orchestrator 编排继续暂缓。
