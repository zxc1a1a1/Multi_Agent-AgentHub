# AgentHub - Multi-Agent Collaboration Platform

AgentHub 是一个基于 IM 交互范式的多 Agent 协作平台。用户通过对话与不同 AI Agent 交互，由 Orchestrator 理解意图、生成计划、调度 Agent 并聚合结果。

## 当前阶段

v1.0 Productization Stage（产品化与能力池接入阶段）

- 后端 Orchestrator vertical slice 已完成（五服务链路已跑通）
- 不再沿用 Phase 8 编号继续开发
- 后续 Milestone 参见 [productization-stage-guide.md](docs/refactor/productization-stage-guide.md)

---

## 当前新架构（主路径）

### 服务拓扑

```text
Frontend / Client
  → Gateway (services/gateway)
  → Orchestrator (services/orchestrator)
  → 10 Specialist Agents:
      code-agent / web-agent / document-agent / vision-agent
      context-agent / test-agent / review-agent / security-agent
      deploy-agent / diff-agent
  → Orchestrator summary (LLM Synthesizer)
  → Gateway SSE
  → Frontend
```

### 运行时链路

```text
Frontend / Client
  → Gateway Public REST (POST /api/agui/run)
  → Gateway → Orchestrator Internal API (stream)
  → Orchestrator RulePlanner → OrchestrationPlan
  → PlanValidator → SingleExecutor / OrderedParallelExecutor
  → A2A Dispatcher → code-agent / web-agent
  → Orchestrator summary aggregation
  → Gateway SSE transformation
  → Frontend SSE consumer
```

### 核心约束

- **Gateway 与 Orchestrator 是两个独立进程**，不得合并
- Frontend 只能通过 Gateway 公开 API 通信，不得直连 Orchestrator 或 Agent
- Gateway 不负责 Agent 编排，不直接调用 Child Agent，不直接调用 LLM Provider
- Orchestrator 是唯一编排层（Planner → Validator → Executor → Dispatcher → Registry）

---

## 新架构主路径

### `services/gateway`

对外公开入口。职责：REST API、SSE stream、用户鉴权、CORS、Conversation/Message 查询入口、调用 Orchestrator 内部 API、将内部事件转为前端 AG-UI 事件。

### `services/orchestrator`

唯一编排层。职责：RulePlanner、OrchestrationPlan 生成、PlanValidator 校验、Agent Registry 查询与 Health Check 过滤、single / ordered_parallel / sequential 执行策略、A2A Dispatcher、fallback / retry、多 Agent 结果聚合。

当前使用 **RulePlanner**（确定性规则），不依赖真实 LLM。

### `services/agents/code-agent`

当前已服务化的代码类 demo Agent。v0.1 mock / deterministic response，实现 `/health`、A2A Server、AgentCard。Dockerfile 已就绪。

### `services/agents/web-agent`

当前已服务化的 Web/UI 类 demo Agent。v0.1 mock / deterministic response，实现 `/health`、A2A Server、AgentCard。Dockerfile 已就绪。

### `docker-compose.new-arch.yml`

新架构主 compose，定义五服务拓扑（frontend-new → gateway-new → orchestrator-new → code-agent-new / web-agent-new）。

### `smoke-new-arch.sh`

新架构 smoke test，覆盖 health check、single code、single web、mixed ordered_parallel、log safety。

---

## 当前 Agent 状态

| Agent | 状态 | 响应模式 | 默认端口 |
|-------|------|----------|----------|
| code-agent | ✅ 已服务化 | Mock / LLM 双模式 | 8081 |
| web-agent | ✅ 已服务化 | Mock / LLM 双模式 | 8082 |
| document-agent | ✅ 已服务化 | Mock / LLM 双模式 | 8083 |
| vision-agent | ✅ 已服务化 | Mock / LLM 双模式 | 8084 |
| context-agent | ✅ 已服务化 | Mock / LLM 双模式 | 8085 |
| test-agent | ✅ 已服务化 | Mock / LLM 双模式 | 8086 |
| review-agent | ✅ 已服务化 | Mock / LLM 双模式 | 8087 |
| security-agent | ✅ 已服务化 | Mock / LLM 双模式 | 8088 |
| deploy-agent | ✅ 已服务化 | Mock / LLM 双模式 | 8089 |
| diff-agent | ✅ 已服务化 | Mock / LLM 双模式 | 8090 |

