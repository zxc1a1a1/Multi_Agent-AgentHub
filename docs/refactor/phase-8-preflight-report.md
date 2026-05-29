# Phase 8 Preflight 完成报告

## 1. 本轮目标
本轮仅完成“迁移前置验收”：
- 子 Agent 能力池盘点；
- 子 Agent 引用扫描；
- 能力整合与分类计划；
- Phase 8 首个 code-agent 迁移最小范围建议。

本轮未执行：
- 不修改旧业务代码；
- 不删除任何 Agent；
- 不移动 `agents/` 旧目录代码；
- 不修改 `docker-compose.yml`、`Makefile`、根 `go.mod/go.sum`；
- 不提交代码。

## 2. 基线确认
- 当前分支：`g`
- 本地最近提交：`f05ec80 feat(refactor): 完成模块解耦主干与独立 Gateway 基础设施`
- 与 `origin/g` 同步状态：
  - `git status -sb` 显示 `## g...origin/g`
  - 本轮尝试 `git fetch origin` 受本机权限/超时影响未完成；
  - 按你确认“仓库已是最新”继续执行，未执行 `pull`。
- `git status --short`：
  - `?? .claude/settings.local.json`（本地未跟踪配置）
  - 未发现本轮前置阶段的额外 tracked 脏改。

## 3. 验证结果
- `pkg/adk`
  - `go test ./...`：通过
  - `go build ./...`：通过
- `pkg/runtime`
  - `go test ./...`：通过
  - `go build ./...`：通过
- `services/gateway`
  - `go test ./...`：通过
  - `go build ./...`：通过
- `agents`
  - `go test ./...`：通过（19 个 `*-agent` + `agents/adk`）
- `server`
  - `go test ./...`：通过

当前阻塞：无。

## 4. 子 Agent 能力池结论
- 扫描识别到 19 个旧 `agents/*-agent`：
  - `agent-builder-agent`、`artifact-agent`、`code-agent`、`context-agent`、`custom-agent`、`deploy-agent`、`diff-agent`、`document-agent`、`file-agent`、`ppt-agent`、`qa-acceptance-agent`、`release-agent`、`review-agent`、`security-agent`、`test-agent`、`version-agent`、`vision-agent`、`web-agent`、`web-research-agent`。
- 结构特征：
  - 全部复用 `agents/adk`，统一暴露 `/health`、`/.well-known/agent.json`、`/a2a/tasks/sendSubscribe`。
  - 大量 Agent handler 为“同构模板”（LLM 流式 + fenced block 解析 + `ctx.AddArtifact`）。
- 核心能力：
  - `code/web/file/document/vision/security/test/review` 可构成首批保留核心池。
- 重复能力：
  - 报告型 Agent（`qa/release/version/deploy`）与 `review/test/security` 有高重叠。
  - `artifact/context/diff/web-research/agent-builder` 更适合作为 tool/skill。
- 暂缓删除：
  - 本轮不删除任何 Agent，先以候选方式标注并绑定前置验证条件。

## 5. 建议保留 / 合并 / 暂缓 / 删除候选

### 保留
- `code-agent`
- `web-agent`
- `file-agent`
- `document-agent`
- `vision-agent`
- `security-agent`
- `test-agent`
- `review-agent`

### 合并为 tool/skill
- `diff-agent`
- `web-research-agent`
- `agent-builder-agent`

### 暂缓
- `custom-agent`
- `ppt-agent`

### 删除候选（仅候选）
- `artifact-agent`
- `context-agent`
- `deploy-agent`
- `qa-acceptance-agent`
- `release-agent`
- `version-agent`

## 6. Phase 8 正式迁移建议
首个 `code-agent` 迁移建议最小范围：
- 新增 `services/agents/code-agent`
- 基于 `pkg/adk` 实现最小 Agent
- 通过 `pkg/adk/a2a` 暴露 `health` 与 `AgentCard`
- 使用 `httptest` 做 A2A round-trip
- 暂不接真实 LLM
- 暂不删旧 `agents` 中的 `code-agent`
- 暂不改 `frontend`
- 暂不改 `docker-compose`

## 7. 风险与注意事项
- 不应直接删除旧 Agent。
- 不应在 Gateway 写死 Agent 路由（当前仅作兼容保留）。
- 不应绕过 AgentCard/Capability 做能力判定。
- 不应让 Orchestrator 继续依赖旧目录结构作为长期方案。
- 删除前必须有替代能力与 contract test 证据。

## 8. 下一步建议
下一步可以进入 Phase 8.1~8.3：新增 services/agents/code-agent 最小迁移骨架、A2A Server 与 round-trip 集成测试，但不删除旧 code-agent。
