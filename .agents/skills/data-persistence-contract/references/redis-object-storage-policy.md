# Redis 与 Object Storage 规则

## Redis

Redis 可用于：

- 缓存。
- 限流。
- 分布式锁。
- 短期任务状态。
- 临时 streaming 状态。

Redis 不得用于：

- 消息唯一事实源。
- Run 唯一事实源。
- Artifact 唯一事实源。
- Agent Registry 唯一事实源。

## Object Storage

Object Storage 可用于：

- 大型 Artifact。
- 图片。
- zip。
- 下载文件。
- 长日志。
- 大型 HTML / 文档。

关系数据库必须保存：

- content_ref。
- mime_type。
- title。
- size。
- checksum 可选。
- owner / conversation / message / run / agent 关联。

## 禁止

- 把对象存储签名 URL 当永久字段。
- 绕过数据库直接把对象地址给前端。
- 没有权限校验就下载。
- 只靠对象存储路径表达归属关系。
