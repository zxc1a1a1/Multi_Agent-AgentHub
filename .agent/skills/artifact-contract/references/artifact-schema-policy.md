# Artifact Schema Policy

## 基础要求

Artifact schema 必须使用 JSON Schema 2020-12。

项目级 schema 文件：

```text
docs/contracts/artifact.schema.json
```

说明文档：

```text
docs/contracts/artifact-schema.md
```

## 最小必填字段

```text
artifactId
type
title
mimeType
links.conversationId
links.messageId
links.runId
version
status
createdAt
```

`content` 与 `contentRef` 至少存在一个。

## 字段命名

- 对外 JSON 字段使用 camelCase。
- `artifactId` 不使用 `id`，避免和 messageId / runId 混淆。
- `type` 表示 Artifact 类型。
- `preview.previewType` 表示预览意图。
- `metadata` 用于类型特定扩展。

## schema 修改规则

任何字段新增、删除、重命名、类型变化，都必须同步更新：

- Markdown 契约。
- JSON Schema。
- Review checklist。
- 示例 Artifact。

不得让实现代码成为事实源。