- 所有 Agent 默认运行在 **Mock 模式**（不依赖 LLM），CI / smoke 可直接通过
- 设置对应 `{PREFIX}_AGENT_LLM_MODEL` 环境变量 + API Key 可切换到 LLM 模式
- 所有 Agent 支持流式输出（`Streaming: true`），通过 A2A SSE 协议与 Orchestrator 通信

---

## 环境依赖

| 依赖 | 最低版本 | 说明 |
|------|----------|------|
| Go | ≥ 1.23 | 后端编译（go.work 要求 1.26，实际兼容 1.23+） |
| Node.js | ≥ 18 | 前端开发 |
| npm | ≥ 9 | 前端包管理 |
| Docker & Docker Compose | latest | 一键部署（可选，本地开发可不用） |

---

## 服务端口规划

| 服务 | 本地开发端口 | Docker 端口 | 说明 |
|------|-------------|-------------|------|
| **前端 (Vite)** | 3000 | 3000 | React 开发服务器 |
| **Gateway** | 8080 | 8080 | 对外 API 入口 |
| **Orchestrator** | 8090 | 8090 | 编排服务 |
| code-agent | 8081 | 8081 | 代码生成 Agent |
| web-agent | 8082 | 8082 | 网页生成 Agent |
| document-agent | 8083 | 8083 | 文档生成 Agent |
| vision-agent | 8084 | 8084 | 图像分析 Agent |
| context-agent | 8085 | 8085 | 上下文压缩 Agent |
| test-agent | 8086 | 8086 | 测试分析 Agent |
| review-agent | 8087 | 8087 | 代码审查 Agent |
| security-agent | 8088 | 8088 | 安全扫描 Agent |
| deploy-agent | 8089 | 8089 | 部署计划 Agent |
| diff-agent | 8090 | 8090 | Diff 生成 Agent |

---

## 环境变量配置

### 一、Orchestrator 编排服务（必读）

Orchestrator 是核心编排层，需要知道所有 Agent 的地址。

| 变量名 | 必填 | 默认值 | 说明 |
|--------|------|--------|------|
| `ORCHESTRATOR_ADDR` | 否 | `:8080` | 编排服务监听地址 |
| `INTERNAL_SERVICE_TOKEN` | 否 | (空) | 内部服务间认证令牌，Gateway 需匹配 |
| `CODE_AGENT_URL` | 否 | `http://127.0.0.1:8081` | code-agent 地址 |
| `WEB_AGENT_URL` | 否 | `http://127.0.0.1:8082` | web-agent 地址 |
| `DOCUMENT_AGENT_URL` | 否 | `http://127.0.0.1:8083` | document-agent 地址 |
| `VISION_AGENT_URL` | 否 | `http://127.0.0.1:8084` | vision-agent 地址 |
| `CONTEXT_AGENT_URL` | 否 | `http://127.0.0.1:8085` | context-agent 地址 |
| `TEST_AGENT_URL` | 否 | `http://127.0.0.1:8086` | test-agent 地址 |
| `REVIEW_AGENT_URL` | 否 | `http://127.0.0.1:8087` | review-agent 地址 |
| `SECURITY_AGENT_URL` | 否 | `http://127.0.0.1:8088` | security-agent 地址 |
| `DEPLOY_AGENT_URL` | 否 | `http://127.0.0.1:8089` | deploy-agent 地址 |
| `DIFF_AGENT_URL` | 否 | `http://127.0.0.1:8090` | diff-agent 地址 |
| `ORCHESTRATOR_AGENT_CARD_LOAD` | 否 | (关) | 设为 `true` 从 Agent 动态加载能力卡片 |
| `REGISTRY_HEALTHCHECK` | 否 | (开) | 设为 `off` 禁用健康检查（CI 模式） |
| `HITL_ENABLED` | 否 | (关) | 设为 `true` 启用高风险任务 HITL 确认 |
| `ORCHESTRATOR_MAX_TASKS` | 否 | `5` | 单计划最大任务数 |

