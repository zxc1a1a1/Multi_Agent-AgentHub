# 子 Agent 能力池盘点

## 1. 盘点范围
- 扫描目录：`agents/`、`server/`、`frontend/`、`docs/`、`pkg/`、`services/`。
- 扫描单文件：`docker-compose.yml`、`Makefile`、`go.work`。
- 扫描类型：`*.go`、`*.md`、`*.json`、`*.yaml`、`*.yml`、`*.ts`、`*.tsx`、`*.js`、`*.jsx`。
- 关注字段：`agentName`、`AgentCard`、`/health`、`/a2a/tasks/sendSubscribe`、`inputModes`、`outputModes`、`artifact type`、skills/capabilities、测试与引用关系。
- 说明：本轮仅做只读盘点与报告输出，不修改 `server/frontend/agents` 代码，不删除任何 Agent。

## 2. 子 Agent 总览表

| agentName | 当前路径 | 当前职责 | 当前入口形式 | 是否有 health | 是否有 AgentCard | 是否有 A2A | inputModes | outputModes | artifactTypes | 主要 tools/skills/capabilities | 测试覆盖情况 | 是否被 server 引用 | 是否被 frontend 引用 | 是否被 docker-compose/Makefile 引用 | 与其他 Agent 的能力重叠 | 初步建议 | 备注 |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| `agent-builder-agent` | `agents/agent-builder-agent` | 生成 Agent Profile 草案 | `main.go` + `adk.LoadConfig` + `adk.NewA2AServer` | 是 | 是 | 是 | `text` | `text,agent_profile` | `agent_profile` | `agent_profile_design`,`agent_scaffold_plan` | `handler_test.go` + `go test ./...` 通过 | 否 | 否 | 否 | 与 `document-agent`、`custom-agent` 的“结构化草案输出”能力重叠 | 合并 | 更适合作为“Agent 设计 tool/skill” |
| `artifact-agent` | `agents/artifact-agent` | 产物清单/分类草案 | 同上 | 是 | 是 | 是 | `text` | `text,artifact_manifest` | `artifact_manifest` | `artifact_manifest`,`artifact_classification` | 同上 | 否 | 否 | 否 | 与 Orchestrator/Artifact Registry 目标能力重叠 | 合并 | 后续更适合沉入 `pkg/runtime`/Orchestrator 侧能力 |
| `code-agent` | `agents/code-agent` | 代码生成/解释/重构，提取 code artifact | 同上 | 是 | 是 | 是 | `text,vision_analysis` | `text,code` | `code` | `code_generation`,`code_review`,`code_refactoring` | 同上 | 是 | 是 | 是 | 与 `review-agent` 在 code review 局部重叠 | 保留 | 当前主链路唯一硬依赖 Agent |
| `context-agent` | `agents/context-agent` | 会话上下文压缩与结构化 context bundle | 同上 | 是 | 是 | 是 | `text` | `text,context_bundle` | `context_bundle` | `context_compression`,`context_selection` | 同上 | 否 | 否 | 否 | 与 Orchestrator 上下文裁剪/摘要能力重叠 | 合并 | 更适合作为编排前置工具 |
| `custom-agent` | `agents/custom-agent` | 受限自定义任务输出（Markdown） | 同上 | 是 | 是 | 是 | `text` | `text,document` | `document` | `custom_agent_execution`,`restricted_profile_task` | 同上 | 否 | 否 | 否 | 与 `document-agent` 高重叠 | 暂缓 | 需先定义多租户安全边界再迁移 |
| `deploy-agent` | `agents/deploy-agent` | 部署计划/回滚说明草案 | 同上 | 是 | 是 | 是 | `text` | `text,deployment_plan` | `deployment_plan` | `deployment_plan`,`environment_checklist`,`rollback_notes` | 同上 | 否 | 否 | 否 | 与 `release-agent`、`qa-acceptance-agent` 报告类能力重叠 | 合并 | 适合并入 release/ops 工具链 |
| `diff-agent` | `agents/diff-agent` | 生成/解释 diff 草案（不执行 apply） | 同上 | 是 | 是 | 是 | `text` | `text,diff` | `diff` | `diff_generation`,`diff_explanation`,`patch_risk_summary` | 同上 | 否 | 否 | 否 | 与 `code-agent` patch 输出、`review/security` 风险分析重叠 | 合并 | 适合作为 code/review 的专用工具 |
| `document-agent` | `agents/document-agent` | Markdown 文档/报告生成 | 同上 | 是 | 是 | 是 | `text,vision_analysis,file_summary` | `text,document` | `document` | `markdown_document`,`technical_report`,`prd_writer`,`handoff_report` | 同上 | 否 | 否 | 否 | 与 `custom-agent`、`ppt-agent` 部分重叠 | 保留 | 文档类入口能力明确 |
| `file-agent` | `agents/file-agent` | 对调用方提供文本做文件摘要 | 同上 | 是 | 是 | 是 | `text,file_ref,extracted_text` | `text,file_summary` | `file_summary` | `file_summary`,`requirement_extraction` | 同上 | 否 | 否 | 否 | 与 `document-agent`、`review-agent` 的摘要链路重叠但定位更底层 | 保留 | 作为 file/document 链路入口价值高 |
| `ppt-agent` | `agents/ppt-agent` | 生成 slide deck JSON 草稿 | 同上 | 是 | 是 | 是 | `text,vision_analysis,file_summary` | `text,slide_deck` | `slide_deck` | `slide_deck_outline`,`presentation_script`,`demo_storyboard` | 同上 | 否 | 否 | 否 | 与 `document-agent` 输出层有重叠 | 暂缓 | Demo 导向能力，先不删除 |
| `qa-acceptance-agent` | `agents/qa-acceptance-agent` | 验收结论与 checklist 草案 | 同上 | 是 | 是 | 是 | `text` | `text,acceptance_report` | `acceptance_report` | `acceptance_check`,`demo_checklist` | 同上 | 否 | 否 | 否 | 与 `test-agent`、`review-agent` 结果汇总重叠 | 合并 | 可收敛为 review/test 流程内模板 |
| `release-agent` | `agents/release-agent` | release 报告/changelog 草案 | 同上 | 是 | 是 | 是 | `text` | `text,release_report` | `release_report` | `release_report`,`changelog_generation`,`release_risk_summary` | 同上 | 否 | 否 | 否 | 与 `version-agent`、`deploy-agent` 重叠 | 合并 | 可并入 release toolset |
| `review-agent` | `agents/review-agent` | 评审结论、问题分级、整改建议 | 同上 | 是 | 是 | 是 | `text` | `text,review_report` | `review_report` | `code_review`,`requirement_coverage_review`,`risk_review` | 同上 | 否 | 否 | 否 | 与 `code-agent` 的 code review 局部重叠 | 保留 | 适合作为独立审查入口 |
| `security-agent` | `agents/security-agent` | 安全扫描与风险报告（含规则预扫描） | 同上 | 是 | 是 | 是 | `text,vision_analysis,file_summary,diff` | `text,security_report` | `security_report` | `secret_scan`,`command_risk_review`,`permission_review` | 同上 | 否 | 否 | 否 | 与 `review-agent` 风险识别局部重叠 | 保留 | 与安全边界契约直接相关 |
| `test-agent` | `agents/test-agent` | 测试日志分析与测试计划建议 | 同上 | 是 | 是 | 是 | `text,vision_analysis` | `text,test_report` | `test_report` | `test_log_analysis`,`build_failure_analysis`,`test_plan_generation` | 同上 | 否 | 否 | 否 | 与 `review-agent` 评审报告重叠 | 保留 | 质量门禁相关核心能力 |
| `version-agent` | `agents/version-agent` | 版本快照与回滚说明草案 | 同上 | 是 | 是 | 是 | `text` | `text,version_snapshot` | `version_snapshot` | `version_summary`,`rollback_plan`,`patch_version_linking` | 同上 | 否 | 否 | 否 | 与 `release-agent`、`deploy-agent` 重叠 | 合并 | 更适合收敛到 release 流程 |
| `vision-agent` | `agents/vision-agent` | 基于 image_ref/extracted_text 的视觉语义分析（不直接看图、不 OCR） | 同上 | 是 | 是 | 是 | `text,image_ref,extracted_text` | `text,vision_analysis` | `vision_analysis` | `image_ref_analysis`,`screenshot_understanding`,`ui_image_analysis`,`error_screenshot_analysis`,`visual_context_summary` | 同上 | 否 | 否 | 否 | 与 `web-agent`、`test-agent` 上游能力耦合 | 保留 | 多模态链路核心上游 |
| `web-agent` | `agents/web-agent` | 网页/UI 生成（HTML/CSS/JS） | 同上 | 是 | 是 | 是 | `text,vision_analysis` | `text,webpage,code` | `webpage` | `web_generation`,`ui_design`,`responsive_layout` | 同上 | 否 | 否 | 否 | 与 `code-agent` 在“产出代码”层有交叠 | 保留 | 网页产物链路核心 |
| `web-research-agent` | `agents/web-research-agent` | 对已提供网页摘录做研究摘要（不主动联网） | 同上 | 是 | 是 | 是 | `text` | `text,web_research` | `web_research` | `web_summary`,`citation_summary` | 同上 | 否 | 否 | 否 | 与 `document-agent` 摘要能力重叠 | 合并 | 更适合作为 research tool/skill |

