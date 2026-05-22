# Object Storage Policy

## 1. 文档目的

本文档定义 AgentHub 中 Object Storage 的使用边界。

Object Storage 用于保存大 Artifact 内容，数据库保存 metadata、object key、访问策略和关联 ID。

## 2. MVP v0.1 状态

MVP v0.1 暂不强制使用对象存储。

MVP 可先将小型 code Artifact 存入数据库字段，但必须保留长期迁移方向。

## 3. Post-MVP 使用场景

Object Storage 用于：

- 大代码包。
- HTML / CSS / JS 压缩包。
- 图片。
- 文档。
- zip。
- 日志文件。
- 大型 Artifact。
- 部署产物。

## 4. 数据库关联

数据库保存：

```text
artifact.id
artifact.conversationId
artifact.messageId
artifact.runId
artifact.type
artifact.title
artifact.fileUrl 或 objectKey
artifact.metadata
artifact.version
```

对象存储保存：

```text
artifact binary / text content
large files
preview bundles
downloadable files
```

## 5. 安全规则

- 私有 Artifact 下载必须鉴权。
- 下载 URL 应短期有效。
- 不允许公开暴露敏感文件。
- 文件类型必须校验。
- 文件大小必须限制。
- 删除 Artifact 时必须处理对象存储清理。
- object storage credentials 不得写进代码、AgentCard、OpenAPI 示例、日志。

## 6. Artifact 映射

Artifact 类型与 Frontend Runtime Skill 的映射由后续 `artifact-contract` 和 `frontend-runtime-skills-contract` 细化。

当前文件只规定对象存储策略。
