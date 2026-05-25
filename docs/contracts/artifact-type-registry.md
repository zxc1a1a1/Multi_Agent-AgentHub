# Artifact Type Registry

## v1.0 Required

| type | mimeType | previewType | 说明 |
|---|---|---|---|
| code | text/plain | code_preview | 代码片段或单文件代码 |
| webpage | text/html | web_preview | HTML 页面或网页片段 |
| markdown | text/markdown | markdown_render | Markdown 文档、说明、报告 |

## Planned

| type | mimeType | previewType | 说明 |
|---|---|---|---|
| document | application/octet-stream | document_preview | 文档 |
| data | application/json | data_preview | 结构化数据 |
| image | image/png | image_preview | 图片 |
| archive | application/zip | download | 打包文件 |
| audio | audio/mpeg | download | 音频 |
| video | video/mp4 | download | 视频 |
| diff | text/x-diff | code_preview | 差异补丁 |
| terminal_log | text/plain | document_preview | 终端日志 |

## 新增类型流程

1. 在本文件登记 type。
2. 更新 `artifact.schema.json`。
3. 更新 `artifact-schema.md`。
4. 增加示例。
5. 增加 Review 检查项。
6. 明确 inline / contentRef 策略。
7. 明确安全风险。
