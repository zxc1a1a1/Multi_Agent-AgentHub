# Versioning Policy

## 规则

Artifact 必须支持版本化。

- version 从 1 开始。
- 新内容不得覆盖旧版本。
- 小修订可以增加 version。
- 独立生成可以创建新 artifactId。
- 历史消息引用必须稳定。

## superseded

当旧版本被新版本替代时，可以设置：

```json
{
  "status": "superseded",
  "supersededBy": {
    "artifactId": "art_001",
    "version": 2
  }
}
```

## 禁止

- 原地覆盖 content。
- 删除旧版本导致历史消息打不开。
- 用 title 区分版本。
