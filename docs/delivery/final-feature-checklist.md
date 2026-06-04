# AgentHub v1.0 最终功能清单

## 1. Gateway API

| 功能 | 状态 | 验证方式 | 备注 |
|------|------|----------|------|
| `GET /health` | ✅ 已完成 | smoke Level 1 / Go 测试 | 返回 `{"status":"ok","service":"gateway"}` |
| `POST /api/conversations` | ✅ 已完成 | Go 集成测试 | 创建会话，返回 id + title |
| `GET /api/conversations` | ✅ 已完成 | Go 测试 | 按 userId 列出会话 |
| `GET /api/conversations/{id}/messages` | ✅ 已完成 | smoke Level 4 / replay 测试 | 从 SQLite/MemoryStore 读取消息 |
| `GET /api/agents` | ✅ 已完成 | Go 测试 | 返回 AgentSummary 列表 |
| `POST /api/chat` | ✅ 已完成 | smoke Level 2/3 | SSE 流式响应 |
| Auth (Bearer Token) | ✅ 已完成 | Go 测试 | Token 泄露测试 |
| CORS | ✅ 已完成 | Go 测试 | 允许跨域 |
| Error sanitization | ✅ 已完成 | Go + smoke 测试 | TextStreamFilter 脱敏 |
| Gateway 不直连 Agent | ✅ 已完成 | CI compose config check | 断言 `gateway-new env` 不含 `AGENT_CODE_URL` 等 |

### 不在 v1.0 范围

- Multi-user RBAC 权限系统
- Rate limiting
- Request logging/monitoring dashboard

## 2. Orchestrator 编排

| 功能 | 状态 | 验证方式 | 备注 |
|------|------|----------|------|
| RulePlanner | ✅ 已完成 | Go 测试 | 关键词规则生成 plan |
| PlanValidator | ✅ 已完成 | Go 测试 | 校验 Agent 存在性/能力/taskId |
| SingleExecutor | ✅ 已完成 | smoke Level 2 | 单 Agent 执行 |
| OrderedParallelExecutor | ✅ 已完成 | smoke Level 3 | 多 Agent 有序并行 |
| A2A Dispatcher | ✅ 已完成 | Go 测试 | A2A 协议调用 Agent |
| Agent Registry | ✅ 已完成 | Go 测试 | 静态注册 + Health Check 过滤 |
| Summary Aggregation | ✅ 已完成 | smoke Level 3 | Orchestrator summary 消息 |
| Fallback | ✅ 已完成 | Go 测试 | Agent 不健康时尝试替代 |

### 不在 v1.0 范围

- LLMPlanner（真实 LLM 计划生成） — 需要 API key + 安全审查
- Sequential executor — 架构支持，v1.0 无 demo 场景
- Real-time run progress via DB — SSE stream 已提供
- Complex retry policy — 有基础 fallback

## 3. RulePlanner

| 功能 | 状态 | 验证方式 | 备注 |
|------|------|----------|------|
| 关键词检测 (web/html/css) → web-agent | ✅ 已完成 | Go 测试 | 确定性规则 |
| 关键词检测 (go/api/http/server) → code-agent | ✅ 已完成 | Go 测试 | 确定性规则 |
| Mixed 检测 → ordered_parallel | ✅ 已完成 | smoke Level 3 | 双关键词触发多 Agent |
| 未知意图处理 | ✅ 已完成 | Go 测试 | 返回错误提示 |

## 4. PlanValidator

| 功能 | 状态 | 验证方式 | 备注 |
|------|------|----------|------|
| Agent 存在性校验 | ✅ 已完成 | Go 测试 | 不存在 Agent → 拒绝 |
| 能力匹配校验 | ✅ 已完成 | Go 测试 | capabilityId 不匹配 → 拒绝 |
| taskId 唯一性校验 | ✅ 已完成 | Go 测试 | 重复 taskId → 拒绝 |
| executionMode 合法性 | ✅ 已完成 | Go 测试 | 非法 mode → 拒绝 |
| Plan 完整性校验 | ✅ 已完成 | Go 测试 | 空 plan/no tasks → 拒绝 |

## 5. code-agent

| 功能 | 状态 | 验证方式 | 备注 |
|------|------|----------|------|
| `/health` endpoint | ✅ 已完成 | smoke Level 1 | `{"status":"ok"}` |
| A2A Server | ✅ 已完成 | Go 测试 | 接收 A2A task |
| AgentCard | ✅ 已完成 | Go 测试 | `.well-known/agent.json` |
| Generate (deterministic mock) | ✅ 已完成 | smoke Level 2 | 返回 Go 代码片段 |
| Dockerfile | ✅ 已完成 | compose up | 独立容器 |
| 不依赖真实 LLM | ✅ 已完成 | CI | 100% deterministic |

## 6. web-agent

| 功能 | 状态 | 验证方式 | 备注 |
|------|------|----------|------|
| `/health` endpoint | ✅ 已完成 | smoke Level 1 | `{"status":"ok"}` |
| A2A Server | ✅ 已完成 | Go 测试 | 接收 A2A task |
| AgentCard | ✅ 已完成 | Go 测试 | `.well-known/agent.json` |
| Generate (deterministic mock) | ✅ 已完成 | smoke Level 2 | 返回 HTML section |
| Unsafe HTML/token filter | ✅ 已完成 | Go 测试 | 安全过滤 |
| Dockerfile | ✅ 已完成 | compose up | 独立容器 |
| 不依赖真实 LLM | ✅ 已完成 | CI | 100% deterministic |

