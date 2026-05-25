# File Upload / Download Policy

## 上传

- 限制大小。
- 校验 MIME type 与扩展名。
- 私有存储。
- 绑定 owner / conversation / run。
- 不直接执行。
- 不直接注入 Prompt。

## 下载

- 鉴权。
- 对象级授权。
- 短期 URL 或受控接口。
- 安全 Content-Type 与 Content-Disposition。
- 不暴露内部路径或签名凭证。
