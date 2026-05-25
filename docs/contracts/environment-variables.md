# Environment Variables Contract

## 必须维护

```text
.env.example
```

## 禁止提交

```text
.env
.env.local
.env.production
secrets/*.txt
```

## 推荐变量

```env
DB_HOST=mysql
DB_PORT=3306
DB_USER=root
DB_PASSWORD=change-me
DB_NAME=agenthub

GATEWAY_PORT=8080
AGENTHUB_API_TOKEN=change-me

LLM_PROVIDER=anthropic
ANTHROPIC_API_KEY=your-api-key-here
ANTHROPIC_MODEL=your-model-here

AGENT_URLS=code-agent=http://code-agent:8081,web-agent=http://web-agent:8082
VITE_API_URL=http://localhost:8080
```

## 规则

- 所有新增变量必须说明用途。
- secret 只能使用占位值。
- 真实 secret 不得进入仓库、镜像或日志。
