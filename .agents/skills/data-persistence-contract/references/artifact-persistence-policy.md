# Artifact 持久化规则

## 目的

Artifact 是 AgentHub 的产物事实源，必须能被历史回放、调试、预览和追踪。

## artifacts 表字段

```text
id
conversation_id
message_id
run_id
agent_name
type
title
mime_type
content
content_ref
metadata
version
status
created_at
updated_at
deleted_at
```

## content 与 content_ref

- 小型文本内容可使用 `content`。
- 大型内容、二进制内容、下载内容必须使用 `content_ref`。
- `content_ref` 必须是后端可控引用，不是永久公开 URL。

## v1.0 类型

至少应支持持久化：

```text
code
webpage
markdown
```

可规划：

```text
document
data
image
zip
log
```

## 关联规则

- Artifact 必须关联 conversation。
- Artifact 应关联 message。
- Artifact 应关联 run。
- 多 Agent 场景下 Artifact 应保存 agent_name。
- 重名 title 不得造成覆盖。

## 禁止

- 产物只存在实时事件中。
- 大文件直接塞数据库。
- 永久公开下载 URL 入库。
- secret、系统 prompt、内部路径入库。
