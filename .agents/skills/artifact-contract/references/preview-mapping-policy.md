# Preview Mapping Policy

## 定义

Preview Mapping 只定义：

```text
artifact.type → previewType
```

它不定义前端组件实现。

## v1.0 映射

| artifact.type | previewType |
|---|---|
| code | code_preview |
| webpage | web_preview |
| markdown | markdown_render |

## planned 映射

| artifact.type | previewType |
|---|---|
| document | document_preview |
| data | data_preview |
| image | image_preview |
| archive | download |

## 规则

- previewType 是预览意图。
- previewType 不是 Artifact 类型。
- previewType 不是组件名。
- Artifact 可以没有可用预览。
- 未知 previewType 必须安全降级。
