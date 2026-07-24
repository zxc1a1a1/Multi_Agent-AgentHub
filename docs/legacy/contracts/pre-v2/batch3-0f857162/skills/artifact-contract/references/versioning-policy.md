# Versioning Policy

## 规则

Artifact 必须支持版本化。

- version 从 1 开始。
- 新内容不得覆盖旧版本。
- 小修订可以增加 version。
- 独立生成可以创建新 artifactId。
- 历史消息引用必须稳定。

## supersedes

当旧版本被新版本替代时，可以设置：

```json
{
  "status": "superseded",
  "supersedes": {
    "artifactId": "art_001",
    "version": 1
  }
}
```

新版本 Artifact 的 `supersedes` 指向被替代的旧版本。字段名对齐 `artifact.schema.json` 中的 `supersedes` 定义。
```

## artifactId 稳定性

- `artifactId` 是跨层级稳定 ID。同一 `artifactId` 的新版本不改变 `artifactId`。
- Public API 的 `id` 和 DB 的 `id`（或 `artifact_id`）均映射到同一个 `artifactId` 值。
- 如需全新独立产物，应创建新 `artifactId`，而不只是增加 `version`。

## 禁止

- 原地覆盖 content。
- 删除旧版本导致历史消息打不开。
- 用 title 区分版本。
- 为新版本分配新的 `artifactId` 而丢失版本链。