## 3. 核心能力分组

### code_generation / code_review / diff / test
- `code-agent`：代码生成、代码产物输出。
- `review-agent`：需求覆盖与风险评审报告。
- `diff-agent`：diff 草案与变更解释。
- `test-agent`：测试日志分析、测试计划建议。
- `qa-acceptance-agent`：验收报告（更偏流程汇总）。

### web_generation / ui_generation / webpage artifact
- `web-agent`：`webpage` 主产物。
- `vision-agent`：提供 `vision_analysis` 作为 UI 还原上游输入。

### file_summary / file_ref / document
- `file-agent`：`file_ref/extracted_text` 到 `file_summary`。
- `document-agent`：文档输出 `document`。
- `custom-agent`：受限 `document` 输出（与 document 能力重叠）。

### vision_analysis / image_ref / extracted_text
- `vision-agent`：多模态语义中间产物 `vision_analysis`。

### security_review
- `security-agent`：密钥/命令风险识别与安全报告。

### ppt / document generation
- `ppt-agent`：`slide_deck` 草案。
- `document-agent`：Markdown 文档。

### deployment / build / release
- `deploy-agent`：部署计划。
- `release-agent`：发布报告。
- `version-agent`：版本快照与回滚计划。

### unknown / legacy / demo-oriented
- `artifact-agent`：manifest 归一化草案（与平台 artifact 契约边界重叠）。
- `context-agent`：context bundle（与 runtime/orchestrator 可复用能力重叠）。
- `agent-builder-agent`：Agent profile 草案。

