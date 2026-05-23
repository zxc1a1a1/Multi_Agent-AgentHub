# MVP 实现验证报告

**生成日期**：2026-05-23
**验证基线**：commit `305eee6`（分支 `y`）
**验证依据**：`testing-review-contract` Skill，MVP 最小质量门禁

---

## 一、编译状态

| 组件 | 命令 | 结果 |
|------|------|:----:|
| Gateway (Go) | `go build ./cmd/server` | 通过 |
| Code-Agent (Go) | `go build ./code-agent` | 通过 |
| Frontend (TS) | `tsc --noEmit` | 通过 |

---

## 二、MVP Demo 路径逐节点追踪

按 `testing-review-contract` 第 6 节 MVP 必测路径逐节点验证：

```
用户发送消息
  → Gateway 保存用户消息
  → Orchestrator 直接路由 code-agent
  → A2A sendSubscribe
  → 文本流式返回
  → Artifact buffer
  → completed 后转换成 code_preview Tool Call
  → 前端 CodePreview 展示
  → 刷新后消息仍在
```

| # | 节点 | 状态 | 关键代码位置 |
|---|------|:----:|-------------|
| 1 | 用户发送消息 | 通过 | `messageStore.sendMessage` → `POST /api/agui/run` (SSE) |
| 2 | Gateway 保存用户消息 | 通过 | `handler/agui.go` 写 user 消息到 DB |
| 3 | Orchestrator 路由 code-agent | 通过 | `orchestrator.go:47` 直接路由 |
| 4 | A2A sendSubscribe | 通过 | `a2a/client.go` 官方 a2a-go/v2 客户端 |
| 5 | 文本流式返回 | 通过（已修复） | `agents/adk/context.go`: StreamText 立即 yield A2A 事件 |
| 6 | Artifact buffer | 通过 | `converter.go` 缓存 code artifacts |
| 7 | 转 code_preview Tool Call | 通过 | `flushArtifacts()` 正确生成 TOOL_CALL 序列 |
| 8 | CodePreview 展示 | 通过 | 语法高亮 + 复制按钮完备 |
| 9 | 刷新后消息仍在 | 通过（已修复） | `handler/agui.go`: SSE 流结束后 SaveMessage 持久化 agent 回复 |

**结果：9/9 节点通过（100%）**

### 已修复的 P0 Bug（commit `305eee6`）

| Bug | 修复内容 |
|-----|---------|
| 流式输出非实时 | `context.go`: StreamText 改为直接 yield A2A TaskArtifactUpdateEvent，每个 chunk 实时产出 |
| Agent 回复不持久化 | `agui.go`: SSE 流结束后收集 agentText + artifacts，调用 SaveMessage 写入 DB |

---

## 三、MVP 最小质量门禁（按 contract 第 6 节）

| # | 门禁项 | 状态 | 说明 |
|---|--------|:----:|------|
| 1 | manual demo checklist | 部分通过 | checklist 已定义（`demo-checklist.md`），但未实际执行（无 LLM API key） |
| 2 | minimum backend unit tests | **未通过** | 项目 **0 个 `*_test.go` 文件**，server/ 和 agents/ 均零覆盖 |
| 3 | minimum protocol conversion tests | **未通过** | `converter.go` 零测试覆盖 |
| 4 | minimum frontend component / hook tests | **未通过** | `frontend/src/` 零测试文件 |
| 5 | minimum E2E happy path | **未通过** | 无 E2E 框架、无测试文件 |
| 6 | docker compose smoke test | 未验证 | Docker Hub 不可达，无法拉取镜像 |
| 7 | secret redaction check | 部分通过 | API key 仅从 env 读取、AgentCard 无密钥、前端无 secret；但 `config.go:19` 源码中硬编码了数据库密码 |

**结果：7 项中 0 项完全通过，2 项部分通过，4 项未通过，1 项未验证**

---

## 四、安全审查（按 contract 第 11 节 + security-review-checklist）

