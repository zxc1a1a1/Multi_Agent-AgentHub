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

## v3 新增变量（占位；权威列表见 `.env.example`）

```env
# 群聊
GROUP_CHAT_MAX_ROUNDS=1
GROUP_CHAT_SPEAKER_MODE=round_robin

# Artifact / 渲染
ARTIFACT_STORE_DIR=./data/artifacts
ARTIFACT_BLOB_DIR=./data/artifacts/blobs
ARTIFACT_METADATA_BACKEND=local
ARTIFACT_S3_ENDPOINT=
ARTIFACT_S3_BUCKET=
ARTIFACT_S3_ACCESS_KEY=
ARTIFACT_S3_SECRET_KEY=
PLAYWRIGHT_RENDER_SCRIPT=
PLAYWRIGHT_NODE_PATH=node
PLAYWRIGHT_RENDER_TIMEOUT_SECONDS=60
WEB_AGENT_ENABLE_CAPTURE=false
ORCHESTRATOR_RENDER_URL=

# 受控 handoff
CODE_AGENT_ENABLE_HANDOFF=false
ORCHESTRATOR_HANDOFF_URL=
CODE_AGENT_HANDOFF_ALLOWLIST=
CODE_AGENT_HANDOFF_MAX_DEPTH=1

# OTel
OTEL_EXPORTER_OTLP_ENDPOINT=
OTEL_TRACES_SAMPLE_RATIO=1

# 工具授权（与 handoff/render 共用服务 token）
ORCHESTRATOR_TOOL_AUTHORIZE_URL=
ORCHESTRATOR_TOOL_RESULT_URL=
ORCHESTRATOR_TOOL_TOKEN=
```

## 规则

- 所有新增变量必须说明用途。
- secret 只能使用占位值。
- 真实 secret 不得进入仓库、镜像或日志。
