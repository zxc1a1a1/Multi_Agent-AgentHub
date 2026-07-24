# Storage Policy

## 分层存储

| 内容 | 推荐存储 |
|---|---|
| 小型文本 | 数据库 JSON 或 artifact 表 |
| 小型代码 | 数据库 JSON 或 artifact 表 |
| 小型 Markdown | 数据库 JSON 或 artifact 表 |
| 小型单页 HTML | 数据库 JSON 或 artifact 表 |
| 大型文件 | contentRef |
| 二进制 | contentRef |
| zip/tar | contentRef |

## 事实源

- Artifact 事实源必须可恢复。
- 内存缓存不是事实源。
- Redis 不应作为唯一事实源。
- 临时 URL 不是事实源。

## 内容引用

Core `contentRef` 应保存稳定对象标识（`kind`、`bucket`、`key`、`sizeBytes`、`sha256`），不是短期可访问链接。

Public API 中的 `contentRef` 字符串是公开授权访问入口 URL。推荐 Public API 使用 `contentUrl` 字段避免混名。

DB `artifacts.content_ref` 列存储 Core contentRef 对象的序列化 JSON。

下载链接应在访问时按权限动态生成。