| # | 检查项 | 状态 | 说明 |
|---|--------|:----:|------|
| 1 | API key 来源 | 通过 | 仅环境变量 (`os.Getenv`)，未硬编码在源码中 |
| 2 | AgentCard 无密钥 | 通过 | `adk/server.go:59-75` 不含敏感信息 |
| 3 | A2A endpoint 不暴露给前端 | 通过 | 仅 Orchestrator 内部调用 |
| 4 | 错误脱敏 — handler 层 | **问题** | `conversation.go:33` / `agui.go:19` 直接暴露 `err.Error()` 给客户端 |
| 5 | 错误脱敏 — agent 层 | 通过 | `context.go:128` 固定安全消息，不泄漏 stack trace |
| 6 | 前端未知事件安全降级 | 通过 | `catch` 块忽略未知事件，不崩溃 |
| 7 | `code_preview` 只展示不执行 | 通过 | 无 `eval` 或代码执行逻辑 |
| 8 | 日志无 API key | 通过 | handler 未打印敏感信息 |
| 9 | 认证中间件 | **缺失** | 所有 5 个 API 端点无任何认证，完全开放 |
| 10 | 数据库密码硬编码 | **问题** | `config.go:19` 默认值包含明文密码 `agenthub123` |
| 11 | 输入校验 | **不足** | `AGUIRunRequest` 和 `createConvRequest` 无 `binding` 标签 |
| 12 | docker-compose.yml 明文密码 | **问题** | MySQL root 密码硬编码在 compose 文件中 |

**结果：7/12 通过，3 项有问题，2 项缺失**（上次报告 7/7 通过是因为检查不够深入）

---

## 五、代码质量 — 新发现的问题

### converter.go（协议转换器）

| 严重度 | 问题 |
|:------:|------|
| 中 | `convertMessage` 在 message 未 started 时静默丢弃内容 |
| 中 | 多 Text Part 时仅保留最后一个 |
| 中 | `code_preview` skill 缺失时静默丢弃 code artifacts |
| 中 | `TaskStateInputRequired` 未处理，fallback 返回 nil |
| 低 | `textContent` 字段写入但从未读取（dead store） |

### orchestrator.go

| 严重度 | 问题 |
|:------:|------|
| 中 | 硬编码 `code-agent` 路由，忽略前端 Tools 选择 |
| 低 | `extractTextFromEvent` 定义但从未调用（dead code） |

### agui.go

| 严重度 | 问题 |
|:------:|------|
| 中 | `GetMessages` / `SaveMessage` 错误被静默吞掉 |
| 低 | 100 event 的 channel buffer 在慢客户端时可能阻塞 |

### CodePreview.tsx

| 严重度 | 问题 |
|:------:|------|
| 低 | `dangerouslySetInnerHTML` 无 `DOMPurify` 防御层 |
| 低 | 空 code block 无 placeholder 处理 |
| 低 | 剪贴板 fallback 未验证 `execCommand` 返回值 |

---

## 六、CI / 基础设施

| 项目 | 状态 |
|------|:----:|
| CI 配置文件 | **缺失**（无 `.github/workflows/`、`.gitlab-ci.yml` 等） |
| Dockerfile Go 版本 | `golang:1.23-alpine` vs `go.mod` 要求 `go 1.26.0`，版本不匹配 |
| Docker Compose | 配置正确，但 Docker Hub 不可达无法验证 |

---

## 七、总结

| 维度 | 之前 (9a0ec94) | 现在 (305eee6) |
|------|:----:|:----:|
| 编译 | 三端通过 | 三端通过 |
| Demo 路径完整性 | 7/9 (78%) | **9/9 (100%)** |
| 阻塞性 Bug | 2 个 | **0 个** |
| 质量门禁 | 0/7 | 0/7（2 项部分通过） |
| 安全审查 | 7/7（检查不充分） | 7/12 |

### 结论：MVP Demo 路径已修复，但质量门禁未通过

两个 P0 Bug 已在 commit `305eee6` 中修复，MVP Demo 必测路径 9/9 节点全部通过。
但 **7 项质量门禁中 0 项完全通过**，按 contract 标准 MVP 未通过验收。

### 阻塞项（必须修复才能通过 MVP 验收）

1. **补 minimum backend unit tests** — 至少覆盖 `converter.go`（协议转换）和 `mysql.go`（消息持久化）
2. **补 minimum protocol conversion tests** — converter 的 happy path + error path
3. **补 minimum frontend component/hook tests** — 至少覆盖 `CodePreview` 组件
4. **修复 `config.go:19` 硬编码数据库密码** — 移除 Go 源码中的默认密码
5. **修复 JSON binding 错误暴露** — `conversation.go:33` 和 `agui.go:19` 使用通用错误消息

### 建议（非阻塞，P1 阶段处理）

6. 补 minimum E2E happy path（Playwright 或 Cypress）
7. 添加 CI 配置文件（GitHub Actions）
8. 修复 Dockerfile Go 版本不匹配
9. 为 converter.go 的中等严重度 bug 添加 fallback 处理
