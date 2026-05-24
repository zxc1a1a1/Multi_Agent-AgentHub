# Artifact Persistence Policy

来源：`docs/contracts/data-model.md`、`docs/contracts/mysql-schema.md`。

- Artifact 不能仅存在于流式事件。
- 小型内容可入库，Post-MVP 大对象进入对象存储。
- 保留 `type/title/content-or-fileUrl/metadata/version` 等字段语义。
- metadata 禁止存储密钥。
