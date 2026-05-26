# AgentHub 子 Agent 能力矩阵

## 1. 当前 Agent 总览

| Agent | 层级 | 默认端口 | 主要职责 | 输入 | 输出 | 主要 Artifact Type | 是否执行外部操作 |
|---|---|---:|---|---|---|---|---|
| code-agent | 基础产品 Agent | 8081 | 根据用户需求生成、解释与改进代码 | text, vision_analysis | text, code | code | 否 |
| web-agent | 基础产品 Agent | 8082 | 生成自包含网页草稿与 UI 原型 | text, vision_analysis | text, webpage, code | webpage | 否 |
| document-agent | 基础产品 Agent | 8083 | 生成 Markdown 文档、README 与报告 | text, vision_analysis, file_summary | text, document | document | 否 |
| custom-agent | 基础产品 Agent | 8084 | 承载受限自定义任务并输出安全文档 | text | text, document | document | 否 |
| vision-agent | 多模态入口 Agent | 8092 | 将图片引用、截图说明和可选提取文本整理为 vision_analysis | text, image_ref, extracted_text | text, vision_analysis | vision_analysis | 否 |
| context-agent | 内部协作 Agent | 8091 | 整理上下文并输出 context bundle | text | text, context_bundle | context_bundle | 否 |
| diff-agent | 内部协作 Agent | 8093 | 生成与审查 unified diff | text | text, diff | diff | 否 |
| review-agent | 内部协作 Agent | 8094 | 进行代码与产物审查并输出评审报告 | text | text, review_report | review_report | 否 |
| test-agent | 内部协作 Agent | 8095 | 分析测试/构建日志并输出测试报告 | text, vision_analysis | text, test_report | test_report | 否（第一版仅分析日志） |
| artifact-agent | 内部协作 Agent | 8096 | 规整多 Agent 输出为统一产物清单 | text | text, artifact_manifest | artifact_manifest | 否 |
| security-agent | 内部协作 Agent | 8097 | 审查密钥泄漏、危险命令与权限风险 | text, vision_analysis, file_summary, diff | text, security_report | security_report | 否（第一版仅文本审查） |
| version-agent | 内部协作 Agent | 8099 | 生成版本快照与回滚建议 | text | text, version_snapshot | version_snapshot | 否（第一版不执行 Git 写操作） |
| qa-acceptance-agent | 内部协作 Agent | 8103 | 根据证据判断是否满足验收标准 | text | text, acceptance_report | acceptance_report | 否 |
| agent-builder-agent | 扩展能力 Agent | 8100 | 生成自定义 Agent Profile 草稿 | text | text, agent_profile | agent_profile | 否（第一版只输出草稿） |
| file-agent | 扩展能力 Agent | 8101 | 分析调用方提供的文件文本并提炼要点 | text, file_ref, extracted_text | text, file_summary | file_summary | 否（第一版不读取本地文件） |
| web-research-agent | 扩展能力 Agent | 8102 | 分析调用方提供的网页资料摘录 | text | text, web_research | web_research | 否（第一版不主动联网） |
| deploy-agent | 扩展能力 Agent | 8104 | 生成部署计划、环境检查与回滚说明 | text | text, deployment_plan | deployment_plan | 否（第一版不执行部署） |
| ppt-agent | 扩展能力 Agent | 8105 | 生成演示文稿大纲与讲解草稿 | text, vision_analysis, file_summary | text, slide_deck | slide_deck | 否（第一版不生成 pptx） |
| release-agent | 扩展能力 Agent | 8106 | 生成发布报告、变更日志与风险说明 | text | text, release_report | release_report | 否（第一版不执行发布） |

## 2. 设计边界

- 子 Agent 只负责 A2A 任务处理。
- 子 Agent 不直接生成 AG-UI 事件。
- 子 Agent 不直接写数据库。
- 子 Agent 不直接读写 `.env`。
- 子 Agent 第一版尽量不执行 shell。
- `test/deploy/release/ppt/web-research/file` 等 Agent 第一版只分析调用方提供内容或生成草稿，不执行真实外部动作。

## 3. Artifact Type 约定

- code-agent: code
- web-agent: webpage
- document-agent: document
- custom-agent: document
- vision-agent: vision_analysis
- context-agent: context_bundle
- review-agent: review_report
- test-agent: test_report
- security-agent: security_report
- diff-agent: diff
- artifact-agent: artifact_manifest
- qa-acceptance-agent: acceptance_report
- version-agent: version_snapshot
- agent-builder-agent: agent_profile
- file-agent: file_summary
- web-research-agent: web_research
- deploy-agent: deployment_plan
- ppt-agent: slide_deck
- release-agent: release_report

## 4. 后续接入建议

- 接入 Orchestrator 路由，将用户意图映射到合适的子 Agent。
- 接入 Agent Registry / AgentCard 发现机制，统一维护 Agent 可用性与能力元数据。
- 接入前端 Agent 列表展示，直接展示中文 description 与能力标签。
- 将重复的 fenced block parser 抽到 ADK，减少各 Agent 解析逻辑重复。
- 对高风险 Agent 增加权限模型和工具审批流程。
- 增加 Docker Compose 服务编排，完善本地一键启动与健康检查链路。

## 5. 多模态输入与附件处理链路

当前大部分 Agent 仍是 text-first。

多模态输入通过 `attachmentRef` / `contentRef` 进入平台。

`image_ref` 首选进入 `vision-agent`。

`file_ref` 首选进入 `file-agent`（后续可接 extractor）。

`vision_analysis` 和 `file_summary` 是连接多模态输入与现有 Agent 的中间产物。

`vision_analysis` 可被 `code-agent`、`web-agent`、`document-agent`、`ppt-agent`、`security-agent`、`test-agent` 消费。

