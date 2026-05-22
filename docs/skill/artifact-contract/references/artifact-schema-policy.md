# Artifact Schema 规则

## 1. 目的

本文定义 AgentHub normalized Artifact 的标准结构。

注意：

```text
AgentHub Artifact ≠ A2A 官方 Artifact
```

AgentHub Artifact 是项目内标准化后的产物对象。

## 2. 标准结构

推荐结构：

```ts
type AgentHubArtifact = {
  artifactId: string;
  type: ArtifactType;
  title: string;
  summary: string;

  content?: unknown;
  contentRef?: ContentRef;

  metadata: Record<string, unknown>;

  conversationId: string;
  messageId: string;
  runId: string;
  a2aTaskId?: string;
  stepId?: string;
  agentName?: string;

  version: number;
  status: "created" | "normalized" | "persisted" | "preview_mapped" | "previewed" | "failed";

  createdAt: string;
  updatedAt: string;
};
```

## 3. 必需字段

长期 Artifact 必须包含：

```text
artifactId
type
title
summary
metadata
conversationId
messageId
runId
version
status
createdAt
updatedAt
```

## 4. content / contentRef

Artifact 必须至少有一个：

```text
content
contentRef
```

长期大型内容应优先使用：

```text
contentRef
```

MVP small code 可以使用：

```text
content
```

## 5. metadata

metadata 必须按 Artifact type 校验。

`code` metadata 至少包含：

```text
language
```

## 6. 禁止事项

不得：

- Artifact 缺少 runId。
- Artifact 缺少 messageId。
- Artifact 缺少 type。
- metadata 任意扩张且不校验。
- 大内容直接塞进 content。
- contentRef 指向未经授权的私有 URL。
