# Artifact / Runtime Test Policy

v1.0 Sprint 示例包含 code、webpage、markdown，但长期测试按 artifactType 和 runtimeCapability 扩展。

## 必测场景

- code artifact 可预览。
- webpage artifact 可 iframe 预览。
- markdown 可渲染。
- unknown artifact type 安全降级。
- missing runtime capability 不生成不可消费 ToolCall。
- large artifact 使用 contentRef。
- iframe sandbox 存在。
