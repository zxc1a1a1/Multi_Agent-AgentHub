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
