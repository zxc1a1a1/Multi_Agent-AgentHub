# Runtime Security Policy

## 通用规则

- Runtime Capability 输入一律不可信。
- 参数通过 schema 校验不代表内容安全。
- 展示型 capability 不得产生服务端副作用。
- 交互型 capability 必须有明确用户动作。
- 副作用型 capability 默认禁用。
- 未知 `toolName` 不得执行。
- 未校验 args 不得传给组件。

## XSS 规则

- 不把未知 HTML 注入主 DOM。
- Markdown 默认禁用 raw HTML。
- 文本展示默认转义。
- URL 字段默认只允许明确协议。
- 需要保留 HTML 结构时必须使用隔离或 sanitization 策略。

## 错误脱敏

用户可见错误不得包含：

- API key。
- Authorization token。
- 内部堆栈。
- 内网地址。
- 完整 system prompt。
- 完整敏感 payload。
