# 环境变量策略

## 必须提交

```text
.env.example
```

## 不得提交

```text
.env
.env.local
.env.production
secrets/*.txt
```

## .env.example 必须包含

- 数据库连接配置。
- Gateway 配置。
- API token 占位符。
- LLM Provider 占位配置。
- Agent URL 或 registry 配置（Orchestrator / Registry 使用，非 Gateway 配置）。
- 前端 API URL。
- 每个变量的用途说明。

## 规则

- `.env.example` 只能使用占位值。
- 新增环境变量必须同步更新 `.env.example`。
- 真实 secret 只能来自本机 `.env`、shell 环境或 secrets。
- 不得在日志或 smoke test 中输出 secret。
- 不得在 Dockerfile 中写真实 secret。

## 示例

```env
DB_HOST=mysql
DB_PORT=3306
DB_USER=root
DB_PASSWORD=change-me
DB_NAME=agenthub

GATEWAY_PORT=8080
AGENTHUB_API_TOKEN=change-me

# Orchestrator / Registry 使用：
AGENT_URLS=code-agent=http://code-agent:8081,web-agent=http://web-agent:8082

LLM_PROVIDER=anthropic
ANTHROPIC_API_KEY=your-api-key-here
```
