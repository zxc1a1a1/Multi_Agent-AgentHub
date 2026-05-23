# MVP code_preview 规则

## 1. 目的

本文定义 MVP 阶段唯一实现的前端 Runtime Skill：`code_preview`。

## 2. 定位

`code_preview` 用于以只读代码预览卡片展示代码内容。

MVP 映射：

```text
toolName = code_preview
component = CodePreview
```

注意：

```text
artifact.type = code → code_preview
```

这个映射由 `artifact-contract` 负责。

本文件只定义 `code_preview` 接收参数后如何在前端展示。

## 3. 参数

```ts
type CodePreviewParams = {
  code: string;
  language: string;
  filename: string;
};
```

JSON Schema 要求：

```text
required = ["code", "language", "filename"]
additionalProperties = false
```

## 4. 注册项

```json
{
  "name": "code_preview",
  "description": "Render read-only code content.",
  "implemented": true,
  "status": "implemented",
  "component": "CodePreview",
  "behavior": "render",
  "blocking": false,
  "requiresConfirmation": false,
  "dangerous": false,
  "allowedInMvp": true,
  "failureMode": "fallback_card"
}
```

## 5. UI 行为

CodePreview 应：

- 显示 filename。
- 显示 language。
- 显示 code。
- 支持复制代码。
- 以只读方式展示。
- 参数非法时显示 fallback。

## 6. 安全规则

`code_preview` 不得：

- 执行代码。
- eval 代码。
- 将代码插入 script。
- 修改服务端状态。
- 访问外部网络。
- 读取本地文件。
- 自动下载文件。
- 触发部署。
- 写入对象存储。

## 7. 错误处理

参数错误时：

```text
不渲染 CodePreview
显示 fallback card
记录 toolCallId / runId / errorCode
不让聊天 UI 崩溃
```

## 8. 禁止事项

不得：

- 把 code_preview 变成代码执行器。
- 把 code_preview 变成文件编辑器。
- 把 code_preview 变成部署按钮。
- 让 copy 行为影响 Agent run 完成状态。
