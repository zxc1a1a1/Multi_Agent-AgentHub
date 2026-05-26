# AgentHub 子 Agent 对接与测试交付说明

## 1. 当前交付范围

当前交付内容是“子 Agent 能力池 + 对接规范 + smoke 测试建议”，不包含以下实现：

- 前端上传实现
- Gateway 新接口实现
- 复杂 Orchestrator 自动编排
- 真实 OCR
- 真实 PPTX 生成
- 真实部署/发布动作

当前 19 个 Agent：

基础产品 Agent：

- code-agent
- web-agent
- document-agent
- custom-agent

内部协作 Agent：

- context-agent
- vision-agent
- review-agent
- test-agent
- security-agent
- diff-agent
- artifact-agent
- qa-acceptance-agent
- version-agent

扩展能力 Agent：

- agent-builder-agent
- file-agent
- web-research-agent
- deploy-agent
- ppt-agent
- release-agent

## 2. 测试职责划分

开发侧已完成：

- `go test ./...`
- `go build` 各 Agent
- `config.yaml` / AgentCard 初步对齐
- artifact type 约定
- 安全边界约束
- 多模态输入/输出契约

测试/前端联调侧需要验证：

- 每个 Agent `health` 是否可访问
- AgentCard 是否能被读取
- 同事 API 是否能调用指定 Agent
- 响应中是否有 text 和 artifact
- artifact type 是否和文档一致
- 前端是否能根据 artifact type 显示预览
- 多模态 `image_ref` 是否能走到 `vision-agent`
- `vision_analysis` 是否能作为下游 Agent 输入

## 3. 全量 Agent 测试矩阵

| Agent | 默认端口 | 输入模式 | 输出模式 | 主要 artifact type | 建议测试方式 | 前端预览组件 | 是否真实执行外部动作 |
|---|---:|---|---|---|---|---|---|
| code-agent | 8081 | text, vision_analysis | text, code | code | 文本生成代码 + 校验 code artifact | code_preview | 否 |
| web-agent | 8082 | text, vision_analysis | text, webpage, code | webpage | 文本/vision_analysis 生成网页 + 校验 webpage artifact | web_preview | 否 |
| document-agent | 8083 | text, vision_analysis, file_summary | text, document | document | 文本/file_summary 生成文档 + 校验 document artifact | markdown_render | 否 |
| custom-agent | 8084 | text | text, document | document | 自定义任务文本输入 + 校验 document artifact | markdown_render | 否 |
| context-agent | 8091 | text | text, context_bundle | context_bundle | 上下文压缩请求 + 校验 JSON artifact | context_card / json_viewer | 否 |
| vision-agent | 8092 | text, image_ref, extracted_text | text, vision_analysis | vision_analysis | 使用 image_ref 元数据 + extractedText + context 做分析 | vision_analysis_card | 否（v0.1 不真实看图、不 OCR） |
| review-agent | 8094 | text | text, review_report | review_report | 输入需求/改动摘要 + 校验审查报告 | report_card | 否 |
| test-agent | 8095 | text, vision_analysis | text, test_report | test_report | 输入日志/报错摘要 + 校验测试报告 | report_card | 否（仅分析日志） |
| security-agent | 8097 | text, vision_analysis, file_summary, diff | text, security_report | security_report | 输入文本/diff/中间产物 + 校验安全报告 | security_report_card | 否（仅审查） |
| diff-agent | 8093 | text | text, diff | diff | 输入改动需求 + 校验 diff artifact | diff_preview | 否（不应用 patch） |
| artifact-agent | 8096 | text | text, artifact_manifest | artifact_manifest | 输入多产物摘要 + 校验 manifest | artifact_manifest_view | 否 |
| qa-acceptance-agent | 8103 | text | text, acceptance_report | acceptance_report | 输入验收证据摘要 + 校验验收报告 | acceptance_card | 否 |
| version-agent | 8099 | text | text, version_snapshot | version_snapshot | 输入版本信息摘要 + 校验版本快照 | version_card | 否（不执行 Git 写操作） |
| agent-builder-agent | 8100 | text | text, agent_profile | agent_profile | 输入 Agent 需求 + 校验 profile 草稿 | yaml_viewer | 否 |
| file-agent | 8101 | text, file_ref, extracted_text | text, file_summary | file_summary | 输入 file_ref 元数据/提取文本 + 校验摘要 | file_summary_card | 否（第一版不读取本地文件） |
| web-research-agent | 8102 | text | text, web_research | web_research | 输入 URL/标题/摘录文本 + 校验研究摘要 | research_card | 否（不主动联网） |
| deploy-agent | 8104 | text | text, deployment_plan | deployment_plan | 输入部署背景 + 校验部署计划 | deployment_plan_card | 否（只生成计划，不真实部署） |
| ppt-agent | 8105 | text, vision_analysis, file_summary | text, slide_deck | slide_deck | 输入资料摘要 + 校验 slide_deck 草稿 | slide_deck_preview | 否（只生成草稿，不真实生成 pptx） |
| release-agent | 8106 | text | text, release_report | release_report | 输入发布证据摘要 + 校验发布报告 | release_report_card | 否（只生成报告，不真实发布） |