**Orchestrator LLM 配置**（启用 LLM Planner + Synthesizer 时必填）：

| 变量名 | 必填 | 默认值 | 说明 |
|--------|------|--------|------|
| `ORCHESTRATOR_LLM_PROVIDER` | 否 | `anthropic` | LLM 提供商：`anthropic` / `openai` |
| `ORCHESTRATOR_LLM_MODEL` | 否 | (空) | 模型名，如 `claude-sonnet-4-20250514` |
| `ORCHESTRATOR_LLM_API_KEY` | 条件 | (空) | API Key（优先级最高） |
| `ORCHESTRATOR_LLM_BASE_URL` | 否 | (空) | 自定义 Base URL |
| `ANTHROPIC_API_KEY` | 条件 | (空) | Anthropic Key（`ORCHESTRATOR_LLM_API_KEY` 未设时回退） |
| `OPENAI_API_KEY` | 条件 | (空) | OpenAI Key（provider=openai 时回退） |

**Orchestrator 韧性配置**（可选，未设置时无额外行为）：

| 变量名 | 必填 | 默认值 | 说明 |
|--------|------|--------|------|
| `DISPATCH_MAX_RETRY` | 否 | (不启用) | 调度失败重试次数，≥1 启用 |
| `DISPATCH_RETRY_BACKOFF_MS` | 否 | `200` | 重试退避基础间隔（毫秒） |
| `DISPATCH_TIMEOUT_MS` | 否 | (不启用) | 单次调度调用超时（毫秒） |
| `DISPATCH_BREAKER_THRESHOLD` | 否 | (不启用) | 熔断器连续失败阈值，≥1 启用 |
| `DISPATCH_BREAKER_COOLDOWN_MS` | 否 | `30000` | 熔断器冷却时间（毫秒） |

---

### 二、Gateway 网关服务

| 变量名 | 必填 | 默认值 | 说明 |
|--------|------|--------|------|
| `GATEWAY_ADDR` | 否 | `:8080` | 网关监听地址 |
| `AGENTHUB_API_TOKEN` | 否 | (空) | 前端 API 认证令牌 |
| `GATEWAY_ENABLE_AUTH` | 否 | 取决于 TOKEN | 是否启用前端认证 |
| `GATEWAY_ALLOWED_ORIGINS` | 否 | `http://localhost:3000,http://127.0.0.1:3000` | CORS 允许来源 |
| `GATEWAY_DEFAULT_AGENT_NAME` | 否 | `code-agent` | 默认 Agent |
| `ORCHESTRATOR_URL` | **是** | (空) | 编排服务地址，如 `http://127.0.0.1:8090` |
| `ORCHESTRATOR_INTERNAL_TOKEN` | 否 | (空) | 编排服务内部令牌，需与 Orchestrator 一致 |
| `AGENTHUB_GATEWAY_STORE` | 否 | `memory` | 存储模式：`memory` / `sqlite` |
| `AGENTHUB_SQLITE_PATH` | 否 | `/data/agenthub.db` | SQLite 路径（store=sqlite 时） |

---

### 三、各 Agent 子服务环境变量

所有 10 个 Agent 遵循相同的环境变量命名规范，仅前缀不同。**不设置 LLM 相关变量时，Agent 默认运行在 Mock 模式**。

**通用模式**（`{PREFIX}` 替换为下表中的值）：

