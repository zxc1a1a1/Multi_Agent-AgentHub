# Volume / Network 策略

## 1. 目的

本文定义 Compose volume 和 network 规则。

## 2. MVP volume

MVP 至少需要：

```text
mysqldata
```

用于 MySQL 数据持久化。

## 3. reset

必须提供清理 volume 的命令：

```text
make docker-reset
```

语义：

```text
docker compose down -v
```

## 4. network

默认 Compose network 即可满足 MVP。

服务之间通过 service name 访问：

```text
mysql
gateway
code-agent
frontend
```

## 5. 禁止事项

不得：

- 硬编码宿主机绝对路径。
- 没有清理 volume 的命令。
- 将数据库数据写入 image。
- 使用 localhost 连接其他容器。
- 在 volume 中存放 secret。
