# markdown_render Capability

## 定义

`markdown_render` 是通用 Markdown 渲染能力，不绑定任何 Agent。

## 参数

```ts
type MarkdownRenderParams = {
  markdown: string
  title?: string
}
```

## 安全规则

- 支持 GFM。
- 默认禁用 raw HTML。
- 链接协议必须限制。
- 代码块只展示，不执行。
- 渲染失败回退纯文本。
- 不得把 Markdown 中的 HTML 当可信 DOM 插入主应用。
