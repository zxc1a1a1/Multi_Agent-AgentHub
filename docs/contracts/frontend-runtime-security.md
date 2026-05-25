# Frontend Runtime Security

## 输入不可信

所有 Tool Call 参数都视为不可信。

参数校验只能证明结构正确，不能证明内容安全。

## HTML / iframe

`web_preview` 必须隔离渲染：

- 不直接注入主应用 DOM。
- 使用 iframe / sandbox / 等价隔离策略。
- 默认不启用 `allow-same-origin`。
- 默认不启用 forms、popups、downloads、top navigation。
- 不默认组合 `allow-scripts allow-same-origin`。

## Markdown

- 默认禁用 raw HTML。
- 链接协议必须限制。
- 渲染失败回退纯文本。

## Code

- 只展示。
- 不执行。
- 不访问网络。
- 不修改服务端状态。

## 错误脱敏

用户可见错误不得包含：

- API key。
- Authorization token。
- 内部堆栈。
- 内网地址。
- 完整 system prompt。
