# Content and Reference Policy

## content

`content` 用于小型 inline 文本内容。

适合：

- 小型代码片段。
- 小型 Markdown。
- 小型单页 HTML。
- 小型 JSON。

不适合：

- zip。
- 图片。
- 二进制。
- 大型日志。
- 大型网页包。
- 私有文件。

## contentRef

`contentRef` 用于后端可控内容引用。

必须具备：

- 可授权。
- 可审计。
- 可校验完整性。
- 不泄漏私有路径或 token。

推荐结构：

```json
{
  "kind": "object_store",
  "bucket": "agenthub-artifacts",
  "key": "conv_001/art_001/file.zip",
  "sizeBytes": 120000,
  "sha256": "..."
}
```

## Public API 投影说明

Core `contentRef` 是对象型后端引用（`kind`、`bucket`、`key`、`sizeBytes`、`sha256`），不是裸字符串。

Public API 中的 `contentRef` 字符串是公开授权访问入口 URL，不是 Core contentRef 对象。推荐 Public API 使用 `contentUrl` 字段避免混名。

DB `artifacts.content_ref` 列存储 Core contentRef 对象的序列化 JSON。

## 禁止

- 裸签名 URL。
- 本地绝对路径。
- 内网地址。
- 含 token 的下载链接。
- 无大小限制的 base64。
- 把裸 URL 字符串作为 Core Artifact.contentRef。