## 7. Frontend 多 Agent UI

| 功能 | 状态 | 验证方式 | 备注 |
|------|------|----------|------|
| IM 聊天 UI | ✅ 已完成 | 前端测试 | 对话列表 + 消息区 |
| Multi-Agent 独立气泡 | ✅ 已完成 | 前端测试 | 不同 messageId → 不同气泡 |
| senderName / senderDisplayName 展示 | ✅ 已完成 | 前端测试 | Web Agent/Code Agent/Orchestrator |
| code preview 渲染 | ✅ 已完成 | 前端测试 | highlight.js |
| web preview 渲染 | ✅ 已完成 | 前端测试 | iframe sandbox |
| Page refresh 消息恢复 | ✅ 已完成 | replay 测试 | 保持 3 条独立消息 |
| Error 安全展示 | ✅ 已完成 | 前端测试 | 不含 token/stack/path |
| Agent 选择器 | ✅ 已完成 | 前端测试 | 下拉选择 Agent |

### 不在 v1.0 范围

- Run History UI
- Agent 头像区域
- @mention 交互

## 8. AG-UI / SSE

| 功能 | 状态 | 验证方式 | 备注 |
|------|------|----------|------|
| RUN_STARTED event | ✅ 已完成 | smoke | 含 runId |
| TEXT_MESSAGE_START/CONTENT/END | ✅ 已完成 | smoke | sender 对象 |
| RUN_FINISHED event | ✅ 已完成 | smoke | 正常结束 |
| RUN_ERROR event | ✅ 已完成 | 前端测试 | error code + message |
| sender.type / sender.name | ✅ 已完成 | smoke Level 3 | agent/orchestrator |
| senderDisplayName | ✅ 已完成 | replay 测试 | 人类可读名称 |

## 9. Persistence

| 功能 | 状态 | 验证方式 | 备注 |
|------|------|----------|------|
| SQLite schema (6 tables) | ✅ 已完成 | Go 测试 | DDL + 12 indexes |
| Migration runner | ✅ 已完成 | Go 测试 | 幂等 |
| SqliteStore CRUD | ✅ 已完成 | Go 测试 | 15 methods |
| PersistenceWriter | ✅ 已完成 | Go 测试 | SSE event → SQLite |
| Memory/sqlite mode switch | ✅ 已完成 | Go 测试 | AGENTHUB_GATEWAY_STORE |
| Container volume persistence | ✅ 已完成 | CI smoke #18 | /data/agenthub.db |

### 不在 v1.0 范围

- Object storage for large artifacts
- Database backup/restore
- MySQL/PostgreSQL support

## 10. Replay / Refresh Recovery

| 功能 | 状态 | 验证方式 | 备注 |
|------|------|----------|------|
| SQLite replay API | ✅ 已完成 | smoke Level 4 | ReplayMessage 格式 |
| MemoryStore 向后兼容 | ✅ 已完成 | Go 测试 | 旧格式兼容 |
| senderDisplayName 映射 | ✅ 已完成 | Go + 前端测试 | web-agent→Web Agent |
| Artifact 解析基础 | ✅ 已完成 | Go 测试 | metadata_json.artifacts |
| 3 agent 独立消息恢复 | ✅ 已完成 | 前端 replay 测试 | 不合并 |

## 11. Failure / Audit

| 功能 | 状态 | 验证方式 | 备注 |
|------|------|----------|------|
| Error sanitization | ✅ 已完成 | Go + 前端测试 | 不含 token/stack/path |
| ListFailedRuns | ✅ 已完成 | Go 测试 | 按 conversation 查询 |
| ListFailedMessages | ✅ 已完成 | Go 测试 | 按 conversation 查询 |
| ListArtifactsByConversation | ✅ 已完成 | Go 测试 | 审计查询 |
| Error code taxonomy | ✅ 已完成 | 代码 | AGUI_INTERNAL 等 |

### 不在 v1.0 范围

- Automatic retry
- Run duration tracking
- Fallback step audit (tracking)

## 12. Docker Compose Demo

| 功能 | 状态 | 验证方式 | 备注 |
|------|------|----------|------|
| 一键启动 5 服务 | ✅ 已完成 | CI smoke | docker compose up --build |
| SQLite named volume | ✅ 已完成 | CI smoke #18 | gateway-data |
| All service healthchecks | ✅ 已完成 | smoke Level 1 | wget /health |
| Deterministic 响应 | ✅ 已完成 | CI | 无 LLM 依赖 |

## 13. GitHub Actions Smoke

| 功能 | 状态 | 验证方式 | 备注 |
|------|------|----------|------|
| Go tests (4 services) | ✅ 已完成 | CI | gateway + orchestrator + agents |
| Compose config check | ✅ 已完成 | CI | 含 gateway no-agent-URL guard |
| Single code-agent smoke | ✅ 已完成 | CI | SSE + agent attribution |
| Single web-agent smoke | ✅ 已完成 | CI | SSE + agent attribution |
| Mixed ordered_parallel smoke | ✅ 已完成 | CI | 3 agents + ordering |
| Replay/persistence sanity | ✅ 已完成 | CI (#18) | GET messages replay check |
| Log safety check | ✅ 已完成 | CI | 无 panic/API key 泄露 |

---

- Created: 2026-06-05
- Step: AgentHub v1.0 Step 5 — Final Feature Checklist
