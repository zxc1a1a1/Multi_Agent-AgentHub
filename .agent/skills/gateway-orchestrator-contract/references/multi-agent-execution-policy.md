# 多 Agent 执行规则

本契约支持任意 2+ Agent，不固定具体 Agent 名称。

## 策略

```text
single
ordered_parallel
sequential
```

## single

一个 run 只有一个 task。

## ordered_parallel

一个 run 可以有多个独立 task。

规则：

- UI 可以展示多个 Agent 参与。
- Orchestrator 可以内部并发或顺序执行。
- Gateway 接收的事件必须可稳定聚合。
- 每个 Agent 输出必须有独立 messageId。
- 不要求 token 级交错。

## sequential

多个 task 存在依赖关系。

规则：

- 后续 task 只能使用前序 task 的脱敏摘要或引用。
- 前序失败时必须根据 fallback 策略决定是否继续。

## 群聊

- conversationType 可以是 group。
- mentions 只是路由提示。
- Orchestrator 必须校验 mentions。
- fallback 后必须记录实际执行 Agent。
