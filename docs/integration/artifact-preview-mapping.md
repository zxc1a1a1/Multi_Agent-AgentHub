# Artifact 与前端预览组件映射说明

| Artifact Type | 产生 Agent | 推荐前端组件 | 是否真实文件 | 是否可下载 | 注意事项 |
|---|---|---|---|---|---|
| code | code-agent | code_preview | 否（默认草稿） | 可选 | 只展示代码，不执行代码 |
| webpage | web-agent | web_preview | 否（默认草稿） | 可选 | 必须使用 sandbox iframe |
| document | document-agent / custom-agent | markdown_render | 否（默认草稿） | 可选 | Markdown 渲染必须防 XSS |
| diff | diff-agent | diff_preview | 否 | 可选 | 第一版不应用 patch |
| context_bundle | context-agent | context_card / json_viewer | 否 | 否 | 作为上下文中间产物 |
| review_report | review-agent | report_card | 否 | 可选 | 结构化审查报告 |
| test_report | test-agent | report_card | 否 | 可选 | 第一版只分析日志 |
| security_report | security-agent | security_report_card | 否 | 可选 | 注意敏感信息脱敏展示 |
| artifact_manifest | artifact-agent | artifact_manifest_view | 否 | 可选 | 用于聚合多产物 |
| acceptance_report | qa-acceptance-agent | acceptance_card | 否 | 可选 | 关注通过/失败/待确认项 |
| version_snapshot | version-agent | version_card | 否 | 可选 | 第一版不执行 Git 写操作 |
| agent_profile | agent-builder-agent | yaml_viewer | 否 | 可选 | Profile 草稿，不创建文件 |
| file_summary | file-agent | file_summary_card | 否 | 可选 | 第一版不读取本地文件 |
| web_research | web-research-agent | research_card | 否 | 可选 | 第一版不主动联网 |
| deployment_plan | deploy-agent | deployment_plan_card | 否 | 可选 | 不是部署结果 |
| slide_deck | ppt-agent | slide_deck_preview | 否（草稿） | 可选 | 不是 pptx 文件 |
| release_report | release-agent | release_report_card | 否 | 可选 | 不是发布动作 |
| vision_analysis | vision-agent | vision_analysis_card | 否（中间产物） | 可选 | v0.1 基于引用/文本推断，不真实看图 |

补充说明：

- `slide_deck` 不是 `pptx`。
- `deployment_plan` 不是部署结果。
- `release_report` 不是发布动作。
- `webpage` 预览必须 sandbox。
- `markdown` 渲染必须防 XSS。
- file download 需要权限校验与有效期控制。