| 变量名 | 必填 | 默认值 | 说明 |
|--------|------|--------|------|
| `{PREFIX}_AGENT_ADDR` | 否 | `:8080` | Agent 监听地址 |
| `{PREFIX}_AGENT_PUBLIC_URL` | 否 | `http://localhost:8080` | Agent 对外公开 URL |
| `{PREFIX}_AGENT_LLM_MODEL` | 否 | (空) | LLM 模型名，不设则 Mock 模式 |
| `{PREFIX}_AGENT_LLM_PROVIDER` | 否 | `anthropic` | LLM 提供商 |
| `{PREFIX}_AGENT_LLM_API_KEY` | 否 | (空) | LLM API Key（优先） |
| `{PREFIX}_AGENT_LLM_BASE_URL` | 否 | (空) | 自定义 Base URL |

**Agent 前缀映射表**：

| Agent | 前缀 | 默认端口 | 入口文件 |
|-------|------|----------|----------|
| code-agent | `CODE` | 8081 | `services/agents/code-agent/cmd/code-agent/main.go` |
| web-agent | `WEB` | 8082 | `services/agents/web-agent/cmd/web-agent/main.go` |
| document-agent | `DOCUMENT` | 8083 | `services/agents/document-agent/cmd/document-agent/main.go` |
| vision-agent | `VISION` | 8084 | `services/agents/vision-agent/cmd/vision-agent/main.go` |
| context-agent | `CONTEXT` | 8085 | `services/agents/context-agent/cmd/context-agent/main.go` |
| test-agent | `TEST` | 8086 | `services/agents/test-agent/cmd/test-agent/main.go` |
| review-agent | `REVIEW` | 8087 | `services/agents/review-agent/cmd/review-agent/main.go` |
| security-agent | `SECURITY` | 8088 | `services/agents/security-agent/cmd/security-agent/main.go` |
| deploy-agent | `DEPLOY` | 8089 | `services/agents/deploy-agent/cmd/deploy-agent/main.go` |
| diff-agent | `DIFF` | 8090 | `services/agents/diff-agent/cmd/diff-agent/main.go` |

**API Key 回退逻辑**（每个 Agent 通用）：
1. 优先读取 `{PREFIX}_AGENT_LLM_API_KEY`
2. provider=openai 时回退到 `OPENAI_API_KEY`
3. 其他 provider 回退到 `ANTHROPIC_API_KEY`

**LLM 模式示例**（以 code-agent 为例）：
```bash
export CODE_AGENT_LLM_MODEL="claude-sonnet-4-20250514"
export CODE_AGENT_LLM_PROVIDER="anthropic"
export ANTHROPIC_API_KEY="sk-ant-xxx"
```

**Mock 模式示例**（默认，无需任何环境变量）：
```bash
# 直接启动即可，Agent 返回确定性模拟响应
go run ./services/agents/code-agent/cmd/code-agent/
```

---

### 四、前端环境变量

| 变量名 | 必填 | 默认值 | 说明 |
|--------|------|--------|------|
| `VITE_AGENTHUB_API_TOKEN` | 否 | (空) | 前端请求 API 的 Bearer Token |

前端 Vite 开发服务器自动将 `/api` 代理到 `http://localhost:8080`（Gateway），无需额外配置。

---

## 本地开发启动

### 前置步骤：安装依赖

```bash
cd Multi_Agent-AgentHub

# 1. 安装前端依赖
cd frontend && npm install && cd ..

# 2. 验证 Go 编译环境
go version  # 应输出 go1.23+
```

### 方式一：全服务 Mock 模式（快速开发，不依赖 LLM）

**推荐首次开发者使用此方式**，无需任何 API Key。

