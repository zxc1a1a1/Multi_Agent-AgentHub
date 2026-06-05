# AgentHub 前后端联调配置指南

## 目录

1. [快速启动（Docker 一键）](#1-快速启动docker-一键)
2. [本地开发模式（无 Docker）](#2-本地开发模式无-docker)
3. [混合模式（Docker 后端 + 本地前端）](#3-混合模式docker-后端--本地前端)
4. [启用 LLM 增强功能](#4-启用-llm-增强功能)
5. [Smoke Test 验收](#5-smoke-test-验收)
6. [服务端口速查](#6-服务端口速查)
7. [常见问题排查](#7-常见问题排查)

---

## 1. 快速启动（Docker 一键）

**前置要求：** Docker + Docker Compose

```bash
# 在项目根目录执行
docker compose -f docker-compose.new-arch.yml up --build
```

等待所有服务 healthy 后访问：

| 服务 | 地址 |
|------|------|
| **前端界面** | http://localhost:3000 |
| Gateway API | http://localhost:8080 |
| Orchestrator | http://localhost:8090 |
| Code Agent | http://localhost:8081 |
| Web Agent | http://localhost:8082 |

**停止并清理：**

```bash
docker compose -f docker-compose.new-arch.yml down -v
```

---

## 2. 本地开发模式（无 Docker）

适合频繁修改代码、需要热重载的开发场景。

### 2.1 前置要求

- Go >= 1.26
- Node.js >= 18, npm >= 9

### 2.2 启动后端（4 个终端窗口）

```bash
# 终端1：启动 Code Agent（端口 8081）
cd services/agents/code-agent
go run ./cmd/code-agent
# 默认监听 :8080（容器内）, 本地会映射到 8081
# 实际监听端口由 CODE_AGENT_ADDR 环境变量决定

# Windows PowerShell:
$env:CODE_AGENT_ADDR=":8081"; go run ./cmd/code-agent

# Linux/Mac:
CODE_AGENT_ADDR=:8081 go run ./cmd/code-agent
```

```bash
# 终端2：启动 Web Agent（端口 8082）
$env:WEB_AGENT_ADDR=":8082"; go run services/agents/web-agent/cmd/web-agent
```

```bash
# 终端3：启动 Orchestrator（端口 8090）
$env:ORCHESTRATOR_ADDR=":8090"; `
$env:CODE_AGENT_URL="http://127.0.0.1:8081"; `
$env:WEB_AGENT_URL="http://127.0.0.1:8082"; `
$env:INTERNAL_SERVICE_TOKEN="dev-internal-token"; `
go run services/orchestrator/cmd/orchestrator
```

```bash
# 终端4：启动 Gateway（端口 8080）
$env:GATEWAY_ADDR=":8080"; `
$env:ORCHESTRATOR_URL="http://127.0.0.1:8090"; `
$env:ORCHESTRATOR_INTERNAL_TOKEN="dev-internal-token"; `
$env:GATEWAY_ENABLE_AUTH="false"; `
$env:GATEWAY_ALLOWED_ORIGINS="http://localhost:5173,http://127.0.0.1:5173"; `
$env:AGENTHUB_GATEWAY_STORE="sqlite"; `
$env:AGENTHUB_SQLITE_PATH="./data/dev-gateway.db"; `
go run services/gateway/cmd/gateway
```

### 2.3 启动前端（第5个终端）

```bash
cd frontend
npm install        # 首次运行
npm run dev        # 启动 Vite 开发服务器 → http://localhost:5173
```

Vite 自动将 `/api` 请求代理到 `http://localhost:8080`（Gateway），无需额外 CORS 配置。

### 2.4 验证后端启动成功

```bash
# 检查所有服务健康状态
curl http://localhost:8081/health    # code-agent
curl http://localhost:8082/health    # web-agent
curl http://localhost:8090/health    # orchestrator
curl http://localhost:8080/health    # gateway
```

---

## 3. 混合模式（Docker 后端 + 本地前端）

后端改动较少、主要调试前端时使用。

### 3.1 启动 Docker 后端（不含前端）

```bash
# 只启动4个后端服务，不启动前端
docker compose -f docker-compose.new-arch.yml up --build code-agent-new web-agent-new orchestrator-new gateway-new
```

### 3.2 启动本地前端

```bash
cd frontend
npm install
npm run dev
# 访问 http://localhost:5173
# Vite 自动代理 /api → http://localhost:8080
```

---

## 4. 启用 LLM 增强功能

> **注意：** 不启用 LLM 时，系统使用 mock 模式，code-agent 和 web-agent 返回固定模板响应。启用 LLM 后，Agent 将调用真实 AI 模型生成响应。

### 4.1 环境变量配置

在启动后端服务前，设置以下环境变量：

```bash
# ==========================================
# 通用：LLM Provider 配置
# ==========================================
# 选择 provider: anthropic 或 openai
# Anthropic（默认）:
export ANTHROPIC_API_KEY=sk-ant-xxxxxxxxxxxxx

# OpenAI:
export OPENAI_API_KEY=sk-xxxxxxxxxxxxx

# ==========================================
# 4.2 启用 LLM Planner（Orchestrator）
# ==========================================
export ORCHESTRATOR_PLANNER=llm                  # 设为 "llm" 启用，不设或"rule"使用规则
export ORCHESTRATOR_LLM_PROVIDER=anthropic       # 默认 anthropic
export ORCHESTRATOR_LLM_MODEL=claude-haiku-4-5-20251001  # 推荐轻量模型降成本
# 可选:
# export ORCHESTRATOR_LLM_BASE_URL=https://api.anthropic.com
# export ORCHESTRATOR_LLM_API_KEY=sk-ant-xxxxx   # 如不设则用 ANTHROPIC_API_KEY

# ==========================================
# 4.3 启用真实 LLM Agent
# ==========================================
# Code Agent:
export CODE_AGENT_LLM_MODEL=claude-sonnet-4-20250514
export CODE_AGENT_LLM_PROVIDER=anthropic
# 可选:
# export CODE_AGENT_LLM_API_KEY=sk-ant-xxxxx
# export CODE_AGENT_LLM_BASE_URL=https://api.anthropic.com

# Web Agent:
export WEB_AGENT_LLM_MODEL=claude-sonnet-4-20250514
export WEB_AGENT_LLM_PROVIDER=anthropic
```

### 4.4 Docker 模式下启用 LLM

编辑 `docker-compose.new-arch.yml` 或在 `docker-compose.override.yml` 中追加环境变量：

```yaml
# docker-compose.override.yml
services:
  orchestrator-new:
    environment:
      ORCHESTRATOR_PLANNER: "llm"
      ORCHESTRATOR_LLM_PROVIDER: "anthropic"
      ORCHESTRATOR_LLM_MODEL: "claude-haiku-4-5-20251001"
      ANTHROPIC_API_KEY: "${ANTHROPIC_API_KEY}"

  code-agent-new:
    environment:
      CODE_AGENT_LLM_MODEL: "claude-sonnet-4-20250514"
      CODE_AGENT_LLM_PROVIDER: "anthropic"
      ANTHROPIC_API_KEY: "${ANTHROPIC_API_KEY}"

  web-agent-new:
    environment:
      WEB_AGENT_LLM_MODEL: "claude-sonnet-4-20250514"
      WEB_AGENT_LLM_PROVIDER: "anthropic"
      ANTHROPIC_API_KEY: "${ANTHROPIC_API_KEY}"
```

启动：

```bash
ANTHROPIC_API_KEY=sk-ant-xxxxx docker compose -f docker-compose.new-arch.yml -f docker-compose.override.yml up --build
```

### 4.5 使用其他模型

```bash
# DeepSeek:
export CODE_AGENT_LLM_PROVIDER=openai
export CODE_AGENT_LLM_MODEL=deepseek-chat
export CODE_AGENT_LLM_BASE_URL=https://api.deepseek.com/v1
export OPENAI_API_KEY=sk-xxxxxxxx

# Ollama 本地:
export CODE_AGENT_LLM_PROVIDER=openai
export CODE_AGENT_LLM_MODEL=llama3
export CODE_AGENT_LLM_BASE_URL=http://localhost:11434/v1
export OPENAI_API_KEY=ollama
```

---

## 5. Smoke Test 验收

### 5.1 运行 Smoke Test（Docker 模式）

```bash
# 后端已通过 Docker 启动后：
bash ./smoke-new-arch.sh
```

Smoke 覆盖：
- 4 个服务健康检查
- 单 code-agent 请求（SSE 流）
- 单 web-agent 请求（SSE 流）
- 混合 ordered_parallel 请求（web + code 同时）
- 多 Agent 输出区分
- 消息回放/持久化
- 日志安全（无 API key 泄漏）

### 5.2 手动 curl 测试

```bash
# 1. 健康检查
curl -s http://localhost:8080/health | jq .

# 2. 列出 Agent
curl -s http://localhost:8080/api/agents | jq .

# 3. 创建会话
curl -s -X POST http://localhost:8080/api/conversations \
  -H "Content-Type: application/json" \
  -d '{"userId":"dev-user","agentName":"code-agent"}' | jq .
# → 获取返回的 id（conversationId）

# 4. 发送消息（SSE 流式响应）
curl -s -N -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{"conversationId":"<替换为上一步的id>","message":"用Go写一个HTTP服务器"}'

# 5. 混合多Agent请求
curl -s -N -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{"conversationId":"<替换>","message":"写一个HTML登录页面，并实现Go API接口"}'

# 6. 回放历史消息
curl -s http://localhost:8080/api/conversations/<conversationId>/messages | jq .
```

### 5.3 Agent 直接调用（绕过 Gateway）

```bash
# 获取 AgentCard
curl -s http://localhost:8081/.well-known/agent.json | jq .

# 通过 A2A JSON-RPC 直接调用 Code Agent
curl -s -X POST http://localhost:8081/a2a/tasks/sendSubscribe \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc":"2.0",
    "method":"tasks/sendSubscribe",
    "id":"1",
    "params":{
      "sessionId":"test-session-1",
      "message":{"role":"user","content":"用Go写一个HTTP服务器"}
    }
  }' | jq .
```

---

## 6. 服务端口速查

| 服务 | Docker 模式 | 本地开发 | 健康检查 |
|------|:----------:|:--------:|----------|
| Frontend | `:3000` | `:5173` (Vite) | `GET /` |
| Gateway | `:8080` | `:8080` | `GET /health` |
| Orchestrator | `:8090` | `:8090` | `GET /health` |
| Code Agent | `:8081` | `:8081` | `GET /health` |
| Web Agent | `:8082` | `:8082` | `GET /health` |

### 环境变量速查表

| 变量 | 服务 | 默认值 | 说明 |
|------|------|--------|------|
| `GATEWAY_ADDR` | Gateway | `:8080` | 监听地址 |
| `GATEWAY_ENABLE_AUTH` | Gateway | `true`/`false` | 鉴权开关 |
| `GATEWAY_ALLOWED_ORIGINS` | Gateway | `http://localhost:3000,...` | CORS 允许源 |
| `AGENTHUB_GATEWAY_STORE` | Gateway | `memory` | `memory` 或 `sqlite` |
| `AGENTHUB_SQLITE_PATH` | Gateway | `/data/agenthub.db` | SQLite 路径 |
| `ORCHESTRATOR_URL` | Gateway | — | Orchestrator 地址 |
| `ORCHESTRATOR_INTERNAL_TOKEN` | Gateway | — | 内部鉴权令牌 |
| `ORCHESTRATOR_ADDR` | Orchestrator | `:8080` | 监听地址 |
| `ORCHESTRATOR_PLANNER` | Orchestrator | `rule` | `rule` 或 `llm` |
| `ORCHESTRATOR_LLM_PROVIDER` | Orchestrator | `anthropic` | LLM 供应商 |
| `ORCHESTRATOR_LLM_MODEL` | Orchestrator | `claude-haiku-4-5-20251001` | Planner 模型 |
| `ORCHESTRATOR_LLM_API_KEY` | Orchestrator | — | LLM API Key |
| `INTERNAL_SERVICE_TOKEN` | Orchestrator | — | 内部鉴权令牌 |
| `CODE_AGENT_URL` | Orchestrator | `http://127.0.0.1:8081` | Code Agent 地址 |
| `WEB_AGENT_URL` | Orchestrator | `http://127.0.0.1:8082` | Web Agent 地址 |
| `CODE_AGENT_ADDR` | Code Agent | `:8080` | 监听地址 |
| `CODE_AGENT_LLM_MODEL` | Code Agent | — | 设为模型名启用 LLM |
| `CODE_AGENT_LLM_PROVIDER` | Code Agent | `anthropic` | LLM 供应商 |
| `CODE_AGENT_LLM_API_KEY` | Code Agent | — | LLM API Key |
| `WEB_AGENT_ADDR` | Web Agent | `:8080` | 监听地址 |
| `WEB_AGENT_LLM_MODEL` | Web Agent | — | 设为模型名启用 LLM |
| `WEB_AGENT_LLM_PROVIDER` | Web Agent | `anthropic` | LLM 供应商 |
| `WEB_AGENT_LLM_API_KEY` | Web Agent | — | LLM API Key |

---

## 7. 常见问题排查

### 7.1 端口冲突

```bash
# Windows 查看端口占用
netstat -ano | findstr :8080
netstat -ano | findstr :8081
netstat -ano | findstr :5173

# 杀死占用进程（替换 PID）
taskkill /PID <PID> /F
```

### 7.2 Go 编译报错

```bash
# 确保在项目根目录（Go workspace 根目录）
cd D:\JavaProject\agent\Multi_Agent-AgentHub

# 检查 Go 版本 >= 1.26
go version

# 下载依赖
go work sync

# 编译检查
go build ./services/gateway/... ./services/orchestrator/... ./services/agents/...
```

### 7.3 前端 npm 报错

```bash
cd frontend
rm -rf node_modules package-lock.json   # 清理
npm install                              # 重新安装
npm run dev                              # 启动
```

### 7.4 CORS 错误

前端浏览器 Console 出现 CORS 错误时：
- **本地开发：** 确保 `GATEWAY_ALLOWED_ORIGINS` 包含 `http://localhost:5173`
- **Docker 模式：** 确保包含 `http://localhost:3000`

### 7.5 SSE 流中断

Gateway 日志中出现 stream error：
- 检查 Orchestrator 是否正常运行
- 检查 Agent 是否正常运行
- 查看 Gateway 日志：`docker compose -f docker-compose.new-arch.yml logs -f gateway-new`

### 7.6 看不到 LLM 编排效果

1. 确认设置了 `ORCHESTRATOR_PLANNER=llm`
2. 确认设置了 `ANTHROPIC_API_KEY` 或 `OPENAI_API_KEY`
3. 查看 Orchestrator 日志确认 Planner 模式：
   ```bash
   docker compose -f docker-compose.new-arch.yml logs orchestrator-new | grep -i planner
   ```
   应看到 `orchestrator planner: LLM mode` 或 `RulePlanner mode`

### 7.7 Agent 使用 mock 而非 LLM

1. 确认设置了 `CODE_AGENT_LLM_MODEL=某个模型名`
2. 确认 API Key 环境变量正确
3. 查看 Agent 日志：
   ```bash
   docker compose -f docker-compose.new-arch.yml logs code-agent-new | grep -i "LLM\|mock"
   ```
   应看到 `code-agent using LLM model: ...` 或 `running in mock mode`
