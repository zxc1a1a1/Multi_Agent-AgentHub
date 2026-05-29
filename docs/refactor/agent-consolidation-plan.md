# 子 Agent 能力整合计划

## 1. 整合目标
- 减少重复 Agent，优先按能力收口，而不是继续按目录扩散。
- 对齐统一声明面：`AgentCard` / `inputModes` / `outputModes` / `artifactTypes`。
- 为后续 Orchestrator 调度提供“按 capability 路由”的稳定基础，逐步摆脱对硬编码 `agentName` 与历史目录结构的依赖。
- 本文属于迁移计划，不执行删除、不移动旧代码。

## 2. 分类规则

### A. 核心保留
- 具备当前主链路价值，或是后续多 Agent 主链路中的核心入口能力。
- 短期必须保持独立服务形态，避免影响演示与回归。

### B. 合并为 tool/skill
- 与其他 Agent 高度同构，主要差异是提示词模板和 artifact 名称。
- 更适合变成可复用工具或 runtime skill，而不是长期独立进程。

### C. 暂缓迁移
- 有业务价值但边界尚未稳定，或安全/产品策略未定。
- 暂不删，先保留并观察实际调用价值。

### D. 删除候选
- 与平台层能力重叠明显，或与其他 Agent 重复度高且无硬依赖。
- 仅作为候选，必须满足删除前检查清单后才可执行。

## 3. 分类结果

| Agent | 建议分类 | 原因 |
|---|---|---|
| `code-agent` | A 核心保留 | 当前 server/frontend/docker/Makefile 都有硬依赖，且是 Phase 8 首个迁移对象。 |
| `web-agent` | A 核心保留 | `webpage` 产物主入口，后续多模态链路关键下游。 |
| `file-agent` | A 核心保留 | `file_ref/extracted_text` 到 `file_summary` 的基础入口。 |
| `document-agent` | A 核心保留 | 文档类产物核心入口，和 file/vision 链路衔接稳定。 |
| `vision-agent` | A 核心保留 | `image_ref` 多模态入口，现有设计中为上游中间产物生成器。 |
| `security-agent` | A 核心保留 | 安全审查能力独立价值高，且契约边界明确。 |
| `test-agent` | A 核心保留 | 质量分析与测试建议能力，适合保留独立入口。 |
| `review-agent` | A 核心保留 | 审查报告能力可与 test/security 形成互补闭环。 |
| `diff-agent` | B 合并为 tool/skill | 与 code/review/security 的 patch 分析能力重叠，可下沉为工具。 |
| `web-research-agent` | B 合并为 tool/skill | 本质是“已提供文本摘要”，可并入 document/review 工具集。 |
| `agent-builder-agent` | B 合并为 tool/skill | 主要产出 profile 草案，适合作为配置生成工具。 |
| `custom-agent` | C 暂缓迁移 | 与 document 输出重叠，但涉及权限/多租户策略，先不合并。 |
| `ppt-agent` | C 暂缓迁移 | Demo 价值存在，但长期定位与文档链路边界仍需验证。 |
| `artifact-agent` | D 删除候选 | 与平台 Artifact 归一化职责重叠明显。 |
| `context-agent` | D 删除候选 | 与 runtime/orchestrator 上下文管理能力重叠。 |
| `deploy-agent` | D 删除候选 | 报告型能力可并入 release/ops toolset。 |
| `qa-acceptance-agent` | D 删除候选 | 与 review/test 汇总能力重叠度高。 |
| `release-agent` | D 删除候选 | 与 version/deploy 报告能力重复。 |
| `version-agent` | D 删除候选 | 与 release/deploy 重复，适合合并后清理。 |

## 4. AgentCard 对齐要求
后续所有保留 Agent 必须显式声明并可被测试验证：
- `name`
- `description`
- `version`
- `url`
- `skills`
- `inputModes`
- `outputModes`
- `artifactTypes`（可由 `outputModes` 与 artifact mapping 统一约束）
- `streaming`
- `health`

要求：
- `GET /health`、`GET /.well-known/agent.json`、`POST /a2a/tasks/sendSubscribe` 行为一致。
- AgentCard 不得包含密钥、内部路径、system prompt。
- 能力判断优先依赖 `skills/inputModes/outputModes`，不依赖 `agentName` 猜测。

## 5. inputModes / outputModes 建议
建议在 Phase 8 迁移后统一中间产物语义集合（以现有契约为准）：
- `text`
- `code`
- `file_ref`
- `file_summary`
- `image_ref`
- `extracted_text`
- `vision_analysis`
- `security_report`
- `test_result`
- `webpage`
- `markdown`
- `artifact_ref`

补充建议：
- 所有报告类输出统一到结构化 JSON + 标准 metadata。
- 新增 mode 前先补 contract schema 与 converter 测试，避免前后端失配。

## 6. 删除前检查清单
删除任何 Agent 前必须同时满足：
- `server` 无引用。
- `frontend` 无引用。
- `docs` / `docker-compose` / `Makefile` 无关键引用。
- `tests` 无依赖。
- AgentCard registry 不再注册该 Agent。
- 已有替代 Agent/tool/skill。
- A2A contract test 通过。
- Gateway 不写死该 Agent。
- 不影响演示路径。

## 7. 建议迁移顺序
结合当前仓库状态，建议顺序如下：
1. `code-agent`
2. `web-agent`
3. `file-agent` / `document-agent`
4. `vision-agent`
5. `security-agent`
6. `test-agent` / `review-agent`
7. 其他低优先级 Agent（`ppt/custom` 暂缓；`artifact/context/release/version/deploy/qa` 先合并再评估删除）
