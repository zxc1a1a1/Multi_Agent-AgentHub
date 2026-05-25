# Parameter Schema Policy

## 目的

所有 Runtime Capability 参数都必须通过 schema 校验后才能传给组件。

## 必须校验

- 必填字段。
- 字段类型。
- 字符串长度。
- 数组长度。
- URL 协议。
- 枚举值。
- 高风险字段，例如 HTML、JS、Markdown、下载 URL。

## 禁止

- 组件直接接收原始 args 字符串。
- 在组件内部才做第一次 JSON.parse。
- schema 校验失败后继续渲染。
- 用 Agent 名称跳过 schema 校验。

## 降级

参数非法时应根据 `failureMode` 降级：

- `error_card`
- `placeholder`
- `text_fallback`
- `hide`
