# iframe Sandbox Policy

## 范围

适用于 web_preview、HTML preview、富媒体预览等能力。

## 规则

- HTML 预览必须与主应用 DOM 隔离。
- 默认使用 iframe sandbox。
- sandbox 权限默认最小化。
- 谨慎添加 allow-scripts、allow-forms、allow-same-origin 等能力。
- iframe 不得读取父页面 token、cookie、localStorage、sessionStorage。
- iframe 不得调用 Gateway 用户 API。
- 预览内容不得获得 service token 或用户 token。

## 禁止

- 直接 innerHTML 渲染不可信 HTML 到主应用 DOM。
- 将前端认证信息注入 iframe。
