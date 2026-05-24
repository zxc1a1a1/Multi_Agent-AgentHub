# AgentHub MVP review2 修复进展与交接报告

## 1. 当前结论

根据 `review2` 文档，本次 MVP 修复工作已经完成主要代码修复：

- review2 的 P0 修复已完成。
- review2 的 P1 修复已完成。
- review2 的 P2 修复已完成。
- 开始前需要确认 `dev` 已经包含本次 review2 修复提交。

当前剩余工作不是继续改 P0 / P1 / P2 代码，而是补跑我本机环境受限未完成的验证项，主要包括 Docker、bash、docker compose 和 smoke-test 相关验证。

这些验证缺口不是已知代码逻辑失败，而是因为我本机 WSL / Docker Desktop / bash 环境异常，无法完整执行相关命令。

如果同事在正常 Docker / WSL / bash 环境下补跑通过，review2 MVP 修复可以认为闭环；如果补跑失败，再根据失败日志做小补丁。

---

## 2. 已完成的 P0 修复

P0 是 review2 中最高优先级的问题。

### 2.1 SSE 解析修复

前端 AG-UI SSE parser 已增强，支持：

- SSE 事件块解析
- 半包 buffer 处理
- 单个 chunk 内多个事件处理
- 多行 `data:` 处理
- malformed JSON 跳过处理

修复目标是避免前端在流式输出时因为 chunk 边界、半包、多事件混合导致解析异常。

### 2.2 LLM HTTP timeout

`code-agent` 的 LLM HTTP 请求已增加 timeout 控制，避免调用外部 LLM 时请求长期阻塞。

修复后不再走无超时的默认 HTTP 请求路径，提升 agent 调用稳定性。

### 2.3 A2A client 复用

Gateway / Orchestrator 侧 A2A client 已按 `agentURL` 复用。

修复目标是避免每次调用 agent 都重复创建 client，降低不必要的连接和资源开销。

### 2.4 threadId / conversation 前置校验

AG-UI `/run` 之前已增加 `threadId` / `conversation` 前置校验。

无效、空值或不存在的 conversation 会提前返回错误，避免无效 threadId 写入消息表或触发后续异常链路。

### 2.5 P0 状态

- P0 代码修复已完成。
- P0 已完成本地测试。

---

## 3. 已完成的 P1 修复

P1 是 review2 中稳定性和生产化相关问题。

### 3.1 Gateway graceful shutdown

Gateway 已增加基于 `http.Server` 的优雅关闭逻辑。

收到 SIGINT / SIGTERM 后，服务会在超时时间内尝试完成已有请求，再退出进程。

### 3.2 CORS 白名单

Gateway 已增加 CORS 白名单配置。

默认支持本地前端开发端口，并支持通过环境变量配置允许的 origin，避免过度开放跨域访问。

### 3.3 code-agent LLMClient 单例化

`code-agent` 中 LLMClient 已改为进程级单例初始化，并注入 handler 使用。

修复目标是避免每次请求重复创建 LLM client，提高稳定性和资源复用能力。

### 3.4 frontend Dockerfile 生产化

前端 Dockerfile 已改为生产构建方式：

- build 阶段使用 Node 构建前端静态资源
- runtime 阶段使用 nginx 提供静态资源
- nginx 监听生产端口

### 3.5 server Dockerfile 多阶段构建

Server Dockerfile 已改为多阶段构建：

- build 阶段编译 Go server
- runtime 阶段使用更轻量镜像运行
- runtime 镜像包含必要的 CA 证书

### 3.6 History 结构化传递

Orchestrator 到 agent 的历史消息传递已增强为结构化 role/content 形式。

用户消息映射为 `user`，agent 消息映射为 `assistant`，当前请求消息也会追加进上下文。

### 3.7 P1 状态

- P1 代码修复已完成。
- P1 已完成本地测试。

---

## 4. 已完成的 P2 修复

P2 是 review2 中稳定性、可维护性、验证补强相关问题。

### 4.1 docker-compose frontend 端口映射修复

`docker-compose.yml` 中 frontend 服务端口映射已从：

```yaml
3000:5173
```

修正为：

```yaml
3000:3000
```

原因是前端生产镜像实际由 nginx 提供服务，监听端口为 `3000`，不再是 Vite dev server 的 `5173`。

### 4.2 docker-compose restart policy

`docker-compose.yml` 中已为以下服务增加：

```yaml
restart: unless-stopped
```

涉及服务：

- frontend
- gateway
- code-agent
- mysql

用于提升本地部署和容器环境中的基础恢复能力。

### 4.3 删除 Orchestrator dead code

已删除 Orchestrator 中不再使用的 `extractTextFromEvent` dead code，并清理相关 import。

