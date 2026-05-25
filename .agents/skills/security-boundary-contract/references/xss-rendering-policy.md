# XSS Rendering Policy

## 原则

用户输入、LLM 输出、Agent 输出、Artifact HTML、Markdown 都是不可信内容。

## 规则

- 不可信内容必须按上下文编码或清洗。
- Markdown 不默认允许危险 HTML。
- HTML 预览必须 iframe sandbox。
- 错误页面不得反射未转义用户输入。
- 外链应使用安全打开策略，防止 tabnabbing。
- CSP 可作为额外防线，但不能替代输出编码和 sandbox。

## 评审问题

- 内容是否来自用户、LLM 或 Agent？
- 是否进入主 DOM？
- 是否允许脚本？
- 是否能访问认证信息？
