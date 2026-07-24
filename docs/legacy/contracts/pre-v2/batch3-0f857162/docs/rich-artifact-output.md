# AgentHub 富媒体输出与 Artifact 预览契约

## 1. 目标与边界

本契约定义子 Agent 如何输出富媒体 Artifact，以及 Gateway、AG-UI、Frontend Runtime Skills 如何识别、传输、预览和下载这些产物。

富媒体输出包括代码、网页、Markdown 文档、diff、slide deck、图片预览、文件卡片、报告、部署计划、发布报告等。

子 Agent 不应把大文件、二进制、图片、PPTX、PDF 直接塞进文本流。

大产物应使用 `artifactRef` 或 `contentRef` 传递引用。

当前多数 Agent 输出的是结构化草稿或可预览内容，不代表真实文件已生成。

真实 pptx/pdf/image/file 生成属于后续文件生成与对象存储能力，不在本轮实现。

## 2. 输出链路

推荐输出链路：

子 Agent
-> A2A Artifact
-> Gateway Artifact Converter
-> AG-UI Artifact Event 或 Tool Result Event
-> Frontend Runtime Skill
-> 用户看到预览卡片 / 沙箱网页 / Markdown 渲染 / diff 视图 / slide deck 预览 / 文件卡片

约束说明：

- `TEXT_MESSAGE_CONTENT` 只承载自然语言流式文本。
- Artifact 内容或引用不应混入普通文本流。
- Gateway 负责将 A2A Artifact 转成前端可消费的 `artifactRef`、`previewSkill`、`metadata`。
- Frontend Runtime Skill 负责最终预览，不执行不可信脚本。

## 3. Artifact Envelope 约定

统一输出包络建议字段：

- `artifactId`
- `type`
- `title`
- `content`
- `contentRef`
- `mimeType`
- `sizeBytes`
- `checksum`
- `previewSkill`
- `downloadable`
- `generatedFile`
- `metadata`
- `provenance`
- `createdBy`
- `createdAt`
- `sourceTaskId`
- `sourceAgent`
- `lifecycle`

约束说明：

- 小型结构化内容可以放 `content`。
- 大文件或二进制必须用 `contentRef`。
- `content` 和 `contentRef` 至少有一个。
- `generatedFile=false` 表示只是结构化草稿，不是真实下载文件。
- `downloadable=true` 只表示前端可提供下载动作，不表示文件已经真实存在。
- `lifecycle` 可包含 `draft`、`previewable`、`generated`、`archived`、`expired`。

## 4. 当前 Agent 输出类型映射

| Agent | Artifact Type | 输出性质 | 是否真实文件 | 推荐 Frontend Runtime Skill | 说明 |
|---|---|---|---|---|---|
| code-agent | code | 代码片段或代码产物 | 非真实文件默认 | code_preview | 以文本代码为主，可后续导出真实文件 |
| web-agent | webpage | HTML 页面预览 | 非真实文件默认 | web_preview | 前端预览必须使用沙箱 |
| document-agent | document | Markdown 文档 | 非真实文件默认 | markdown_render | 默认展示结构化草稿 |
| custom-agent | document | Markdown 文档 | 非真实文件默认 | markdown_render | 受安全边界约束的文档输出 |
| context-agent | context_bundle | JSON 上下文包 | 非真实文件默认 | json_viewer / context_card | 供下游 Agent 消费的压缩上下文 |
| review-agent | review_report | JSON 审查报告 | 非真实文件默认 | report_card | 包含问题、风险、建议 |
| test-agent | test_report | JSON 测试报告 | 非真实文件默认 | report_card | 第一版仅分析调用方日志 |
| security-agent | security_report | JSON 安全报告 | 非真实文件默认 | security_report_card | 包含敏感风险与策略建议 |
| diff-agent | diff | unified diff | 非真实文件默认 | diff_preview | 第一版不应用 patch |
| artifact-agent | artifact_manifest | 产物清单 | 非真实文件默认 | artifact_manifest_view | 汇总多 Agent 输出 |
| qa-acceptance-agent | acceptance_report | 验收报告 | 非真实文件默认 | acceptance_card | 给出通过/失败/待确认项 |
| version-agent | version_snapshot | 版本快照 | 非真实文件默认 | version_card | 第一版不执行 Git 写操作 |
| agent-builder-agent | agent_profile | YAML 配置草稿 | 非真实文件默认 | yaml_viewer | 输出 profile 草稿，不创建文件 |
| file-agent | file_summary | 文件摘要 | 非真实文件默认 | file_summary_card | 第一版只分析调用方提供内容 |
| web-research-agent | web_research | 网页资料摘要 | 非真实文件默认 | research_card | 第一版不主动联网 |
| deploy-agent | deployment_plan | 部署计划 | 非真实文件默认 | deployment_plan_card | 第一版只输出计划 |
| ppt-agent | slide_deck | slide deck JSON 草稿 | 非真实 PPTX | slide_deck_preview | 仅草稿，不表示已生成真实文件 |
| release-agent | release_report | 发布报告 | 非真实发布包 | release_report_card | 第一版只输出报告 |

## 5. 富媒体类型定义

