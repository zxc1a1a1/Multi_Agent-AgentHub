# Component Binding Policy

## 绑定方式

前端通过 Registry 将 `toolName` 绑定到组件或执行器。

外部协议不得直接传 React 组件名。

## 规则

- `toolName` 是外部稳定标识，由 `preview.previewType` 决定。
- `previewType` 应优先等于 Runtime `toolName`。
- `artifact.type` 不是 `toolName`。
- `outputMode` 不是 `toolName`。
- `component` 是前端内部实现细节。
- 缺失组件必须 fallback。
- 组件抛错必须被 Error Boundary 或等价机制隔离。
- 组件不得依赖 Agent 名称判断渲染分支。
- 不得通过 `agentName` 推断 outputMode 或选择 toolName。

### 三层映射链

```text
outputMode → artifact.type → previewType → toolName → component
```

前端只关心 `toolName → component` 这一段。`outputMode`、`artifact.type`、`previewType` 由后端契约链决定，不得在前端直接绑定。

## 示例

```ts
registry['code_preview'] = {
  component: 'CodePreview',
  behavior: 'render',
  riskLevel: 'low',
  failureMode: 'error_card',
}
```
