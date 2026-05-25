# Artifact Lifecycle Policy

## 状态

Artifact 生命周期状态：

```text
pending
normalizing
ready
failed
superseded
deleted
```

## 状态含义

| 状态 | 含义 | 是否可预览 |
|---|---|---|
| pending | 已创建，内容未完成 | 否 |
| normalizing | 正在标准化或校验 | 否 |
| ready | 可用 | 是 |
| failed | 生成或标准化失败 | 否 |
| superseded | 已被新版本替代 | 可按历史策略 |
| deleted | 已删除或不可用 | 否 |

## 规则

- 只有 ready 可预览。
- failed 不能被展示为成功产物。
- deleted 应保留 tombstone，避免历史消息断链。
- superseded 不等于 deleted。
- 任何状态变更应可追踪。
