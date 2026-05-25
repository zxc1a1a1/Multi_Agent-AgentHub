# Artifact Type Registry

## 三层边界

```text
outputMode → artifact.type → previewType → toolName
```

- `outputMode`：Agent 声明可产出的语义类别（上游）。
- `artifact.type`：平台归一化后的产物类型（本 Registry 管理）。
- `toolName`：前端执行/渲染能力名（下游，由 previewType 决定）。

三者不得直接等同。所有转换必须通过映射表表达。

映射链示例：

| outputMode | artifact.type | previewType | toolName |
|---|---|---|---|
| `code` | `code` | `code_preview` | `code_preview` |
| `webpage` | `webpage` | `web_preview` | `web_preview` |
| `document` | `document` 或 `markdown` | `document_preview` 或 `markdown_render` | `document_preview` 或 `markdown_render` |
| `text` | 无 Artifact | — | `markdown_render` / StreamingText |

`artifact.type` 由 ArtifactRegistry 在归一化时根据 ArtifactDraft 中的 `type` 字段、`metadata`（如 `language`、`format`）、推断的 `mimeType`、以及 Agent 声明的 `outputMode` 等信息综合确定。

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
| archive | application/zip | file_download | 打包文件 |
| audio | audio/mpeg | file_download | 音频 |
| video | video/mp4 | file_download | 视频 |
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
8. 明确 outputMode → artifact.type → previewType → toolName 映射链。

## 禁止

- 不得把 `artifact.type` 直接当 `toolName`。
- 不得把 `outputMode` 直接当 `artifact.type`（除非映射表显式声明兼容）。
- 不得把 `outputMode` 直接当 `toolName`。
- 不得通过 `agentName` 推断 outputMode 或选择 toolName。
