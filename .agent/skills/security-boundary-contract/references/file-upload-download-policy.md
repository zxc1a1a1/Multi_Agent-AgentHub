# File Upload / Download Policy

## file_upload

上传文件必须：

- 限制大小。
- 校验 MIME type。
- 校验扩展名。
- 私有存储。
- 绑定 owner / conversation / run。
- 不直接执行。
- 不直接注入 Prompt。
- 必要时安全扫描。

## file_download

下载文件必须：

- 鉴权。
- 对象级授权。
- 使用短期 URL 或受控下载接口。
- 设置明确 Content-Type。
- 使用安全 Content-Disposition。
- 不暴露内部存储路径或签名凭证。
