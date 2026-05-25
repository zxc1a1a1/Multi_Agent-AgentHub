# Strategy 规则

支持三种策略：

| strategy | 含义 |
|---|---|
| `single` | 一个 task |
| `ordered_parallel` | 多个独立 task，输出按稳定顺序聚合 |
| `sequential` | 多个有依赖 task |

## 兼容

历史 `parallel` 语义应迁移为 `ordered_parallel`。

## 禁止

- 没有 message/task 隔离时做 token 级交错输出。
- sequential 忽略 dependsOn。
- strategy 不在枚举内仍执行。