该修复不改变 Orchestrator 主流程，也不改变 A2A / AG-UI 协议语义。

### 4.4 DB 连接池配置环境变量化

DB 连接池参数已支持环境变量配置：

- `DB_MAX_OPEN_CONNS`
- `DB_MAX_IDLE_CONNS`
- `DB_CONN_MAX_LIFETIME_SECONDS`

默认值如下：

- `DB_MAX_OPEN_CONNS=20`
- `DB_MAX_IDLE_CONNS=5`
- `DB_CONN_MAX_LIFETIME_SECONDS=300`

空值、非数字、负数、0 等非法值会回退默认值。

该修复不修改数据库 schema。

### 4.5 REST conversation_id UUID 校验

`GET /api/conversations/:id/messages` 已增加 conversation id UUID 校验。

修复后：

- 空 id 返回 400
- 非法 UUID 返回 400
- 错误响应使用脱敏文案
- 不暴露 SQL、表名、stack trace 等内部信息

### 4.6 smoke-test 增加 AG-UI SSE 验证

`smoke-test.sh` 已增加 `/api/agui/run` 最小 SSE 链路验证。

逻辑包括：

- 未设置 `AGENTHUB_API_TOKEN` 时，验证接口返回 401
- 设置 `AGENTHUB_API_TOKEN` 时，通过环境变量读取 token
- 调用 `/api/agui/run`
- 验证响应是 SSE 流式链路
- 允许无真实 LLM Key 场景下通过 `RUN_ERROR` 判断协议链路可达

注意：脚本只读取环境变量，不写死真实 token。

### 4.7 messageStore streaming 改为 per-conversation

前端 `messageStore` 中 streaming 状态已从全局状态改为按 conversation 维护。

已调整为：

- `streamingByConversation`
- `abortControllersByConversation`
- `isStreaming(conversationId)`
- `stopStreaming(conversationId)`

修复后，不同 conversation 的流式状态和取消控制互不覆盖。

### 4.8 React Error Boundary

前端已新增 `ErrorBoundary`，并在根组件处包裹 `App`。

当 React 渲染出现未捕获异常时，会展示 fallback 页面，并提供 reload 按钮执行：

```ts
window.location.reload()
```

### 4.9 parseCodeBlocks 稳定性增强

`code-agent` 中 `parseCodeBlocks` 已从单条正则整体解析改为按行扫描 fenced code block。

增强内容包括：

- 支持标准代码块
- 支持无文件名代码块
- 支持多个代码块
- 支持无代码块场景
- 对语言名增加合法字符校验
- 仅当独立成行的三反引号出现时闭合代码块，降低误截断风险
- 在注释 / prompt 中明确 MVP 限制：代码块内部不应输出未转义且独立成行的三反引号

### 4.10 P2 状态

- P2 代码修复已完成。
- P2 已完成本地 Go / 前端测试。

---

## 5. 已完成的本地验证

本地已经完成以下验证。

### 5.1 agents 验证

```powershell
cd agents
go test ./...
go build ./code-agent
```

结果：通过。

### 5.2 server 验证

```powershell
cd server
go test ./...
go build ./cmd/server
```

结果：通过。

### 5.3 frontend 验证

```powershell
cd frontend
npm test -- --run
npm run build
```

结果：通过。

其中前端 build 出现 chunk size warning，这是 Vite 的体积提示，不是构建失败。

### 5.4 构建产物清理

已清理以下构建产物：

- `agents/code-agent.exe`
- `server/server.exe`
- `frontend/dist`

这些文件没有提交。

### 5.5 敏感信息扫描

已执行敏感信息扫描，未发现真实密钥泄漏。

确认没有提交：

- `.env`
- 真实 API Key
- 真实 Token
- 真实数据库密码
- 真实 `DATABASE_URL`
- 私钥
- `node_modules`
- `dist`
- `.exe` 构建产物

扫描中出现的以下内容属于环境变量占位或测试假值，不是真实密钥：

- `${AGENTHUB_API_TOKEN}`
- `${MYSQL_ROOT_PASSWORD}`
- `os.Getenv("AGENTHUB_API_TOKEN")`
- `test-token`
- 单元测试中的本地 `DATABASE_URL`

---

## 6. 还需要同事根据 review2 补做的事情

由于我本机 WSL / Docker Desktop / bash 环境异常，以下验证没有完整执行，需要同事在环境正常的机器上补跑。

这些是验证缺口，不是已知代码逻辑失败。

### 6.1 拉取最新 dev 分支

直接从 `dev` 分支拉取即可。

同事开始前需要确认 `dev` 已经包含本次 review2 修复提交。

### 6.2 bash 语法检查

需要补跑：

```bash
bash -n smoke-test.sh
```

