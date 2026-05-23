# Task handler 契约

## 1. 目的

本文定义 AgentHub 子 Agent Task handler 的输入输出规则。

Task handler 是子 Agent 的核心业务入口。

## 2. 推荐签名

推荐抽象：

```text
HandleTask(ctx, task) error
```

## 3. handler 负责

Task handler 负责：

- 读取用户消息。
- 读取任务上下文。
- 调用 LLM 或工具。
- 通过 `ctx.StreamText` 输出文本。
- 通过 `ctx.AddArtifact` 添加 Artifact。
- 尊重取消信号。
- 返回错误。
- 不泄漏敏感信息。

## 4. handler 不负责

Task handler 不负责：

- 选择哪个 Agent 执行。
- 生成 ExecutionPlan。
- 定义 AG-UI 事件。
- 渲染前端组件。
- 定义 Artifact 完整 schema。
- 持久化大型 Artifact。
- 管理数据库 DDL。

## 5. MVP code-agent handler

MVP `code-agent` handler 应执行：

```text
读取用户消息
→ 构造 LLM 请求
→ 流式接收 LLM 文本
→ ctx.StreamText(chunk)
→ 收集完整回复
→ 解析代码块
→ ctx.AddArtifact(type=code)
```

## 6. 取消和超时

handler 必须尊重 context cancellation。

长时间任务必须可取消。

工具调用必须有 timeout。

## 7. 禁止事项

不得：

- 忽略取消信号。
- 把大内容塞进 `ctx.StreamText`。
- 直接写 AG-UI 事件。
- 直接调用前端 Runtime Skill。
- 在错误中暴露 stack trace 或 secret。
