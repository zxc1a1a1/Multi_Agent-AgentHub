---
name: docker-compose-delivery
description: "用于定义 AgentHub 本地开发、Demo 演示和验收测试的一键启动交付契约，包括 Compose 文件、服务拓扑、多 Child Agent 服务、环境变量、secrets、healthcheck、profiles、Makefile、smoke test、volume/network、日志、清理和 Demo checklist。"
---

# docker-compose-delivery

## 1. Skill 目的

本 Skill 用于规范 AgentHub 项目的本地交付、Demo 演示和验收测试环境。

它回答以下问题：

- `compose.yaml` / `docker-compose.yml` 应如何组织？
- 本地 Demo 至少应启动哪些服务？
- 新增 Child Agent 服务时 Compose 应如何扩展？
- `.env.example` 应如何维护？
- secret 不得如何泄漏？
- healthcheck 应如何编写？
- `depends_on` 与 readiness 应如何处理？
- Makefile 应暴露哪些稳定命令？
- `smoke-test.sh` 应验证哪些路径？
- volume / network / logs / reset / clean 应如何约束？
- Demo 交付前必须检查什么？

一句话：

**Docker Compose 是 AgentHub 本地开发、演示和验收的可重复交付入口；任何服务只要进入 Demo 链路，就必须能被 Compose 启动、健康检查、日志排障和 smoke test 验证。**

---

## 2. 独立性原则

本 Skill 是独立交付契约。

本 Skill 不要求读者先阅读其他 Skill 才能理解。

本 Skill 只定义本地 Compose 交付规则，不展开：

- 应用业务逻辑。
- 子 Agent 内部实现。
- 前后端协议字段。
- 数据库表结构细节。
- 前端组件实现。
- LLM Provider 调用细节。
- 生产部署平台。
- Kubernetes / Helm / Terraform。
- CI/CD 全流程。

如果某个服务需要进入 Compose，本 Skill 只关心：

- 它叫什么。
- 如何构建或拉取镜像。
- 如何配置环境变量。
- 如何在 Compose 网络中被访问。
- 是否有 healthcheck。
- 是否能被 smoke test 验证。
- 是否泄漏 secret。

---

## 3. 当前阶段识别

当前开发阶段不应被永久写死在本 Skill 中。

MVP v0.1 已完成，仅作为历史兼容和回归测试基线。

当前 v1.0 交付目标是：

- 支持 2+ Child Agent 的本地启动模式。
- 支持 `frontend`、`gateway`、`mysql` 和多个 Child Agent 服务。
- 支持 Agent 服务通过 Compose service name 被 Gateway 访问。
- 支持 healthcheck 全覆盖。
- 支持 `.env.example` 完整说明。
- 支持 `docker compose up --build` 一键启动。
- 支持 Makefile 封装常用命令。
- 支持 `smoke-test.sh` 验收核心服务。
- 支持 Demo 场景稳定运行。

本 Skill 不固定具体 Agent 名称。

示例可以出现 `code-agent`、`web-agent`、`doc-agent`、`search-agent` 等，但示例不是唯一允许列表。

---

## 4. 本 Skill 负责什么

本 Skill 负责：

- Compose 文件策略。
- 本地服务拓扑。
- Child Agent 服务模式。
- 环境变量文件规则。
- secrets 使用规则。
- healthcheck 规则。
- `depends_on` 与 readiness 规则。
- frontend 生产镜像规则。
- profiles 规则。
- Makefile 命令规则。
- smoke test 规则。
- volume / network 规则。
- logs / reset / clean 规则。
- Demo checklist。
- 交付 Review checklist。

---

## 5. 本 Skill 不负责什么

本 Skill 不负责：

- 生产部署架构。
- 云服务资源创建。
- Kubernetes、Helm、Terraform。
- 业务 API 设计。
- Agent 内部协议。
- 数据库 schema 设计。
- 前端组件实现。
- 认证授权策略细节。
- LLM Provider 接入细节。
- 观测系统完整设计。

