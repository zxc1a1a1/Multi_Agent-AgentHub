# Artifact Review Checklist

## Schema

- [ ] 是否更新 `docs/contracts/artifact-schema.md`？
- [ ] 是否更新 `docs/contracts/artifact.schema.json`？
- [ ] 是否使用 JSON Schema 2020-12？
- [ ] 是否定义必填字段？
- [ ] 是否定义 content / contentRef 规则？

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

## Identity

- [ ] 是否有 artifactId？
- [ ] 是否有 conversationId / messageId / runId？
- [ ] 多 Agent 场景是否保留来源？
- [ ] 重名文件是否不会冲突？

## Lifecycle

- [ ] 是否有 status？
- [ ] 是否只有 ready 可预览？
- [ ] 是否支持版本？
- [ ] 删除是否不破坏历史引用？

## Security

- [ ] 是否没有 API key？
- [ ] 是否没有 token？
- [ ] 是否没有本地路径？
- [ ] 是否没有内网地址？
- [ ] 是否没有 system prompt？
