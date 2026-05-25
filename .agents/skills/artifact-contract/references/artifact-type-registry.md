# Artifact Type Registry

## 目的

Artifact Type Registry 用于统一管理 AgentHub 中允许出现的 `artifact.type`。

`artifact.type` 是平台归一化后的产物类型。ArtifactRegistry 在归一化时根据 ArtifactDraft 中的 `type` 字段、`metadata`（如 `language`、`format`）、推断的 `mimeType`、以及 Agent 声明的 `outputMode` 等信息综合确定最终的 `artifact.type`。

未注册的类型不得进入持久化、预览、下载或对外展示流程。

## 三层边界

```text
outputMode → artifact.type → previewType → toolName
```

- `outputMode`：Agent 声明可产出的语义类别（上游，不属于本 Registry）。
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

## v1.0 Required Types

| type | 说明 | 默认 mimeType | 默认 previewType | inline 策略 |
|---|---|---|---|---|
| `code` | 代码片段或单文件代码 | `text/plain` | `code_preview` | 小内容允许 |
| `webpage` | HTML 页面或网页片段 | `text/html` | `web_preview` | 小型单页允许 |
| `markdown` | Markdown 文档、说明、报告 | `text/markdown` | `markdown_render` | 小内容允许 |

## Planned Types

| type | 说明 | 默认 mimeType | 默认 previewType | inline 策略 |
|---|---|---|---|---|
| `document` | 通用文档 | `application/octet-stream` | `document_preview` | 默认不允许 |
| `data` | JSON / CSV / 结构化数据 | `application/json` | `data_preview` | 小 JSON 允许 |
| `image` | 图片 | `image/png` | `image_preview` | 不允许 |
| `archive` | zip / tar 等归档 | `application/zip` | `file_download` | 不允许 |

## 新增类型要求

新增 Artifact 类型时必须说明：

- 类型名。
- 典型用途。
- MIME 类型。
- 是否可 inline。
- 是否必须 contentRef。
- 默认 previewType。
- 最大建议大小。
- 安全风险。
- metadata 字段。
- schema 示例。

## 禁止

- 不得用组件名作为 type。
- 不得用事件名作为 type。
- 不得用 Agent 名称作为 type。
- 不得用文件扩展名直接作为 type，除非它确实是稳定产物类别。
- 不得把 artifact.type 直接当 toolName。
- 不得把 outputMode 直接当 toolName。
- 不得把 outputMode 直接当 artifact.type（除非映射表显式声明兼容）。
- 不得通过 agentName 推断 outputMode 或选择 toolName。
- 不得使用 `download` 作为 previewType（已废弃，统一使用 `file_download`）。
