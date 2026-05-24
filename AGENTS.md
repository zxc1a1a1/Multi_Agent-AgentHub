# Codex 全局工作规则



## 1. 最高安全底线：绝不泄漏密钥

任何情况下都不允许创建、读取、修改、提交、推送真实密钥。

禁止把以下真实内容写入代码、测试、文档、日志、提交信息、PR 描述、注释、配置文件、临时文件或任何可被 Git 追踪的文件中：

- 真实 API Key
- 真实 Token
- 真实数据库密码
- 真实 `DATABASE_URL`
- 真实 `.env` 内容
- 私钥文件
- SSH 私钥
- 云服务凭证
- OpenAI / Anthropic / GitHub / DeepSeek / 阿里云 / 腾讯云等平台密钥

以下文件永远不能提交：

- `.env`
- `.env.local`
- `.env.development`
- `.env.production`
- `.env.test`
- `id_rsa`
- `id_ed25519`
- `*.pem`
- `*.key`
- `*.p12`
- `*.pfx`

`.env.example` 可以提交，但只能包含空值或明显占位符，例如：

```env
OPENAI_API_KEY=
ANTHROPIC_API_KEY=
DB_PASSWORD=
AGENTHUB_API_TOKEN=
DATABASE_URL=
```

不要在 `.env.example` 中写真实格式的长密钥。  
避免使用看起来像真实密钥的示例值。

---

## 2. 每次 commit / push 前必须做安全检查

在执行 `git commit` 或 `git push` 前，必须检查：

```powershell
git status --short
git diff --staged --name-only
git diff --staged
git ls-files .env
```

必须确认：

- `.env` 没有被 Git 跟踪
- `.env` 没有出现在 staged 文件中
- 没有真实 API Key
- 没有真实 Token
- 没有真实数据库密码
- 没有真实 `DATABASE_URL`
- 没有私钥内容
- 没有构建产物，例如 `dist/`、`node_modules/`、`*.exe`

建议执行敏感关键词扫描：

```powershell
git diff --staged | Select-String "OPENAI_API_KEY|ANTHROPIC_API_KEY|AGENTHUB_API_TOKEN|DB_PASSWORD|MYSQL_ROOT_PASSWORD|DATABASE_URL|PRIVATE KEY|BEGIN RSA"
git diff --staged | Select-String "sk-[A-Za-z0-9_-]{20,}"
git ls-files .env
```

如果发现真实密钥、真实密码、真实 token 或 `.env` 被提交，必须立刻停止，不允许继续 commit 或 push。

---

## 3. AgentHub 项目专用规则

当当前目录是 AgentHub 项目时，必须遵守本节规则。

AgentHub 是一个多 Agent 协作平台，核心链路包括：

```text
用户消息
→ Frontend
→ Gateway
→ Orchestrator
→ Child Agent
→ A2A / AG-UI 流式返回
→ Artifact / CodePreview 展示
```

修改代码前必须先判断应该使用哪些 Skills。

仓库级 Skills 位置：

```text
.agents/skills
```

本机个人 Skills 位置：

```text
~/.agents/skills
```

AgentHub 项目的长期事实源：

```text
docs/contracts
docs/architecture
docs/skill
```

不要凭记忆改协议。  
涉及协议、数据结构、事件、数据库、鉴权、测试、Docker 时，必须先参考对应 Skill 和 contract 文档。

---

## 4. AgentHub Skills 使用规则

处理 AgentHub 任务时，优先使用以下 Skills。

### 4.1 架构相关

使用：

```text
project-architecture
```

适用场景：

- 修改系统架构
- 修改模块边界
- 修改 Frontend / Gateway / Orchestrator / Child Agent 职责
- 判断 MVP 范围
- 判断某个功能属于 MVP 还是 Post-MVP

禁止：

- 未经确认重写整体架构
- 提前实现 Post-MVP 大功能
- 把 Gateway、Orchestrator、Child Agent 职责混在一起

---

### 4.2 代码风格与协作规范

使用：

```text
code-style-and-conventions
ai-collaboration-workflow
```

适用场景：

- 编写 Go / TypeScript / React / JSON / SQL
- 生成 commit message
- 做代码审查
- 拆分任务批次
- 防止 scope creep
- 约束 Codex 修改范围

要求：

- 复杂任务先分析，不要直接修改
- 大任务分批做
- 每批只改允许范围内的文件
- commit message 使用 Conventional Commit 格式
- commit message 必须基于 `git diff --staged`
- 不夸大完成度

---

### 4.3 REST API / 前后端接口

使用：

```text
platform-api-contract
```

适用场景：

- 修改 Frontend 到 Gateway 的 REST API
- 修改请求 / 响应结构
- 修改 OpenAPI schema
- 修改 auth header
- 修改分页、错误格式、状态码
- 修改前端 API client

要求：

- 不要把 REST API 和 AG-UI 事件混用
- 不要把 Gateway-Orchestrator 内部协议暴露给前端
- 对外错误消息必须脱敏
- 不返回内部 stack trace、数据库错误、DSN、密钥、token

---

### 4.4 AG-UI 流式事件

使用：

