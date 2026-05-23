# AgentHub - Multi-Agent Collaboration Platform

AgentHub 是一个基于 IM 交互范式的多 Agent 协作平台。用户通过对话与不同 AI Agent 交互，获得流式回复和代码预览等产物。

当前为 **MVP v0.1** 版本，实现单 Agent（code-agent）+ 代码预览的端到端链路。

## 架构概览

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

**核心协议：**
- **AG-UI**：Frontend ↔ Gateway（SSE 流式事件）
- **A2A**：Gateway ↔ Code-Agent（基于 [a2a-go/v2](https://github.com/a2aproject/a2a-go) 官方协议库）

## 环境依赖

| 依赖 | 版本要求 | 说明 |
|------|----------|------|
| Go | >= 1.26 | 后端 + Agent 编译 |
| Node.js | >= 18 | 前端开发 |
| npm | >= 9 | 前端包管理 |
| Docker & Docker Compose | latest | 一键部署（可选） |
| MySQL | 8.0 | 数据库（Docker 方式会自动启动） |

## 快速开始

### 方式一：Docker 一键启动（推荐）

只需要 Docker 环境和一个 Anthropic API Key：

```bash
# 1. 克隆项目
git clone <repo-url>
cd Multi_Agent-AgentHub

# 2. 配置环境变量
cp .env.example .env
# 编辑 .env，填入你的 ANTHROPIC_API_KEY
# ANTHROPIC_API_KEY=sk-ant-xxxxx

# 3. 一键启动所有服务
make docker-up
```

启动后访问：
- **前端界面**：http://localhost:3000
- **Gateway API**：http://localhost:8080
- **Code-Agent**：http://localhost:8081/health

停止服务：
```bash
make docker-down
```

### 方式二：本地开发

适合需要热重载和调试的开发场景。

#### 1. 启动 MySQL

```bash
# 使用 Docker 启动 MySQL（推荐）
docker run -d \
  --name agenthub-mysql \
  -e MYSQL_ROOT_PASSWORD=agenthub123 \
  -e MYSQL_DATABASE=agenthub \
  -p 3306:3306 \
  -v $(pwd)/init.sql:/docker-entrypoint-initdb.d/init.sql \
  mysql:8.0 \
  --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci

# 等待 MySQL 就绪
sleep 10
```

或者连接已有的 MySQL 实例，手动执行 `init.sql` 初始化表结构。

#### 2. 配置环境变量

```bash
cp .env.example .env
```

编辑 `.env` 文件：
```env
DATABASE_URL=root:agenthub123@tcp(localhost:3306)/agenthub?charset=utf8mb4&parseTime=True&loc=Local
ANTHROPIC_API_KEY=sk-ant-xxxxx
GATEWAY_PORT=8080
CODE_AGENT_PORT=8081
AGENT_CODE_URL=http://localhost:8081
```

#### 3. 安装依赖

```bash
make install
```

这会执行：
- `cd frontend && npm install` — 前端依赖
- `cd server && go mod tidy` — Gateway 依赖
- `cd agents && go mod tidy` — Agent 依赖

#### 4. 启动所有服务

```bash
# 同时启动前端、Gateway、Code-Agent（3个进程并行）
make dev
```

或者分别启动（便于查看各服务日志）：

```bash
# 终端1: 前端 (http://localhost:5173)
make dev-frontend

# 终端2: Gateway (http://localhost:8080)
make dev-server

# 终端3: Code-Agent (http://localhost:8081)
make dev-agent
```

本地开发时前端运行在 5173 端口，Vite 自动代理 `/api` 请求到 Gateway 的 8080 端口。

## 环境变量说明

### 基础配置

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `DATABASE_URL` | `root:agenthub123@tcp(localhost:3306)/agenthub?...` | MySQL 连接字符串 |
| `GATEWAY_PORT` | `8080` | Gateway 服务端口 |
| `CODE_AGENT_PORT` | `8081` | Code-Agent 服务端口 |
| `AGENT_CODE_URL` | `http://localhost:8081` | Gateway 连接 Agent 的地址 |

### LLM Provider 配置

通过 `LLM_PROVIDER` 环境变量选择使用的模型服务：

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `LLM_PROVIDER` | `anthropic` | 选择 Provider：`anthropic` 或 `openai` |

**使用 Anthropic（默认）：**

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `ANTHROPIC_API_KEY` | （必填） | Anthropic API 密钥 |
| `ANTHROPIC_MODEL` | `claude-sonnet-4-20250514` | Claude 模型名称 |
| `ANTHROPIC_BASE_URL` | `https://api.anthropic.com` | API 地址 |

**使用 OpenAI 或兼容服务：**

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `OPENAI_API_KEY` | （必填） | OpenAI API 密钥 |
| `OPENAI_MODEL` | `gpt-4o` | 模型名称 |
| `OPENAI_BASE_URL` | `https://api.openai.com/v1` | API 地址 |

**OpenAI 兼容服务示例：**

```bash
# DeepSeek
LLM_PROVIDER=openai
OPENAI_API_KEY=your-deepseek-key
OPENAI_MODEL=deepseek-chat
OPENAI_BASE_URL=https://api.deepseek.com/v1

# Ollama (本地模型)
LLM_PROVIDER=openai
OPENAI_API_KEY=ollama
OPENAI_MODEL=llama3
OPENAI_BASE_URL=http://localhost:11434/v1

# vLLM / LocalAI
LLM_PROVIDER=openai
OPENAI_API_KEY=not-needed
OPENAI_MODEL=your-model-name
OPENAI_BASE_URL=http://localhost:8000/v1
```

## 项目结构

```
├── frontend/                  # React 前端
│   ├── src/
│   │   ├── components/        # UI 组件
│   │   ├── agui/              # AG-UI SSE 客户端
│   │   ├── stores/            # Zustand 状态管理
│   │   └── services/          # REST API 调用
│   └── package.json
│
├── server/                    # Go Gateway + Orchestrator
│   ├── cmd/server/main.go     # 入口
│   ├── internal/
│   │   ├── handler/           # HTTP handlers (REST + SSE)
│   │   ├── orchestrator/      # 编排 + A2A→AG-UI 协议转换
│   │   ├── a2a/               # A2A 客户端 (基于 a2a-go/v2)
│   │   ├── store/             # MySQL 数据操作
│   │   └── model/             # 数据模型
│   └── go.mod
│
├── agents/                    # Agent 子服务
│   ├── adk/                   # ADK Runtime (基于 a2a-go/v2)
│   │   ├── context.go         # ctx.StreamText / ctx.AddArtifact
│   │   ├── server.go          # A2A Server (a2asrv)
│   │   └── llm.go             # Anthropic LLM 客户端
│   ├── code-agent/            # MVP 唯一 Agent
│   │   ├── main.go
│   │   ├── handler.go         # 核心逻辑: LLM流式 + 代码解析
│   │   └── config.yaml        # AgentCard 配置
│   └── go.mod
│
├── docs/                      # 架构文档 + 契约
│   ├── contracts/             # 协议契约定义
│   └── skill/                 # Claude Code 开发契约
│
├── .claude/skills/            # Claude Code Skills (开发时约束)
├── docker-compose.yml         # 一键部署
├── init.sql                   # 数据库初始化
├── Makefile                   # 常用命令
└── .env.example               # 环境变量模板
```

## 技术栈

| 层 | 技术 |
|----|------|
| 前端 | React 18 + TypeScript + Tailwind CSS + Vite |
| 状态管理 | Zustand |
| 代码高亮 | highlight.js |
| Gateway | Go + Gin |
| 协议库 | [a2a-go/v2](https://github.com/a2aproject/a2a-go) (A2A 官方协议) |
| LLM | Anthropic Claude API (模型可配置) |
| 数据库 | MySQL 8.0 |
| 部署 | Docker Compose |

## 开发命令

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

## 验收 Checklist

- [ ] 打开网页能看到对话列表
- [ ] 能新建对话（选择 code-agent）
- [ ] 发送消息后 Agent 流式回复（逐字出现）
- [ ] 代码块以预览卡片形式展示（语法高亮）
- [ ] 能复制代码
- [ ] 能多轮对话（上下文连续）
- [ ] 刷新页面后消息不丢失（MySQL 持久化）
- [ ] Docker Compose 一键启动所有服务

## API 测试用例（curl）

三个核心协议端点，服务启动后可直接用 curl 验证全链路。

### 1. AgentCard — Agent 自描述发现

```bash
curl -s http://localhost:8081/.well-known/agent.json | jq .
```

预期返回：

```json
{
  "name": "code-agent",
  "description": "Generates, refactors, and reviews code.",
  "version": "0.1.0",
  "capabilities": { "streaming": true },
  "defaultInputModes": ["text/plain"],
  "defaultOutputModes": ["text/plain"],
  "skills": [
    {
      "id": "code_generation",
      "name": "Code Generation",
      "description": "Generate, refactor, and review code"
    }
  ]
}
```

### 2. A2A 协议 — 主调子（Gateway → Code-Agent）

Gateway（8080）内部也是走这个协议调用 Agent（8081）。直接 curl 子 Agent 验证 A2A JSON-RPC + SSE 流式返回：

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

预期 SSE 事件流（依次出现）：

```
data: {"jsonrpc":"2.0","id":"test-001","result":{"id":"<task-id>","status":{"state":"submitted"},...}}

data: {"jsonrpc":"2.0","id":"test-001","result":{"id":"<task-id>","status":{"state":"working"}}}

data: {"jsonrpc":"2.0","id":"test-001","result":{"id":"<task-id>","artifact":{...,"parts":[{"type":"text","text":"package "}]}}}

...（逐 token 流式文本）...

data: {"jsonrpc":"2.0","id":"test-001","result":{"id":"<task-id>","artifact":{...,"metadata":{"type":"code","language":"go"},...}}}

data: {"jsonrpc":"2.0","id":"test-001","result":{"id":"<task-id>","status":{"state":"completed"}}}
```

### 3. AG-UI 协议 — 前端流式展示（Gateway → 浏览器）

这是前端实际消费的 SSE 端点，Gateway 内部将 A2A 事件转换为此格式：

```bash
# 注意：如果未设 AGENTHUB_API_TOKEN，本地开发默认免鉴权
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

预期 SSE 事件流（依次出现）：

```
data: {"type":"RUN_STARTED","runId":"test-run-001"}

data: {"type":"TEXT_MESSAGE_START","messageId":"<msg-id>"}

data: {"type":"TEXT_MESSAGE_CONTENT","messageId":"<msg-id>","content":"package "}

...（逐 token 流式文本）...

data: {"type":"TEXT_MESSAGE_END","messageId":"<msg-id>"}

data: {"type":"TOOL_CALL_START","toolCallId":"<tc-id>","toolName":"code_preview"}

data: {"type":"TOOL_CALL_ARGS","toolCallId":"<tc-id>","content":"{\"code\":\"...\",\"language\":\"go\",\"filename\":\"main.go\"}"}

data: {"type":"TOOL_CALL_END","toolCallId":"<tc-id>"}

data: {"type":"RUN_FINISHED"}
```

> **三个端点的关系**：用户浏览器 →（AG-UI, :8080）→ Gateway 协议转换 →（A2A, :8081）→ Code-Agent → LLM。Agent 启动时通过 AgentCard（:8081）声明能力，Gateway 通过 AgentCard 发现可用的 Agent。

## 常见问题

**Q: Anthropic API Key 从哪里获取？**
A: 访问 https://console.anthropic.com/ 创建 API Key。

**Q: 可以使用其他模型吗？**
A: 可以。支持两种 Provider：
- **Anthropic**：通过 `ANTHROPIC_MODEL` 切换模型（`claude-opus-4-20250514`、`claude-haiku-4-20250514` 等）
- **OpenAI / 兼容服务**：设置 `LLM_PROVIDER=openai`，支持 GPT-4o、DeepSeek、Ollama 本地模型等任何 OpenAI API 兼容服务

**Q: 本地开发时前端如何连接 Gateway？**
A: Vite 配置了 `/api` 代理到 `http://localhost:8080`，无需额外配置 CORS。

**Q: Docker 启动后 MySQL 初始化失败？**
A: 执行 `make docker-down` 清理 volume 后重新 `make docker-up`。

**Q: 如何查看 Agent 的 AgentCard？**
A: 访问 http://localhost:8081/.well-known/agent.json
