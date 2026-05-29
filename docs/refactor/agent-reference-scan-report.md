# 子 Agent 引用扫描报告

## 1. 扫描命令
本轮实际执行的核心扫描命令（节选）：

```powershell
git branch --show-current
git status -sb
git log --oneline -5

rg -n -S "code-agent|web-agent|file-agent|vision-agent|security-agent|test-agent|review-agent|ppt-agent|doc-agent|document-agent|agentName|AgentCard|A2A|a2a|sendSubscribe|\.well-known/agent\.json|/health" server
rg -n -S "code-agent|web-agent|file-agent|vision-agent|security-agent|test-agent|review-agent|ppt-agent|doc-agent|document-agent|agentName|AgentCard|A2A|a2a|sendSubscribe|\.well-known/agent\.json|/health" frontend
rg -n -S "code-agent|web-agent|file-agent|vision-agent|security-agent|test-agent|review-agent|ppt-agent|doc-agent|document-agent|agentName|AgentCard|A2A|a2a|sendSubscribe|\.well-known/agent\.json|/health" docs
rg -n -S "agent|code-agent|web-agent|file-agent|vision-agent|security-agent|test-agent|review-agent|ppt-agent|doc-agent|document-agent|a2a|health" docker-compose.yml
rg -n -S "agent|code-agent|web-agent|file-agent|vision-agent|security-agent|test-agent|review-agent|ppt-agent|doc-agent|document-agent|a2a|health" Makefile

rg -n "LoadConfig\(|NewA2AServer\(|_PORT|Run\(" agents
rg -n "AddArtifact" agents
rg -n "Multi_Agent-AgentHub/agents" agents
```

## 2. server 引用
- `server/internal/config/config.go`：
  - `Agents` 映射只配置 `code-agent`。
  - `AGENT_CODE_URL` 指向 `code-agent` 地址。
- `server/internal/orchestrator/orchestrator.go`：
  - 当前路由逻辑写死 `o.agents["code-agent"]`。
  - 注释中明确 MVP 直连 code-agent。
- `server/internal/handler/conversation.go`：
  - 创建会话默认 `agentName = "code-agent"`。
  - `/api/agents` 仅返回 `code-agent`。
- `server/internal/a2a/*`：
  - 使用官方 `a2a-go/v2` 客户端与事件类型。
  - 负责 A2A 调用与事件传输，不直接暴露前端协议。
- `server/internal/orchestrator/converter.go`：
  - 明确执行 A2A -> AG-UI 事件转换。
  - 当前 artifact flush 逻辑偏向 `code_preview` MVP 路径。

结论：`server` 运行链路对 `code-agent` 仍有强硬编码依赖。

## 3. agents 内部引用
- 19 个 `*-agent` 均依赖 `agents/adk`，统一模式为：
  - `adk.LoadConfig("config.yaml")`
  - `adk.NewA2AServer(config, taskHandler)`
  - `server.Run(":"+PORT)`
- 公共端点来自 `agents/adk/server.go`：
  - `GET /health`
  - `GET /.well-known/agent.json`
  - `POST /`
  - `POST /a2a/tasks/sendSubscribe`
- 互相依赖情况：
  - 未发现 Agent 之间直接 import 或直接调用。
  - 当前是“共享 ADK runtime + 独立 handler 模板”的松耦合结构。
- 测试结构：
  - 每个 Agent 至少有 `handler_test.go`。
  - `agents/adk` 有较完整的并发、AgentCard、A2A 兼容与安全测试。

## 4. frontend 引用
- `frontend/src/components/ConversationList.tsx`：
  - 新建会话写死 `create("code-agent")`。
- `frontend/e2e/mocks.ts`：
  - 会话、消息、agents fixture 均写死 `code-agent`。
  - SSE mock 主要围绕 `code_preview` 路径。
- `frontend/src/services/api.ts`：
  - `createConversation(agentName)` 接口是通用的，但 UI 层当前传固定值。

结论：前端仍以单 `code-agent` 体验为默认入口。

## 5. docs 引用
- 高密度引用区：
  - `docs/integration/*`（多 Agent handoff、artifact preview、multimodal）
  - `docs/contracts/*`（A2A/Artifact/AG-UI/安全与测试清单）
  - `docs/reports/*`（历史验证报告）
  - `docs/refactor/*`（迁移规划）
- 文档对多个 Agent 名称有大量示例与流程链路说明（如 `vision -> web`、`file -> document`、`security -> review`）。

结论：文档层引用广泛，后续任何删除都必须配套文档收敛，避免“代码已迁移、文档仍指旧路径”。

## 6. docker-compose / Makefile 引用
- `docker-compose.yml`：
  - 仅声明 `code-agent` 服务。
  - `gateway` 通过 `AGENT_CODE_URL=http://code-agent:8081` 调用。
- `Makefile`：
  - `dev-agent` 直接运行 `agents/code-agent`。
  - `build-check` 直接构建 `./code-agent`。

结论：本地开发与演示脚本当前围绕 `code-agent` 单点设计。

## 7. 高风险引用
- 风险 1：`server/internal/orchestrator/orchestrator.go` 写死 `code-agent`。
  - 后果：删除或迁移路径变更后，运行链路直接失败。
- 风险 2：`server/internal/handler/conversation.go` 默认与列表接口写死 `code-agent`。
  - 后果：前端会话创建与 Agent 列表不一致。
- 风险 3：`frontend/src/components/ConversationList.tsx` 写死 `code-agent`。
  - 后果：即使后端支持多 Agent，前端入口仍单点。
- 风险 4：`docker-compose.yml` / `Makefile` 写死 `agents/code-agent`。
  - 后果：迁移到 `services/agents/code-agent` 后启动脚本会断。
- 风险 5：docs 大量历史引用。
  - 后果：团队协作认知与事实源偏差，导致错误回归操作。

## 8. 建议处理方式
- 对 `server` 引用：
  - 保留：短期保留 `code-agent` 硬依赖以保证 MVP 演示。
  - 迁移：在 Phase 8.1~8.3 先新增新路径 Agent，再改路由为 capability/registry 驱动。
  - 删除前置条件：新路由回归测试 + A2A round-trip 测试通过。
- 对 `frontend` 引用：
  - 保留：短期允许默认 `code-agent`。
  - 替换：后续改为 `/api/agents` 动态列表驱动，而非硬编码。
  - 删除前置条件：UI 新建会话与历史会话展示支持多 Agent。
- 对 `agents` 内部：
  - 保留：共享 `agents/adk` 骨架稳定，先迁移一个样板再推广。
  - 合并：模板化报告 Agent 优先收口成 tools/skills。
  - 删除前置条件：替代工具已接入编排并通过契约测试。
- 对 `docs/docker-compose/Makefile`：
  - 迁移：等新路径 Agent 骨架稳定后，再同步脚本与文档。
  - 删除前置条件：演示脚本、smoke test、README 一致性校验通过。