```bash
# Terminal 1: 启动 Gateway（端口 8080）
ORCHESTRATOR_URL="http://127.0.0.1:8090" \
  go run ./services/gateway/cmd/gateway/

# Terminal 2: 启动 Orchestrator（端口 8090）
ORCHESTRATOR_ADDR=":8090" \
  go run ./services/orchestrator/cmd/orchestrator/

# Terminal 3: 启动 code-agent（端口 8081）
CODE_AGENT_ADDR=":8081" \
CODE_AGENT_PUBLIC_URL="http://127.0.0.1:8081" \
  go run ./services/agents/code-agent/cmd/code-agent/

# Terminal 4: 启动 web-agent（端口 8082）
WEB_AGENT_ADDR=":8082" \
WEB_AGENT_PUBLIC_URL="http://127.0.0.1:8082" \
  go run ./services/agents/web-agent/cmd/web-agent/

# Terminal 5: 启动前端（端口 3000）
cd frontend && npm run dev
```

**验证**：访问 http://localhost:3000，发送消息测试。

---

### 方式二：全服务 LLM 模式（启用真实 AI）

```bash
# 设置通用 LLM 环境变量
export ANTHROPIC_API_KEY="sk-ant-xxx"

# Gateway + Orchestrator（同上，增加 LLM Planner）
ORCHESTRATOR_ADDR=":8090" \
ORCHESTRATOR_LLM_MODEL="claude-sonnet-4-20250514" \
ORCHESTRATOR_LLM_API_KEY="$ANTHROPIC_API_KEY" \
  go run ./services/orchestrator/cmd/orchestrator/ &

ORCHESTRATOR_URL="http://127.0.0.1:8090" \
  go run ./services/gateway/cmd/gateway/ &

# 各 Agent 启用 LLM
CODE_AGENT_ADDR=":8081" CODE_AGENT_LLM_MODEL="claude-sonnet-4-20250514" \
  go run ./services/agents/code-agent/cmd/code-agent/ &

WEB_AGENT_ADDR=":8082" WEB_AGENT_LLM_MODEL="claude-sonnet-4-20250514" \
  go run ./services/agents/web-agent/cmd/web-agent/ &

# 前端
cd frontend && npm run dev
```

---

### 方式三：启动指定 Agent 子集

Orchestrator 按需连接 Agent，未启动的 Agent 会在调度时触发 fallback。

```bash
# 最小子集：仅 code-agent + document-agent + review-agent
# Terminal 1: Orchestrator
ORCHESTRATOR_ADDR=":8090" go run ./services/orchestrator/cmd/orchestrator/

# Terminal 2: Gateway
ORCHESTRATOR_URL="http://127.0.0.1:8090" go run ./services/gateway/cmd/gateway/

# Terminal 3-5: 三个 Agent
CODE_AGENT_ADDR=":8081" go run ./services/agents/code-agent/cmd/code-agent/ &
DOCUMENT_AGENT_ADDR=":8083" go run ./services/agents/document-agent/cmd/document-agent/ &
REVIEW_AGENT_ADDR=":8087" go run ./services/agents/review-agent/cmd/review-agent/ &
```

---

### 方式四：启动全部 12 个服务

```bash
#!/bin/bash
# 启动全部服务脚本（保存为 start-all.sh）
set -e

export ANTHROPIC_API_KEY="${ANTHROPIC_API_KEY:-}"

# Gateway
ORCHESTRATOR_URL="http://127.0.0.1:8090" \
  go run ./services/gateway/cmd/gateway/ &

# Orchestrator
ORCHESTRATOR_ADDR=":8090" \
ORCHESTRATOR_LLM_MODEL="${ORCHESTRATOR_LLM_MODEL:-}" \
  go run ./services/orchestrator/cmd/orchestrator/ &

# 10 Agent（全部 Mock 模式）
for agent in code web document vision context test review security deploy diff; do
  prefix=$(echo "$agent" | tr '[:lower:]' '[:upper:]' | tr '-' '_')
  port=$((8080 + $(echo "code web document vision context test review security deploy diff" | tr ' ' '\n' | grep -n "^$agent$" | cut -d: -f1)))
  eval "${prefix}_AGENT_ADDR=\":${port}\" ${prefix}_AGENT_PUBLIC_URL=\"http://127.0.0.1:${port}\" \
    go run ./services/agents/${agent}-agent/cmd/${agent}-agent/" &
done

# 前端
cd frontend && npm run dev &

wait
```