```text
agui-event-contract
```

适用场景：

- 修改 SSE 流式输出
- 修改 AG-UI events
- 修改 run lifecycle
- 修改 text streaming
- 修改 tool call event
- 修改 frontend event reducer
- 修改 CodePreview 相关事件消费

要求：

- AG-UI 是前端运行时事件协议
- 不要直接把 A2A 原始事件透传给前端
- 错误事件必须脱敏
- 不允许泄漏内部 agent URL、stack trace、API key、数据库信息

---

### 4.5 Gateway-Orchestrator 内部协议

使用：

```text
gateway-orchestrator-contract
```

适用场景：

- 修改 Gateway 到 Orchestrator 的内部调用
- 修改 run lifecycle
- 修改 OrchestratorEvent
- 修改 Gateway 将 OrchestratorEvent 映射到 AG-UI 的逻辑
- 修改 cancellation / approval / tool result 流程

要求：

- 长期架构中 Orchestrator 输出内部事件
- Gateway 负责转换为 AG-UI Event
- MVP 可以保留轻量化实现，但必须标明边界
- 不要把 Gateway 内部 endpoint 暴露成前端公开 API

---

### 4.6 A2A / Child Agent 协议

使用：

```text
a2a-agent-contract
```

适用场景：

- 修改 Orchestrator 到 Child Agent 的 A2A 调用
- 修改 AgentCard
- 修改 `sendSubscribe`
- 修改 task 状态
- 修改 artifacts
- 修改 agent health check
- 修改 Child Agent endpoint

要求：

- A2A 是 Orchestrator 与 Child Agent 之间的协议
- 前端不直接调用 A2A
- A2A endpoint 不应暴露给浏览器
- AgentCard 不得包含密钥
- A2A error 返回给前端前必须经过脱敏转换

---

### 4.7 前端 Runtime Skills / CodePreview

使用：

```text
frontend-runtime-skills-contract
artifact-contract
```

适用场景：

- 修改 CodePreview
- 修改 web_preview
- 修改 confirm_action
- 修改 Tool Call 渲染
- 修改 frontend skill registry
- 修改 artifact 到 preview 的映射
- 修改 tool result

要求：

- `code_preview` 只展示代码，不执行代码
- 不允许 `eval`
- 不允许 `new Function`
- 不允许执行用户生成的脚本
- 可疑 HTML / JS 只能作为代码文本展示
- artifact schema 变更必须同步测试

---

### 4.8 数据持久化

使用：

```text
data-persistence-contract
```

适用场景：

- 修改数据库表
- 修改 MySQL / PostgreSQL schema
- 修改 conversation / message / run / task / artifact 存储
- 修改 migration
- 修改 Redis / Object Storage 规划
- 修改 ID 字段
- 修改持久化测试

要求：

- MVP 当前使用 MySQL
- 不要随意引入 PostgreSQL / Redis / Object Storage 实现
- Post-MVP 表只能作为规划，不能误当作 MVP 必需实现
- 数据库密码只能来自环境变量
- 不允许在源码中硬编码数据库密码

---

### 4.9 Intent Orchestration

使用：

```text
intent-orchestration-contract
```

适用场景：

- 修改意图识别
- 修改 agent routing
- 修改 execution plan
- 修改 fallback 策略
- 修改多 Agent 编排
- 修改 planner validation

要求：

- MVP 可保留直接路由 code-agent
- 不要为了优化提前重写复杂多 Agent 编排
- 如果修改路由策略，必须保证 MVP demo 主链路不破坏

---

### 4.10 ADK Runtime / Child Agent Runtime

使用：

```text
adk-runtime-contract
```

适用场景：

- 修改 Child Agent runtime
- 修改 agent execution context
- 修改 tool adapter
- 修改 streaming output
- 修改 artifact output
- 修改 agent lifecycle

要求：

- 不要破坏 A2A compatibility
- streaming 必须保持实时
- artifact 输出必须能被 Orchestrator/Gateway 转换成前端可消费事件

---

### 4.11 安全边界

使用：

```text
security-boundary-contract
```

适用场景：

- 修改 auth
- 修改 fixed token / JWT
- 修改 secret management
- 修改 API key 读取
- 修改 error redaction
- 修改 sandbox
- 修改 file upload
- 修改 run_command
- 修改 confirm_action

要求：

- MVP 固定 Token 可以接受
- 不要擅自引入复杂用户系统
- API key 只从环境变量读取
- 错误对外必须脱敏
- 日志中不得打印密钥
- `.env` 永远不能提交
- AgentCard 不能包含密钥
- 前端不能拿到后端密钥

---

### 4.12 LLM Provider

使用：

```text
llm-provider-contract
```

适用场景：

- 修改 LLM provider
- 修改模型配置
- 修改 prompt template
- 修改 retry / fallback
- 修改 provider error handling
- 修改 streaming model output

要求：

- 不要写入真实 API key
- provider key 必须来自环境变量
- 错误返回前必须脱敏
- 不要把 provider 原始错误完整暴露给前端

---

### 4.13 可观测性与调试

使用：

```text
observability-debugging-contract
```

适用场景：

