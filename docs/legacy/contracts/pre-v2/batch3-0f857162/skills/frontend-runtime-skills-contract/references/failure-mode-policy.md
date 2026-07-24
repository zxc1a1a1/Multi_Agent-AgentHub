# Failure Mode Policy

## 可选策略

| failureMode | 说明 |
|---|---|
| `hide` | 静默隐藏，仅记录开发日志 |
| `placeholder` | 显示占位卡片 |
| `error_card` | 显示错误卡片 |
| `text_fallback` | 回退纯文本展示 |

## 规则

- 单个 capability 失败不得导致整个页面白屏。
- 高风险组件必须被隔离。
- 面向用户的错误必须脱敏。
- 开发日志不得打印 token、API key 或完整敏感 payload。