`file_summary` 可被 `document-agent`、`ppt-agent`、`security-agent` 消费。

`vision-agent` v0.1 只处理调用方提供的 `image_ref` 元数据、用户描述和 `extractedText`。

`vision-agent` v0.1 不直接读取图片、不访问 `contentRef`、不执行 OCR；后续可在 Gateway/ADK/LLMClient 支持真实多模态模型后升级。

不是所有 Agent 都直接处理图片或文件，第一版继续采用“入口 Agent + 中间产物 + 下游消费”模式。

| 输入类型 | 首选入口 | 中间产物 | 下游 Agent | 说明 |
|---|---|---|---|---|
| UI 截图 | vision-agent v0.1 | vision_analysis | web-agent, code-agent, document-agent | 基于调用方提供的引用/描述/提取文本生成结构化视觉上下文 |
| 报错截图 | vision-agent v0.1 | vision_analysis | test-agent, review-agent, security-agent | 优先整理报错文本与界面上下文，再做诊断和风险审查 |
| 普通图片 | vision-agent v0.1 | vision_analysis | document-agent, ppt-agent, security-agent | 第一版不直接看图，作为文本化视觉摘要输入 |
| 文档/PDF/PPT 附件 | file-agent（后续可接 extractor） | file_summary | document-agent, ppt-agent, review-agent, qa-acceptance-agent | 先做结构化摘要，再做报告与验收判断 |
| 网页资料 | web-research-agent | web_research / file_summary | document-agent, ppt-agent, review-agent | 第一版仅分析调用方提供内容，不主动联网抓取 |
| 含敏感信息截图或文件 | security-agent | security_report（可叠加 vision_analysis/file_summary） | review-agent, qa-acceptance-agent, release-agent | 高风险内容优先经过安全审查与脱敏策略 |

## 6. 富媒体输出与 Artifact 预览链路

当前 Agent 输出不仅是文本，还包括 Artifact。

Artifact 是网页、文档、代码、diff、报告、slide deck、文件卡片等富媒体结果的统一载体。

`TEXT_MESSAGE_CONTENT` 只用于流式自然语言。

富媒体内容通过 `artifactRef` / `contentRef` 和 `previewSkill` 交给前端。

前端 Runtime Skill 负责渲染。

`slide_deck` 是 PPT 草稿，不等于真实 pptx 文件。

`deployment_plan` 是部署计划，不等于真实部署结果。

`release_report` 是发布报告，不等于真实发布动作。

`vision_analysis` 是中间富媒体 Artifact，可被 `web-agent`、`code-agent`、`ppt-agent`、`security-agent`、`test-agent`、`document-agent` 消费。

| Artifact Type | 产生 Agent | 内容性质 | 推荐预览组件 | 是否真实文件 | 注意事项 |
|---|---|---|---|---|---|
| code | code-agent | 代码片段/代码草稿 | code_preview | 否（默认） | 仅展示，不执行 |
| webpage | web-agent | HTML 页面草稿 | web_preview | 否（默认） | 必须 sandbox 预览 |
| document | document-agent/custom-agent | Markdown 文档草稿 | markdown_render | 否（默认） | 渲染需防 XSS |
| diff | diff-agent | unified diff 文本 | diff_preview | 否（默认） | 第一版不应用 patch |
| context_bundle | context-agent | JSON 上下文包 | context_card/json_viewer | 否（默认） | 供下游 Agent 消费 |
| review_report | review-agent | JSON 审查报告 | report_card | 否（默认） | 含风险与改进建议 |
| test_report | test-agent | JSON 测试报告 | report_card | 否（默认） | 第一版只分析日志 |
| security_report | security-agent | JSON 安全报告 | security_report_card | 否（默认） | 高风险内容需脱敏 |
| artifact_manifest | artifact-agent | JSON 产物清单 | artifact_manifest_view | 否（默认） | 统一管理多 Agent 结果 |
| acceptance_report | qa-acceptance-agent | JSON 验收报告 | acceptance_card | 否（默认） | 需标注人工确认项 |
| version_snapshot | version-agent | JSON 版本快照 | version_card | 否（默认） | 第一版不执行 Git 写操作 |
| agent_profile | agent-builder-agent | YAML Profile 草稿 | yaml_viewer | 否（默认） | 不直接创建目录或文件 |
| file_summary | file-agent | JSON 文件摘要 | file_summary_card | 否（默认） | 第一版不读取本地文件 |
| web_research | web-research-agent | JSON 网页资料摘要 | research_card | 否（默认） | 第一版不主动联网 |
| deployment_plan | deploy-agent | JSON 部署计划 | deployment_plan_card | 否（默认） | 不代表部署已执行 |
| slide_deck | ppt-agent | JSON 演示文稿草稿 | slide_deck_preview | 否（默认） | 不代表 pptx 已生成 |
| release_report | release-agent | JSON 发布报告 | release_report_card | 否（默认） | 不代表发布已执行 |
| vision_analysis | vision-agent v0.1 | JSON 视觉分析中间产物 | vision_analysis_card | 否（默认） | 第一版基于调用方提供的引用/描述/提取文本推断 |

## 7. 对接与测试交付说明

- 本目录提供子 Agent 能力池。
- 同事 API 可以直接调用 A2A 子 Agent。
- 所有 Agent 都需要做 `health` / `AgentCard` smoke。
- 代表性 Agent 需要做 artifact 输出深测。
- 多模态入口优先调用 `vision-agent` 和 `file-agent`。
- 复杂 Orchestrator 编排后续再做。
- 当前主要用于验证 Agent 可调用、多模态通道和 Artifact 预览。
