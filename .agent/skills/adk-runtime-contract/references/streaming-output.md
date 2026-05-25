# streaming-output

## 目的

本文定义 ADK Runtime 的流式文本输出规则。

## 基本规则

`ctx.StreamText(chunk)` 只用于用户可读文本流。

适合输出：

- 说明。
- 分析。
- 步骤。
- 摘要。
- Markdown 文本。
- 当前进度的自然语言描述。

不适合输出：

- 大型代码文件的唯一事实源。
- 完整 HTML 的唯一事实源。
- 二进制内容。
- 前端 Tool Call 参数。
- A2A 原始 JSON。
- AG-UI 原始 JSON。

## Runtime 映射

```text
ctx.StreamText(chunk) → A2A text event → AG-UI TEXT_MESSAGE_CONTENT
```

## Chunk 规则

- Runtime 不保证 chunk 是完整句子。
- Runtime 不保证 chunk 是完整 Markdown block。
- Orchestrator / Frontend 必须支持任意 chunk 切分。
- Handler 不应依赖 chunk 边界表达语义。

## Artifact 关系

如果内容需要结构化预览，应同时输出 Artifact。

例如：

- 代码预览：输出 `code` Artifact。
- 网页预览：输出 `webpage` Artifact。
- Markdown 文档下载或独立预览：输出 `document` Artifact。

## 禁止事项

- 不得把 API key 输出到文本流。
- 不得把完整 system prompt 输出到文本流。
- 不得在文本流中伪造 A2A event。
- 不得在文本流中伪造 AG-UI Tool Call。
