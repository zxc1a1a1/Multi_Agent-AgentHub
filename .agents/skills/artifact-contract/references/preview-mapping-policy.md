# Preview Mapping Policy

## 定义

Preview Mapping 定义三层映射：

```text
outputMode → artifact.type → previewType → Frontend Runtime toolName
```

其中：
- `outputMode`（上游）是 Agent 声明的语义类别。
- `artifact.type` 是平台归一化后的产物类型。
- `previewType` 应优先等于 Frontend Runtime Registry 中的 `toolName`。

三层不得直接等同，所有转换必须通过映射表表达。

它不定义前端组件实现。

## v1.0 映射（required）

| artifact.type | previewType | toolName |
|---|---|---|
| `code` | `code_preview` | `code_preview` |
| `webpage` | `web_preview` | `web_preview` |
| `markdown` | `markdown_render` | `markdown_render` |

## planned 映射

| artifact.type | previewType | toolName |
|---|---|---|
| `document` | `document_preview` | `document_preview` |
| `data` | `data_preview` | `data_preview` |
| `image` | `image_preview` | `image_preview` |
| `archive` | `file_download` | `file_download` |

## 完整三层映射链（含上游 outputMode）

| outputMode | artifact.type | previewType | toolName | 说明 |
|---|---|---|---|---|
| `code` | `code` | `code_preview` | `code_preview` | 代码类产物 |
| `webpage` | `webpage` | `web_preview` | `web_preview` | 网页类产物 |
| `document` | `document` 或 `markdown` | `document_preview` 或 `markdown_render` | `document_preview` 或 `markdown_render` | 归一化时根据 metadata.format 确定 artifact.type |
| `text` | 无 Artifact | — | `markdown_render` / StreamingText | 纯文本流 |

## 规则

- `previewType` 是预览意图。
- `previewType` 应优先等于 Runtime toolName。
- `previewType` 不是 Artifact 类型。
- `artifact.type` 不是 `toolName`。
- `outputMode` 不是 `toolName`。
- `outputMode` 不是 `artifact.type`（除非映射表显式声明兼容）。
- `previewType` 不是组件名。
- 不得使用 `download` 作为 previewType（已废弃，统一使用 `file_download`）。
- 不得通过 `agentName` 推断 outputMode 或选择 toolName。
- `document_preview` / `data_preview` 当前为 planned/reserved，不要求实现。
- Artifact 可以没有可用预览。
- 未知 previewType 必须安全降级。

### Public API 投影

Core 中 `previewType` 嵌套在 `preview` 对象下（含 `previewType`、`available`、`reason`）。

Public API DTO 将 `previewType` 扁平化为顶层字段。`preview.available` 和 `preview.reason` 是 Core 内部字段，不直接暴露给 Public API。
