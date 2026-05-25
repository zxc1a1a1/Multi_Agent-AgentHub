# Artifact Review Checklist

## Schema

- [ ] 是否更新 `docs/contracts/artifact-schema.md`？
- [ ] 是否更新 `docs/contracts/artifact.schema.json`？
- [ ] 是否使用 JSON Schema 2020-12？
- [ ] 是否定义必填字段？
- [ ] `content` / `contentRef` 是否至少存在一个？

## Type Registry

- [ ] 新类型是否登记到 `artifact-type-registry.md`？
- [ ] 是否有 mimeType？
- [ ] 是否有 previewType？
- [ ] 是否有 inline 策略？
- [ ] 是否有安全说明？

## Content

- [ ] 大内容是否没有进入 `content`？
- [ ] 二进制是否没有 base64 inline？
- [ ] 私有文件是否使用受控 contentRef？
- [ ] contentRef 是否不是裸签名 URL？

## Identity

- [ ] 是否有 artifactId？
- [ ] 是否有 conversationId？
- [ ] 是否有 messageId？
- [ ] 是否有 runId？
- [ ] 多 Agent 情况是否保留 source？

## Lifecycle

- [ ] 是否设置 status？
- [ ] 是否只有 ready 可预览？
- [ ] 重新生成是否保留旧版本？
- [ ] deleted 是否保留 tombstone？

## Security

- [ ] 是否没有 API key？
- [ ] 是否没有 token？
- [ ] 是否没有本地路径？
- [ ] 是否没有内网地址？
- [ ] 是否没有 system prompt？
- [ ] 是否没有未脱敏隐私？
