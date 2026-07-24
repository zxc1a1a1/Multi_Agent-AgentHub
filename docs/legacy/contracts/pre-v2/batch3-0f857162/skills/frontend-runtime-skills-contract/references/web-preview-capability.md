# web_preview Capability

## 定义

`web_preview` 是通用 HTML/Web 预览能力，不绑定任何 Agent。

## 参数

```ts
type WebPreviewParams = {
  html: string
  css?: string
  js?: string
  title?: string
  filename?: string
}
```

## 安全规则

- 不得把 HTML 直接注入主应用 DOM。
- 必须使用 iframe / sandbox / 等价隔离策略。
- 默认不得启用 `allow-same-origin`。
- 不得默认启用 `allow-forms`。
- 不得默认启用 `allow-popups`。
- 不得默认启用 `allow-downloads`。
- 不得默认启用 `allow-top-navigation`。
- 不得默认组合 `allow-scripts allow-same-origin`。
- 渲染失败必须显示错误卡片或占位。

## 推荐 sandbox

```html
<iframe sandbox="allow-scripts" />
```