---

## Docker 一键启动

```bash
# 构建并启动所有服务
docker compose -f docker-compose.new-arch.yml up --build

# 后台运行
docker compose -f docker-compose.new-arch.yml up -d --build

# 停止
docker compose -f docker-compose.new-arch.yml down

# 查看日志
docker compose -f docker-compose.new-arch.yml logs -f
```

启动后访问：
- **前端界面**：http://localhost:3000
- **Gateway API**：http://localhost:8080
- **Orchestrator**：http://localhost:8090
- **各 Agent**：http://localhost:8081 ~ 8090

---

## 测试命令

```bash
# 全量测试（新架构核心服务，deterministic）
go test ./services/orchestrator/... ./services/gateway/... \
  ./services/agents/code-agent/... ./services/agents/web-agent/... \
  -count=1

# Orchestrator 单包测试（含 race 检测）
go test -race ./services/orchestrator/... -count=1

# ADK 基础库测试
go test ./pkg/adk/... -count=1

# 前端单元测试
cd frontend && npm test

# 前端 E2E 测试
cd frontend && npm run test:e2e

# Smoke 验收测试
./smoke-new-arch.sh

# Doctor 环境检查
./doctor-new-arch.sh
```

---

## 验证命令

### 健康检查

```bash
# Gateway
curl -s http://localhost:8080/health | jq .

# Orchestrator
curl -s http://localhost:8090/health | jq .

# 各 Agent
for port in $(seq 8081 8090); do
  echo "Port $port: $(curl -s -o /dev/null -w '%{http_code}' http://localhost:$port/health 2>/dev/null || echo 'DOWN')"
done
```

### AgentCard 发现

```bash
# 查看 Agent 能力描述
curl -s http://localhost:8081/.well-known/agent.json | jq .
```

### A2A 协议直接调用 Agent

```bash
curl -s -N -X POST http://localhost:8081/ \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tasks/sendSubscribe",
    "id": "1",
    "params": {
      "sessionId": "test-session",
      "message": {"role": "user", "content": "用 Go 写一个 hello world"}
    }
  }'
```

### AG-UI 协议完整链路测试

```bash
curl -s -N -X POST http://localhost:8080/api/agui/run \
  -H "Content-Type: application/json" \
  -d '{
    "threadId": "test-conv-001",
    "runId": "test-run-001",
    "messages": [
      {"role": "user", "content": "帮我写一个 Go HTTP 服务"}
    ]
  }'
```

---

## 常见问题

**Q: Agent 未启动时，Orchestrator 会怎样？**
A: Orchestrator 的 Health Checker 会检测 Agent 不可用，Planner 不会再调度到该 Agent。如果已调度的任务目标 Agent 不可达，会触发 `ORCHESTRATOR_AGENT_UNAVAILABLE` 错误并通过 fallback 机制降级。

**Q: Mock 模式和 LLM 模式如何切换？**
A: Agent 和 Orchestrator 都通过环境变量控制。不设置 `*_LLM_MODEL` 时自动使用 Mock 模式，无需 API Key。设置后自动切换到 LLM 模式。

**Q: 如何只启动部分 Agent 进行开发？**
A: Orchestrator 的 Health Checker 会识别哪些 Agent 在线，Planner 根据可用 Agent 生成计划。只需启动你需要的 Agent 即可。

**Q: 前端开发时如何连接后端？**
A: Vite 开发服务器自动将 `/api` 请求代理到 `http://localhost:8080`（Gateway），无需额外配置 CORS。

**Q: 如何添加新的 Agent？**
A: 参考 `services/agents/code-agent/` 的完整文件结构，创建 `agent.go`、`prompt.go`、`tools.go`、`server.go`、`cmd/{name}/main.go`、`go.mod`，然后在 Orchestrator 的 `main.go` 中注册 AgentEndpoint。详见现有 Agent 模式。

