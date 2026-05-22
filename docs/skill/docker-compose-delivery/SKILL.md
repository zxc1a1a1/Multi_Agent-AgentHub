---
name: docker-compose-delivery
description: 当定义、实现、修改或审查 AgentHub 的 Compose 本地交付、Demo 环境、.env.example、Makefile、healthcheck、smoke test、profiles、volumes 或 secrets 时，使用本 Skill。
---

# docker-compose-delivery

## 1. 目的

本 Skill 定义 AgentHub 的 Docker Compose 本地交付与 Demo 环境契约。

本 Skill 的定位是：

```text
AgentHub 本地 Demo、开发复现、交付验收的一键启动契约
```

目标：

- 让开发者、评审、演示人员可以用标准命令稳定启动 AgentHub。
- 确保 frontend、gateway、code-agent、mysql 的本地链路可复现。
- 确保 `.env.example`、Compose 文件、Makefile、smoke test 一致。
- 确保服务健康检查和启动顺序可靠。
- 确保 Demo 前可以一键验证核心链路。
- 避免真实 secret 被提交、写入 image 或暴露到日志。
- 避免把本地 Compose 误写成生产部署方案。

## 2. 官方标准优先级

涉及 Compose 文件格式、CLI 行为、service、network、volume、healthcheck、profiles、secrets 时，以 Docker Compose 官方文档和 Compose Specification 为准。

优先级：

```text
Docker Compose 官方规范
> AgentHub docker-compose-delivery contract
> 项目中随手写的 compose 实现
```

规则：

- 优先使用 `docker compose`，不要新增依赖旧命令 `docker-compose`。
- Compose 文件应遵守 Compose Specification。
- `depends_on` 只能表达启动顺序，不等于服务 ready。
- 服务 ready 必须通过 `healthcheck` 或 wait 脚本验证。
- `.env` 可用于 Compose variable interpolation，但真实 secret 不得提交。
- 正式开发阶段敏感信息应优先使用 Docker secrets 或等价安全机制。

## 3. 文件位置说明

### 项目级 Contract 文件

以下文件属于 AgentHub 项目仓库，是项目级事实源：

```text
<repo-root>/docs/contracts/docker-compose-delivery.md
<repo-root>/docs/contracts/docker-compose-delivery.schema.json
```

### 当前 Skill 的参考文件

以下文件属于当前 Coding Agent Skill：

```text
<current-skill-dir>/references/compose-file-policy.md
<current-skill-dir>/references/environment-policy.md
<current-skill-dir>/references/healthcheck-policy.md
<current-skill-dir>/references/service-topology.md
<current-skill-dir>/references/makefile-commands.md
<current-skill-dir>/references/smoke-test-policy.md
<current-skill-dir>/references/volume-network-policy.md
<current-skill-dir>/references/demo-checklist.md
```

如果当前 Skill 安装在 Claude Code 项目目录中，则 `<current-skill-dir>` 通常是：

```text
<repo-root>/.claude/skills/docker-compose-delivery
```

## 4. 适用场景

当进行以下工作时，启用本 Skill：

- 新增或修改 `compose.yaml` / `docker-compose.yml`。
- 新增或修改 `.env.example`。
- 新增或修改 `Makefile`。
- 新增或修改 `scripts/smoke-test.sh`。
- 新增或修改 `scripts/wait-for-health.sh`。
- 新增或修改 service healthcheck。
- 新增或修改 service ports。
- 新增或修改 Docker volume / network。
- 新增或修改 Compose profiles。
- 新增或修改 Docker secrets。
- 新增或修改本地 Demo checklist。
- 审查 Compose 是否泄漏 secret。
- 审查 `docker compose up` 是否能一键启动 MVP Demo。

## 5. 长期契约基线

长期架构中，AgentHub 必须提供可复现的本地交付环境。

长期 contract 必须定义：

- Compose 文件命名。
- service 命名。
- service topology。
- build context。
- ports。
- environment variables。
- `.env.example`。
- healthcheck。
- `depends_on` + `service_healthy`。
- network。
- volume。
- profiles。
- secrets。
- Makefile 命令。
- smoke test。
- logs 命令。
- reset / clean 命令。
- Demo checklist。

推荐交付文件：

```text
compose.yaml
compose.override.yaml
.env.example
Makefile
scripts/smoke-test.sh
scripts/wait-for-health.sh
```

MVP 兼容文件：

```text
docker-compose.yml
```

正式开发阶段推荐使用：

```text
compose.yaml
```

