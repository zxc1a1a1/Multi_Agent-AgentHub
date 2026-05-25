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
- Core `artifactId` 不使用 `id`，避免和 messageId / runId 混淆。
- Public API DTO 中 `id` 是 `artifactId` 的公开投影。DB `artifacts.id` 存储 `artifactId`。
- `type` 表示 Artifact 类型。
- Core `preview.previewType` 表示预览意图。Public API DTO 扁平化为 `previewType`。
- `links` 在 Core 中是嵌套对象（`links.conversationId` 等），在 Public API DTO 中扁平化为顶层字段。
- `metadata` 用于类型特定扩展。

## ArtifactDraft 与 Core Artifact

标准 Artifact schema 定义的是 **Core Artifact**（归一化后的完整产物对象）。

Child Agent / ADK Handler 输出的 **ArtifactDraft** 不是 Core Artifact：

- ArtifactDraft 只包含 `type`、`title`、`content`（或 `contentRefDraft`）、`metadata`。
- Core Artifact 的 `artifactId`、`mimeType`、`source.*`、`links.*`、`preview.*`、`version`、`status`、`createdAt` 由 Orchestrator / ArtifactRegistry 归一化时生成。
- ArtifactDraft 不得包含平台字段。
- 不得把 ArtifactDraft 当作 Core Artifact。

## schema 修改规则

任何字段新增、删除、重命名、类型变化，都必须同步更新：

- Markdown 契约。
- JSON Schema。
- Review checklist。
- 示例 Artifact。

不得让实现代码成为事实源。
