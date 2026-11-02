# XSS / Rendering Policy

## 原则

用户输入、LLM 输出、Agent 输出、Markdown、HTML Artifact 都不可信。

## 规则

- Markdown 默认不允许危险 HTML。
- HTML 预览使用 iframe sandbox。
- 不可信 HTML 不直接进入主应用 DOM。
- 外链安全打开。
- 错误页面不反射未转义输入。
- CSP 是额外防线，不能替代编码和隔离。

## 实现状态（v3 A5-07）

- `StreamingText` 使用 `react-markdown` + `remark-gfm`，未启用 `rehype-raw`；
  原始 HTML 按文本处理。测试：`StreamingText.test.tsx` XSS 用例。
- `CodePreview` 已改为 Shiki token 渲染，不再使用 `dangerouslySetInnerHTML`；
  `sanitizeHighlightHtml` 保留给未来 HTML 高亮路径（允许 `span/br` 与
  `hljs-*`）。测试：`sanitizeHtml.test.ts`、`CodePreview.test.tsx`。
- `WebPreview` 使用无 `allow-scripts` 的 sandbox iframe，并注入
  `default-src 'none'` CSP；源码 tab 以文本渲染。
- 证据为单元测试，不是浏览器真机验证；Mermaid/KaTeX/Markdown agent 的渲染安全
  仍属于 A5-03b/A5-04 的未完成范围。