本地 Compose 可以模拟或承载这些能力，但不代表生产部署承诺。

---

## 6. 官方规范优先级

当本 Skill 与 Docker 官方行为冲突时，以 Docker Compose Specification 和 Docker 官方文档为准。

本 Skill 默认使用：

```text
docker compose
```

而不是旧版：

```text
docker-compose
```

但仓库可以为了兼容保留 `docker-compose.yml`。

必须避免：

- 依赖已废弃字段。
- 写只在某个人机器上可用的配置。
- 把生产集群要求混进本地 Demo Compose。
- 让本地 Compose 成为不可维护的多环境大杂烩。

---

## 7. Compose 文件策略

推荐事实源：

```text
compose.yaml
```

兼容文件：

```text
docker-compose.yml
```

可选覆盖：

```text
compose.override.yaml
compose.dev.yaml
compose.demo.yaml
compose.mock.yaml
```

规则：

- 不应同时维护两个内容分叉的主 Compose 文件。
- 如果保留 `docker-compose.yml`，它应与 `compose.yaml` 等价，或仅作为兼容入口。
- 本地开发覆盖项应放入 override 文件。
- Demo 默认路径应简单明确。
- 不应把生产部署配置写入本地 Compose 主文件。
- `docker compose config` 必须能通过。
- 新增服务时必须同步更新拓扑说明、环境变量、healthcheck 和 smoke test。

---

## 8. Service Topology

基础服务：

```text
frontend
gateway
mysql
child-agent services, one or more
```

可选服务：

```text
mock-llm
mock-agent
redis
object-storage
observability
reverse-proxy
```

规则：

- 默认 Profile 必须足以运行主 Demo。
- 可选服务不得破坏默认 `docker compose up`。
- 服务之间必须通过 Compose service name 通信。
- 容器内不得使用 `localhost` 访问其他容器。
- 宿主机访问服务时才使用 `localhost:port`。
- 新增 Child Agent 时不得要求修改 Gateway 源码才能被发现。

容器内访问示例：

```text
http://gateway:8080
http://mysql:3306
http://some-agent:8083
```

宿主机访问示例：

```text
http://localhost:3000
http://localhost:8080
```

---

## 9. Child Agent Service Pattern

本 Skill 不固定具体 Agent 名称。

任何 Child Agent 服务进入 Compose 时，必须满足：

- 有稳定 service name。
- 有独立 `build` 或 `image`。
- 有明确内部端口。
- 提供 `/health`。
- 可通过 Compose service name 被 Gateway 访问。
- 不把 secret 写入 image。
- 可以被 smoke test 单独检查。
- 日志能标识 agent name。
- 新增或删除该服务不应破坏其他服务启动。

示例：

```yaml
services:
  some-agent:
    build:
      context: ./agents
      dockerfile: some-agent/Dockerfile
    expose:
      - "8083"
    environment:
      - AGENT_PORT=8083
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8083/health"]
      interval: 10s
      timeout: 5s
      retries: 10
```

建议：

- 默认只使用 `expose` 暴露给 Compose 网络。
- 需要宿主机调试时才使用 `ports`。
- Agent 服务端口不得硬编码到 Gateway 源码。
- Agent URL 应通过环境变量或配置文件提供给 Gateway。

---

## 10. Gateway 与 Agent 发现配置

Gateway 应通过配置发现多个 Agent。

v1.0 兼容方式可以保留：

```text
AGENT_CODE_URL=http://code-agent:8081
AGENT_WEB_URL=http://web-agent:8082
```

推荐通用方式：

```text
AGENT_URLS=code-agent=http://code-agent:8081,web-agent=http://web-agent:8082,doc-agent=http://doc-agent:8083
```

或：

```text
AGENT_REGISTRY_FILE=/app/config/agents.yaml
```

规则：

