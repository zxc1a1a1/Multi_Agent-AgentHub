# Artifact Runtime Security

## 风险分级

低风险：
- markdown_render
- code_preview 只展示不执行

中风险：
- file_download
- terminal_output
- generated_document

高风险：
- web_preview
- file_upload
- run_command
- deploy_action
- file_overwrite
- external_publish

## 规则

- Artifact 内容不可信。
- Artifact metadata 不得包含 secret。
- 大型 Artifact 必须走 contentRef 或受控下载。
- code preview 默认只展示不执行。
- markdown 必须防 XSS。
- web preview 必须 iframe sandbox。
- 高风险 Runtime Capability 必须权限校验、确认和审计。