| 类型 | 推荐 content 格式 | 推荐 mimeType | 推荐 previewSkill | 是否允许 contentRef | 安全注意事项 |
|---|---|---|---|---|---|
| code | 纯文本代码 | text/plain | code_preview | 是 | 不执行代码，只展示文本 |
| webpage | HTML/CSS/JS 草稿 | text/html | web_preview | 是 | 必须 iframe sandbox |
| document | Markdown 文本 | text/markdown | markdown_render | 是 | 渲染前消毒，防 XSS |
| diff | unified diff 文本 | text/x-diff | diff_preview | 是 | 防止误触发自动应用 patch |
| slide_deck | JSON 草稿 | application/json | slide_deck_preview | 是 | 仅草稿，不等于 pptx_file |
| image_preview | 图片引用或结构化预览信息 | image/* | image_preview | 是 | 使用 contentRef，不内嵌 base64 |
| file_card | 文件元数据 | application/json | file_card | 是 | 文件名与来源需脱敏校验 |
| file_download | 下载描述与授权信息 | application/json | file_download | 是 | 必须权限校验和过期控制 |
| vision_analysis | JSON 分析结果 | application/json | vision_analysis_card | 是 | 检测文本需做敏感扫描 |
| file_summary | JSON 摘要结果 | application/json | file_summary_card | 是 | 不应伪造“已读取本地文件” |
| artifact_manifest | JSON 清单 | application/json | artifact_manifest_view | 是 | 记录来源与生命周期 |
| deployment_plan | JSON 计划 | application/json | deployment_plan_card | 是 | 不得伪造已执行部署 |
| deployment_status | JSON 状态 | application/json | deployment_plan_card | 是 | 仅在有可信执行链路时使用 |
| release_report | JSON 报告 | application/json | release_report_card | 是 | 不得伪造已发布结果 |
| report | JSON/Markdown 报告 | application/json 或 text/markdown | report_card | 是 | 风险与证据分离展示 |
| chart | JSON 图表数据 | application/json | chart_view | 是 | 数据来源需可追溯 |
| json | JSON 文本 | application/json | json_viewer | 是 | Schema 校验与字段白名单 |
| yaml | YAML 文本 | application/yaml | yaml_viewer | 是 | 防止注入不可信执行语义 |

重点约束：

- `slide_deck` 是结构化演示文稿草稿，不等于 `pptx_file`。
- `pptx_file` 是未来真实文件产物，应使用 `contentRef`。
- `document` 默认 Markdown，不等于 `pdf_file`。
- `pdf_file` 是未来真实文件产物，应使用 `contentRef`。
- `webpage` 预览必须沙箱。
- `image_preview` 应使用 `contentRef`，不内嵌 base64。
- `file_download` 必须有权限和过期控制。

## 6. AG-UI 输出事件建议

项目级事件语义建议：

- `ARTIFACT_CREATED`
- `ARTIFACT_UPDATED`
- `ARTIFACT_PREVIEW_READY`
- `ARTIFACT_FAILED`
- `ARTIFACT_EXPIRED`

每类事件推荐字段：

- `artifactId`
- `artifactRef`
- `type`
- `title`
- `previewSkill`
- `metadata`
- `taskId`
- `agentName`
- `status`

约束说明：

- 不在 AG-UI 文本流中传输大二进制。
- AG-UI 事件只传引用、状态和预览元数据。
- 前端通过 `artifactRef` / `contentRef` 获取可预览内容或展示结构化内容。

## 7. Frontend Runtime Skills 映射

- `code` -> `code_preview`
- `webpage` -> `web_preview`
- `document` -> `markdown_render`
- `diff` -> `diff_preview`
- `slide_deck` -> `slide_deck_preview`
- `image_preview` -> `image_preview`
- `file_card` -> `file_card`
- `file_download` -> `file_download`
- `vision_analysis` -> `vision_analysis_card`
- `file_summary` -> `file_summary_card`
- `security_report` -> `security_report_card`
- `deployment_plan` -> `deployment_plan_card`
- `release_report` -> `release_report_card`
- `artifact_manifest` -> `artifact_manifest_view`

说明：

- 前端 Runtime Skill 只负责展示和受控交互。
- 不执行 Agent 输出中的任意脚本。
- `web_preview` 必须 iframe sandbox。
- `markdown_render` 必须防 XSS。
- `file_download` 需要权限校验。
- `slide_deck_preview` 只渲染结构化草稿，不假装已经生成 pptx。

## 8. 安全边界

- 禁止把真实密钥放进 artifact content。
- 禁止把 `.env` 内容作为 Artifact 输出。
- 禁止前端执行不可信脚本。
- HTML/webpage 预览必须沙箱。
- Markdown 渲染必须消毒。
- 文件下载链接必须有权限、过期时间、最小可见范围。
- 日志中不记录完整授权 URL。
- `contentRef` 不应是本地 `file://` 路径。
- 产物中的敏感内容应先经过 `security-agent` 或安全策略检查。
- `deployment_status` 不应伪造真实部署结果。
- `release_report` 不应伪造真实发布结果。
- `slide_deck` 不应伪造真实 pptx 生成结果。

## 9. 与多模态输入契约的关系

- `multimodal-attachment.md` 解决“用户给平台什么”。
- `rich-artifact-output.md` 解决“Agent 给用户什么”。
- `vision-analysis-artifact.md` 是图片理解后的中间产物。
- `file_summary` 是文件理解后的中间产物。
- 输入侧 `contentRef` 和输出侧 `artifactRef` 可以形成闭环。
- 现有 18 个 Agent 继续作为能力池，通过 Artifact 输出富媒体结果。

## 10. MVP 与后续演进

MVP：

- 继续使用现有 Artifact Type。
- 子 Agent 输出结构化 `content`。
- 大产物预留 `contentRef`。
- 前端先做卡片和结构化预览。
- PPT 先是 `slide_deck` 草稿，不生成真实 pptx。
- 部署先是 `deployment_plan`，不执行真实部署。
- 发布先是 `release_report`，不执行真实发布。

后续：

- 对象存储
- 真实文件生成
- PPTX/PDF 导出
- 图片生成
- 文件下载
- artifact versioning
- artifact lifecycle
- preview permission
- artifact search
- artifact sharing