- 新增 Agent 服务时，不应要求修改 Gateway 源码。
- 新增 Agent 服务时，必须同步更新 `.env.example`。
- 新增 Agent 服务时，必须同步更新 smoke test。
- 可选 Agent 未启动时，不应导致整个基础栈无法启动，除非它属于当前 Demo 必需服务。
- Gateway 日志应能显示已加载的 Agent 配置，但不得输出 secret。

---

## 11. Environment Policy

必须提交：

```text
.env.example
```

不得提交：

```text
.env
.env.local
.env.production
secrets/*.txt
任何含真实 token 的 override 文件
```

`.env.example` 必须说明：

- 数据库连接变量。
- Gateway 端口。
- API token 占位值。
- LLM Provider 占位变量。
- Agent URL / Agent registry 配置。
- 前端 API base URL。
- 是否必须设置。
- 默认值是否仅用于本地 Demo。
- 变量用途。

示例：

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

规则：

- `.env.example` 只能使用占位值。
- 真实 key 只能放在本机 `.env`、shell 环境或 Compose secrets 中。
- 不得在 Dockerfile 中 `ENV` 真实 secret。
- 不得在 README、日志、smoke test 输出中暴露真实 secret。
- 新增环境变量必须同步更新 `.env.example`。

---

## 12. Secrets Policy

本地 Demo 可以使用 `.env` 注入 secret，但不得提交真实 secret。

更安全的方式是使用 Compose secrets 或外部 secret manager。

规则：

- API key 不得写入镜像。
- API key 不得写入仓库。
- API key 不得写入 smoke test 输出。
- API key 不得进入 container logs。
- secrets 文件不得被 git 跟踪。
- 生产部署不得依赖 `.env.example`。
- 如使用 Compose secrets，服务只应挂载自己需要的 secret。

可接受的本地做法：

```text
.env             # 本机私有，gitignore
secrets/*.txt    # 本机私有，gitignore
```

禁止：

```dockerfile
ENV ANTHROPIC_API_KEY=sk-...
```

---

## 13. Healthcheck Policy

所有长期运行服务必须有 healthcheck：

- `mysql`
- `gateway`
- `frontend`
- 每个启用的 Child Agent 服务

healthcheck 必须：

- 快速。
- 稳定。
- 可重复。
- 不修改业务数据。
- 不调用真实 LLM。
- 不依赖公网。
- 不输出 secret。
- 能区分服务未启动和服务异常。

示例：

```yaml
healthcheck:
  test: ["CMD", "wget", "-qO-", "http://localhost:8080/health"]
  interval: 10s
  timeout: 5s
  retries: 10
```

数据库示例：

```yaml
healthcheck:
  test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
  interval: 5s
  timeout: 5s
  retries: 10
```

规则：

- Gateway 应等待数据库 healthy。
- Frontend 可以等待 Gateway started 或 healthy，取决于部署方式。
- Agent healthcheck 不应触发昂贵模型调用。
- 可选 Agent unhealthy 不应拖垮整个栈，除非当前 Demo 必需。

---

## 14. depends_on 与 readiness 规则

`depends_on` 只能解决启动顺序，不能天然保证服务 ready。

需要等待依赖 ready 时，应使用 healthcheck + `condition: service_healthy`。

示例：

```yaml
gateway:
  depends_on:
    mysql:
      condition: service_healthy
```

规则：

- 不得使用固定 `sleep 10` 替代 readiness。
- 服务启动失败必须能从日志中定位。
- Gateway 初始化应能处理 Agent 暂时不可用。
- Agent Registry 类加载不应因单个可选 Agent 不健康而 panic。
- 数据库 migration 或 init 必须明确在何处执行。

---

## 15. Frontend Image Policy

Demo profile 中的前端容器应使用生产构建镜像。

允许模式：

- Node 构建 + `serve dist`。
- Node 构建 + nginx 静态服务。
- dev profile 中可使用 Vite dev server。

规则：

- Demo 默认路径不得依赖 Vite dev server。
- 前端镜像应使用多阶段构建。
- API base URL 必须在 Compose 网络和宿主机访问两种场景下说明清楚。
- 如果 nginx 代理 `/api`，必须避免破坏流式接口。
- SSE / streaming 路径不得被默认 proxy buffering 卡住。
- 前端容器默认宿主机端口建议为 `3000`。