## 4. 能力重复与合并机会
- `review-agent`、`test-agent`、`qa-acceptance-agent`、`release-agent`、`version-agent` 都是“文本输入 -> JSON 报告 artifact”模板，处理链路高度同构，可收敛为统一 report toolset + 不同模板。
- `diff-agent` 与 `code-agent` 在 patch/diff 生成能力上重叠，且 `security-agent`/`review-agent` 已能消费 diff，适合改为 `code-agent` 可调用工具而非独立进程。
- `artifact-agent` 与长期目标中的 Artifact Registry/Normalizer 职责重叠，更适合并入平台层（`pkg/runtime` 或 Orchestrator 归一化流程）。
- `context-agent` 与 runtime 层的 pruning/session/context 管理能力有功能交集，建议并入编排前置工具。
- `document-agent` 与 `custom-agent` 在 `document` 输出高度重叠，后者建议作为受限 profile/runtime policy，而不是独立长期核心 Agent。

## 5. 第一批建议保留 Agent
- `code-agent`：当前 server/frontend/docker/Makefile 主链路硬依赖，且是首个迁移目标。
- `web-agent`：网页产物核心能力，后续 Phase 8.2/8.3 可直接跟进 code-agent 迁移模板。
- `file-agent`：文件类输入入口，承接 `file_ref/extracted_text`。
- `document-agent`：文档类核心输出入口（与 file/vision 下游衔接清晰）。
- `vision-agent`：多模态入口，支撑 image_ref 链路。
- `security-agent`：安全审查专用能力，契约价值高。
- `test-agent`：质量分析能力，测试与验收闭环关键。
- `review-agent`：评审能力与 test/security 互补。

## 6. 第一批删除候选（仅候选，不执行删除）

### `artifact-agent`（删除候选）
- 可能删除原因：职责与平台 Artifact 归一化目标重叠。
- 当前引用情况：代码运行链路无强依赖（主要 docs 引用）。
- 替代能力：`pkg/runtime` + Orchestrator Artifact 归一化流程。
- 删除前验证：契约测试覆盖 artifact 映射；docs/演示链路迁移完成。

### `context-agent`（删除候选）
- 可能删除原因：与 runtime context/pruning 能力重叠。
- 当前引用情况：运行链路无强依赖（主要 docs 引用）。
- 替代能力：runtime plugin / orchestrator context preprocessing。
- 删除前验证：上下文裁剪与回放测试不回退。

### `version-agent` / `release-agent` / `qa-acceptance-agent`（删除候选）
- 可能删除原因：报告类模板重复，适合合并成统一工具集。
- 当前引用情况：运行链路无硬编码依赖（主要 docs 引用）。
- 替代能力：review/test/security + release toolset 模板化输出。
- 删除前验证：保留同等报告字段，关键 demo 路径与契约测试通过。

### `deploy-agent`（删除候选）
- 可能删除原因：与 release/version 报告能力重复且不执行真实部署。
- 当前引用情况：运行链路无强依赖（主要 docs 引用）。
- 替代能力：release tool/skill 中的 deployment section。
- 删除前验证：部署计划输出字段迁移完成且安全审查仍通过。