**Q: `DISPATCH_MAX_RETRY` 等韧性参数需要设置吗？**
A: 本地开发不需要。这些是生产环境或压测时的可选配置，未设置时 Orchestrator 使用默认行为（单次调用、无熔断）。

**Q: Anthropic API Key 从哪里获取？**
A: 访问 https://console.anthropic.com/ 创建 API Key。

**Q: 可以使用其他模型吗？**
A: 可以。通过 `LLM_PROVIDER` 切换 Anthropic / OpenAI 兼容服务（DeepSeek、Ollama 等），配合 `LLM_BASE_URL` 指向自定义端点。

---

## 旧路径边界

| 路径 | 定位 |
|------|------|
| `server/` | Legacy 路径（MVP v0.1 Gateway + 内嵌 Orchestrator），不再作为新功能主入口 |
| `agents/` | 旧 Agent 能力池，已被 `services/agents/*` 替代 |
| `docker-compose.yml` | Legacy compose，不基于此扩展新功能 |
| Gateway 直连 Agent | 旧路径，**禁止扩展**。新功能必须走 Gateway → Orchestrator → Agent |

---

## 后续开发入口

当前处于 AgentHub v1.0 Productization Stage。后续 Milestone 主线：

1. CI / Smoke 正式化
2. 前端 Multi-Agent SSE 展示优化
3. Conversation / Run / Message 持久化
4. Artifact 与 Runtime Preview 落地
5. AgentCard / Registry 契约化
6. LLMPlanner 受控接入（feature flag）

详见：
- [productization-stage-guide.md](docs/refactor/productization-stage-guide.md)
- [current-architecture-state.md](docs/refactor/current-architecture-state.md)
- [legacy-boundary.md](docs/refactor/legacy-boundary.md)

---

## Legacy / Historical Notes

以下内容来自 MVP v0.1 时期，仅作为历史参考。当前主路径见上方新架构说明。

---

## Legacy / Historical Notes

以下内容来自 MVP v0.1 时期，仅作为历史参考。当前主路径见上方新架构说明。

<details>
<summary>MVP v0.1 历史架构（点击展开）</summary>

### 历史架构概览

```
Frontend (React + AG-UI SSE)
       │
       ▼
Gateway (Go/Gin, port 8080)
  └─ Orchestrator (内嵌, A2A↔AG-UI 协议转换)
       │
       ▼  A2A Protocol (a2a-go/v2)
Code-Agent (Go, port 8081)
  └─ Anthropic Claude API (流式)
       │
       ▼
MySQL 8 (持久化)
```

### 历史环境依赖

| 依赖 | 版本要求 | 说明 |
|------|----------|------|
| Go | >= 1.26 | 后端 + Agent 编译 |
| Node.js | >= 18 | 前端开发 |
| npm | >= 9 | 前端包管理 |
| Docker & Docker Compose | latest | 一键部署 |
| MySQL | 8.0 | 数据库 |

### 历史项目结构

```
├── frontend/                  # React 前端
│   ├── src/
│   │   ├── components/        # UI 组件
│   │   ├── agui/              # AG-UI SSE 客户端
│   │   ├── stores/            # Zustand 状态管理
│   │   └── services/          # REST API 调用
│   └── package.json
│
├── server/                    # Legacy Go Gateway + 内嵌 Orchestrator
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── handler/           # HTTP handlers
│   │   ├── orchestrator/      # 编排 + A2A↔AG-UI 协议转换
│   │   ├── a2a/               # A2A 客户端
│   │   ├── store/             # MySQL 数据操作
│   │   └── model/             # 数据模型
│   └── go.mod
│
├── agents/                    # 旧 Agent 能力池
│   ├── adk/                   # 旧 ADK Runtime
│   │   ├── context.go
│   │   ├── server.go
│   │   └── llm.go
│   ├── code-agent/            # 旧版 code-agent
│   │   ├── main.go
│   │   ├── handler.go
│   │   └── config.yaml
│   └── go.mod
│
├── services/                  # 新架构主路径（当前）
│   ├── gateway/
│   ├── orchestrator/
│   └── agents/
│       ├── code-agent/
│       └── web-agent/
│
├── docs/                      # 架构文档 + 契约
│   ├── contracts/
│   ├── architecture/
│   └── refactor/
│
├── docker-compose.yml         # Legacy compose
├── docker-compose.new-arch.yml # 新架构 compose
└── smoke-new-arch.sh           # 新架构 smoke
```

