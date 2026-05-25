# Artifact Review Checklist

## Schema

- [ ] 是否更新 `docs/contracts/artifact-schema.md`？
- [ ] 是否更新 `docs/contracts/artifact.schema.json`？
- [ ] 是否使用 JSON Schema 2020-12？
- [ ] 是否定义必填字段？
- [ ] 是否定义 content / contentRef 规则？
- [ ] 是否同步 Core↔Public API DTO 字段映射？

## Type

- [ ] artifact.type 是否已注册？
- [ ] previewType 是否已注册？
- [ ] 是否没有用组件名当 type？
- [ ] 是否没有用 agentName 判断 type？

## Content

- [ ] 小内容是否合理 inline？
- [ ] 大内容是否使用 contentRef？
- [ ] 是否没有 inline zip / image / 二进制？
- [ ] 是否没有裸露私有 URL？
- [ ] Core contentRef 是否为对象型引用（非裸 URL 字符串）？
- [ ] Public API contentRef 是否为公开授权访问入口 URL？
- [ ] 是否提供 contentUrl 字段用于公开访问地址？

## Identity

- [ ] 是否有 artifactId？
- [ ] 是否有 conversationId / messageId / runId？
- [ ] Public API 的 id 是否为 artifactId 的公开投影？
- [ ] Public API 扁平 conversationId/messageId/runId 是否从 links.* 投影？
- [ ] 多 Agent 场景是否保留来源？
- [ ] 重名文件是否不会冲突？

## Lifecycle

- [ ] 是否有 status？
- [ ] status 是否为 Core 枚举：pending/normalizing/ready/failed/superseded/deleted？
- [ ] 是否只有 ready 可预览？
- [ ] 是否支持版本？
- [ ] version 是否为 integer？
- [ ] 删除是否不破坏历史引用？
- [ ] 如有 legacy created 状态，是否有明确的 Core 状态映射？

## Normalization

- [ ] Child Agent / ADK 输出的是否是 ArtifactDraft？
- [ ] ArtifactDraft 是否只包含 `type`、`title`、`content`（或 `contentRefDraft`）、`metadata`？
- [ ] ArtifactDraft 是否没有 `artifactId`、`version`、`links.*`、`source.*`、`preview.*`、`status`、`createdAt`？
- [ ] 是否有归一化流程将 ArtifactDraft 转为 Core Artifact？
- [ ] 是否没有把 ArtifactDraft 当作 Core Artifact？

## Security

- [ ] 是否没有 API key？
- [ ] 是否没有 token？
- [ ] 是否没有本地路径？
- [ ] 是否没有内网地址？
- [ ] 是否没有 system prompt？
