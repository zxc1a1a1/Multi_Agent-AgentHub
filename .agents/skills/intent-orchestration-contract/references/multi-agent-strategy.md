# 多 Agent 编排策略

## 1. 目的

本文定义 `single`、`parallel`、`sequential` 的编排规则。

MVP 阶段只实现 `single`。

## 2. single

`single` 表示只执行一个目标 Agent 任务。

MVP 阶段只允许：

```text
strategy = single
```

规则：

- tasks 长度应为 1。
- task.agent 必须存在。
- task.skill 必须存在。
- 执行结果直接作为 run 结果。

## 3. parallel

`parallel` 表示多个任务并行执行。

正式开发阶段启用前，必须定义：

- 并发限制。
- 每个 task 的 timeout。
- 聚合规则。
- 部分失败策略。
- Artifact 合并规则。
- trace 规则。

## 4. sequential

`sequential` 表示任务按依赖顺序执行。

正式开发阶段启用前，必须定义：

- `dependsOn`。
- 依赖图校验。
- 循环依赖检测。
- 中间结果传递规则。
- 下游失败策略。
- 取消策略。

## 5. 聚合规则

多 Agent 阶段必须定义聚合策略，例如：

- preserve_order。
- summarize。
- merge_artifacts。
- select_best。
- user_choose。

## 6. 禁止事项

不得：

- 在 MVP 阶段启用 parallel。
- 在 MVP 阶段启用 sequential。
- 执行存在循环依赖的 plan。
- 忽略 parallel 部分失败。
- 忽略 sequential 依赖失败。
- 未定义聚合规则就合并多 Agent 输出。
