# AgentHub Docker 一键启动

## 最简（Anthropic）

```bash
# Linux / macOS
LLM_API_KEY=sk-ant-xxx docker compose -f docker-compose.new-arch.yml up --build

# Windows PowerShell
$env:LLM_API_KEY="sk-ant-xxx"; docker compose -f docker-compose.new-arch.yml up --build

# Windows CMD
set LLM_API_KEY=sk-ant-xxx && docker compose -f docker-compose.new-arch.yml up --build
```

## 命令行注入所有格式

### Anthropic

```bash
LLM_API_KEY=sk-ant-xxx docker compose -f docker-compose.new-arch.yml up --build
```

### OpenAI

```bash
LLM_API_KEY=sk-xxx LLM_PROVIDER=openai LLM_MODEL=gpt-4o docker compose -f docker-compose.new-arch.yml up --build
```

### DeepSeek

```bash
LLM_API_KEY=sk-xxx LLM_PROVIDER=openai LLM_MODEL=deepseek-chat LLM_BASE_URL=https://api.deepseek.com/v1 docker compose -f docker-compose.new-arch.yml up --build
```

### 通义千问（阿里云）

```bash
LLM_API_KEY=sk-xxx LLM_PROVIDER=openai LLM_MODEL=qwen-plus LLM_BASE_URL=https://dashscope.aliyuncs.com/compatible-mode/v1 docker compose -f docker-compose.new-arch.yml up --build
```

### 火山引擎（字节跳动）

```bash
LLM_API_KEY=xxx LLM_PROVIDER=openai LLM_MODEL=ep-xxx LLM_BASE_URL=https://ark.cn-beijing.volces.com/api/v3 docker compose -f docker-compose.new-arch.yml up --build
```

可选模型：`deepseek-v3-0324`、`deepseek-r1-0528`、`doubao-1.5-pro-256k`、`doubao-1.5-lite-32k`

### Ollama 本地模型

```bash
LLM_API_KEY=ollama LLM_PROVIDER=openai LLM_MODEL=llama3 LLM_BASE_URL=http://localhost:11434/v1 docker compose -f docker-compose.new-arch.yml up --build
```

### vLLM / LocalAI

```bash
LLM_API_KEY=not-needed LLM_PROVIDER=openai LLM_MODEL=your-model LLM_BASE_URL=http://localhost:8000/v1 docker compose -f docker-compose.new-arch.yml up --build
```

## 环境变量说明

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `LLM_API_KEY` | — | **必填**，三个服务共用 |
| `LLM_PROVIDER` | `anthropic` | `anthropic` \| `openai` |
| `LLM_MODEL` | 不设用内置默认 | 模型名，不设则 agent 跑 mock 模式 |
| `LLM_BASE_URL` | — | 自定义 endpoint，OpenAI 兼容 API 必填 |

Per-service 覆盖（按需）：

| 变量 | 覆盖范围 |
|------|---------|
| `ORCHESTRATOR_LLM_MODEL` | 只改编排器 |
| `CODE_AGENT_LLM_MODEL` | 只改代码 agent |
| `WEB_AGENT_LLM_MODEL` | 只改网页 agent |

## Windows PowerShell 多行

```powershell
$env:LLM_API_KEY="sk-xxx"
$env:LLM_PROVIDER="openai"
$env:LLM_MODEL="gpt-4o"
$env:LLM_BASE_URL="https://api.deepseek.com/v1"
docker compose -f docker-compose.new-arch.yml up --build
```

## 后台运行 + 查看日志

```bash
LLM_API_KEY=sk-ant-xxx docker compose -f docker-compose.new-arch.yml up --build -d
docker compose -f docker-compose.new-arch.yml logs -f
```

## 停止

```bash
docker compose -f docker-compose.new-arch.yml down
```

## 端口

| 服务 | 端口 |
|------|------|
| Gateway | `8080` |
| Orchestrator | `8090` |
| code-agent | `8081` |
| web-agent | `8082` |
| Frontend | `3000` |