示例原则：

```text
dev profile: 开发热更新
demo profile: 生产构建静态服务
```

---

## 16. Profiles Policy

建议 Profile：

```text
default
dev
demo
mock
extra-agents
observability
storage
```

含义：

- `default`：主 Demo 最小必需服务。
- `dev`：热更新、bind mount、调试端口。
- `demo`：稳定演示环境。
- `mock`：mock LLM / mock Agent。
- `extra-agents`：可选更多 Child Agent。
- `observability`：日志、指标、追踪。
- `storage`：Redis、Object Storage 等可选依赖。

规则：

- 新增 profile 前必须说明用途。
- 默认 profile 不应过重。
- 可选 profile 不应影响默认启动。
- profile 名称应稳定、简短。
- 文档必须写清启动方式和清理方式。

---

## 17. Makefile Commands

推荐命令：

```makefile
docker-up
docker-down
docker-build
docker-reset
docker-logs
docker-ps
smoke-test
smoke-test-ci
demo-up
demo-reset
```

语义：

```text
make docker-up        # 启动本地栈
make docker-down      # 停止本地栈，不删除数据
make docker-build     # 构建镜像
make docker-reset     # 停止并删除 volume，危险操作
make docker-logs      # 查看日志
make docker-ps        # 查看服务状态
make smoke-test       # 本地 smoke test
make smoke-test-ci    # CI 友好 smoke test
make demo-up          # Demo profile 启动
make demo-reset       # Demo 数据清理
```

规则：

- Makefile 只是命令封装，不应隐藏破坏性行为。
- `reset` / `clean` 必须明确会删除 volume。
- 默认 `up` 不应删除数据。
- 命令名应稳定。
- 失败时应返回非零退出码。
- 不存在的命令不得在文档中声称可用。

---

## 18. Smoke Test Policy

Smoke test 应验证本地交付是否可用，不应依赖真实 LLM 随机输出。

建议分层：

### Level 0: Compose config

```text
docker compose config
```

### Level 1: Service health

- mysql healthy
- gateway `/health`
- frontend reachable
- 每个启用的 Child Agent `/health`

### Level 2: Registry / API

- Gateway 能列出启用的 Agent。
- v1.0 Demo profile 至少有 2 个 healthy Agent，除非当前 profile 明确是单 Agent 回归。

### Level 3: Minimal run

- 创建会话。
- 发送最小消息。
- 收到流式响应或确定性 mock 响应。

### Level 4: Demo path

- 单 Agent 对话。
- 群聊或多 Agent 场景。
- 产物元数据或预览入口存在。

规则：

- smoke test 不能输出 secret。
- smoke test 失败必须显示失败项。
- smoke test 应可在本机和 CI 中运行。
- 网络超时应有合理上限。
- 不得只检查容器 started，必须检查服务可用。

---

## 19. Volume / Network Policy

Volume 规则：

- MySQL 数据必须使用 named volume。
- Demo 数据可通过 reset 命令清理。
- 默认 up 不删除 volume。
- 不得使用本机绝对路径作为默认 volume。
- dev profile 可以使用 bind mount。
- secret 文件不得作为 repo-tracked volume 挂载。

Network 规则：

- 默认使用 Compose project network。
- 服务间通信使用 service name。
- 容器间不得使用 `localhost` 调其他服务。
- 只有需要宿主机访问的服务才映射 `ports`。
- Agent 服务优先 `expose` 给内部网络。
- 网络别名必须有明确理由。

---

## 20. Logs / Reset / Clean

日志规则：

- 所有服务必须输出到 stdout / stderr。
- 日志不得包含 secret。
- 日志应包含 service name 或 agent name。
- Makefile 应提供查看日志命令。
- 排障文档应说明如何看 gateway、agent、mysql、frontend 日志。

清理规则：

