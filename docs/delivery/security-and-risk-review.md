# AgentHub v1.0 安全与风险审计

## 1. Secret 边界

### 1.1 已确认不存在于仓库

| 类型 | 状态 | 验证方式 |
|------|------|----------|
| 真实 API key (sk-ant-*, sk-proj-*, etc.) | ✅ 不存在 | `rg` 全仓库扫描 |
| OPENAI_API_KEY 真实值 | ✅ 不存在 | `git diff` + `rg` 扫描 |
| ANTHROPIC_API_KEY 真实值 | ✅ 不存在 | 同上 |
| DATABASE_URL 真实连接串 | ✅ 不存在 | 代码中无此变量 |
| 私钥 (BEGIN RSA / BEGIN OPENSSH) | ✅ 不存在 | 全仓库扫描 |
| MySQL/MariaDB password | ✅ 不适用 | v1.0 不依赖 MySQL |
| 内部服务 token (INTERNAL_SERVICE_TOKEN) | ⚠️ 值为 `dev-internal-token` | 仅用于 demo，非生产密钥 |

### 1.2 环境变量名引用（非泄露）

文档和代码中出现的 `OPENAI_API_KEY`、`ANTHROPIC_API_KEY`、`DATABASE_URL` 均为环境变量名引用，**不含真实值**。这些出现在：
- README.md 历史文档（legacy compose 配置说明）
- 旧 `server/` 和 `agents/` 中的历史配置模板

### 1.3 .env 不入库

仓库中无 `.env` 文件。所有配置通过 Docker Compose environment 或代码中的 default 常量提供。

## 2. 接口安全

### 2.1 Auth

- Gateway 支持 Bearer Token 鉴权（`GATEWAY_ENABLE_AUTH` + `AGENTHUB_API_TOKEN`）
- Composer demo 默认 `GATEWAY_ENABLE_AUTH=false`（本地 demo 环境）
- Auth token 泄露测试已覆盖（Go 测试验证 error 响应不包含 token）
- Orchestrator 内部 API 使用 `ORCHESTRATOR_INTERNAL_TOKEN`，compose 中值为 `dev-internal-token`

### 2.2 CORS

- Gateway CORS middleware 支持可配置 `GATEWAY_ALLOWED_ORIGINS`
- 默认 `http://localhost:3000,http://127.0.0.1:3000`
- Go 测试覆盖

## 3. Error Sanitization

- **Gateway 层**：`TextStreamFilter` 过滤 error message 中的敏感信息（API key、stack trace、文件路径、panic）
- **Frontend 层**：`sanitizeErrorText()` 二次过滤
- **测试覆盖**：Go 测试 + 前端测试（`sk-*` token、`panic`、file path、`OPENAI_API_KEY` 检测）
- **smoke 覆盖**：Log Safety Level 5 验证 compose logs 无 panic/fatal/API key 泄露

## 4. 进程边界安全

| 规则 | 状态 | 验证方式 |
|------|------|----------|
| Gateway 不直连 Agent | ✅ 遵守 | CI compose config check 断言 |
| Frontend 不直连 Orchestrator | ✅ 遵守 | 架构审查 |
| Frontend 不直连 Agent | ✅ 遵守 | 架构审查 |
| Gateway 不 import Orchestrator 业务包 | ✅ 遵守 | 代码审查 |
| Gateway 不直接调用 LLM Provider | ✅ 遵守 | 代码审查 |
| Orchestrator 不暴露给浏览器 | ✅ 遵守 | 架构审查 |
| Child Agent 不反向调用 Gateway | ✅ 遵守 | 代码审查 |

## 5. CI 安全

- CI 不依赖真实外部服务（LLM API、OCR、文件解析等）
- CI 使用 deterministic mock workload
- CI log safety 自动检查
- 无 secret 通过 CI 注入

## 6. SQLite 数据安全

- SQLite 数据存储在 Docker named volume（`gateway-data`），不暴露到外部
- Demo 数据为 mock 响应，不含用户隐私
- 错误消息经过 `TextStreamFilter` 脱敏后才写入 SQLite
- 环境变量 `AGENTHUB_SQLITE_PATH` 不使用真实路径之外的敏感信息

## 7. 当前 v1.0 风险

| 风险 | 等级 | 说明 | 缓解 |
|------|------|------|------|
| SQLite 并发写限制 | 低 | v1.0 单用户 demo，无并发问题 | WAL 模式启用；后续可换 PostgreSQL |
| Auth token 为 `dev-internal-token` | 低 | 仅 demo 环境 | 生产环境需替换 |
| 无 rate limiting | 低 | v1.0 demo 不暴露公网 | 生产环境需添加 |
| 无 HTTPS | 低 | Docker compose 本地网络 | 生产环境需 nginx/TLS |
| RulePlanner 关键词匹配有限 | 低 | 仅覆盖 web/code 场景 | v1.1 LLMPlanner |
| 无审计日志 | 低 | v1.0 demo 无需合规审计 | 后续可接入 |

## 8. 已知限制

1. **单用户 demo**：无多用户认证和会话隔离
2. **Deterministic mock**：Agent 返回固定内容，不适用真实场景
3. **2 Agent**：当前仅 code-agent + web-agent 已服务化
4. **无对象存储**：Artifact 内容元数据存储在 SQLite 中，不支持大文件
5. **无 HTTPS**：仅 HTTP 明文通信（容器内网）
6. **基础 error handling**：无 retry backoff、circuit breaker
7. **无监控**：无可观测性 dashboard

## 9. 比赛演示注意事项

1. **不暴露内部端口到公网** — compose 映射的端口仅用于本地访问
2. **演示结束后** — `docker compose down -v` 清理数据卷
3. **不要将 demo 部署到生产环境** — 使用 `dev-internal-token` 和 `GATEWAY_ENABLE_AUTH=false`
4. **API key 检查** — 演示前再次确认无真实 key 写入仓库
5. **前端构建** — 确认 `npm run build` 产物已生成（如通过 compose 则自动构建）

---

- Created: 2026-06-05
- Step: AgentHub v1.0 Step 5 — Security & Risk Review
