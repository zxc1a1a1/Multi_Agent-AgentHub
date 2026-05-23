# 流式输出规则

## 1. 目的

本文定义 `ctx.StreamText` 的使用边界。

流式文本用于用户可读的实时回复，不是 Artifact 事实源。

## 2. 允许输出

`ctx.StreamText` 可以输出：

- Agent 回复文本。
- 任务进度说明。
- 代码解释。
- 用户可见状态。
- 简短提示。

## 3. 不允许输出

`ctx.StreamText` 不得输出：

- 大型代码包。
- 图片内容。
- zip 内容。
- 大型日志。
- 私有文件内容。
- API key。
- access token。
- refresh token。
- system prompt。
- stack trace。
- 内部服务地址。
- 对象存储私有地址。
- 数据库连接字符串。

## 4. 与 Artifact 的关系

如果内容是任务产物，应使用：

```text
ctx.AddArtifact
```

而不是：

```text
ctx.StreamText
```

MVP 中，代码解释可以 stream，代码产物应通过 `code` Artifact 表达。

## 5. 禁止事项

不得：

- 用 `ctx.StreamText` 代替 Artifact。
- 用 `ctx.StreamText` 传输大型内容。
- 用 `ctx.StreamText` 传输 secret。
- 用 `ctx.StreamText` 传输内部错误详情。