目标是确认 `smoke-test.sh` 没有 shell 语法错误。

### 6.3 docker compose 配置检查

需要补跑：

```bash
docker compose config
```

目标是确认 `docker-compose.yml` 配置合法，服务、环境变量、端口映射没有 compose 解析错误。

### 6.4 Docker 镜像构建

需要补跑：

```bash
docker build -t agenthub-frontend-test ./frontend
docker build -t agenthub-gateway-test ./server
```

目标是确认 frontend / server 的生产 Dockerfile 可以在正常 Docker 环境下构建成功。

### 6.5 docker compose 一键启动

需要补跑：

```bash
docker compose up --build
```

目标是确认 MVP 依赖服务可以通过 compose 启动，包括：

- frontend
- gateway
- code-agent
- mysql

### 6.6 smoke-test.sh

需要在服务启动后补跑：

```bash
./smoke-test.sh
```

如果脚本需要 token，请只在本地环境变量中设置测试 token，不要写入文件。

PowerShell 示例：

```powershell
$env:AGENTHUB_API_TOKEN="<本地测试token>"
```

Bash 示例：

```bash
export AGENTHUB_API_TOKEN="<本地测试token>"
```

不要把真实 token 写进 `.env`、脚本、README 或提交内容。

### 6.7 如果补验证失败

如果 Docker / smoke-test 验证失败，请保留完整日志，包括：

- 命令
- 当前分支
- 最新 commit
- 失败输出
- 服务日志
- Docker / compose 版本

然后把日志发回来，再基于失败点做小补丁。

---

## 7. 同事拉取方式

同事后续直接从 `dev` 分支拉取。

```powershell
git fetch origin
git switch dev
git pull origin dev
git log --oneline -10
```

注意：

- 我会先把本次修复提交到 `g`。
- 然后我会把 `g` 同步 / 合并到 `dev`。
- 同事不要基于旧 `dev` 开始开发。
- 同事开始前先确认 `dev` 的最新提交里包含 review2 P2 修复提交。
- 如果 `dev` 暂时还没有同步完成，先等我同步完成后再拉 `dev`。

---

## 8. 暂时不要开始的新功能

在 Docker / smoke-test 补验证通过之前，暂时不要先做以下后续 v1.0 Sprint 功能：

- web-agent
- planner
- registry
- 群聊
- web_preview
- markdown_render

原因：

- 这些属于后续 v1.0 Sprint，不属于本次 review2 MVP 修复收尾。
- 当前优先级是先让 MVP review2 修复验证闭环。
- 如果在 Docker / smoke-test 还没补验证前直接上新功能，后续问题定位会更复杂。

---

## 9. 提交与安全说明

本次 review2 修复遵守以下安全边界：

- 不提交 `.env`
- 不提交真实 API Key
- 不提交真实 Token
- 不提交真实数据库密码
- 不提交真实 `DATABASE_URL`
- 不提交私钥
- 不提交 `node_modules`
- 不提交 `frontend/dist`
- 不提交 `.exe` 构建产物

`smoke-test.sh` 只从环境变量读取：

```bash
AGENTHUB_API_TOKEN
```

测试中的 `test-token` 和测试 `DATABASE_URL` 仅为单元测试占位值，不是生产凭据。

---

## 10. 给同事的命令参考

### 10.1 拉取 dev

```powershell
git fetch origin
git switch dev
git pull origin dev
git log --oneline -10
```

### 10.2 agents 验证

```powershell
cd agents
go test ./...
go build ./code-agent
```

### 10.3 server 验证

```powershell
cd server
go test ./...
go build ./cmd/server
```

### 10.4 frontend 验证

```powershell
cd frontend
npm test -- --run
npm run build
```

### 10.5 Docker / smoke 验证

```bash
bash -n smoke-test.sh
docker compose config
docker build -t agenthub-frontend-test ./frontend
docker build -t agenthub-gateway-test ./server
docker compose up --build
```

服务启动后再执行：

```bash
./smoke-test.sh
```

如需 token，只能使用本地环境变量设置测试 token：

```bash
export AGENTHUB_API_TOKEN="<本地测试token>"
```

不要把真实 token 写进任何文件，也不要提交 token。

---

## 11. 总结

当前 review2 的 P0 / P1 / P2 代码修复已经完成。

你现在最需要做的是：

1. 拉取最新 `dev`。
2. 确认 `dev` 已包含本次 review2 修复提交。
3. 补跑 bash / docker / docker compose / smoke-test 验证。
4. 如果补验证通过，review2 MVP 修复可以认为闭环。
5. 如果补验证失败，把完整日志发回来，再做小补丁。

在这些验证完成前，不建议开始 web-agent、planner、registry、群聊、web_preview、markdown_render 等后续 v1.0 功能。