但可以保留 `docker-compose.yml` 作为兼容入口。

## 6. MVP 约束

MVP 阶段只要求以下服务：

```text
frontend
gateway
code-agent
mysql
```

MVP 服务端口：

```text
frontend: 3000
gateway: 8080
code-agent: 8081
mysql: 3306
```

MVP 必需环境变量：

```text
DATABASE_URL
ANTHROPIC_API_KEY
GATEWAY_PORT
CODE_AGENT_PORT
AGENT_CODE_URL
```

MVP 最小命令：

```text
make docker-up
make docker-down
make docker-reset
make logs
make smoke
make dev
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
- Production reverse proxy。
- Kubernetes。
- Helm。
- Terraform。
- 多 profile 部署。

## 7. 阶段演进规则

### 7.1 MVP 阶段

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

MVP 阶段不得：

- 提交真实 API key。
- 把 API key bake 进 image。
- 把 secret 写进 Dockerfile ENV。
- 只靠 sleep 等 MySQL。
- 没有 healthcheck。
- 没有清理命令。
- smoke test 依赖真实 LLM 随机输出。
- 把生产部署要求混进 MVP Compose。

### 7.2 P1 / 正式开发阶段

P1 阶段逐步启用：

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
- logs / reset / seed 命令。

### 7.3 后续扩展阶段

后续可以增加：

- `mock-llm` profile。
- `observability` profile。
- `object-storage` profile。
- `redis` profile。
- `multi-agent` profile。
- `demo-seed` profile。

新增 profile 前必须更新 contract 和 README。

## 8. 本 Skill 负责

本 Skill 负责：

- Compose 文件策略。
- `.env.example` 规则。
- 本地服务拓扑。
- 服务端口约定。
- service healthcheck。
- `depends_on` 与 ready 判断。
- Makefile 命令。
- smoke test。
- volume / network 策略。
- Docker secrets 规则。
- profiles 规则。
- Demo checklist。
- 本地日志和清理命令。

## 9. 本 Skill 不负责

本 Skill 不负责：

- Kubernetes。
- Helm。
- Terraform。
- 云部署。
- 生产高可用。
- CI/CD 全流程。
- 灰度发布。
- 监控告警平台。
- 安全策略完整定义。
- 数据库 schema 设计。
- 应用业务逻辑。
- Dockerfile 具体优化细节。
- Go / TypeScript 代码风格。

涉及以上内容时，只引用相关 contract，不在本 Skill 中重新定义。

## 10. Contract first 规则

任何新增或修改 Compose 服务、端口、环境变量、healthcheck、Makefile 命令、smoke test、profiles、volume、network 或 secrets 前，必须先更新：

```text
<repo-root>/docs/contracts/docker-compose-delivery.md
<repo-root>/docs/contracts/docker-compose-delivery.schema.json
```

未更新 contract 的 Compose 交付变更不得接受。

## 11. 核心规则

### Compose 文件

MVP 可使用：

```text
docker-compose.yml
```

正式开发推荐：

```text
compose.yaml
```

Compose services 必须至少包含：

```text
frontend
gateway
code-agent
mysql
```

### Environment

必须提交：

```text
.env.example
```

不得提交：

```text
.env
```

`.env.example` 只能包含占位值，不得包含真实 secret。

### Healthcheck

MVP 至少要求：

```text
mysql healthcheck
gateway /health
code-agent /health
```

`gateway` 不得在 MySQL 未 ready 时错误启动后静默失败。

### Makefile

必须提供稳定命令：

```text
make docker-up
make docker-down
make docker-reset
make logs
make smoke
make dev
```

### Smoke Test

smoke test 必须验证：

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

## 12. 禁止事项

Coding Agent 不得：

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
- 在公开日志里输出 secret。
- 在 CI 中使用真实生产 API key。

## 13. 相关契约

本 Skill 只引用以下契约，不重新定义它们：

- `testing-review-contract`：负责 smoke test、CI quality gates 和 Demo 测试策略。
- `security-boundary-contract`：负责 secret、权限、沙箱和敏感信息保护。
- `data-persistence-contract`：负责 MySQL、Redis、Object Storage 和数据迁移策略。
- `adk-runtime-contract`：负责 code-agent /health、A2A Server、AgentCard。
- `llm-provider-contract`：负责 LLM Provider secret、mock provider 和 fallback。
- `observability-debugging-contract`：负责 logs、traceId、metrics 和排障。
- `platform-api-contract`：负责 Gateway API 和 `/health`。