- `down` 不删除 volume。
- `reset` 可以删除 volume，但必须明确危险。
- 不得让普通启动命令自动清空数据库。
- Demo reset 应可重复执行。
- 清理命令失败时不得假装成功。

---

## 21. Demo Checklist

Demo 前必须检查：

- `docker compose config` 通过。
- `.env.example` 完整。
- 本机 `.env` 已配置必要占位变量。
- `docker compose up --build` 可启动。
- mysql healthy。
- gateway healthy。
- frontend 可访问。
- 至少 2 个当前启用的 Child Agent healthy。
- smoke test 通过。
- 前端使用 Demo / production-like build。
- 日志无 secret。
- 停止和重启后数据仍符合预期。
- reset 命令可清理演示数据。
- README 或交付文档说明启动步骤。

---

## 22. 禁止事项

禁止：

- 提交真实 `.env`。
- 提交真实 API key。
- Dockerfile `ENV` 写真实 secret。
- Compose 文件写死本机绝对路径。
- 容器间使用 `localhost` 调其他容器。
- 用 `sleep` 替代 readiness。
- 长期运行服务缺少 healthcheck。
- smoke test 依赖真实 LLM 随机输出。
- 默认 up 后还需要手动启动某个必需服务。
- 新增 Agent 但不加 healthcheck。
- 新增 Agent 但不更新 `.env.example`。
- 新增 Agent 但不更新 smoke test。
- 新增 Agent 但不更新 Makefile / 文档。
- 把本地 Compose 当生产部署承诺。
- 默认 profile 启动过重的可选组件。
- reset 命令无提示删除数据。
- 日志输出 token、API key、数据库密码。

---

## 23. Review Checklist

### Compose

- 是否有 `compose.yaml` 或 `docker-compose.yml`？
- 是否能通过 `docker compose config`？
- 是否没有真实 secret？
- 是否没有本机绝对路径？
- 容器间是否使用 service name？
- 需要宿主机访问的服务是否才映射 ports？

### 服务

- frontend 是否可访问？
- gateway 是否可访问？
- mysql 是否 healthy？
- enabled child agents 是否都有 `/health`？
- 新增 Agent 是否有 build/image/env/healthcheck？
- Agent 服务是否不固定为唯一允许列表？

### 依赖

- gateway 是否等待 mysql healthy？
- frontend 是否正确访问 gateway？
- gateway 是否通过配置发现 agents？
- 可选 Agent 不健康是否不会拖垮整栈？

### 环境

- `.env.example` 是否完整？
- `.env` 是否未提交？
- secret 是否未写进 Dockerfile ENV？
- LLM key 是否只使用占位？
- 新增变量是否有说明？

### Smoke

- 是否有 `smoke-test.sh`？
- 是否检查 compose config？
- 是否检查服务健康？
- 是否检查 enabled agents？
- 是否不依赖真实 LLM 随机输出？
- 失败时是否返回非零退出码？

### Demo

- `make docker-up` 是否一键启动？
- `make docker-down` 是否可停止？
- `make docker-reset` 是否明确删除 volume？
- `make docker-logs` 是否方便排障？
- README 是否说明启动和清理步骤？

---

## 24. 完成定义

本 Skill 视为完成，当且仅当：

- `SKILL.md` 明确本地 Compose 交付边界。
- 不再把 MVP 单 Agent 服务拓扑作为当前限制。
- 支持 2+ Child Agent 的通用服务模式。
- 不固定具体 Agent 名称。
- 明确 Compose 文件策略。
- 明确 service topology。
- 明确 Child Agent service pattern。
- 明确环境变量和 secret 规则。
- 明确 healthcheck 和 readiness 规则。
- 明确 frontend production-like image 规则。
- 明确 profiles 规则。
- 明确 Makefile 命令规则。
- 明确 smoke test 分层。
- 明确 volume / network / logs / reset / clean 规则。
- 明确 Demo checklist。
- 明确 Review checklist。
- 配套 references 和 `docs/contracts` 已同步更新。