### 历史快速开始（Legacy Compose）

```bash
cp .env.example .env
# 编辑 .env，填入 ANTHROPIC_API_KEY
make docker-up
```

### 历史开发命令

```bash
make install       # 安装所有依赖
make dev           # 启动所有服务（并行）
make dev-frontend  # 仅启动前端
make dev-server    # 仅启动 Gateway
make dev-agent     # 仅启动 Code-Agent
make docker-up     # Docker 一键启动
make docker-down   # 停止并清理 Docker
make build-check   # 验证所有模块编译通过
```

### 历史环境变量说明

**基础配置：**

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `DATABASE_URL` | `root:agenthub123@tcp(localhost:3306)/agenthub?...` | MySQL 连接字符串 |
| `GATEWAY_PORT` | `8080` | Gateway 服务端口 |
| `CODE_AGENT_PORT` | `8081` | Code-Agent 服务端口 |
| `AGENT_CODE_URL` | `http://localhost:8081` | Gateway 连接 Agent 的地址 |

**LLM Provider 配置：**

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `LLM_PROVIDER` | `anthropic` | 选择 Provider：`anthropic` 或 `openai` |

Anthropic 配置：`ANTHROPIC_API_KEY`、`ANTHROPIC_MODEL`、`ANTHROPIC_BASE_URL`

OpenAI 配置：`OPENAI_API_KEY`、`OPENAI_MODEL`、`OPENAI_BASE_URL`

### 历史技术栈

| 层 | 技术 |
|----|------|
| 前端 | React 18 + TypeScript + Tailwind CSS + Vite |
| 状态管理 | Zustand |
| 代码高亮 | highlight.js |
| Gateway | Go + Gin |
| 协议库 | a2a-go/v2 (A2A 官方协议) |
| LLM | Anthropic Claude API |
| 数据库 | MySQL 8.0 |
| 部署 | Docker Compose |

### 历史 API 测试用例（curl）

**AgentCard — Agent 自描述发现：**

```bash
curl -s http://localhost:8081/.well-known/agent.json | jq .
```

**A2A 协议 — Gateway → Code-Agent：**

```bash
curl -s -N -X POST http://localhost:8081/ \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tasks/sendSubscribe",
    "id": "test-001",
    "params": {
      "message": {
        "role": "user",
        "parts": [
          {"type": "text", "text": "用 Go 写一个 hello world"}
        ]
      }
    }
  }'
```

**AG-UI 协议 — 前端流式展示：**

```bash
curl -s -N -X POST http://localhost:8080/api/agui/run \
  -H "Content-Type: application/json" \
  -d '{
    "threadId": "test-conv-001",
    "runId": "test-run-001",
    "messages": [
      {"role": "user", "content": "用 Go 写一个 hello world"}
    ],
    "tools": [
      {"name": "code_preview"}
    ]
  }'
```

### 历史常见问题

**Q: Anthropic API Key 从哪里获取？**
A: 访问 https://console.anthropic.com/ 创建 API Key。

**Q: 可以使用其他模型吗？**
A: 可以。通过 `LLM_PROVIDER` 切换 Anthropic / OpenAI 兼容服务（DeepSeek、Ollama 等）。

**Q: 本地开发时前端如何连接 Gateway？**
A: Vite 配置了 `/api` 代理到 `http://localhost:8080`，无需额外配置 CORS。

**Q: Docker 启动后 MySQL 初始化失败？**
A: 执行 `make docker-down` 清理 volume 后重新 `make docker-up`。

</details>
