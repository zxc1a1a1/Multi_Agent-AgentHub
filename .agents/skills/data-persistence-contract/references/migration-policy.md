# Migration Policy

来源：`docs/contracts/migration-policy.md`。

- migration 必须版本化。
- schema 变更需同步 data model / OpenAPI / 相关 contract。
- 不在生产库手工改且不留记录。
- 可逆性（up/down）或不可逆说明必须明确。