## 4. 推荐最小验收路径

建议采用“全量 smoke + 代表链路深测”。

全量 smoke：

- health
- AgentCard
- 启动端口
- inputModes/outputModes

代表链路深测：

1. 代码生成：`text -> code-agent -> code artifact -> code_preview`
2. 网页生成：`text -> web-agent -> webpage artifact -> web_preview`
3. 文档生成：`text -> document-agent -> document artifact -> markdown_render`
4. 多模态截图理解：`image_ref + extractedText -> vision-agent -> vision_analysis -> vision_analysis_card`
5. 截图转网页：`image_ref -> vision-agent -> vision_analysis -> web-agent -> webpage`
6. 资料生成 PPT 草稿：`file_ref/extracted_text -> file-agent -> file_summary -> ppt-agent -> slide_deck`
7. 安全检查：`text/diff/vision_analysis/file_summary -> security-agent -> security_report`
8. Diff 预览：`text -> diff-agent -> diff artifact -> diff_preview`
9. 验收判断：`需求 + artifact/test/review 摘要 -> qa-acceptance-agent -> acceptance_report`

## 5. 启动方式建议

PowerShell 示例：

```powershell
cd "D:\研究生\竞赛\Multi_Agent-AgentHub\agents\vision-agent"
go run .
```

说明：

- 每个 Agent 独立启动。
- 默认端口见测试矩阵。
- 可用对应环境变量覆盖端口（如 `VISION_AGENT_PORT`）。
- 不要提交构建出来的 exe。
- 测试后清理 exe。

## 6. Health 与 AgentCard 检查

所有 Agent 都应支持：

- `GET /health`
- `GET /.well-known/agent.json`

PowerShell 示例：

```powershell
Invoke-RestMethod -Method GET "http://localhost:8092/health"
Invoke-RestMethod -Method GET "http://localhost:8092/.well-known/agent.json"
```

AgentCard 建议至少检查：

- `name`
- `description`
- `skills`
- `inputModes`
- `outputModes`
- `streaming`
- `url` / `supportedInterfaces`

## 7. Artifact 验收规则

- TEXT 只用于自然语言。
- artifact 用于代码、网页、文档、diff、报告、slide_deck、vision_analysis 等富媒体结果。
- 每个 artifact 至少检查 `type/title/content/metadata`。
- 大文件未来使用 `contentRef`，不直接塞 base64。
- 前端根据 artifact type 或 previewSkill 显示组件。

## 8. 安全验收规则

- 不允许 `.env` 被提交。
- 不允许真实 API Key、Token、密码、私钥出现在输出或文档。
- 不允许 Agent 执行未声明的 shell。
- 不允许 `file-agent` / `vision-agent` 读取本地路径。
- 不允许 `web-research-agent` 主动联网。
- 不允许 `deploy-agent` 真实部署。
- 不允许 `release-agent` 真实发布。
- 不允许 `ppt-agent` 声称生成真实 pptx。
- HTML 预览必须 sandbox。
- Markdown 渲染要防 XSS。
