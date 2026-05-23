# Artifact Registry 规则

## 1. 目的

本文定义 AgentHub Artifact 类型注册表。

Artifact type 是产物事实类型，不是前端组件名，也不是 toolName。

## 2. 长期 Artifact 类型

长期统一 Artifact 类型：

```text
code
webpage
diff
file
image
deploy
document
terminal
chart
```

## 3. 状态

每个 Artifact type 必须声明状态：

```text
implemented
reserved
disabled
deprecated
```

含义：

```text
implemented:
  当前已实现完整链路。

reserved:
  长期预留，当前不可执行完整链路。

disabled:
  临时关闭。

deprecated:
  已废弃，但可能兼容读取。
```

## 4. MVP 规则

MVP 阶段：

```text
code:
  implemented = true
  allowedInMvp = true

其他类型:
  implemented = false
  status = reserved
  allowedInMvp = false
```

## 5. 注册项建议

```ts
type ArtifactTypeRegistration = {
  type: ArtifactType;
  description: string;
  implemented: boolean;
  status: "implemented" | "reserved" | "disabled" | "deprecated";
  allowedInMvp: boolean;
  allowInlineContent: boolean;
  requiresContentRef: boolean;
  requiresObjectStorage: boolean;
  metadataSchema: JsonSchema;
};
```

## 6. 禁止事项

不得：

- 使用未注册 Artifact type。
- 把 toolName 当作 Artifact type。
- 把 React Component 名称当作 Artifact type。
- MVP 阶段启用非 code Artifact。
- implemented=false 的 Artifact 进入完整预览链路。
