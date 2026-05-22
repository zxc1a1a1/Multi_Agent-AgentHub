# Docker Compose Delivery Contract

版本：v0.1-p1  
适用项目：AgentHub - 多 Agent 协作平台  
适用阶段：MVP 本地交付 + P1 正式开发演进  
事实源文件：

```text
<repo-root>/docs/contracts/docker-compose-delivery.md
<repo-root>/docs/contracts/docker-compose-delivery.schema.json
```

## 1. 目的

本文定义 AgentHub 的项目级 Docker Compose 本地交付与 Demo 环境契约。

定位：

```text
AgentHub 本地 Demo、开发复现、交付验收的一键启动契约
```

## 2. 官方标准优先

涉及 Compose 文件格式、CLI 行为、service、network、volume、healthcheck、profiles、secrets 时，以 Docker Compose 官方文档和 Compose Specification 为准。

优先级：

```text
Docker Compose 官方规范
> AgentHub docker-compose-delivery contract
> 项目中随手写的 compose 实现
```

## 3. Contract first 规则

任何新增或修改 Compose 服务、端口、环境变量、healthcheck、Makefile 命令、smoke test、profiles、volume、network 或 secrets 前，必须先更新：

```text
<repo-root>/docs/contracts/docker-compose-delivery.md
<repo-root>/docs/contracts/docker-compose-delivery.schema.json
```

未更新 contract 的 Compose 交付变更不得接受。

## 4. 阶段演进规则

### MVP 阶段

MVP 阶段只要求：

```text
frontend + gateway + code-agent + mysql
docker compose up 一键启动
.env.example
Makefile
MySQL healthcheck
Gateway /health
code-agent /health
smoke-test
```

MVP 不要求：

- Redis。
- PostgreSQL。
- Object Storage。
- Jaeger。
- Prometheus。
- Grafana。
- Nginx。
- TLS。
- Kubernetes。
- Helm。
- Terraform。

### P1 / 正式开发阶段

逐步启用：

- `compose.yaml`。
- `compose.override.yaml`。
- profiles: `dev` / `demo` / `observability` / `mock-llm`。
- Docker secrets。
- gateway healthcheck。
- code-agent healthcheck。
- mysql healthcheck。
- `scripts/smoke-test.sh`。
- `scripts/wait-for-health.sh`。
- `docker compose config` 校验。
- volume 清理策略。

## 5. MVP 服务

MVP 必须包含：

```text
frontend
gateway
code-agent
mysql
```

MVP 端口：

```text
frontend: 3000
gateway: 8080
code-agent: 8081
mysql: 3306
```

## 6. MVP 环境变量

必需：

```text
DATABASE_URL
ANTHROPIC_API_KEY
GATEWAY_PORT
CODE_AGENT_PORT
AGENT_CODE_URL
```

必须提交：

```text
.env.example
```

不得提交：

```text
.env
```

## 7. Makefile

必须提供：

```text
make docker-up
make docker-down
make docker-reset
make logs
make smoke
make dev
```

## 8. Smoke Test

必须验证：

```text
docker compose config
services start
mysql healthy
gateway /health
code-agent /health
frontend reachable
minimal API path
no obvious 500
```

CI / smoke 默认不得依赖真实 LLM 随机输出。

## 9. 禁止事项

不得：

- 提交真实 `.env`。
- 提交真实 API key。
- 把 secret 写进 Dockerfile `ENV`。
- 把 secret bake 进 image。
- 让 `docker compose up` 后还需要手动启动某个服务。
- 只靠 `sleep` 等 MySQL。
- 没有 healthcheck。
- 没有 smoke test。
- 没有清理命令。
- smoke test 依赖真实 LLM 随机输出。
- 把生产部署要求混进 MVP Compose。
- 把 `docker-compose.yml` 写死成唯一正式部署方案。
- 在 Compose 文件里硬编码本机绝对路径。
