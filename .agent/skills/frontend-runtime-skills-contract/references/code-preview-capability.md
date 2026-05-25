# code_preview Capability

## 定义

`code_preview` 是通用代码展示能力，不绑定任何 Agent。

## 参数

```ts
type CodePreviewParams = {
  code: string
  language: string
  filename?: string
  title?: string
}
```

## 安全规则

- 只展示代码。
- 不执行代码。
- 不插入 script。
- 不访问网络。
- 不修改服务端状态。
- 大代码应折叠或截断。
- `language` 不合法时降级为纯文本。
