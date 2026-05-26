# 附件安全策略补充

- 附件安全检查应覆盖：`mimeType/size` 校验、SSRF 防护、路径穿越防护、密钥扫描、日志脱敏。
- 禁止 `file://`、`localhost`、内网 metadata URL 作为可访问目标。
- OCR 或提取文本结果也必须经过 secret scan。
- `contentRef` 不应暴露永久公开 URL，应具备权限与时效控制。
