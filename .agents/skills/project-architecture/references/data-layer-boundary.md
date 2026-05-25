# Data Layer Boundary

本 Skill 只定义数据职责，不固定数据库产品。

## Primary Relational Database

可承载：

- users。
- conversations。
- participants。
- messages。
- agents。
- runs。
- artifacts。
- tool calls。

## Cache / Ephemeral Store

可承载：

- run transient state。
- health cache。
- online state。
- short-lived stream state。

## Object Storage

可承载：

- large artifacts。
- files。
- generated packages。
- images。
- documents。

## 禁止

- 将大产物长期塞入普通 message content。
- 依赖单进程内存保存跨请求关键状态。
- 在架构层固化 MySQL / PostgreSQL / Redis 为唯一长期实现。
