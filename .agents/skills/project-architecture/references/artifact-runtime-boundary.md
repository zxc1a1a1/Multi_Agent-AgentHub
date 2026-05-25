# Artifact / Runtime Capability Boundary

Artifact 是非纯文本产物的通用表达。Runtime Capability 是前端渲染产物的通用机制。

## 规则

- 非纯文本产物必须通过 Artifact 或等价引用表达。
- 大产物不得塞入普通消息文本。
- Artifact 必须能关联 runId 与 messageId。
- Runtime Capability 根据产物类型或 tool call 渲染。
- 新增产物类型不应要求修改项目总架构。

## v1.0 示例

- code。
- webpage。
- markdown。

这些是 Sprint 示例，不是长期全集。
