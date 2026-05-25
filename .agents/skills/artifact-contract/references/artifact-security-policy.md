# Artifact Security Policy

## 禁止进入 Artifact 的内容

- API key。
- Authorization token。
- Cookie。
- 数据库连接串。
- 私有对象存储签名 URL。
- 本地绝对路径。
- 内网地址。
- 未脱敏个人信息。
- 完整 system prompt。
- 原始内部堆栈。

## HTML / webpage

webpage Artifact 是可预览内容，不代表可信执行内容。

规则：

- 不在本契约中授予浏览器权限。
- 不把 HTML 视为可信代码。
- 不在 Artifact 中嵌入访问后端的 secret。
- 不保存用户 token 到 HTML。

## contentRef

contentRef 必须：

- 后端可验证。
- 后端可授权。
- 后端可审计。
- 不直接暴露永久公开地址。
