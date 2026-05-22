# Versioning 规则

## 1. 目的

本文定义 Artifact 版本化规则。

长期 Artifact 必须可版本化。

## 2. 长期规则

长期要求：

- 同一 message 可有多个 artifacts。
- 同一 artifact 可有多个 version。
- 重新生成不能覆盖旧版本。
- version 必须可追踪 `runId / a2aTaskId / stepId / agentName`。
- 每个版本必须有 createdAt / updatedAt。
- 每个版本必须能追踪内容位置或 contentRef。

## 3. 推荐字段

```text
artifactId
version
parentArtifactId
supersedesArtifactId
createdAt
updatedAt
createdByRunId
createdByA2aTaskId
createdByAgentName
```

## 4. MVP 简化

MVP 可以简化为：

```text
version = 1
不实现 regenerate
不实现 artifact history
不实现 supersedes
```

但不得在正式开发中继续假设 Artifact 永远只有一个版本。

## 5. 禁止事项

不得：

- 重新生成时覆盖旧 Artifact。
- 无法追踪 Artifact 来源 run。
- 无法判断前端预览的是哪个版本。
- 用 filename 代替 artifactId。