- 修改 logs
- 修改 traceId / requestId / runId / taskId
- 修改 metrics
- 修改 debug view
- 修改 error traces

要求：

- 日志可用于调试，但不能打印密钥
- trace id 应贯穿 Gateway / Orchestrator / Agent
- 错误分类要稳定
- 前端错误展示要安全降级

---

### 4.14 测试与验收

使用：

```text
testing-review-contract
```

适用场景：

- 添加或修改测试
- 修复 MVP verification report
- 添加 contract test
- 添加 frontend component test
- 添加 backend unit test
- 添加 E2E
- 添加 CI
- 做最终验收

要求：

- 先明确对应的质量门禁
- 不为了测试重写业务逻辑
- 不连接真实外部服务
- 不读取真实 `.env`
- 优先使用 mock / fake / sqlmock
- 测试不应包含真实 API key 或真实密码

---

### 4.15 Docker / Compose / Delivery

使用：

```text
docker-compose-delivery
```

适用场景：

- 修改 Dockerfile
- 修改 docker-compose.yml
- 修改 .env.example
- 修改本地启动脚本
- 修改 healthcheck
- 修改 smoke test
- 修改 service ports

要求：

- docker-compose 不写真实密码
- 密码通过环境变量注入
- `.env.example` 只能有空值或占位符
- Dockerfile Go 版本应与 go.mod 对齐
- 不提交本地构建产物

---

## 5. 工作流程

对于复杂任务，必须按以下流程：

### 第一步：分析

先阅读相关文件和 Skill，只输出分析报告，不修改文件。

分析报告需要说明：

- 当前问题是否真实存在
- 涉及哪些文件
- 应使用哪些 Skills
- 建议修改范围
- 不应修改哪些内容
- 是否存在安全风险

### 第二步：制定最小计划

计划必须明确：

- 本批次目标
- 允许修改的文件
- 禁止修改的文件
- 测试命令
- 安全检查命令
- commit 范围

### 第三步：执行修改

只修改计划允许的文件。

不允许随意扩大范围。

如果发现必须修改额外文件，先停止并说明原因。

### 第四步：验证

根据项目类型运行对应验证。

AgentHub 常用验证命令如下。

Frontend：

```powershell
cd frontend
npm test -- --run
npm run build
```

Server：

```powershell
cd server
go test ./...
go build ./cmd/server
```

Agents：

```powershell
cd agents
go build ./code-agent
```

### 第五步：安全检查

commit 前必须检查：

```powershell
git status --short
git diff --staged --name-only
git diff --staged
git ls-files .env
```

敏感信息扫描：

```powershell
git diff --staged | Select-String "OPENAI_API_KEY|ANTHROPIC_API_KEY|AGENTHUB_API_TOKEN|DB_PASSWORD|MYSQL_ROOT_PASSWORD|DATABASE_URL|PRIVATE KEY|BEGIN RSA"
git diff --staged | Select-String "sk-[A-Za-z0-9_-]{20,}"
```

### 第六步：生成 commit message

commit message 必须：

- 使用 Conventional Commit
- 基于 `git diff --staged`
- 不夸大完成度
- 不写真实密钥
- 不写真实密码
- 不写真实环境变量值
- 用中文描述主要改动
- 写清楚实际执行过的验证

示例：

```text
test(frontend): 补充 CodePreview 组件最小测试

- 新增 CodePreview 组件测试，覆盖正常渲染、空 code 和缺省字段兼容
- 覆盖复制按钮主路径与 fallback 行为
- 补充 Vitest 与 Testing Library 配置

验证：
- frontend 目录下 npm test -- --run 通过
- frontend 目录下 npm run build 通过
```

---

## 6. 修改边界

默认不要做以下事情，除非用户明确要求：

- 不重写项目架构
- 不重写前端状态管理
- 不重写 Gateway
- 不重写 Orchestrator
- 不重写 Child Agent
- 不重写数据库层
- 不引入复杂 JWT 用户系统
- 不提前实现 Post-MVP 多 Agent 编排
- 不切换包管理器
- 不删除队友文件
- 不格式化无关文件
- 不提交构建产物
- 不提交 `node_modules`
- 不提交 `dist`
- 不提交 `*.exe`

---

## 7. Git 操作规则

修改前先看：

```powershell
git status
git branch -vv
```

如果工作区不干净，先说明当前状态，不要直接覆盖。

同步远程分支时，只有在用户明确允许后才使用：

```powershell
git reset --hard origin/<branch>
```

因为 `reset --hard` 会丢弃本地未提交修改。

提交时不要使用：

```powershell
git add .
```

优先逐个添加本批次文件，例如：

```powershell
git add frontend/src/components/CodePreview.tsx
git add frontend/src/components/CodePreview.test.tsx
```

push 前确认：

```powershell
git status
git log --oneline -5
```

---

## 8. 输出风格

回答用户时：

- 使用中文
- 直接给步骤
- 多给可复制命令
- 少讲抽象概念
- 明确告诉用户“现在做哪一步”
- 如果有风险，先说明风险
- 如果不确定，不要假装确定
- 如果验证失败，必须如实说明失败原因
